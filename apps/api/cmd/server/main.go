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

	gauges    := handlers.NewGaugeHandler(pool, enricher, p)
	reaches   := handlers.NewReachHandler(pool, asker).WithPoller(p)
	watchlist := handlers.NewWatchlistHandler(pool)
	admin     := handlers.NewAdminHandler(pool)
	nldiH    := handlers.NewNLDIHandler(pool).WithAnthropicKey(cfg.AnthropicAPIKey)
	// Warm the reach map cache immediately, then refresh every poll cycle.
	reaches.WarmCache(context.Background())
	reaches.StartCacheRefresh(pollerCtx, pollInterval.USGS)
	trips         := handlers.NewTripHandler(pool, describer)
	contributions := handlers.NewContributionHandler(pool)
	var importEmbedder *ai.Embedder
	if cfg.VoyageAPIKey != "" {
		importEmbedder = ai.NewEmbedder(cfg.VoyageAPIKey)
	}
	imports := &handlers.Import{
		Pool:              pool,
		CacheWarmer:       func() { reaches.WarmCache(context.Background()) },
		CenterlineFetcher: reaches.BackgroundFetchCenterline,
		Embedder:          importEmbedder,
		MetadataSyncer:    func() { p.SyncMetadataNow(context.Background()) },
	}
	// LoadAppRoles queries user_roles for the authenticated user on each request.
	// Runs after Optional/Required so the user ID is already in context.
	loadAppRoles := auth.LoadAppRoles(func(r *http.Request, userID string) ([]string, error) {
		rows, err := pool.Query(r.Context(),
			`SELECT role FROM user_roles WHERE user_id = $1`, userID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var roles []string
		for rows.Next() {
			var role string
			if rows.Scan(&role) == nil {
				roles = append(roles, role)
			}
		}
		return roles, nil
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Optional: attaches user claims when a valid Bearer token is present,
		// but anonymous (device_id) requests still flow through.
		r.Use(auth.Optional(verifier))
		// LoadAppRoles enriches context with DB roles for authenticated users.
		r.Use(loadAppRoles)
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
		r.Get("/reaches/{slug}/flow-ranges", reaches.GetFlowRanges)
		r.Post("/reaches/{slug}/ask", reaches.Ask)
		r.Post("/ask", reaches.GlobalAsk)

		// Authenticated user routes — require a valid Supabase JWT.
		r.Group(func(r chi.Router) {
			r.Use(auth.Required(verifier))
			r.Get("/watchlist", watchlist.List)
			r.Post("/watchlist", watchlist.Add)
			r.Delete("/watchlist/{gaugeId}", watchlist.Remove)
		})

		// Data admin routes — require data_admin or site_admin role.
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireDataAdmin)
			r.Put("/reaches/{slug}/flow-ranges", reaches.SetFlowRanges)
			r.Delete("/reaches/{slug}", reaches.Delete)
			r.Post("/reaches/{slug}/fetch-centerline", reaches.FetchCenterline)
			r.Delete("/reaches/{slug}/centerline", reaches.ClearCenterline)
			r.Post("/import/kmz", imports.ImportKMZ)
			r.Put("/admin/reaches/{slug}/river", admin.AssignReachToRiver)
			r.Get("/admin/rivers", admin.ListRivers)
			r.Get("/admin/rivers/{riverSlug}", admin.GetRiver)
			r.Post("/admin/rivers", admin.CreateRiver)
			r.Put("/admin/rivers/{riverSlug}", admin.UpdateRiver)
			r.Delete("/admin/rivers/{riverSlug}", admin.DeleteRiver)
			r.Get("/admin/nldi/watershed", nldiH.WatershedExplorer)
			r.Get("/admin/nldi/upstream-tributaries", nldiH.UpstreamTributaries)
			r.Get("/admin/nldi/downstream", nldiH.DownstreamMainstem)
			r.Post("/admin/reaches", nldiH.CreateReach)
			r.Get("/admin/reaches/{slug}", nldiH.GetAdminReach)
			r.Post("/admin/reaches/{slug}/generate-description", nldiH.GenerateDescription)
			r.Patch("/admin/reaches/{slug}", nldiH.PatchReach)
			r.Put("/admin/reaches/{slug}/meta", nldiH.UpdateReachMeta)
			r.Post("/admin/reaches/{slug}/nldi-centerline", nldiH.UpdateReachCenterline)
			r.Post("/admin/reaches/{slug}/nldi-centerline-by-comid", nldiH.UpdateReachCenterlineByComID)
		})

		// Site admin only — role management.
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAdmin)
			r.Get("/admin/users/roles", admin.ListUserRoles)
			r.Post("/admin/users/roles", admin.AssignRole)
			r.Delete("/admin/users/roles/{roleId}", admin.RevokeRole)
		})

		// Authenticated user — own role info.
		r.Group(func(r chi.Router) {
			r.Use(auth.Required(verifier))
			r.Get("/admin/me/roles", admin.GetMyRoles)
		})

		r.Post("/reaches/{slug}/contributions", contributions.CreateContribution)
		r.Post("/reaches/{slug}/trip-reports", contributions.CreateTripReport)
		r.Get("/reaches/{slug}/trip-reports", contributions.ListTripReports)
		r.Get("/trip-reports/{slug}", contributions.GetTripReport)
		r.Patch("/trip-reports/{slug}", contributions.PatchTripReport)
		r.Delete("/trip-reports/{slug}", contributions.DeleteTripReport)
		r.Post("/proximity-events", contributions.CreateProximityEvent)

		r.Post("/trips", trips.Create)
		r.Get("/trips", trips.List)
		r.Get("/trips/{id}", trips.Get)
		r.Patch("/trips/{id}", trips.Patch)
		r.Post("/trips/{id}/describe", trips.Describe)
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
