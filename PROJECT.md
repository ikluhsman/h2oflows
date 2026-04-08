# H2OFlows — Project Context & Decision Log

> This document captures the full planning conversation and all key decisions
> made during the initial project design phase. Keep it at the repo root or
> in docs/ as a reference for onboarding contributors and your future self.
>
> Last updated: March 2026

---

## Project identity

| Item | Value |
|---|---|
| Project name | H2OFlows |
| Canonical domain | h2oflow.org |
| Redirect | h2omotion.org → h2oflow.org |
| GitHub org | h2oflow |
| License | Apache 2.0 |
| Status | Pre-launch, active development |

**Naming principle:** No hardcoded brand strings anywhere in source code. App name, domain, and org name live in environment config only. Rebranding is a one-line change.

---

## What we're building and why

The whitewater community currently uses a fragmented stack: USGS/Grafana dashboards for streamflow, Facebook groups for conditions and word-of-mouth, Google Sheets for permit trip logistics, manual KML files for trip maps, and Discord for real-time communication. Nothing talks to anything else.

American Whitewater has the closest thing to a unified platform but their reach data is proprietary and their tools are outdated. RiverApp is commercial and closed. There is no open, community-owned alternative.

H2OFlows is the open data layer the whitewater community should have had for years. The core insight: **this is a data platform, not a social network**. Discord, Facebook, and SMS handle the social layer well enough. The gap is structured, queryable, geographically precise river data tied to real-time conditions — accessible to anyone via a free API.

---

## The wedge feature

**Multi-gauge aggregate dashboard.** A paddler checks 3 rivers every morning before deciding where to go. Right now that's 3 tabs, 3 different graph scales, manual mental comparison. H2OFlows shows all 3 on one graph, same scale, with community-defined flow range bands (green = optimal, yellow = high, red = too much) overlaid. One glance answers the question.

This is the feature that gets the first 30 users. Everything else follows.

---

## Target users

- Solo kayakers and small crews checking flows for day trips
- Overnight multi-day trip groups (up to ~8 paddlers)
- Permit trip coordinators for Grand Canyon, Gates of Lodore, Westwater, Cataract Canyon (8–16+ paddlers, weeks of planning)
- Local paddling communities who want to replace Facebook groups for conditions reporting
- Developers and other apps who want to build on open river data via the API

Initial beta group: ~20–30 Colorado paddler friends. Colorado-centric launch, expand from there.

---

## What this is NOT

- Not a social network (Discord/Facebook already do this)
- Not a replacement for recreation.gov permit applications
- Not a commercial product (core platform stays free and open)
- Not trying to compete with RiverApp feature-for-feature at launch

---

## Tech stack decisions

### Backend: Go + Chi

**Why Go:** Excellent concurrency model for polling multiple gauge APIs on schedules, strong standard library means fewer dependencies, single-binary deployment makes ECS Fargate operations trivial. Learning curve for the primary developer but porting existing USGS/DWR logic is the ideal first Go project.

**Why Chi:** Lightweight, idiomatic, no magic. Good FOSS contributor community.

**Key Go packages:**
- `chi` — router
- `pgx` — PostgreSQL driver
- `golang-migrate` — migrations
- `go-redis` — Redis client

### Database: PostgreSQL + PostGIS

River data is fundamentally geographic. PostGIS is the only correct choice. Enables real spatial queries: "show me reaches within 50 miles of Colorado Springs that are class III or below at current flows."

### Frontend: Nuxt 4 + Nuxt UI Pro + Tailwind

**Why Nuxt 4:** Primary developer knows Vue/Nuxt well. SSR means reach pages are indexable — organic SEO is the primary discovery mechanism for a FOSS project with no marketing budget. Strong PWA support via @vite-pwa/nuxt.

**Why Nuxt UI Pro:** Polished dashboard layouts, stat cards, data tables, and navigation components built on Tailwind and Radix. For a project where the UI needs to feel slick to attract users, starting with Pro components and customizing beats building on raw Tailwind. Per-project license is reasonable.

