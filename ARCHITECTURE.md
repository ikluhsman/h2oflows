# H2OFlow — Architecture & Vision

> Open source whitewater platform, built by paddlers for paddlers.
> https://h2oflow.org

---

## What this is

H2OFlow is a community-owned, open source platform for whitewater enthusiasts. At its core it is a **streamflow data platform** — a beautiful, fast, customizable gauge dashboard that works better than anything else available, backed by an open reach registry that the entire whitewater community can build on and query freely.

It is not a social network. Discord, Facebook, and SMS fill that role well enough. H2OFlow fills the gap those platforms can't: structured, geographically precise river data tied to real-time conditions, accessible via a clean public API that any service can consume.

The platform is free, open source, and will remain so. The data belongs to the community.

---

## The problem we're solving

Right now a Colorado paddler checking conditions before a morning run opens multiple browser tabs — USGS for one river, DWR for another, a Facebook group for word-of-mouth conditions, maybe a Grafana dashboard they built themselves. There is no single place that aggregates favorite gauges into one view, overlays community-defined flow ranges, or surfaces hazard warnings alongside live CFS numbers.

American Whitewater has reach data but keeps it locked. RiverApp is commercial and closed. USGS is authoritative but raw. The geographic data that defines where paddlers actually put on and take out rivers doesn't exist in any open, structured form.

H2OFlow builds that open data layer — and gives it back to the community as a free API.

---

## Guiding principles

**Streamflow first.** The gauge dashboard is the entry point and the core value. Everything else builds on top of it.

**Data as community asset.** The reach registry, flow ranges, conditions reports, and hazard data belong to the community. No proprietary lock-in. Everything is exportable and queryable.

**Local-first.** Trip data lives on the user's device first. The app works offline at the put-in where there's no signal. Cloud sync is optional.

**Open API.** All community-contributed data is freely accessible via API to anyone with an account or token. H2OFlow is the data infrastructure. The app is just the first consumer of its own API.

**Complexity scales with the trip.** A gauge check should take 3 seconds. A day trip 30 seconds. A permit trip has every tool it needs. The UI adapts to the use case.

**FOSS commitment is real.** Self-hosting is a first-class deployment option. `docker compose up` should produce a running instance in under 10 minutes.

**No hardcoded brand strings.** App name, domain, and org live in config only. Rebranding or repo migration is a one-line change.

---

## Product scope

### Core — Streamflow platform
The wedge. Usable without an account.

- Live gauge dashboard — CFS, stage, trend, sparklines
- Multi-gauge aggregate view — see 3 rivers on one graph, same scale
- Community-defined flow range bands overlaid on graphs (too low / minimum / fun / optimal / pushy / flood)
- Saved gauge dashboards (account required for persistence)
- SMS / push / Discord alerts on user-defined thresholds
- Reach pages with current conditions, recent trip reports, hazard warnings
- Public API for all reach and gauge data

### Reach registry
The open geographic data layer that doesn't currently exist in open form.

- Put-in / take-out coordinates with confidence scoring
- Rapid inventory with locations and difficulty ratings
- Flow-dependent difficulty ratings (class varies by CFS)
- Community conditions board — word-of-mouth reports, auto-expires 7 days
- Hazard warnings with flow context
- Passive put-in/take-out detection from opt-in app usage

### Trip planning (builds on core)
- Day trip planner — reach lookup, current conditions, shareable link
- Overnight trip planner — multi-day itinerary, roster, basic food notes
- Permit trip coordinator — full roster, roles, gear matrix, food planner, outfitter integration, cost splitting, shuttle coordination
- Trip export — markdown, PDF, KML/GPX, hosted blog post with geotagged photo map

### Back-burner (post v1)
- Commercial outfitter API (paid tier)
- Photo / video storage (subscription tier, Cloudflare R2)
- Native iOS / Android apps via Capacitor (PWA first)
- International gauge API integrations (Environment Canada, CDEC, etc.)

---

## Tech stack

### Backend

| Layer | Choice |
|---|---|
| Language | Go |
| Router | Chi |
| Database | PostgreSQL + PostGIS |
| Migrations | golang-migrate |
| Cache | Redis |
| Object storage | Cloudflare R2 (hosted) / any S3-compatible (self-hosted) |
| Reverse proxy | Traefik |
| Auth | JWT + email/password (no social login) |

