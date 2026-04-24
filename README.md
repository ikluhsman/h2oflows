# H2OFlows

A streamflow data platform for whitewater paddlers. Snap in your favorite gauges, get live CFS with flow status bands, compare rivers side by side, and ask plain-English questions about any run. Backed by a free open reach registry and public API built from community data.

See [ARCHITECTURE.md](ARCHITECTURE.md) for full technical design.

---

## Features

### Gauge dashboard
- Personal watchlist of USGS and Colorado DWR gauges — saved locally, synced to your account when signed in
- Live CFS readings refreshed every 60 seconds
- Flow status bands (below recommended / runnable / above recommended) overlaid on each gauge card
- Named flow band label — know at a glance whether your run is on
- Four density views — **Compact**, **Comfortable**, **Full**, and **List** — so you can tune the dashboard to phone, tablet, or wall display
- Show/hide dashboard map, with the preference persisted across sessions
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
- Animated scroll-to-top button (GSAP)

### AI river assistant (RAG)
- Ask plain-English questions about any reach: *"What's Browns Canyon like at 800 cfs?"*
- Answers grounded in reach-specific embedded content (descriptions, rapids, flow ranges, access) — never hallucinates rapid names or distances not in the source data
- Per-reach chat panel on every reach page
- Global search on the landing page — identifies the reach from free text, then answers
- Powered by Voyage AI embeddings + pgvector similarity search + Claude Haiku

### Landing page
- Hero with linked feature pills — **Real-time Gauges**, **AI Flow Intel**, **GPS Trip Tracking** — each links directly to the relevant feature
- Global AI ask prompt: type a question about any Colorado river and get an answer on the spot
- Dashboard and Map quick-nav buttons
- App Store and Google Play placeholder badges (native apps coming soon)

### Admin reach authoring (NHD Explorer)
- Step-based pin picker in the admin panel: click the map to place put-in and take-out, auto-snapped to the nearest NHD reach via USGS NLDI
- Upstream flowlines (blue), downstream mainstem (teal), and upstream USGS gauges (amber) drawn on a MapLibre topo map after each snap
- NLDI centerline fetched between the two pins, trimmed to exact extent via PostGIS `ST_LineSubstring`, and stored alongside the reach geometry
- Re-pin existing reaches: update any reach's centerline from the admin panel without modifying its access points — used to fix or improve centerlines on already-imported reaches
- Creates reaches with slug, name, river, class, put-in/take-out ComIDs, and length in miles; 409 on slug collision
- Basemap switcher (Street / Topo / Satellite) on all admin maps

### Data pipeline
- Reaches seeded with AI-generated descriptions, rapid inventories, access points, and flow ranges (all marked `ai_seed`, confidence-scored)
- OSM centerline fetch for each reach using the Overpass API; NLDI centerline fetch available as an alternative (`--centerlines=nldi` on import)
- Best-effort NHD ComID capture on every OSM centerline fetch — stored for future NLDI switch-over without re-snapping
- Polling tiers: trusted reaches always polled, demand-tier gauges polled when recently viewed, cold gauges skipped until requested
- USGS and Colorado DWR gauge import commands

---

## Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.23, Chi, PostgreSQL 16 + PostGIS + pgvector |
| AI | Anthropic Claude Haiku (RAG answers, reach seeding, search enrichment) |
| Embeddings | Voyage AI `voyage-3` (1024-dim, stored in pgvector) |
| Frontend | Nuxt 4, Nuxt UI Pro, Tailwind CSS, MapLibre GL, uPlot, GSAP, Pinia |
| Auth | Supabase Auth (email, OAuth) — server-side watchlist sync |
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
      kmlimport/      KMZ/KML importer, OSM + NLDI centerline sync
      nldi/           USGS NLDI API client (snap, navigate, mainstem merge)
      osm/            Overpass API client + reach centerline fetch
      poller/         Gauge polling scheduler (trusted/demand/cold tiers)
      config/         Environment config
    migrations/       golang-migrate SQL files (060 migrations)
  web/                Nuxt 4 frontend
    app/
      pages/          Landing, dashboard, explore, reach detail, trips
      components/
        map/          DashboardMap, ReachesMap, ReachMap, NHDExplorerMap (MapLibre)
        gauge/        GaugeCard, GaugeGraph, GaugeSparkline
      composables/    useAuth, useWatchlistSync, useWatchlistRefresh, useTrips, useGaugeGraph
      stores/         Pinia — watchlist (localStorage + server sync)
packages/
  gauge-core/         Gauge source adapter interface + USGS/DWR implementations
```

---

## Data sources

- **USGS Water Services API** — no API key, covers most of the US
- **Colorado DWR telemetry** — CDSS API, abbreviation-based station IDs
- **USGS NLDI (Network Linked Data Index)** — snaps coordinates to NHD ComIDs, navigates upstream/downstream flowlines, discovers upstream USGS gauge sites; used for admin reach authoring and centerline generation
- **OpenStreetMap Overpass API** — fallback centerline source; longest river/stream waterway within a bounding box
- **AI seeder (Claude)** — generates reach descriptions, rapid inventories, access points, and flow ranges from training knowledge; all output is marked `data_source='ai_seed'` and confidence-scored

Community corrections and verified data take precedence over AI-seeded content once contributed.

---

## Roadmap

### Trip reports + social sharing
Trip "tracking" is really trip **reporting**. A paddler creates a quick trip report — reach, date, flow impression, conditions, optional photos — and shares it to **AW**, **Instagram**, **Facebook**, or keeps it private. AW is one destination among many; the user chooses where their data goes.

### Passive telemetry
Lightweight, opt-in telemetry that confirms flow bands at scale. Proximity pings near known put-ins/take-outs, gauge-view interest signals, and post-trip flow confirmations — all low-friction taps that feed aggregate flow-band accuracy without requiring GPS.

### Native mobile apps
Capacitor-based iOS and Android apps for background GPS recording, offline trip sync, and push notifications for flow alerts. Web app continues as the primary dashboard.

### SEO + Open Graph
Dynamic OG images for reaches, gauges, trip reports, and the homepage — so shared links look great on social media with flow status, reach name, and conditions.