**Why uPlot for gauge graphs:** Renders 100k+ data points smoothly. Extremely lightweight. Handles time-series data better than Chart.js or Recharts. Ideal for the multi-gauge aggregate view.

**Why MapLibre GL:** FOSS Mapbox fork. No API key required. Vector tiles render beautifully. `vue-maplibre-gl` wrapper makes it natural in a Nuxt app.

### PWA → App Store strategy

PWA first. When ready for app stores, wrap with Capacitor (Ionic team) — still the same Nuxt/Vue PWA underneath, just with a thin native container. Gets "download on App Store" credibility without maintaining a separate native codebase. Native apps only if PWA proves insufficient for specific features.

### Infrastructure

| Component | Choice | Notes |
|---|---|---|
| Reverse proxy | Traefik | Docker-native, automatic TLS |
| Cache | Redis | Gauge data TTLs, alert queue |
| Object storage | Cloudflare R2 | No egress fees, S3-compatible. |
| Auth | JWT + email/password | No social login — paddlers are privacy-conscious |
| Discord bot | Go (same monorepo) | Phase 1: incoming webhooks only. Phase 2: slash commands. |

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
│   ├── gauge-core/           gauge source adapters
│   │   ├── interface.go      GaugeSource interface
│   │   ├── usgs.go           USGS NWIS adapter
│   │   └── dwr.go            Colorado DWR adapter
│   ├── river-data/           reach schema, seed data, OSM import tools
│   └── ui/                   shared Vue components if needed
├── infra/
│   ├── docker-compose.yml
│   └── traefik/
└── docs/
    ├── architecture.md       full technical architecture
    ├── api.md                public API reference
    └── contributing.md
```

---

## Data model decisions

### The three-layer model

```
Public commons     → read-only, no auth, community-contributed
Local device       → no account required, full read/write, works offline
Cloud sync         → authenticated, optional, backup + collaboration
```

Trip data is local-first. The app must work at the put-in with no signal.

### Key design decisions

**`cfs_at_run` stamped on trip_reports** is the killer feature. Trip reports indexed to actual water conditions rather than just dates. "Show me trip reports for the Numbers between 800-1000 CFS" becomes a real query.

**`flow_ranges` are community-editable per gauge.** The difference between "minimum" and "optimal" on a specific run is local knowledge. This table encodes it as structured data.

**`reach_flow_difficulty` links difficulty to flow bands.** A run that's class III at 500 CFS and class V at 2500 CFS is two different rivers. Nobody models this well currently.

**`reach_conditions` auto-expires after 7 days.** Conditions intel goes stale. The staleness indicator is part of the UI — "reported 6 days ago at 340 CFS, currently 580 CFS" tells you exactly how much to trust it.

**Difficulty stored as floats, rendered as Roman numerals.** `3.0` → "III", `4.5` → "IV+", `5.0` → "V". Enables real numeric queries and filtering.

**Reach relevance scored by usage, not editorial curation.** Searches, views, trip reports, gauge activity, and watchlist counts feed a nightly-computed score. Popular reaches float up naturally.

### Put-in/take-out detection

Passive crowdsourced GIS. Opt-in explicitly. Raw GPS tracks never leave the device — only derived point candidates are contributed. Manual markers from verified local paddlers score highest and anchor the system. Passive detection fills in the long tail — the obscure creek run that 8 people do every spring that was never manually defined.

---

## Gauge data sources

**Primary sources (launch):**
- USGS Water Services API — `https://waterservices.usgs.gov/nwis/iv/` — no API key, open, covers most of US. Parameter `00060` = CFS discharge, `00065` = gage height.
- Colorado DWR API — existing Prometheus exporter logic to be ported to Go adapter

**Reference:** Developer has existing USGS and DWR Prometheus blackbox exporters that serve as the specification for the Go adapter implementations.

**Future sources:**
- CDEC (California)
- Environment Canada
- Manual/community readings for reaches with no telemetry
- Word-of-mouth source type for informal reports

**Plugin interface:**
```go
type GaugeSource interface {
    FetchReading(externalID string) (*Reading, error)
    FetchHistory(externalID string, since time.Time) ([]*Reading, error)
    Name() string
    SourceType() SourceType
}
```