Go was chosen for single-binary deployment (critical for self-hosting), excellent concurrency for gauge polling across multiple sources, and a strong FOSS contributor ecosystem. The `gauge-core` package implements a plugin interface — adding a new data source is writing one adapter file.

### Frontend

| Layer | Choice |
|---|---|
| Framework | Nuxt 4 |
| UI library | Nuxt UI Pro |
| Styling | Tailwind CSS (via Nuxt UI) |
| Maps | MapLibre GL + vue-maplibre-gl |
| Charts | uPlot (fast time-series rendering for gauge graphs) |
| State | Pinia |
| Composables | VueUse |
| PWA | @vite-pwa/nuxt |
| App store wrapper | Capacitor (later) |

Nuxt 4 with SSR means reach pages and gauge dashboards are fully indexable — organic SEO is the primary discovery mechanism for a community project with no marketing budget. Nuxt UI Pro provides polished dashboard layouts, stat cards, and data table components that would otherwise be built from scratch.

uPlot is specifically chosen for gauge graphs — it renders 100k+ data points smoothly, is extremely lightweight, and handles the time-series use case better than Chart.js or Recharts.

### Discord bot

Small Go service in the same monorepo. Phase 1 uses simple incoming webhooks (no bot OAuth required). Phase 2 adds slash commands via the Discord bot API.

---

## Monorepo structure

```
h2oflow/
├── apps/
│   ├── api/                  Go backend
│   │   ├── cmd/server/       main entrypoint
│   │   ├── internal/
│   │   │   ├── handlers/     Chi route handlers
│   │   │   ├── models/       database models
│   │   │   ├── poller/       gauge polling scheduler
│   │   │   └── alerts/       threshold alert dispatch
│   │   └── migrations/       golang-migrate SQL files
│   ├── web/                  Nuxt 4 frontend
│   │   ├── pages/
│   │   ├── components/
│   │   ├── composables/
│   │   └── stores/           Pinia
│   └── discord-bot/          Go webhook + slash command service
├── packages/
│   ├── gauge-core/           gauge source adapter implementations
│   │   ├── usgs.go
│   │   ├── dwr.go
│   │   └── interface.go
│   ├── river-data/           reach schema, seed data, OSM import tools
│   └── ui/                   shared Vue components (if needed beyond Nuxt UI)
├── infra/
│   ├── docker-compose.yml
│   └── traefik/
└── docs/
    ├── architecture.md       this file
    ├── api.md                public API reference
    └── contributing.md
```

---

## Gauge plugin architecture

All gauge sources implement a common Go interface.

```go
type GaugeSource interface {
    FetchReading(externalID string) (*Reading, error)
    FetchHistory(externalID string, since time.Time) ([]*Reading, error)
    Name() string
    SourceType() SourceType
}

type Reading struct {
    ExternalID string
    Value      float64
    Unit       string
    Timestamp  time.Time
    QualCode   string
}
```

Current adapters: `USGSSource`, `ColoradoDWRSource`
Planned: `CDECSource`, `EnvironmentCanadaSource`, `ManualSource`, `CommunitySource`

The poller runs on a configurable schedule per source, writes readings to PostgreSQL, invalidates Redis cache, and fires webhooks for threshold alerts.

USGS base URL: `https://waterservices.usgs.gov/nwis/iv/`
Key params: `sites={site_code}`, `parameterCd=00060` (CFS discharge), `00065` (stage), `format=json`

---

## Data architecture

### Three-layer model

```
Public commons          read-only, no auth required
  River reach registry
  Gauge telemetry + history
  Flow range definitions
  Published trip reports
  Hazard warnings
  Reach conditions board
  Incident reports

Local device            no account required, full read/write
  Saved gauge dashboards
  Trip plans (active and completed)
  Food plans and gear lists
  Personal reach notes
  Cached reach/gauge data for offline use

Cloud sync              authenticated, optional
  Backup of local trip data
  Roster collaboration
  Trip publishing and blog export
  Photo storage (subscription)
  Anonymous usage telemetry (opt-in, feeds reach scoring)
```

### Core schema (simplified)

