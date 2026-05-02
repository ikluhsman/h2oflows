# H2OFlows Roadmap

Current state as of May 2026. Phase 1 (gauge dashboard + reach pages + AI assistant) is complete. Backend routes exist for trip reports, trips, contributions, and proximity events — the frontend for those is stub. Everything below is unbuilt or incomplete.

---

## Phase 2 — Community data layer

*The contribution pipeline. Turn solo users into a data network.*

### Trip reports (backend done, frontend stub)

- **Filing UI** on each reach page — reach, date, craft, flow impression, conditions freetext, optional photos
- CFS at run auto-stamped from gauge reading at `run_date` (gauge closest to put-in)
- `class_felt` slider — "how did this feel at that flow?" — feeds flow-band accuracy over time
- Published reports visible on reach page with flow context (was it runnable? pushy?)
- Privacy toggle: private (default) / community / public
- **Social sharing** — one-tap share to Instagram/Facebook/SMS; shared link renders reach name, CFS, flow band, and photo as OG image (see Phase 3 SEO)
- Trip report slug at `/trip-reports/{slug}` — shareable, crawlable

### Community conditions board

- Short-lived intel posted to any reach: word-of-mouth, personal, outfitter, Discord source
- Auto-expires 7 days
- Runnable boolean — quick gut check from the paddler who just got off the water
- Surfaced on reach page above the fold, sorted by recency

### Hazard warnings UI

- Backend routes (`GET/POST /reaches/{slug}/hazards`) already exist
- Reach page section: active hazards with type badge, description, CFS at report
- Report-a-hazard form: type (strainer/sieve/undercut/low-head dam/other), location on map, description
- Admin: mark resolved, add resolution note

### Proximity events + passive telemetry

- Backend route (`POST /proximity-events`) exists
- Mobile web: opt-in proximity detection near known put-ins/take-outs
- Aggregate signals feed put-in/take-out confidence scoring (see ARCHITECTURE.md)
- Never send raw GPS — only derived point candidates
- Settings page: toggle telemetry contribution on/off

---

## Phase 2b — Calculated gauges

*User-defined math on top of real gauge data. The power-user gauge feature.*

### What it is

A calculated gauge is a named formula combining one or more real gauges into a derived reading — displayed as a gauge card on the dashboard with its own flow bands, graph, and sparkline. Examples:

- **Sum two tributaries**: `cache-la-poudre-canyon + cache-la-poudre-above-rustic` → estimated mainstem below confluence
- **Ratio**: `gauge-A / gauge-B` → fraction of historical median, normalized runability
- **Offset**: `arkansas-nathrop - 150` → adjusted reading accounting for known diversion
- **Stage conversion**: `gauge-A * 3.7 + 42` → custom CFS estimate from a stage-only gauge using a local rating curve

### Formula engine

- Supported ops: `+`, `-`, `*`, `/`, `()`
- Operands: gauge external IDs or calculated gauge IDs (composable)
- Named constants: user-defined scalars stored per formula
- Validated at save time — all referenced gauges must resolve
- Readings computed on-the-fly from latest polled values; no separate polling tier needed
- Historical graph: reconstructed from stored readings for all referenced gauges (back-filled to earliest common timestamp)

### Data model

```sql
CREATE TABLE calculated_gauges (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug         TEXT UNIQUE NOT NULL,        -- user-chosen, URL-safe
  owner_id     UUID REFERENCES users(id),
  name         TEXT NOT NULL,
  formula      TEXT NOT NULL,               -- e.g. "gauge_a + gauge_b * 0.8"
  unit         TEXT DEFAULT 'cfs',
  description  TEXT,
  public       BOOLEAN DEFAULT FALSE,
  created_at   TIMESTAMPTZ DEFAULT NOW()
);

-- References to real gauges used in formula (for dependency tracking)
CREATE TABLE calculated_gauge_inputs (
  calculated_gauge_id  UUID REFERENCES calculated_gauges(id) ON DELETE CASCADE,
  gauge_id             UUID REFERENCES gauges(id),
  PRIMARY KEY (calculated_gauge_id, gauge_id)
);
```

Flow ranges (bands) on a calculated gauge work identically to real gauges — same `flow_ranges` table, same `gauge_id` FK.

### API

```
POST   /calculated-gauges              create (auth required)
GET    /calculated-gauges/{slug}       public if owner marked public
PATCH  /calculated-gauges/{slug}       owner only
DELETE /calculated-gauges/{slug}       owner only
GET    /calculated-gauges/{slug}/readings   computed from inputs' stored readings
GET    /gauges/{id}/calculated         list public calculated gauges that reference this gauge
```

### Dashboard integration

- Calculated gauges appear as first-class cards on the dashboard — same `GaugeCard` component, tagged with a formula icon
- "Build a calculation" button in gauge search modal — opens formula builder
- Formula builder: searchable gauge picker, drag-to-build expression, live preview of current computed value
- Private by default; toggle to public makes it discoverable

### Social / sharing

