# Dependency Graph

## Most Imported Files (change these carefully)

- `encoding/json` — imported by **26** files
- `net/http` — imported by **24** files
- `net/url` — imported by **7** files
- `net/http/httptest` — imported by **4** files
- `os/signal` — imported by **1** files
- `crypto/rand` — imported by **1** files
- `encoding/base32` — imported by **1** files
- `archive/zip` — imported by **1** files
- `encoding/xml` — imported by **1** files
- `apps/web/app/composables/useDiurnalPattern.ts` — imported by **1** files

## Import Map (who imports what)

- `encoding/json` ← `apps/api/cmd/backfill-comids/main.go`, `apps/api/internal/ai/asker.go`, `apps/api/internal/ai/describer.go`, `apps/api/internal/ai/discoverer.go`, `apps/api/internal/ai/embedder.go` +21 more
- `net/http` ← `apps/api/cmd/backfill-comids/main.go`, `apps/api/cmd/server/main.go`, `apps/api/internal/ai/embedder.go`, `apps/api/internal/auth/middleware.go`, `apps/api/internal/elevation/elevation.go` +19 more
- `net/url` ← `apps/api/cmd/backfill-comids/main.go`, `apps/api/internal/nldi/client.go`, `apps/api/internal/nldi/geo.go`, `apps/api/internal/nldi/nhd.go`, `apps/api/internal/osm/osm.go` +2 more
- `net/http/httptest` ← `apps/api/internal/kmlimport/nldi_test.go`, `apps/api/internal/nldi/client_test.go`, `packages/gauge-core/dwr_test.go`, `packages/gauge-core/usgs_test.go`
- `os/signal` ← `apps/api/cmd/server/main.go`
- `crypto/rand` ← `apps/api/internal/handlers/contributions.go`
- `encoding/base32` ← `apps/api/internal/handlers/contributions.go`
- `archive/zip` ← `apps/api/internal/kmlimport/kmlimport.go`
- `encoding/xml` ← `apps/api/internal/kmlimport/kmlimport.go`
- `apps/web/app/composables/useDiurnalPattern.ts` ← `apps/web/app/composables/useDiurnalCache.ts`