```sql
-- Geographic anchor for everything
CREATE TABLE reaches (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug            TEXT UNIQUE NOT NULL,
  name            TEXT NOT NULL,
  put_in          GEOGRAPHY(POINT, 4326),
  take_out        GEOGRAPHY(POINT, 4326),
  centerline      GEOGRAPHY(LINESTRING, 4326),
  class_min       NUMERIC(3,1),
  class_max       NUMERIC(3,1),
  class_at_low    NUMERIC(3,1),
  class_at_high   NUMERIC(3,1),
  character       TEXT, -- creeking/pool-drop/continuous/big-water/flatwater
  length_mi       NUMERIC(6,2),
  region          TEXT,
  created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Gauge sources bound to reaches
CREATE TABLE gauges (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reach_id        UUID REFERENCES reaches(id),
  external_id     TEXT NOT NULL, -- USGS site code, DWR abbrev, etc
  source          TEXT NOT NULL, -- usgs/dwr/cdec/manual/community
  name            TEXT,
  location        GEOGRAPHY(POINT, 4326),
  param_code      TEXT DEFAULT '00060', -- CFS
  active          BOOLEAN DEFAULT TRUE
);

-- Community-defined flow bands
CREATE TABLE flow_ranges (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  gauge_id        UUID REFERENCES gauges(id),
  label           TEXT NOT NULL, -- too_low/minimum/fun/optimal/pushy/high/flood
  min_cfs         NUMERIC(10,2),
  max_cfs         NUMERIC(10,2),
  class_modifier  NUMERIC(3,1)
);

-- Auto-stamped with real gauge reading at time of run
CREATE TABLE trip_reports (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reach_id        UUID REFERENCES reaches(id),
  user_id         UUID REFERENCES users(id),
  run_date        DATE NOT NULL,
  cfs_at_run      NUMERIC(10,2),
  class_felt      NUMERIC(3,1),
  craft_type      TEXT,
  conditions      TEXT,
  notes           TEXT,
  published       BOOLEAN DEFAULT FALSE,
  created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Point hazards on a reach
CREATE TABLE hazards (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reach_id        UUID REFERENCES reaches(id),
  location        GEOGRAPHY(POINT, 4326),
  hazard_type     TEXT, -- strainer/sieve/undercut/low-head-dam/other
  description     TEXT NOT NULL,
  cfs_at_report   NUMERIC(10,2),
  reported_by     UUID REFERENCES users(id),
  active          BOOLEAN DEFAULT TRUE,
  created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Short-lived conditions intel
CREATE TABLE reach_conditions (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reach_id        UUID REFERENCES reaches(id),
  source_type     TEXT, -- gauge/personal/word-of-mouth/discord/outfitter
  summary         TEXT NOT NULL,
  runnable        BOOLEAN,
  reported_by     UUID REFERENCES users(id),
  expires_at      TIMESTAMPTZ DEFAULT NOW() + INTERVAL '7 days',
  created_at      TIMESTAMPTZ DEFAULT NOW()
);
```

### Difficulty rating model

Stored as floats, rendered as Roman numerals in the UI.

```
class_min    3.0   → "III"
class_max    4.0   → "IV"
4.5          → "IV+"
5.0          → "V"
```

Enables real queries: "show me runs that are class III or below at current flows."

### Reach relevance scoring

No editorial curation. Usage signals importance. Score computed nightly.

```
reach_relevance_score =
  (search_count × 1)
  + (view_count_30d × 2)
  + (trip_report_count × 5)
  + (active_gauge_count × 10)
  + (watchlist_count × 3)
  + (api_request_count × 1)
```

### Put-in / take-out detection

Opt-in only. Raw GPS tracks never leave the device — only derived point candidates are contributed.

```
location_confidence_score
  manual_marker_placed      weight: 100  (user explicitly pinned it)
  vehicle_stop_cluster      weight: 40   (multiple users stopped here)
  trip_session_start        weight: 30   (app session began here)
  road_access_proximity     weight: 20   (OSM road access nearby)
  unique_user_count         weight: 10   (per additional unique user)
  repeat_user_bonus         weight: 15   (returning paddlers)
```

---

## Public API

Freely accessible. Token required for writes and elevated rate limits.

```
GET  /api/v1/reaches
GET  /api/v1/reaches/{slug}
GET  /api/v1/reaches/{slug}/gauges
GET  /api/v1/reaches/{slug}/conditions
GET  /api/v1/reaches/{slug}/reports
GET  /api/v1/reaches/{slug}/hazards
GET  /api/v1/gauges/{id}/readings
GET  /api/v1/gauges/{id}/readings?from=&to=
GET  /api/v1/gauges/{id}/flow-ranges
GET  /api/v1/regions/{slug}/reaches
POST /api/v1/reaches/{slug}/conditions   (authenticated)
POST /api/v1/reaches/{slug}/hazards      (authenticated)
POST /api/v1/reaches/{slug}/reports      (authenticated)
```

