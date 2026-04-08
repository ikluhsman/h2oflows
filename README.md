# H2OFlows

A streamflow data platform for whitewater paddlers. Snap in your favorite gauges, get live CFS with flow status bands, compare rivers side by side, and ask plain-English questions about any run. Backed by a free open reach registry and public API built from community data.

See [ARCHITECTURE.md](ARCHITECTURE.md) for full technical design and [DECISIONS.md](DECISIONS.md) for the reasoning behind key choices.

---

## Features

### Gauge dashboard
- Personal watchlist of USGS and Colorado DWR gauges, persisted across sessions
- Live CFS readings refreshed every 60 seconds
- Flow status bands (below recommended / runnable / above recommended) overlaid on each gauge card
- Named flow band label — know at a glance whether your run is on
- Side-by-side multi-gauge comparison graph for reaches with more than one relevant gauge
- Gauges grouped by reach with a link through to the full reach page

### Interactive maps
- MapLibre GL maps across dashboard, explore, and reach detail views
- Reach centerlines drawn from OSM data, colored by difficulty (green / blue / black / double-black)
- Dashboard map auto-fits to the geographic bounding box of your saved gauges
- Gauge markers show current CFS and flow status color
- Street / Topo / Satellite basemap toggle on all three maps (Esri tile services)
- Topo default uses USGS 7.5-minute quad sheets — contours, river names, access roads

### Reach pages
- Curated reach data: description, difficulty, put-in/take-out access, rapid inventory
- 48-hour flow graph with flow band overlays
- Seasonal stats showing historical median CFS by month
- Hazard and conditions feeds
- KML export of reach geometry

### AI river assistant (RAG)
- Ask plain-English questions about any reach: *"What's Browns Canyon like at 800 cfs?"*
- Answers grounded in reach-specific embedded content (descriptions, rapids, flow ranges, access) — never hallucinates rapid names or distances not in the source data
- Per-reach chat panel on every reach page
- Global search on the landing page — identifies the reach from free text, then answers
- Powered by Voyage AI embeddings + pgvector similarity search + Claude Haiku

### Data pipeline
- 32 Colorado reaches seeded with AI-generated descriptions, rapid inventories, access points, and flow ranges (all marked `ai_seed`, confidence-scored)
- OSM centerline fetch for each reach using the Overpass API
- Polling tiers: trusted reaches always polled, demand-tier gauges polled when recently viewed, cold gauges skipped until requested
- USGS and Colorado DWR gauge import commands

---

## Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.23, Chi, PostgreSQL 16 + PostGIS + pgvector |
| AI | Anthropic Claude Haiku (RAG answers, reach seeding, search enrichment) |
| Embeddings | Voyage AI `voyage-3` (1024-dim, stored in pgvector) |
| Frontend | Nuxt 4, Nuxt UI Pro, Tailwind CSS, MapLibre GL, uPlot, Pinia |
| Maps | Esri public tile services (Street / USA Topo / World Imagery) |

---

## Running locally

### Prerequisites

- Go 1.26+ (`/usr/local/go/bin/go version`)
- PostgreSQL 16 + PostGIS (`sudo apt install postgresql-16-postgis-3`)
- Node 20+ + npm

### Setup

```sh
# 1. Create the database
sudo -u postgres createdb h2oflow

# 2. Copy and fill in environment variables
cp apps/api/.env.example apps/api/.env
# Required: DATABASE_URL, ANTHROPIC_API_KEY, VOYAGE_API_KEY

# 3. Start the API (auto-runs migrations on startup)
cd apps/api
set -a && source .env && set +a
go run ./cmd/server

# 4. In a separate terminal, start the frontend
cd apps/web
npm install
npm run dev
```

API runs on `:8080`, web on `:3000`.

### Seeding reach data

```sh
cd apps/api
set -a && source .env && set +a

# Seed the 19 hand-picked Front Range CO reaches + AI-generated content
go run ./cmd/seed-reaches

# Re-seed a reach (overwrites AI content, keeps community data)
RESEED=true go run ./cmd/seed-reaches

# Bulk import USGS gauges by state
go run ./cmd/seed-usgs-states
```

### Importing reach data from KMZ

Build a Google My Map of a river (rapids, put-ins, take-outs, parking) and export as KMZ:

```sh
cd apps/api
go run ./cmd/import-kml -file /path/to/your-export.kmz
```

See [docs/kmz-import-guide.md](docs/kmz-import-guide.md) for the folder and pin naming conventions the importer expects.

---

## Project structure

```
apps/
  api/                Go backend
    cmd/
      server/         Main entrypoint (Chi router, migrations, poller)
      seed-reaches/   Upserts Front Range reaches + AI-generated content
      embed-reaches/  Embeds reach content chunks into pgvector
      seed-flow-ranges/   Seeds flow bands for gauge+reach pairs
      seed-usgs-states/   Bulk USGS gauge import
    internal/
      ai/             Claude + Voyage AI (RAG asker, reach seeder, search enrichment)
      handlers/       HTTP route handlers
      osm/            Overpass API client + reach centerline fetch
      poller/         Gauge polling scheduler (trusted/demand/cold tiers)
      config/         Environment config
    migrations/       golang-migrate SQL files (031 migrations)
  web/                Nuxt 4 frontend
    app/
      pages/          Landing, dashboard, explore, reach detail
      components/
        map/          DashboardMap, ReachesMap, ReachMap (MapLibre)
        gauge/        GaugeCard, GaugeGraph, GaugeSparkline
      composables/    useWatchlistRefresh, useGaugeGraph
      stores/         Pinia — watchlist (persisted to localStorage)
packages/
  gauge-core/         Gauge source adapter interface + USGS/DWR implementations
```

---

## Data sources

- **USGS Water Services API** — no API key, covers most of the US
- **Colorado DWR telemetry** — CDSS API, abbreviation-based station IDs
- **AI seeder (Claude)** — generates reach descriptions, rapid inventories, access points, and flow ranges from training knowledge; all output is marked `data_source='ai_seed'` and confidence-scored

Community corrections and verified data take precedence over AI-seeded content once contributed.