Adding a new data source = writing one file implementing this interface.

---

## Reach data bootstrap strategy

American Whitewater's dataset is proprietary. The open alternative:

1. **Seed from OpenStreetMap** — whitewater tagging schema (`whitewater:section_grade`, put-in/take-out nodes) exists and is open licensed. Incomplete and inconsistent but importable.
2. **Manual verification by beta group** — top 50 Colorado runs defined by the ~30 paddler beta group in an initial sprint. These are rivers everyone knows personally.
3. **Passive detection over time** — opt-in GPS data from app usage converges on additional put-in/take-out points organically.
4. **Relevance scoring** — usage data surfaces which reaches matter to the community without editorial curation.

Colorado first. The beta group can personally verify every reach in the initial seed dataset.

---

## AI contribution layer

**Principle:** AI reads what users naturally write and maps it to database fields. Humans confirm every output. The AI never writes to the database autonomously.

**Post-trip flow:**
1. User writes natural trip notes
2. API call sends notes + current reach database state to AI
3. AI returns structured JSON — hazard candidates, flow range confirmations, put-in suggestions
4. Frontend renders as contextual confirmation cards
5. User taps confirm/skip on each card
6. Confirmed items write to database

**Also generates:** Blog post drafts from trip metadata + auto-stamped flow data + weather summary + food log + user notes. User edits rather than writes from scratch.

**Not a chatbot.** Contextual nudge cards at natural moments in the trip flow.

---

## Discord integration

Community context: Colorado Whitewater Discord server already exists and has the target community. H2OFlows doesn't replace it — it feeds data into it.

**Phase 1 (launch) — explicit commands only:**
```
!hflow hazard arkansas-numbers "new strainer pine creek river left"
!hflow conditions poudre-mishawaka 340 "tobin clean, picnic washed out"
!hflow flow arkansas-numbers
!hflow alert set cache-la-poudre 150 250
!hflow help
```
Every write command returns a one-click web confirmation before touching the database.

**Phase 2 — keyword nudges in designated channels only**
**Phase 3 — AI-assisted extraction with human confirmation**
**Phase 4 — auto-post (only if community explicitly wants it, may never happen)**

Phase 3 with human confirmation is likely the correct permanent state. Data quality depends on it.

**Outbound alerts push to `#h2oflow-alerts` (read-only bot channel):**
- Hazard warnings with flow context
- Flow threshold alerts
- Weekly conditions digest (optional)

---

## Trip export formats

| Format | Use case |
|---|---|
| Markdown | Static site generators, Obsidian, any text tool |
| PDF | Printable pre-trip binder or post-trip archive |
| KML | Google Earth — developer used to do this manually for Grand Canyon trips |
| GPX | Gaia GPS, CalTopo, backcountry navigation tools |
| Hosted trip page | Public shareable URL with embedded map, flow graph, photos, video |

**Hosted trip page includes:**
- River, dates, party size, craft types
- Flow conditions with graph (auto-stamped)
- Weather summary for trip dates (Open-Meteo API, free)
- Day-by-day narrative (user-written)
- Geotagged photo map (EXIF GPS → auto-pinned, river-following line from reach centerline)
- Embedded YouTube/Vimeo (no raw video hosting)
- Food log (if food planner used)
- Conditions summary and incident notes

---

## Public API design

H2OFlows is the open data layer. The app is just the first consumer of its own API.

Every endpoint built for the frontend is also a public API endpoint. API token issuance starts in Phase 3.

**Philosophy:** Data flows freely. Rate limits prevent abuse, not access. Attribution requested culturally, not enforced technically (OSM model).

**Downstream uses envisioned:**
- Discord bots (the Colorado Whitewater server can embed live gauge widgets)
- Slack integrations for paddling crews
- Third-party trip planning tools
- Academic streamflow research
- Commercial outfitter tools (commercial tier, back-burner)

---

## Subscription / sustainability model

Core platform: **free forever.** The FOSS commitment is real.