Rate limits: 1000 req/hour free, elevated for community tools, commercial tier for outfitters.
Attribution requested: "data sourced from H2OFlow community (h2oflow.org)"

---

## AI contribution layer

Lowers contribution friction to near zero. Reads natural trip notes, maps to database fields, surfaces confirmation cards. Never writes to the database without explicit user action.

```
[ Trip complete ]

A few things from your notes we can add to the community database:

  ✦ Hazard at Pine Creek rapid — new strainer river left
    → Log as hazard warning?  [ Yes ]  [ Edit ]  [ Skip ]

  ✦ Vallie Bridge put-in not yet verified
    → Drop a pin?  [ Open map ]  [ Skip ]

  ✦ You ran this at 850 CFS — community shows 800–1000 as optimal
    → Confirm?  [ Confirm ]  [ Adjust ]  [ Skip ]
```

Also generates blog post drafts from trip metadata + auto-stamped flow data + user notes.

---

## Discord integration

### Phase 1 — Commands only (launch)
```
!hflow hazard arkansas-numbers "new strainer pine creek river left"
!hflow conditions poudre-mishawaka 340 "tobin clean, picnic washed out"
!hflow flow arkansas-numbers
!hflow alert set cache-la-poudre 150 250
!hflow help
```
Every write returns a one-click confirmation link before touching the database.

### Phase 2 — Keyword nudges
Watches designated channels for: strainer, hazard, portage, pin, washed out, undercut. Prompts author gently. Never auto-posts.

### Phase 3 — AI extraction
AI reads natural language, identifies conditions reports, prompts author to confirm. Human confirms every post.

### Outbound alerts
```
🚨 Hazard — Arkansas / Numbers
Pine Creek Rapid · strainer river left
Reported at 920 CFS (currently 950, rising)
→ h2oflow.org/reaches/arkansas-numbers/hazards
```

---

## Trip export formats

- **Markdown** — static site generators, Obsidian
- **PDF** — printable trip binder
- **KML / GPX** — Google Earth, Gaia GPS, CalTopo
- **Hosted trip page** — flow graph, geotagged photo map, embedded YouTube/Vimeo, food log, conditions summary

Photos auto-pinned from EXIF GPS. River-following line drawn using reach centerline geometry.

---

## Self-hosting

Target: `git clone` → `docker compose up` → running in under 10 minutes.
Self-hosted instances can optionally federate reach edits and hazard data back to the public commons with user consent.

---

## Reach data bootstrap

Seed from OpenStreetMap whitewater data (`whitewater:section_grade`). Colorado first. Top 50 Colorado runs manually verified by beta group in initial sprint. Passive detection fills in from there.

---

## Build order

**Phase 1 — Gauge dashboard (months 1–2)**
Port USGS/DWR integrations to `gauge-core` Go package. Build public gauge dashboard in Nuxt 4 with Nuxt UI Pro — live CFS, sparklines, multi-gauge aggregate view, flow range bands. Deploy to h2oflow.org.

**Phase 2 — Reach registry (months 2–3)**
Reach data model + PostGIS. Community editor. Seed Colorado reaches. Bind gauges to reaches. Flow ranges editable. Conditions board and hazard warnings on reach pages.

**Phase 3 — Accounts & community (months 3–4)**
Email/password auth. Trip report filing with auto-stamped CFS. Saved dashboards and alerts. SMS/push notifications. Discord bot phase 1.

**Phase 4 — Trip planning (months 4–5)**
Day trip and overnight planner. Roster. Basic food notes. Export (markdown, PDF, KML/GPX). AI post-trip extraction cards.

**Phase 5 — Permit trip module (months 5–6+)**
Full permit coordination. Outfitter integration. Food planner with dietary restrictions and shopping list. Gear matrix. Cost splitting. Shuttle coordination. Hosted trip page.

**Phase 6 — Public API (parallel to phase 3+)**
Token issuance. Rate limiting. Public API docs at h2oflow.org/developers.

---

## Contributing

**Non-developer contributions:**
- Add or verify reach data for rivers you know
- Define flow ranges for local gauges
- File trip reports with honest conditions
- Post hazard warnings when you encounter them

**Developer contributions:**
- Issues tagged `good-first-issue` are the entry point
- Gauge source adapters are the easiest first contribution
- All PRs require passing tests and a clear description

---

*Last updated: March 2026. Evolves with community input.*
