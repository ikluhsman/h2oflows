package poller

import (
	"context"
	"log"
	"sync"
	"time"

	gauge "github.com/h2oflow/h2oflow/packages/gauge-core"
	"github.com/jackc/pgx/v5/pgxpool"
)

// readingRetention is how long gauge readings are kept in the DB.
// gauge_readings is a rolling cache — historical graphs proxy to source APIs directly.
const readingRetention = 48 * time.Hour

// pollConcurrency is the max number of parallel FetchReading calls per source.
// Keeps us from hammering USGS or DWR with hundreds of simultaneous requests.
const pollConcurrency = 10

// consecutiveFailuresThreshold is the number of consecutive poll failures before
// a gauge is automatically marked 'inactive' by the poller.
const consecutiveFailuresThreshold = 5

// Poller fetches gauge readings on a schedule and writes them to the database.
// Each GaugeSource runs on its own ticker so sources with different poll
// intervals don't block each other.
type Poller struct {
	db      *pgxpool.Pool
	sources []sourceConfig
}

type sourceConfig struct {
	source   gauge.GaugeSource
	interval time.Duration
}

func New(db *pgxpool.Pool) *Poller {
	return &Poller{db: db}
}

// Register adds a GaugeSource to the poller with the given poll interval.
func (p *Poller) Register(source gauge.GaugeSource, interval time.Duration) {
	p.sources = append(p.sources, sourceConfig{source: source, interval: interval})
}

// Run starts all registered source pollers and the reading pruner.
// Blocks until ctx is cancelled.
func (p *Poller) Run(ctx context.Context) {
	// Sync gauge metadata (location, HUC, basin, watershed) on startup.
	// Runs in background — doesn't block the first poll cycle.
	go p.syncAllMetadata(ctx)

	// One goroutine per source, plus the pruner.
	for _, sc := range p.sources {
		go p.runSource(ctx, sc)
	}
	go p.runPruner(ctx)
	<-ctx.Done()
}

// runSource polls one GaugeSource on its configured interval.
func (p *Poller) runSource(ctx context.Context, sc sourceConfig) {
	// Poll immediately on startup, then on each tick.
	p.pollSource(ctx, sc)

	ticker := time.NewTicker(sc.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.pollSource(ctx, sc)
		}
	}
}

// pollSource loads all active gauges for this source, fetches their current
// reading in parallel, and writes the results to gauge_readings.
func (p *Poller) pollSource(ctx context.Context, sc sourceConfig) {
	sourceType := string(sc.source.SourceType())
	gauges, err := p.loadGauges(ctx, sourceType)
	if err != nil {
		log.Printf("poller[%s]: load gauges: %v", sourceType, err)
		return
	}
	if len(gauges) == 0 {
		return
	}

	log.Printf("poller[%s]: polling %d gauges", sourceType, len(gauges))

	// Bounded concurrency — a semaphore channel.
	sem := make(chan struct{}, pollConcurrency)
	var wg sync.WaitGroup

	for _, g := range gauges {
		wg.Add(1)
		sem <- struct{}{}
		go func(g dbGauge) {
			defer wg.Done()
			defer func() { <-sem }()
			p.fetchAndStore(ctx, sc.source, g)
		}(g)
	}
	wg.Wait()
}

// fetchAndStore fetches one reading and writes it to the DB.
func (p *Poller) fetchAndStore(ctx context.Context, src gauge.GaugeSource, g dbGauge) {
	reading, err := src.FetchReading(ctx, g.externalID)
	if err != nil {
		p.recordFailure(ctx, g.id, err)
		return
	}

	if err := p.writeReading(ctx, g.id, *reading); err != nil {
		log.Printf("poller: write reading for %s/%s: %v", src.Name(), g.externalID, err)
		return
	}
	p.recordSuccess(ctx, g.id)
}

// writeReading inserts a reading into gauge_readings and updates the
// denormalized current_cfs on the gauges row so search results reflect
// live data without a join.
// ON CONFLICT DO NOTHING: the poller may overlap ticks and the source API
// may return the same timestamped value twice — that's fine, ignore it.
func (p *Poller) writeReading(ctx context.Context, gaugeID string, r gauge.Reading) error {
	_, err := p.db.Exec(ctx, `
		INSERT INTO gauge_readings (gauge_id, value, unit, timestamp, qual_code, provisional)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (gauge_id, timestamp) DO NOTHING
	`, gaugeID, r.Value, string(r.Unit), r.Timestamp, r.QualCode, r.Provisional)
	if err != nil {
		return err
	}
	// Keep gauges.current_cfs in sync — only advance forward in time.
	_, err = p.db.Exec(ctx, `
		UPDATE gauges
		SET current_cfs = $2
		WHERE id = $1
		  AND (last_reading_at IS NULL OR last_reading_at <= $3)
	`, gaugeID, r.Value, r.Timestamp)
	return err
}