Sustainability path:
- **Supporter tier ($3-5/month):** Photo storage, supporter badge. Framed as "support the project, get photo storage as thanks" — not feature gating.
- **Commercial API tier (back-burner):** Outfitters building client-facing tools. Elevated rate limits, SLA.

Photo storage on R2 is cheap enough at community scale that the supporter tier covers infrastructure costs without needing many subscribers.

---

## Build order (6-month plan)

### Phase 1 — Gauge dashboard (months 1–2)
**Goal:** Something real at h2oflow.org that a Colorado paddler finds useful today.

- Stand up monorepo, Docker Compose, Traefik
- Port USGS/DWR exporter logic to `gauge-core` Go package (first Go project)
- Build public gauge dashboard in Nuxt 4 / Nuxt UI Pro
  - Live CFS display with sparklines
  - Multi-gauge aggregate view (the wedge feature)
  - Community flow range bands overlaid on graphs
  - Mobile-first, PWA-ready
- Deploy to h2oflow.org
- No account required for any read access

**Success metric:** Beta paddlers use it instead of their own Grafana dashboards.

### Phase 2 — Reach registry (months 2–3)
- Reach data model + PostGIS setup
- Simple community editor (web UI)
- OSM whitewater data import tooling
- Manual seed of top 50 Colorado reaches by beta group
- Bind gauges to reaches
- Community-editable flow ranges
- Reach pages: conditions board, hazard warnings, recent trip reports

### Phase 3 — Accounts & community layer (months 3–4)
- Email/password auth, JWT
- Trip report filing with auto-stamped CFS
- Saved gauge dashboards and alert thresholds
- SMS / push notifications
- Discord bot phase 1 (commands + outbound alerts)
- Contribution scoring
- API token issuance

### Phase 4 — Trip planning (months 4–5)
- Day trip and overnight planner
- Roster management
- Basic food notes
- Trip export (markdown, PDF, KML/GPX)
- AI post-trip extraction and confirmation cards
- Geotagged photo map on trip pages

### Phase 5 — Permit trip module (months 5–6+)
- Full permit trip coordination
- Outfitter integration
- Complete food planner (dietary restrictions, shopping list, weight/bulk estimates)
- Gear matrix with outfitter-provided items flagged
- Cost splitting and expense ledger
- Shuttle coordination
- Hosted trip page / blog export

### Phase 6 — Public API (runs parallel to phase 3+)
- Token issuance and management
- Rate limiting
- Public API documentation at h2oflow.org/developers
- API changelog

---

## First steps (this week)

1. Register h2oflow.org
2. Create GitHub org `h2oflow`
3. Create repo `h2oflow` (monorepo)
4. Add `docs/architecture.md` (see architecture.md)
5. Write minimal `README.md` (vision statement, link to architecture doc)
6. Add `LICENSE` (Apache 2.0)
7. Stub `docker-compose.yml`
8. Start `packages/gauge-core/` — port existing USGS exporter to Go

---

## Key references

- USGS Water Services API: https://waterservices.usgs.gov/rest/IV-Service.html
- Colorado DWR: https://dwr.state.co.us/rest/get/help
- OpenStreetMap whitewater tagging: https://wiki.openstreetmap.org/wiki/Whitewater_sports
- MapLibre GL: https://maplibre.org
- vue-maplibre-gl: https://vue-maplibre-gl.web.app
- uPlot: https://github.com/leeoniya/uPlot
- Nuxt UI Pro: https://ui.nuxt.com/pro
- golang-migrate: https://github.com/golang-migrate/migrate
- Chi router: https://github.com/go-chi/chi
- Capacitor (PWA → App Store): https://capacitorjs.com

---

## Open questions (decide as they come up)

- OSM whitewater data import: how to handle inconsistent tagging schema across regions
- Offline sync conflict resolution when same trip edited on two devices
- Discord bot hosting — same server as API or separate lightweight instance
- Open-Meteo vs other weather API for auto-stamping weather on trip reports
- Whether Phase 4 photo map requires a separate tile server or MapLibre handles it from reach centerline geometry alone

---

*This document is a living record of project decisions. Update it as the project evolves.*
