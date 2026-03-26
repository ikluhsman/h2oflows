package poller

import (
	"context"
	"log"
	"time"

	gauge "github.com/h2oflow/h2oflow/packages/gauge-core"
	"github.com/jackc/pgx/v5/pgxpool"
)

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

// Run starts all registered source pollers. Blocks until ctx is cancelled.
// TODO: implement — load active gauges from DB, poll each, write readings,
// update last_reading_at and consecutive_failures, fire alerts on thresholds.
func (p *Poller) Run(ctx context.Context) {
	for _, sc := range p.sources {
		go func(sc sourceConfig) {
			ticker := time.NewTicker(sc.interval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					log.Printf("polling %s (not yet implemented)", sc.source.Name())
				}
			}
		}(sc)
	}
	<-ctx.Done()
}
