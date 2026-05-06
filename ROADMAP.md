# H2OFlows Roadmap

Current state as of May 2026. Phase 1 (gauge dashboard + reach pages + AI assistant) is complete. Backend routes exist for trip reports, trips, contributions, and proximity events — the frontend for those is stub. Everything below is unbuilt or incomplete.

---

## Phase 2 — Pilot polish + personal data layer

*Attractive interface + pilot onboarding. Private user reaches and custom gauges before any community/social features.*

### 2.1 — Flow band simplification

Three bands replace the existing five. Fixed names and colors — users only set CFS threshold values. `craft_type` column dropped from `flow_ranges` (kayak/raft/sup distinction not used in pilot).

| Band | Color | Stored values |
|---|---|---|
| `low` | red | `max_value` only |
| `running` | green | `min_value` + `max_value` |
| `high` | blue | `min_value` only |

**Migration from existing 5-tier schema** (`below_recommended` / `low_runnable` / `runnable` / `high_runnable` / `above_recommended`):

- `low.max` = `below_recommended.max`
- `running.min` = `COALESCE(low_runnable.min, runnable.min, high_runnable.min)` — lowest available bottom across the runnable tiers
- `running.max` = `COALESCE(above_recommended.min, high_runnable.max, runnable.max)` — highest available top
- `high.min` = same value as `running.max` (boundary mirrors)
- `low_runnable`, `runnable`, `high_runnable` collapse into the single `running` band — all three runnable tiers fold together

**Coloring rule (web):**
- reading ≤ `running.min` → red (low)
- reading ≥ `running.max` → blue (high)
- else → green (running)

`low.max` and `high.min` are persisted for future visual gradient or admin reference; primary classification uses `running.min` / `running.max`.

**Migration 000068 highlights:**
- Drop `craft_type` column from `flow_ranges`; replace `(reach_id, label, craft_type)` unique constraint with `(reach_id, label)`
- Aggregate per-reach into 3 rows; preserve `data_source` (default `manual`) and `verified` flag
- Replace CHECK constraint: `label IN ('low','running','high')`
- Temporary `legacy_band_data JSONB` column retained on modified rows during migration window for rollback. Dropped after Phase 2 ships.

**UI sweep:** admin reach form, gauge modal, flow badges, ReachMap pins, GaugeCard, Sparkline, graph thresholds.

---

### 2.2 — Admin reach workflow

**Rivers tab restructure:**
- Group: state → basin → river → reach
- Pagination 10 / 50 / 100, default 50
- "Needs review" sub-section at top: rivers with `verified = false` (auto-created from user reach saves in 2.4)

**New reach flow (progressive, admin mode):**

1. Click "New reach" → enter pick-anchor mode immediately, no toggle required
2. Helper: "Find the start point for your river or creek. Tap the river as close to the start point as possible."
3. Anchor selected → "Pick another point" and "Clear" buttons appear. Re-pick replaces anchor; clear resets map.
4. Helper updates: "Tap the river as close to the put-in (starting point) as possible. Try satellite view to find the boat ramp."
5. Take-out selected → auto-trim and preview centerline immediately. No "Save flowlines" button.
6. Auto GNIS lookup → display "Looks like Trout Creek"
7. Full admin form: slug, common name, class, description, multi-day, permit, flow band thresholds, gauge
8. Click "Save reach" → GNIS confirm prompt ("Trout Creek, basin: South Platte, state: CO?" with manual override) → river auto-created with `verified = false` if no GNIS match → redirect to reach detail page
9. If gauge is new to system → warn "This gauge was just added. Polling starts within ~15 minutes."

User reach flow (2.4) reuses map steps 1–6, then a slim form.

---

### 2.3 — Custom gauges

A custom gauge is a named sum or difference of real gauges. Produces a CFS reading. Private to owner. Stored in its own table — distinct from admin singular `gauges`, which remain a separate concept.

