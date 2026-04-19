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
// 7 days gives the graph enough history for the 12/24/48h windows plus context.
const readingRetention = 7 * 24 * time.Hour

// backfillWindow is how far back to seed readings for a gauge that has none.
const backfillWindow = 7 * 24 * time.Hour

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

	// Backfill 7 days of history for any gauges with no recent readings.
	go p.backfillAll(ctx)

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
	// Keep gauges.current_cfs and flow_status in sync — only advance forward in time.
	_, err = p.db.Exec(ctx, `
		UPDATE gauges
		SET current_cfs = $2,
		    flow_status  = COALESCE(
		        (SELECT CASE
		                    WHEN fr.label IN ('running','high') THEN 'runnable'
		                    WHEN fr.label = 'too_low'           THEN 'caution'
		                    WHEN fr.label = 'very_high'         THEN 'flood'
		                    ELSE 'unknown'
		                END
		         FROM flow_ranges fr
		         WHERE fr.gauge_id = $1
		           AND fr.craft_type = 'general'
		           AND (fr.min_cfs IS NULL OR $2 >= fr.min_cfs)
		           AND (fr.max_cfs IS NULL OR $2 <  fr.max_cfs)
		         ORDER BY fr.min_cfs ASC NULLS FIRST
		         LIMIT 1),
		        'unknown'
		    )
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
//
// Reach-linked gauges are skipped: they're already polled every cycle, so
// touching them just churns last_requested_at without changing behaviour.
func (p *Poller) TouchRequested(ctx context.Context, gaugeID string) {
	_, err := p.db.Exec(ctx,
		`UPDATE gauges SET last_requested_at = NOW() WHERE id = $1 AND reach_id IS NULL`,
		gaugeID,
	)
	if err != nil {
		log.Printf("poller: touch requested for %s: %v", gaugeID, err)
	}
}

// FetchNowIfStale fetches a single reading synchronously from the upstream
// source if the gauge's most recent reading is older than maxAge (or absent).
// Returns true if a fresh reading was written.
//
// Used by handlers serving reach detail pages so the user sees current data
// on first view rather than waiting for the next poll tick. Bounded by a
// short timeout so a hung upstream API never blocks the page response.
func (p *Poller) FetchNowIfStale(ctx context.Context, gaugeID string, maxAge time.Duration) bool {
	// Look up source + external_id + last reading freshness in one shot.
	var (
		sourceType   string
		externalID   string
		lastReadingAt *time.Time
	)
	err := p.db.QueryRow(ctx, `
		SELECT g.source, g.external_id, g.last_reading_at
		FROM   gauges g
		WHERE  g.id = $1
	`, gaugeID).Scan(&sourceType, &externalID, &lastReadingAt)
	if err != nil {
		return false
	}
	if lastReadingAt != nil && time.Since(*lastReadingAt) < maxAge {
		return false
	}

	// Find the registered source for this gauge.
	var src gauge.GaugeSource
	for _, sc := range p.sources {
		if string(sc.source.SourceType()) == sourceType {
			src = sc.source
			break
		}
	}
	if src == nil {
		return false
	}

	// Bound the upstream call so a slow USGS response can't stall the handler.
	fetchCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	reading, err := src.FetchReading(fetchCtx, externalID)
	if err != nil {
		p.recordFailure(ctx, gaugeID, err)
		return false
	}
	if err := p.writeReading(ctx, gaugeID, *reading); err != nil {
		log.Printf("poller: on-demand write for %s: %v", gaugeID, err)
		return false
	}
	p.recordSuccess(ctx, gaugeID)
	return true
}

// --- Historical backfill ----------------------------------------------------

// backfillAll seeds readings for gauges with no data or with gaps in their
// recent history. Runs once on startup in the background.
// Safe to re-run — ON CONFLICT DO NOTHING skips already-stored readings.
func (p *Poller) backfillAll(ctx context.Context) {
	for _, sc := range p.sources {
		p.backfillSource(ctx, sc)
	}
}

type backfillTarget struct {
	dbGauge
	since time.Time
}

func (p *Poller) backfillSource(ctx context.Context, sc sourceConfig) {
	sourceType := string(sc.source.SourceType())

	// Find gauges that either:
	//   (a) have no readings in the last 7 days, OR
	//   (b) have a gap > 2 hours anywhere in the last 7 days
	// For (b) we fetch from just before the earliest gap so we only pull what's missing.
	rows, err := p.db.Query(ctx, `
		WITH window_readings AS (
		    SELECT gauge_id, timestamp,
		           LEAD(timestamp) OVER (PARTITION BY gauge_id ORDER BY timestamp) AS next_ts
		    FROM   gauge_readings
		    WHERE  timestamp > NOW() - INTERVAL '7 days'
		),
		earliest_gaps AS (
		    -- Mid-stream gap: two consecutive readings more than 2 h apart
		    SELECT gauge_id, MIN(timestamp) - INTERVAL '5 minutes' AS since
		    FROM   window_readings
		    WHERE  EXTRACT(EPOCH FROM (next_ts - timestamp)) > 7200
		    GROUP  BY gauge_id
		),
		trailing_gap AS (
		    -- Trailing gap: most recent reading was > 2 h ago (e.g. backend was down)
		    -- LEAD returns NULL for the last row so the earliest_gaps CTE misses this case.
		    SELECT gauge_id, MAX(timestamp) - INTERVAL '5 minutes' AS since
		    FROM   gauge_readings
		    WHERE  timestamp > NOW() - INTERVAL '7 days'
		    GROUP  BY gauge_id
		    HAVING MAX(timestamp) < NOW() - INTERVAL '2 hours'
		)
		SELECT g.id, g.external_id,
		       COALESCE(
		           LEAST(eg.since, tg.since),
		           eg.since,
		           tg.since,
		           NOW() - INTERVAL '7 days'
		       ) AS fetch_since
		FROM   gauges g
		LEFT   JOIN earliest_gaps eg ON eg.gauge_id = g.id
		LEFT   JOIN trailing_gap  tg ON tg.gauge_id = g.id
		WHERE  g.source = $1
		  AND  g.status NOT IN ('retired', 'inactive')
		  AND  (
		           g.reach_id IS NOT NULL
		           OR g.last_requested_at > NOW() - $2::interval
		       )
		  AND  (
		           -- no readings at all in the window
		           NOT EXISTS (
		               SELECT 1 FROM gauge_readings gr
		               WHERE  gr.gauge_id = g.id
		                 AND  gr.timestamp > NOW() - INTERVAL '7 days'
		               LIMIT 1
		           )
		           OR eg.gauge_id IS NOT NULL
		           OR tg.gauge_id IS NOT NULL
		       )
	`, sourceType, demandWindow)
	if err != nil {
		log.Printf("poller: backfill query [%s]: %v", sourceType, err)
		return
	}
	defer rows.Close()

	var targets []backfillTarget
	for rows.Next() {
		var t backfillTarget
		if err := rows.Scan(&t.id, &t.externalID, &t.since); err != nil {
			continue
		}
		targets = append(targets, t)
	}
	if err := rows.Err(); err != nil {
		return
	}
	if len(targets) == 0 {
		return
	}

	log.Printf("poller: backfilling %d %s gauges", len(targets), sourceType)

	for _, t := range targets {
		select {
		case <-ctx.Done():
			return
		default:
		}

		readings, err := sc.source.FetchHistory(ctx, t.externalID, t.since)
		if err != nil {
			log.Printf("poller: backfill fetch %s/%s: %v", sourceType, t.externalID, err)
			continue
		}
		if len(readings) == 0 {
			continue
		}

		if err := p.bulkWriteReadings(ctx, t.id, readings); err != nil {
			log.Printf("poller: backfill write %s/%s: %v", sourceType, t.externalID, err)
		} else {
			log.Printf("poller: backfilled %d readings for %s/%s", len(readings), sourceType, t.externalID)
		}

		// Small pause to avoid hammering the source API.
		time.Sleep(200 * time.Millisecond)
	}
}

// bulkWriteReadings inserts a batch of historical readings in a single transaction
// and updates current_cfs / flow_status from the most recent one.
// Readings must be oldest-first (as returned by FetchHistory).
func (p *Poller) bulkWriteReadings(ctx context.Context, gaugeID string, readings []*gauge.Reading) error {
	if len(readings) == 0 {
		return nil
	}

	tx, err := p.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, r := range readings {
		if _, err := tx.Exec(ctx, `
			INSERT INTO gauge_readings (gauge_id, value, unit, timestamp, qual_code, provisional)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (gauge_id, timestamp) DO NOTHING
		`, gaugeID, r.Value, string(r.Unit), r.Timestamp, r.QualCode, r.Provisional); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	// Update current_cfs with the most recent reading (oldest-first slice → last element).
	last := readings[len(readings)-1]
	return p.writeReading(ctx, gaugeID, *last)
}

// --- Metadata sync ----------------------------------------------------------

// SyncMetadataNow runs the metadata sync immediately for all sources.
// Safe to call at any time — skips gauges that already have complete metadata.
// Used by the import handler so newly imported gauges get basin/location data
// without waiting for the next server restart.
func (p *Poller) SyncMetadataNow(ctx context.Context) {
	p.syncAllMetadata(ctx)
}

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

	// Bulk-propagate basin from gauges → rivers for any river still missing it.
	// Runs after all source syncs so it catches:
	//   (a) gauges just synced above, and
	//   (b) gauges already fully synced whose river was recently re-created.
	p.propagateRiverBasins(ctx)
}

// propagateRiverBasins fills rivers.basin for any river that is missing it by
// joining through reaches to the primary gauge's watershed_name. Safe to run
// repeatedly — only touches rivers where basin IS NULL.
func (p *Poller) propagateRiverBasins(ctx context.Context) {
	tag, err := p.db.Exec(ctx, `
		UPDATE rivers rv
		SET    basin = g.watershed_name
		FROM   reaches re
		JOIN   gauges  g ON g.id = re.primary_gauge_id
		WHERE  re.river_id        = rv.id
		  AND  rv.basin           IS NULL
		  AND  g.watershed_name   IS NOT NULL
	`)
	if err != nil {
		log.Printf("poller: propagate river basins: %v", err)
		return
	}
	if tag.RowsAffected() > 0 {
		log.Printf("poller: set basin on %d river(s) from gauge watershed data", tag.RowsAffected())
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
		  AND  (location IS NULL OR huc8 IS NULL OR basin_name IS NULL OR state_abbr IS NULL OR state_abbr = '' OR elevation_ft IS NULL OR watershed_name IS NULL)
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
// watershed_name is always set to the canonical basin label (e.g. "South Platte",
// "Arkansas", "Colorado") which is source-agnostic and used by the UI for grouping.
// basin_name retains the raw HUC2-derived name from USGS where available.
func (p *Poller) applyMetadata(ctx context.Context, gaugeID string, site *gauge.SiteMetadata) {
	basinName, _ := gauge.HUCNames(site.HUCCode) // HUC2 name for basin_name column only

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
			site.CanonicalBasin,
			site.ElevationFt, // nullable
		}
	} else {
		args = []any{
			gaugeID,
			site.HUCCode,
			site.StateCode,
			basinName,
			site.CanonicalBasin,
			site.ElevationFt, // nullable
		}
	}

	var sql string
	if site.Location != nil {
		sql = `
			UPDATE gauges SET
				` + locExpr + `
				huc8           = $4,
				state_abbr     = $5,
				basin_name     = COALESCE(NULLIF($6, ''), basin_name),
				watershed_name = NULLIF($7, ''),
				elevation_ft   = COALESCE(elevation_ft, $8)
			WHERE id = $1
		`
	} else {
		sql = `
			UPDATE gauges SET
				huc8           = $2,
				state_abbr     = $3,
				basin_name     = COALESCE(NULLIF($4, ''), basin_name),
				watershed_name = NULLIF($5, ''),
				elevation_ft   = COALESCE(elevation_ft, $6)
			WHERE id = $1
		`
	}

	if _, err := p.db.Exec(ctx, sql, args...); err != nil {
		log.Printf("poller: apply metadata for %s: %v", gaugeID, err)
	}

	// Propagate canonical basin to any river whose reaches use this gauge as
	// primary. Skips rivers that already have an explicit basin set.
	if site.CanonicalBasin != "" {
		if _, err := p.db.Exec(ctx, `
			UPDATE rivers rv
			SET    basin = $2
			FROM   reaches re
			WHERE  re.river_id         = rv.id
			  AND  re.primary_gauge_id = $1
			  AND  rv.basin            IS NULL
		`, gaugeID, site.CanonicalBasin); err != nil {
			log.Printf("poller: propagate basin to river for gauge %s: %v", gaugeID, err)
		}
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
//   - It is associated with a reach (always polled — these are the load-bearing
//     gauges that back reach pages), OR
//   - It was actively requested by a user within the demand window
//
// This keeps the poll set small. USGS has ~10,000 gauges in Colorado alone;
// we have no business polling gauges that nobody is looking at and that don't
// belong to any reach.
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
		      reach_id IS NOT NULL
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