// recordSuccess resets consecutive_failures and updates last_reading_at.
// If the gauge was auto-managed to inactive due to failures, restore it to active.
func (p *Poller) recordSuccess(ctx context.Context, gaugeID string) {
	_, err := p.db.Exec(ctx, `
		UPDATE gauges
		SET last_reading_at      = NOW(),
		    consecutive_failures = 0,
		    status               = CASE
		        WHEN auto_managed = TRUE AND status = 'inactive' THEN 'active'
		        ELSE status
		    END
		WHERE id = $1
	`, gaugeID)
	if err != nil {
		log.Printf("poller: record success for %s: %v", gaugeID, err)
	}
}

// recordFailure increments consecutive_failures. If the gauge is auto-managed
// and crosses the threshold, it is marked inactive automatically.
func (p *Poller) recordFailure(ctx context.Context, gaugeID string, fetchErr error) {
	log.Printf("poller: fetch failed for gauge %s: %v", gaugeID, fetchErr)
	_, err := p.db.Exec(ctx, `
		UPDATE gauges
		SET consecutive_failures = consecutive_failures + 1,
		    status               = CASE
		        WHEN auto_managed = TRUE
		             AND consecutive_failures + 1 >= $2
		             AND status NOT IN ('retired', 'seasonal')
		        THEN 'inactive'
		        ELSE status
		    END
		WHERE id = $1
	`, gaugeID, consecutiveFailuresThreshold)
	if err != nil {
		log.Printf("poller: record failure for %s: %v", gaugeID, err)
	}
}

// runPruner deletes readings older than readingRetention once per day.
// gauge_readings is a rolling cache — we don't store history, source APIs do.
func (p *Poller) runPruner(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.pruneOldReadings(ctx)
		}
	}
}

func (p *Poller) pruneOldReadings(ctx context.Context) {
	cutoff := time.Now().Add(-readingRetention)
	tag, err := p.db.Exec(ctx, `DELETE FROM gauge_readings WHERE timestamp < $1`, cutoff)
	if err != nil {
		log.Printf("poller: prune readings: %v", err)
		return
	}
	if tag.RowsAffected() > 0 {
		log.Printf("poller: pruned %d readings older than %s", tag.RowsAffected(), readingRetention)
	}
}

// TouchRequested marks a gauge as recently requested, bringing it into the
// demand-driven poll window. Call this whenever the API serves gauge data to
// a user — search results, detail page, watchlist load.
// Fire-and-forget: errors are logged but not returned to the caller.
func (p *Poller) TouchRequested(ctx context.Context, gaugeID string) {
	_, err := p.db.Exec(ctx,
		`UPDATE gauges SET last_requested_at = NOW() WHERE id = $1 AND featured = FALSE`,
		gaugeID,
	)
	if err != nil {
		log.Printf("poller: touch requested for %s: %v", gaugeID, err)
	}
}

// --- Metadata sync ----------------------------------------------------------

// syncAllMetadata fetches site metadata (location, HUC, state) from USGS for
// any gauges that are missing it. Runs once on startup. Safe to re-run.
func (p *Poller) syncAllMetadata(ctx context.Context) {
	for _, sc := range p.sources {
		discoverer, ok := sc.source.(gauge.SiteDiscoverer)
		if !ok {
			continue
		}
		sourceType := string(sc.source.SourceType())
		if err := p.syncSourceMetadata(ctx, sourceType, discoverer); err != nil {
			log.Printf("poller: metadata sync [%s]: %v", sourceType, err)
		}
	}
}

