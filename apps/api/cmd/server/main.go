package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	gauge "github.com/h2oflow/h2oflow/packages/gauge-core"
	"github.com/h2oflow/h2oflow/apps/api/internal/ai"
	"github.com/h2oflow/h2oflow/apps/api/internal/auth"
	"github.com/h2oflow/h2oflow/apps/api/internal/config"
	"github.com/h2oflow/h2oflow/apps/api/internal/db"
	"github.com/h2oflow/h2oflow/apps/api/internal/handlers"
	"github.com/h2oflow/h2oflow/apps/api/internal/poller"
)

func main() {
	cfg := config.Load()

	// Run migrations before accepting traffic.
	if err := runMigrations(cfg); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	pool, err := db.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000", "https://*.h2oflows.org", "https://h2oflows.org", "https://*.netlify.app"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
		MaxAge:         300,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	// AI search enrichment — optional, degrades gracefully if key is absent.
	var enricher *ai.SearchEnricher
	if cfg.AnthropicAPIKey != "" {
		enricher = ai.NewSearchEnricher(cfg.AnthropicAPIKey)
	} else {
		log.Println("ANTHROPIC_API_KEY not set — AI search enrichment disabled")
	}

	// River assistant (RAG) — requires both Voyage and Anthropic keys.
	var asker *ai.ReachAsker
	if cfg.VoyageAPIKey != "" && cfg.AnthropicAPIKey != "" {
		asker = ai.NewReachAsker(pool, cfg.VoyageAPIKey, cfg.AnthropicAPIKey)
	} else {
		log.Println("VOYAGE_API_KEY not set — river assistant (/ask) disabled")
	}

	// Start gauge poller — runs in background, survives HTTP errors.
	pollInterval := cfg.ParsePollInterval()
	p := poller.New(pool)
	p.Register(gauge.NewUSGSSource(cfg.USGSAPIKey), pollInterval.USGS)
	p.Register(gauge.NewDWRSource(), pollInterval.DWR)
	pollerCtx, stopPoller := context.WithCancel(context.Background())
	go p.Run(pollerCtx)

	var describer *ai.TripDescriber
	if cfg.AnthropicAPIKey != "" {
		describer = ai.NewTripDescriber(pool, cfg.AnthropicAPIKey)
	}

	// Supabase JWT verifier — optional. When unset, all requests stay anonymous
	// and the existing device_id flow continues to work unchanged.
	var verifier *auth.Verifier
	if cfg.SupabaseJWKSURL != "" {
		v, err := auth.NewVerifier(context.Background(), cfg.SupabaseJWKSURL)
		if err != nil {
			log.Fatalf("auth: %v", err)
		}
		verifier = v
		log.Printf("auth: Supabase JWT verification enabled (%s)", cfg.SupabaseJWKSURL)
	} else {
		log.Println("SUPABASE_JWKS_URL not set — auth middleware disabled, all requests anonymous")
	}

	gauges  := handlers.NewGaugeHandler(pool, enricher, p)
	reaches := handlers.NewReachHandler(pool, asker)
	// Warm the reach map cache immediately, then refresh every poll cycle.
	reaches.WarmCache(context.Background())
	reaches.StartCacheRefresh(pollerCtx, pollInterval.USGS)
	trips   := handlers.NewTripHandler(pool, describer)
	imports := &handlers.Import{Pool: pool}
	r.Route("/api/v1", func(r chi.Router) {
		// Optional: attaches user claims when a valid Bearer token is present,
		// but anonymous (device_id) requests still flow through.
		r.Use(auth.Optional(verifier))
		r.Get("/gauges/search", gauges.Search)
		r.Get("/gauges/batch", gauges.BatchGet)
		r.Get("/gauges/{id}/readings", gauges.GetReadings)
		r.Get("/gauges/{id}/flow-ranges", gauges.GetFlowRanges)
		r.Get("/gauges/{id}/seasonal", gauges.GetSeasonalStats)

		r.Get("/reaches/map/all", reaches.MapAll)
		r.Get("/reaches/map", reaches.Map)
		r.Get("/reaches", reaches.List)
		r.Get("/reaches/{slug}", reaches.Get)
		r.Get("/reaches/{slug}/conditions", reaches.GetConditions)
		r.Get("/reaches/{slug}/hazards", reaches.GetHazards)
		r.Post("/reaches/{slug}/fetch-centerline", reaches.FetchCenterline)
		r.Delete("/reaches/{slug}/centerline", reaches.ClearCenterline)
		r.Post("/reaches/{slug}/ask", reaches.Ask)
		r.Post("/ask", reaches.GlobalAsk)

		r.Post("/trips", trips.Create)
		r.Get("/trips", trips.List)
		r.Get("/trips/{id}", trips.Get)
		r.Patch("/trips/{id}", trips.Patch)
		r.Post("/trips/{id}/describe", trips.Describe)

		r.Post("/import/kmz", imports.ImportKMZ)
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: r,
	}

	// Start server in background, shut down gracefully on SIGINT/SIGTERM.
	go func() {
		log.Printf("starting %s on :%s", cfg.AppName, cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	stopPoller() // stop poller before draining HTTP so in-flight DB writes finish

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
}

func runMigrations(cfg config.Config) error {
	// golang-migrate expects a file:// URL for the source and a pgx5:// URL for the DB.
	src := "file://" + cfg.MigrationsPath

	// golang-migrate's pgx/v5 driver uses "pgx5://" as the scheme.
	dbURL := "pgx5://" + stripScheme(cfg.DatabaseURL)

	m, err := migrate.New(src, dbURL)
	if err != nil {
		return fmt.Errorf("new: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("up: %w", err)
	}
	return nil
}

// stripScheme removes a leading "postgres://" or "postgresql://" so we can
// replace it with "pgx5://" for golang-migrate's pgx/v5 driver.
func stripScheme(url string) string {
	for _, prefix := range []string{"postgresql://", "postgres://"} {
		if len(url) > len(prefix) && url[:len(prefix)] == prefix {
			return url[len(prefix):]
		}
	}
	return url
}