**Operations:** `+` and `-` only. No multiply, divide, parens, or constants — additive/subtractive watershed flow modeling only.

**Standalone:** can exist without a reach. Dashboard card shows computed CFS + custom-gauge icon (calc icon), no sparkline. Clicking opens a modal with a stacked graph of all contributing real gauges. Single-input custom gauges allowed (acts as a labeled passthrough; modal shows one trace).

**Colorization:** raw CFS only on standalone card — no band color without a reach. When a custom gauge backs a user reach, the reach's flow band thresholds determine card color.

**Data model (migration 000070):**

```sql
CREATE TABLE custom_gauges (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id        TEXT NOT NULL,
  slug            TEXT NOT NULL,
  name            TEXT NOT NULL,
  description     TEXT,
  note            TEXT,
  unit            TEXT NOT NULL DEFAULT 'cfs',
  last_value_cfs  NUMERIC,
  last_value_at   TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (owner_id, slug)
);
CREATE INDEX custom_gauges_owner_idx ON custom_gauges (owner_id);

CREATE TABLE custom_gauge_inputs (
  custom_gauge_id  UUID NOT NULL REFERENCES custom_gauges(id) ON DELETE CASCADE,
  position         SMALLINT NOT NULL,
  gauge_id         UUID NOT NULL REFERENCES gauges(id) ON DELETE RESTRICT,
  sign             SMALLINT NOT NULL CHECK (sign IN (-1, 1)),
  PRIMARY KEY (custom_gauge_id, position)
);
CREATE INDEX custom_gauge_inputs_gauge_idx ON custom_gauge_inputs (gauge_id);
```

`owner_id` is `TEXT` to match Supabase auth user IDs — same pattern as `user_roles` and `user_watchlists`. No `users` table in DB.

`gauge_id` uses `ON DELETE RESTRICT`: prevents accidental loss of a custom gauge's input when an admin deletes the underlying gauge — admin must explicitly migrate or break the formula first.

No `public` flag. No subscriber tracking. No flow ranges on custom gauges — bands belong to reaches.

**Slug uniqueness:** scoped per owner (`owner_id, slug`). Form checks availability before save and blocks on collision.

**Delete:** hard delete, cascades inputs. Blocked if a user reach currently uses this gauge — form warns "Reach `xyz` uses this gauge. Reassign reach gauge or delete the reach first."

**Polling integration:**

Current poller (mig 000065) polls only gauges subscribed via `gauge_reach_associations`. Custom gauge inputs and user-reach gauges must also drive polling.

Migration 000072 — replace polling source with a union view:

```sql
CREATE OR REPLACE VIEW polled_gauge_ids AS
  SELECT DISTINCT gauge_id FROM gauge_reach_associations
  UNION
  SELECT DISTINCT gauge_id FROM custom_gauge_inputs
  UNION
  SELECT DISTINCT primary_gauge_id AS gauge_id
    FROM user_reaches WHERE primary_gauge_id IS NOT NULL;
```

Poller selects from `polled_gauge_ids` instead of `gauge_reach_associations` directly. Adding a custom gauge auto-enrolls its inputs. Cascade delete on inputs auto-de-enrolls gauges no longer needed by anyone.

**Custom gauge value computation:**

After each poll cycle, a worker pass recomputes every custom gauge:

- `value = SUM(latest_reading × sign)` over inputs
- writes `last_value_cfs` and `last_value_at` on `custom_gauges`
- if any input gauge has `poll_health` worse than `healthy` (see 2.5), the worker still computes a value but flags it stale — UI shows "depends on stale gauge: [name]"

**Formula builder UI:**
- Searchable real gauge picker (by name, river, station ID)
- Add gauges row by row with +/- toggle
- Drag handles to reorder rows
- Live preview of computed current value
- Note field (owner-visible, editable — not RAG-indexed)
- Save → owner-only, no public toggle

**API:**

