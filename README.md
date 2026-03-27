# H2OFlow

A streamflow data platform for whitewater paddlers. Snap in your favorite gauges, get live CFS with flow status bands, compare rivers side by side, and track sessions. Backed by a free open reach registry and public API built from community data.

See [ARCHITECTURE.md](ARCHITECTURE.md) for full technical design and [DECISIONS.md](DECISIONS.md) for the reasoning behind key choices.

---

## Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.26, Chi, PostgreSQL 16 + PostGIS |
| AI | Anthropic Claude (reach seeding, search enrichment) |
| Frontend | Nuxt 4, Nuxt UI v4, Tailwind CSS, uPlot, Pinia |
| Maps | MapLibre GL (planned) |

Redis and Docker Compose are planned for production deployment but not required for local development.

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
# Required: DATABASE_URL, ANTHROPIC_API_KEY

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

---

## Project structure

```
apps/
  api/                Go backend
    cmd/
      server/         Main entrypoint (Chi router, migrations, poller)
      seed-reaches/   Upserts Front Range reaches + AI-generated content
      seed-flow-ranges/  Seeds flow bands for gauge+reach pairs
      seed-state-reaches/ Broader CO inventory
      seed-usgs-states/   Bulk USGS gauge import
    internal/
      ai/             Claude seeder (reach descriptions, rapids, access, flow ranges)
      handlers/       HTTP route handlers
      poller/         Gauge polling scheduler (trusted/demand/cold tiers)
      config/         Environment config
    migrations/       golang-migrate SQL files (026 migrations)
  web/                Nuxt 4 frontend
    app/
      pages/          Dashboard (index), reach detail
      components/     GaugeCard, GaugeSearchModal, graphs, sparklines
      composables/    useWatchlistRefresh, useGaugeGraph, useTripRecording
      stores/         Pinia — watchlist (persisted to localStorage)
packages/
  gauge-core/         Gauge source adapter interface + USGS/DWR/HUC implementations
```

---

## Data sources

- **USGS Water Services API** — no API key, covers most of the US
- **Colorado DWR telemetry** — CDSS API, abbreviation-based station IDs
- **AI seeder (Claude)** — generates reach descriptions, rapid inventories, access points, and flow ranges from training knowledge; all output is marked `data_source='ai_seed'` and confidence-scored

Community corrections and verified data take precedence over AI-seeded content once contributed.
