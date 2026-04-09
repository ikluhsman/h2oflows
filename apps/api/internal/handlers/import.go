package handlers

import (
	"io"
	"net/http"

	"github.com/h2oflow/h2oflow/apps/api/internal/kmlimport"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Import provides the KMZ/KML file import endpoint.
type Import struct {
	Pool        *pgxpool.Pool
	CacheWarmer func() // optional; called after a successful import to refresh the map cache
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

	jsonResponse(w, http.StatusOK, res)
}