```
POST   /me/custom-gauges
GET    /me/custom-gauges
GET    /me/custom-gauges/{slug}
PATCH  /me/custom-gauges/{slug}
DELETE /me/custom-gauges/{slug}
GET    /me/custom-gauges/{slug}/readings
```

All routes require auth. Slug resolved against the authenticated session's user — no `{handle}` in URL needed. Owner check enforced on every path.

**Readings computation:** on-the-fly from latest polled values of contributing gauges. Historical graph: reconstructed by joining stored `gauge_readings` over a common timestamp window across inputs.

**Watchlists extended:** migration 000074 adds `custom_gauge_id` (nullable) to `user_watchlists` so users can pin custom gauges to dashboard the same way as real gauges. CHECK enforces that exactly one of `gauge_id` / `custom_gauge_id` is set.

**Export / share via payload (no DB sharing):**

Tapping "Share" on a custom gauge generates a portable payload — a snapshot of the formula only, not a DB record. Recipient imports it as their own independent copy. Pattern is similar to Grafana's dashboard JSON export/import.

Payload format (compact JSON, base64url-encoded for URL transport):

```json
{
  "v": 1,
  "n": "Cache la Poudre Confluence Estimate",
  "d": "Optional description",
  "i": [
    {"s": 1, "g": "USGS:09058000"},
    {"s": 1, "g": "USGS:09060500"},
    {"s": -1, "g": "USGS:09057500"}
  ]
}
```

`g` = gauge external ID prefixed by source (`USGS:`, `DWR:`) — resolves across any user's account. Import fails with a clear error if a gauge isn't in the system; offers to add it from USGS/DWR before retry.

Share modal options:
- "Copy as message" — human-readable text + import link
- "Copy import link" — raw URL (`/import/gauge?d=<base64>`)
- Social intents (Twitter, SMS, Discord) — text + link

Import flow: link opens formula builder pre-filled. Slug collision prompts user to rename before save. The payload has no reference back to the original — once imported, edits diverge.

QR code sharing deferred to a later phase.

---

### 2.4 — User-defined reaches

Private reaches any authenticated user can create. Stored in a separate table from curated `reaches` so curated and user spaces never cross-contaminate by query oversight.

**Rivers stay shared.** When a user saves a reach for a river not in `rivers`, the row is auto-created with `verified = false`. Admin Rivers tab surfaces unverified rows for review. No `owner_id` on rivers — rivers are physical entities, deduped by GNIS lookup across all users.

Migration 000069 adds `verified BOOLEAN NOT NULL DEFAULT FALSE` to `rivers`. Existing curated rivers backfilled to `true`.

**Schema (migration 000071):**

```sql
CREATE TABLE user_reaches (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id          TEXT NOT NULL,
  slug              TEXT NOT NULL,
  name              TEXT NOT NULL,
  river_id          UUID REFERENCES rivers(id) ON DELETE SET NULL,
  put_in            GEOGRAPHY(POINT, 4326) NOT NULL,
  take_out          GEOGRAPHY(POINT, 4326) NOT NULL,
  centerline        GEOGRAPHY(LINESTRING, 4326),
  primary_gauge_id  UUID REFERENCES gauges(id) ON DELETE SET NULL,
  custom_gauge_id   UUID REFERENCES custom_gauges(id) ON DELETE SET NULL,
  note              TEXT,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (owner_id, slug),
  CHECK (primary_gauge_id IS NULL OR custom_gauge_id IS NULL)
);
CREATE INDEX user_reaches_owner_idx ON user_reaches (owner_id);
CREATE INDEX user_reaches_river_idx ON user_reaches (river_id);

CREATE TABLE user_reach_flow_ranges (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_reach_id  UUID NOT NULL REFERENCES user_reaches(id) ON DELETE CASCADE,
  label          TEXT NOT NULL CHECK (label IN ('low','running','high')),
  min_value      NUMERIC,
  max_value      NUMERIC,
  UNIQUE (user_reach_id, label)
);
```

