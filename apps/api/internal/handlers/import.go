package handlers

import (
	"context"
	"io"
	"log"
	"net/http"

	"github.com/h2oflow/h2oflow/apps/api/internal/ai"
	"github.com/h2oflow/h2oflow/apps/api/internal/kmlimport"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Import provides the KMZ/KML file import endpoint.
type Import struct {
	Pool               *pgxpool.Pool
	CacheWarmer        func()             // optional; called after a successful import to refresh the map cache
	CenterlineFetcher  func(slug string)  // optional; called for each imported reach to auto-fetch its river line
	Embedder           *ai.Embedder       // optional; when set, auto-embeds imported reaches for the AI assistant
	MetadataSyncer     func()             // optional; called after import to sync gauge basin/location metadata immediately
}

// ImportKMZ handles POST /api/v1/import/kmz
// Accepts multipart/form-data with a "file" field containing a KML or KMZ file.
// Returns a JSON import result with per-reach counts and a log of actions taken.
func (h *Import) ImportKMZ(w http.ResponseWriter, r *http.Request) {
	// 32 MB max upload.
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid multipart form: "+err.Error())
		return
	}

	f, _, err := r.FormFile("file")
	if err != nil {
		errorResponse(w, http.StatusBadRequest, "missing 'file' field")
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "read file: "+err.Error())
		return
	}

	doc, err := kmlimport.ParseKMLBytes(data)
	if err != nil {
		errorResponse(w, http.StatusUnprocessableEntity, "parse KML: "+err.Error())
		return
	}

	imp := kmlimport.New(h.Pool, false)
	res, err := imp.Import(r.Context(), doc)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "import: "+err.Error())
		return
	}

	// Rewarm the map cache so imported reaches appear on the map immediately.
	if h.CacheWarmer != nil {
		go h.CacheWarmer()
	}

	// Sync gauge metadata (basin, watershed, location) immediately so newly
	// imported DWR gauges get their basin assigned without a server restart.
	if h.MetadataSyncer != nil {
		go h.MetadataSyncer()
	}

	// Auto-fetch centerlines for all imported reaches in the background.
	if h.CenterlineFetcher != nil {
		for slug := range res.Reaches {
			h.CenterlineFetcher(slug)
		}
	}

	// Auto-embed imported reaches for the AI assistant. Runs in a background
	// goroutine so it doesn't block the import response. Rate-limiting is
	// applied between reaches to respect the Voyage free-tier (3 RPM).
	if h.Embedder != nil && len(res.Reaches) > 0 {
		slugs := make([]string, 0, len(res.Reaches))
		for slug := range res.Reaches {
			slugs = append(slugs, slug)
		}
		pool := h.Pool
		embedder := h.Embedder
		go func() {
			// Resolve slugs → reach IDs.
			ids := make([]string, 0, len(slugs))
			for _, slug := range slugs {
				var id string
				err := pool.QueryRow(context.Background(),
					`SELECT id FROM reaches WHERE slug = $1`, slug).Scan(&id)
				if err != nil {
					log.Printf("auto-embed: lookup slug %q: %v", slug, err)
					continue
				}
				ids = append(ids, id)
			}
			if len(ids) == 0 {
				return
			}
			embedded, skipped, err := ai.EmbedReaches(context.Background(), pool, embedder, ids, true)
			if err != nil {
				log.Printf("auto-embed: %v", err)
				return
			}
			log.Printf("auto-embed: %d chunks embedded, %d skipped for %d reach(es)", embedded, skipped, len(ids))
		}()
	}

	jsonResponse(w, http.StatusOK, res)
}