- Public calculated gauges get a canonical URL: `/gauges/calculated/{slug}`
- OG image (Phase 3): formula name, current computed value, flow band, contributing gauge names
- On each real gauge page: **"Community calculations"** section lists public calculated gauges that reference this gauge — name, description, owner, current value
- "Add to my dashboard" button on any public calculated gauge — one tap, no re-entry of formula
- "Fork this calculation" — copy to own account, edit formula

### Discovery

- `/explore/calculations` — browse public calculated gauges, filterable by river/region
- Linked from reach pages when a calculated gauge is the primary gauge for that reach (admin-assignable)
- Search includes calculated gauge names alongside real gauges

---

## Phase 3 — SEO + Open Graph

*Organic discovery. No marketing budget — make every shared link count.*

### Dynamic OG images

- `/og/reaches/{slug}.png` — reach name, river, class, current CFS, flow band color, reach centerline thumbnail
- `/og/trip-reports/{slug}.png` — reach name, date, CFS at run, conditions summary, optional user photo
- `/og/gauges/{id}.png` — gauge name, current CFS, sparkline, flow status
- Generated server-side (Go + `gg` or headless Chromium); cached in Cloudflare R2

### Reach page SSR meta

- `<title>`, `og:title`, `og:description`, `og:image` populated from reach data + live gauge reading
- Structured data (`application/ld+json`): `Place`, `Event` (for trip reports)
- Canonical URLs for reach slugs

### Shareable links

- Trip report share → OG image with conditions + CFS
- Gauge alert share → "Browns Canyon is running at 850 CFS (optimal)" + link
- Dashboard snapshot URL — encodes current watchlist + gauge readings as shareable link (no account required)
- KML export already exists on reach pages; add GPX

---

## Phase 4 — Public API

*H2OFlows is infrastructure. The app is just the first consumer.*

### Token issuance

- API token tied to user account — issued from profile/settings page
- Token scopes: `read` (public data), `write` (contributions), `elevated` (higher rate limits)
- Tokens stored hashed; revocable from settings

### Rate limiting

- Anonymous: 100 req/hour
- Free token (`read`): 1000 req/hour
- Community contributor: 5000 req/hour (auto-granted on first verified trip report)
- Commercial/outfitter tier: paid, negotiated

### Versioned public endpoints

All under `/api/v1/`. Currently functional but undocumented:

```
GET  /reaches                       paginated, filterable by region/class/state
GET  /reaches/{slug}
GET  /reaches/{slug}/gauges
GET  /reaches/{slug}/conditions
GET  /reaches/{slug}/trip-reports
GET  /reaches/{slug}/hazards
GET  /reaches/{slug}/flow-ranges
GET  /gauges/{id}/readings
GET  /gauges/{id}/readings?from=&to=
GET  /gauges/{id}/flow-ranges
GET  /gauges/{id}/seasonal
POST /reaches/{slug}/conditions      (write token)
POST /reaches/{slug}/hazards         (write token)
POST /reaches/{slug}/trip-reports    (write token)
```

New endpoints needed:
```
GET  /regions                        list states/basins with reach counts
GET  /regions/{slug}/reaches
GET  /reaches?bbox={w,s,e,n}         geographic filter
GET  /gauges?near={lat},{lng}&r={km} proximity search
```

### API docs

- OpenAPI 3.1 spec generated from route annotations or hand-maintained
- Hosted at `/api/docs` — Swagger UI or Scalar
- Attribution: "data sourced from H2OFlows community (h2oflow.org)"

---

## Phase 5 — American Whitewater integration

*Close the loop with the upstream data source.*

### Inbound: AW reach import

- AW exposes a public JSON API (`www.americanwhitewater.org/content/River/list/`)
- Import script: map AW reach ID → H2OFlows slug, pull description + rapids + access
- Store with `data_source='aw_import'`, `aw_reach_id` foreign key
- Diff against existing AI-seeded content; flag conflicts for human review
- One-time bulk import + periodic sync (weekly cron)

### Outbound: contribution pipeline back to AW

- When a trip report, hazard, or conditions update is published on H2OFlows, offer one-tap **"Also post to AW"**
- AW has a submission form API (undocumented but used by their mobile app); reverse-engineer or coordinate directly
- If AW API isn't available: generate formatted AW submission text + deep-link to AW's web form, pre-populated
- Track `aw_synced_at` on contributions — don't double-post
- User controls which data they share externally; default off

### AW reach linking

- Admin tool: search AW by river/state, link AW reach ID to H2OFlows slug
- Linked reaches show "Also on American Whitewater" badge + link
- AW gauge associations imported and cross-referenced with our USGS IDs

---

## Phase 6 — Data admin roles (scoped)

*Trusted local stewards, not just global admins.*

### Role model

Current: `data_admin` (global) and `site_admin` (global). Needed: scoped trust.

```
site_admin           global — full access, role assignment
data_admin           global — all reach/river data
basin_admin          scoped to a drainage basin (e.g. Arkansas River basin)
state_admin          scoped to a state (e.g. Colorado)
reach_steward        scoped to one or more specific reaches
```