// syncSourceMetadata fetches and stores metadata for gauges of one source
// that are missing location or HUC data.
func (p *Poller) syncSourceMetadata(ctx context.Context, sourceType string, discoverer gauge.SiteDiscoverer) error {
	// Load gauges that are missing location or basin context.
	rows, err := p.db.Query(ctx, `
		SELECT id, external_id
		FROM   gauges
		WHERE  source = $1
		  AND  status != 'retired'
		  AND  (location IS NULL OR huc8 IS NULL OR basin_name IS NULL OR state_abbr IS NULL OR state_abbr = '')
		LIMIT  500
	`, sourceType)
	if err != nil {
		return err
	}
	defer rows.Close()

	var targets []dbGauge
	for rows.Next() {
		var g dbGauge
		if err := rows.Scan(&g.id, &g.externalID); err != nil {
			return err
		}
		targets = append(targets, g)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(targets) == 0 {
		return nil
	}

	log.Printf("poller: syncing metadata for %d %s gauges", len(targets), sourceType)

	// Fetch in batches of 100 (USGS site service limit per request).
	const batchSize = 100
	for i := 0; i < len(targets); i += batchSize {
		end := i + batchSize
		if end > len(targets) {
			end = len(targets)
		}
		batch := targets[i:end]

		ids := make([]string, len(batch))
		for j, g := range batch {
			ids[j] = g.externalID
		}

		sites, err := discoverer.DiscoverSites(ctx, gauge.DiscoverOptions{
			SiteIDs:    ids,
			ActiveOnly: false,
		})
		if err != nil {
			log.Printf("poller: discover sites batch: %v", err)
			continue
		}

		// Build a lookup from external_id → site metadata.
		siteByID := make(map[string]*gauge.SiteMetadata, len(sites))
		for _, s := range sites {
			siteByID[s.ExternalID] = s
		}

		for _, g := range batch {
			site, ok := siteByID[g.externalID]
			if !ok {
				continue
			}
			p.applyMetadata(ctx, g.id, site)
		}
	}
	return nil
}

// applyMetadata writes discovered site metadata to the gauges table.
func (p *Poller) applyMetadata(ctx context.Context, gaugeID string, site *gauge.SiteMetadata) {
	basinName, watershedName := gauge.HUCNames(site.HUCCode)

	var locExpr string
	var args []any

	if site.Location != nil {
		locExpr = "location = ST_MakePoint($2, $3)::geography,"
		args = []any{
			gaugeID,
			site.Location.Lng, site.Location.Lat,
			site.HUCCode,
			site.StateCode,
			basinName,
			watershedName,
		}
	} else {
		locExpr = ""
		args = []any{
			gaugeID,
			site.HUCCode,
			site.StateCode,
			basinName,
			watershedName,
		}
	}

	var sql string
	if site.Location != nil {
		sql = `
			UPDATE gauges SET
				` + locExpr + `
				huc8           = $4,
				state_abbr     = $5,
				basin_name     = $6,
				watershed_name = COALESCE(watershed_name, $7)
			WHERE id = $1
		`
	} else {
		sql = `
			UPDATE gauges SET
				huc8           = $2,
				state_abbr     = $3,
				basin_name     = $4,
				watershed_name = COALESCE(watershed_name, $5)
			WHERE id = $1
		`
	}

	if _, err := p.db.Exec(ctx, sql, args...); err != nil {
		log.Printf("poller: apply metadata for %s: %v", gaugeID, err)
	}
}

// --- DB helpers -------------------------------------------------------------

type dbGauge struct {
	id         string
	externalID string
}

// demandWindow is how long a non-featured gauge stays in the poll set after
// its last user request. After this period with no activity it is silently
// dropped — the source API remains the source of truth for historical data.
const demandWindow = 7 * 24 * time.Hour

// loadGauges returns gauges for the given source that should be polled this tick.
//
// A gauge is included if:
//   - It is featured (always polled — these are the curated backbone of the app), OR
//   - It was actively requested by a user within the demand window
//
// This keeps the poll set small. USGS has ~10,000 gauges in Colorado alone;
// we have no business polling gauges that nobody is looking at.
func (p *Poller) loadGauges(ctx context.Context, sourceName string) ([]dbGauge, error) {
	rows, err := p.db.Query(ctx, `
		SELECT id, external_id
		FROM   gauges
		WHERE  source = $1
		  AND  status IN ('active', 'seasonal', 'maintenance')
		  AND  (
		      -- seasonal: only poll when within the season window
		      status != 'seasonal'
		      OR seasonal_start_mmdd IS NULL
		      OR TO_CHAR(NOW(), 'MM-DD') BETWEEN seasonal_start_mmdd AND seasonal_end_mmdd
		  )
		  AND (
		      featured = TRUE
		      OR last_requested_at > NOW() - $2::interval
		  )
	`, sourceName, demandWindow)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []dbGauge
	for rows.Next() {
		var g dbGauge
		if err := rows.Scan(&g.id, &g.externalID); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}