A user reach uses **either** a real gauge or a custom gauge — never both. CHECK enforces it.

**Slug rules:** auto-generated from name on first save, editable. Unique per owner, not globally — two users can each own a `clear-creek-section-1`. Server validates uniqueness against `(session.owner_id, slug)` before insert.

**Slim form (user mode, post-map flow):**
- Reach name (required)
- Optional note
- Gauge selection: real gauge picker OR "My Gauges" (custom gauges)
- 3 flow band threshold values (low max, running min, running max — high min auto-mirrors running max)
- Omits: slug input (auto-generated, optional override link), common name, class definition, description, multi-day, permit

**Save flow:**
- GNIS confirm prompt same as admin flow
- River auto-created with `verified = false` if no GNIS match
- New gauge warning same as admin flow ("Polling starts within ~15 minutes.")
- Redirect to user reach detail page after save

**Reach detail page (user reach):**
- Shows computed gauge reading with flow band color, reach map, note field (editable for owner)
- "Add to dashboard" button
- 404 for non-owner (no existence leak)
- `noindex, nofollow` meta

**URLs:**
- Curated: `/reaches/{slug}` — public, indexed
- User reach: `/my/reaches/{slug}` — owner auth required, slug resolved against session owner_id; not addressable by any other user

**Delete:** hard delete. Dashboard cards referencing the reach removed silently. Associated custom gauge (if any) survives in owner's library — only the reach link is broken.

**"My Reaches" page (avatar menu):**
- Not a top-level nav tab — lives under avatar menu
- Layout: state → basin → river → reach grouping, same as admin Rivers tab
- Pagination 10 / 50 / 100, default 50
- Per-row actions: edit, delete, add to dashboard

**"My Gauges" page (avatar menu):**
- Lists owner's custom gauges
- Per-row actions: edit, delete, share (payload), add to dashboard
- Same pagination

**Explore page change:** "+" button made prominent so users without admin access discover reach creation. Links to user reach creation flow (slim form path).

**Trip reports / hazards / conditions (Phase 2b):** writes blocked against `user_reaches`. Community data layer applies only to curated `reaches` so moderation surface stays bounded. User reaches remain personal-use only.

---

### 2.5 — Polling resilience

Gauges are not manually retired — sources (USGS, DWR via NLDI) decide when a gauge stops reporting. We surface poll health instead of curating gauge lifecycle.

**Schema (migration 000073):**

```sql
ALTER TABLE gauges
  ADD COLUMN consecutive_poll_failures INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN last_poll_failure_at      TIMESTAMPTZ,
  ADD COLUMN last_poll_success_at      TIMESTAMPTZ,
  ADD COLUMN poll_health               TEXT NOT NULL DEFAULT 'healthy'
    CHECK (poll_health IN ('healthy','degraded','stale','unreachable'));
CREATE INDEX gauges_poll_health_idx ON gauges (poll_health) WHERE poll_health <> 'healthy';
```

**Poller logic (15-minute cadence — pilot baseline):**

| Failures | Health | Action |
|---|---|---|
| 0 | `healthy` | normal cadence |
| 2 (~30 min) | `degraded` | normal cadence; UI badge appears |
| 4 (~1 hr) | `stale` | normal cadence; reach pages show "data stale" banner |
| 48 (~12 hr) | `unreachable` | back off to 1× per hour; admin alert in Rivers tab |
| 7 days unreachable | `unreachable` | log warning; flag on admin dashboard for swap |

Success at any state resets `consecutive_poll_failures = 0`, sets `last_poll_success_at`, returns to `healthy`.

**UI surfacing:**
- Gauge card: "stale" / "unreachable" badge with last successful timestamp when not healthy
- Reach detail: banner above gauge graph when reach's gauge is `stale` or worse
- Admin Rivers tab: per-river health summary; reaches needing gauge swap surfaced
- Custom gauge: if any input gauge unhealthy, the computed value flagged stale on the dashboard card