### Implementation

- `user_roles` table gains optional `scope_type` (`basin|state|reach`) and `scope_id`
- Auth middleware: `RequireDataAdmin` checks scope before allowing reach mutations
- Admin UI: assign `reach_steward` role to a user + select reaches they steward
- Basin/state scopes defined by PostGIS containment check on reach put-in geometry
- Site admins can grant scoped roles; scoped admins cannot grant roles

### Steward features

- Stewards receive email digest of new trip reports, hazard warnings, and conditions posts for their reaches
- Stewards can verify/reject AI-seeded content on their reaches
- Stewards can close resolved hazards
- "Maintained by [name]" attribution on reach pages for verified stewards

---

## Phase 7 — Alerts + Discord

*Push notifications when the river comes up.*

### User-defined flow alerts

- Alert creation: gauge ID + threshold (min CFS, max CFS, or named flow band)
- Delivery channels: email (Phase 1), SMS (Phase 2, Twilio), push (Phase 2, PWA), Discord DM (Phase 3)
- Alert deduplication: don't re-fire until gauge crosses threshold again after going out of range
- Alert stored in DB; evaluated by poller on each gauge refresh

### Discord bot — Phase 1 (webhook, no OAuth)

Commands via text in designated channels:
```
!hflow flow arkansas-numbers
!hflow conditions poudre-mishawaka 340 "tobin clean, picnic washed out"
!hflow hazard arkansas-numbers "new strainer pine creek river left"
!hflow alert set cache-la-poudre 150 250
```
Every write returns confirmation link before touching DB.

Outbound alerts to subscribed channels:
```
🚨 Hazard — Arkansas / Numbers
Pine Creek Rapid · strainer river left
Reported at 920 CFS (currently 950, rising)
→ h2oflow.org/reaches/arkansas-numbers/hazards
```

### Discord bot — Phase 2 (slash commands + keyword nudges)

- Slash commands registered via Discord app
- Keyword watcher: strainer, hazard, portage, pin, washed out, undercut — nudges author to log it
- Never auto-posts; always prompts human confirmation

---

## Phase 8 — Trip planning

*From quick day trips to full permit expeditions.*

### Day trip planner

- Reach lookup → current conditions summary → shareable link
- Simple itinerary: date, reach, crew size, shuttle plan
- Link-only sharing (no account required to view)

### Overnight trip planner

- Multi-day itinerary builder
- Roster with roles (trip lead, safety, shuttle driver)
- Basic food notes per day
- Export: markdown / PDF / GPX

### AI post-trip extraction

After trip marked complete, AI reads trip notes and surfaces contribution cards:

```
  ✦ Hazard at Pine Creek rapid — new strainer river left
    → Log as hazard warning?  [ Yes ]  [ Edit ]  [ Skip ]

  ✦ You ran this at 850 CFS — community shows 800–1000 as optimal
    → Confirm flow band?  [ Confirm ]  [ Adjust ]  [ Skip ]
```

Never writes to DB without explicit user action.

### Trip export formats

- Markdown (Obsidian, static sites)
- PDF (printable trip binder)
- KML / GPX (Gaia GPS, CalTopo, Google Earth)
- Hosted trip page: flow graph, geotagged photo map, embedded video, food log, conditions summary

---

## Phase 9 — Permit trip module

*Full expedition coordination. Post-v1.*

- Full roster with roles, emergency contacts, dietary restrictions
- Gear matrix: who brings what, weight tracking
- Food planner: per-day meals, quantities, cook assignments
- Cost splitting: gear rental, shuttle, food, permit fees
- Shuttle coordination: vehicle assignments, meetup times, parking logistics
- Outfitter integration: guided trip roster management, paid API tier
- Permit tracking: application deadlines, lottery status, permit scan storage

---

## Phase 10 — Native mobile apps

*PWA first; native for GPS and push.*

- Capacitor-based iOS and Android apps wrapping the Nuxt PWA
- Background GPS for passive put-in/take-out detection (opt-in)
- Offline reach + gauge cache — works at the put-in without signal
- Push notifications for flow alerts
- Photo capture tied to trip reports — EXIF GPS auto-pins to reach map
- App Store and Google Play distribution

---

## Non-goals (intentionally out of scope)

- Social graph / follows / likes — Discord, Instagram, and SMS do this
- Photo/video hosting as a primary feature — R2 storage for trip reports only, not a media platform
- Outfitter booking / transactional flows — outfitter API for data only; booking stays on their platforms
- International reach registry — US-first until data model is proven; gauge adapters already extensible

---

## Gauge adapter backlog

New sources require one file in `packages/gauge-core`:

| Source | Priority | Notes |
|---|---|---|
| CDEC (California) | High | Covers Sierra + N. California runs |
| Environment Canada | Medium | BC, Alberta, Quebec paddling |
| USGS stage-only gauges | Medium | Paramter `00065` instead of `00060` |
| Manual / community gauge | Low | Spreadsheet-defined readings for ungauged runs |
