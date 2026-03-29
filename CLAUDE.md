# H2OFlow — Claude Code Guide

## Project docs

- [PROJECT.md](PROJECT.md) — vision, user goals, build order, open questions
- [ARCHITECTURE.md](ARCHITECTURE.md) — tech stack, data model, guiding principles
- [DECISIONS.md](DECISIONS.md) — ADR log for non-obvious decisions

## Repo layout

```
apps/api/          Go backend (Chi, pgx v5, PostGIS)
  cmd/server/      main entrypoint + router
  internal/
    handlers/      HTTP handlers
    osm/           Overpass API client + way-chaining algorithm
    kmlimport/     KMZ/KML importer
    poller/        gauge polling scheduler
    ai/            Claude search enrichment
  migrations/      golang-migrate SQL files (numbered, never edit old ones)

apps/web/          Nuxt 4 frontend (Nuxt UI Pro, MapLibre, uPlot)
  app/pages/       file-based routing
  app/components/
    map/ReachMap.vue   reach detail map (MapLibre, HTML pin markers)
    gauge/             gauge cards, graphs, sparklines

packages/
  gauge-core/      GaugeSource interface + USGS/DWR adapters

.claude/memory/    persistent AI memory (committed for project/* types)
```

## Stack notes

- **Go**: uses `go.work` workspace; build with `/usr/local/go/bin/go build ./apps/api/...`
- **pgx v5**: returns `text` columns as `[]byte` when scanned into `[]byte` — never add `::json` cast to `ST_AsGeoJSON()` output or pgx will try to base64-decode it
- **PostGIS**: use `ST_GeomFromGeoJSON($1)::geography` to store GeoJSON — `ST_GeogFromGeoJSON` does not exist in this version
- **MapLibre markers**: use `transition:scale` not `transition:transform` — MapLibre repositions via `transform`, transitioning it causes pins to float/lag during pan/zoom

## Environment

```
DATABASE_URL=postgres://h2oflow:h2oflow@localhost:5432/h2oflow?sslmode=disable
APP_PORT=8080
```

API server: `cd apps/api && /usr/local/go/bin/go run ./cmd/server`
Web dev: `cd apps/web && npm run dev`

## Conventions

- Migrations: sequential numbered files (`000027_*.up.sql` / `*.down.sql`), never edit existing ones
- Reach slugs are the canonical reach identifier across the codebase
- Flow difficulty stored as floats (`3.5`), rendered as Roman numerals (`III+`)
- Colorado rivers flow west→east, so `MIN(lng)` = most upstream, `MAX(lng)` = most downstream