No automatic retirement — user / admin decides whether to swap a reach's gauge. The existing `gauges.status` enum (`active|seasonal|inactive|retired|maintenance`) remains untouched and continues to serve manual admin lifecycle decisions; `poll_health` is orthogonal.

---

### 2.6 — Discovery + dashboard distinctions

**Add gauge / add reach search:**
- Default tab: curated h2oflows reaches/gauges
- Second tab: "My Reaches & Gauges" — owner-only personal items
- Import button next to search bar: "Import from share code" → payload paste dialog
- No public/community tab — sharing is point-to-point via payload only

**Dashboard card icons:**
- Curated reach card: H2OFlows badge
- User reach card: just a regular "user" icon, like the blacked-out headshot default avatar
- Curated gauge card: H2OFlows logo
- Custom gauge card: calc icon + "calculated" label, no sparkline (single trace only on click-through modal)

---

### Migration sequence

```
000068_flow_bands_three_tier.up.sql        (2.1: 5→3, drop craft_type)
000069_rivers_verified_flag.up.sql         (rivers.verified for review queue)
000070_custom_gauges.up.sql                (custom_gauges + custom_gauge_inputs)
000071_user_reaches.up.sql                 (user_reaches + user_reach_flow_ranges)
000072_polled_gauge_ids_view.up.sql        (poll source = union view)
000073_gauges_poll_health.up.sql           (2.5 health columns)
000074_user_watchlists_custom_gauge.up.sql (watchlist custom_gauge_id column)
```

Each migration self-contained, reversible. Order matters: 68 first (band format change touches admin form before any new table references flow ranges); 70 + 71 must precede 72 (view depends on both); 74 depends on 70.

---

## Phase 2b — Community data layer

*The contribution pipeline. Turn solo users into a data network. Deferred from original Phase 2.*

### Trip reports (backend done, frontend stub)

- Filing UI on each reach page — reach, date, craft, flow impression, conditions freetext, optional photos
- CFS at run auto-stamped from gauge reading at `run_date` (gauge closest to put-in)
- `class_felt` slider — "how did this feel at that flow?" — feeds flow-band accuracy over time
- Published reports visible on reach page with flow context (was it runnable? pushy?)
- Privacy toggle: private (default) / community / public
- Social sharing — one-tap share to Instagram/Facebook/SMS; shared link renders reach name, CFS, flow band, and photo as OG image (see Phase 3 SEO)
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

## Phase 3 — SEO + Open Graph

*Organic discovery. No marketing budget — make every shared link count. Curated content only.*

### Dynamic OG images

- `/og/reaches/{slug}.png` — reach name, river, class, current CFS, flow band color, reach centerline thumbnail
- `/og/trip-reports/{slug}.png` — reach name, date, CFS at run, conditions summary, optional user photo
- `/og/gauges/{id}.png` — gauge name, current CFS, sparkline, flow status
- Generated server-side (Go + `gg` or headless Chromium); cached in Cloudflare R2

User reaches and custom gauges excluded — non-permanent pages, no indexing.

### Reach page SSR meta

- `<title>`, `og:title`, `og:description`, `og:image` populated from reach data + live gauge reading
- Structured data (`application/ld+json`): `Place`, `Event` (for trip reports)
- Canonical URLs for curated reach slugs

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

- When a trip report, hazard, or conditions update is published on H2OFlows, offer one-tap "Also post to AW"
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
- Public sharing of user-defined reaches or custom gauges — private only; share formula payloads via message instead

---

## Gauge adapter backlog

New sources require one file in `packages/gauge-core`:

| Source | Priority | Notes |
|---|---|---|
| CDEC (California) | High | Covers Sierra + N. California runs |
| Environment Canada | Medium | BC, Alberta, Quebec paddling |
| USGS stage-only gauges | Medium | Parameter `00065` instead of `00060` |
| Manual / community gauge | Low | Spreadsheet-defined readings for ungauged runs |
