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

## Phase 2b — Reports + multi-dashboards + theming

*Contribution layer rebuilt around a unified Reports concept. Trip reports, hazard warnings, and conditions board collapse into one entity. Adds tabbed dashboards and a theme picker. Re-planned from `NewFeatures.md` 2026-05-06; supersedes the original 2b split.*

The existing `trip_reports`, `hazards`, and `reach_conditions` tables stay live during 2b development (frontend was stub anyway), then drop once routes are migrated. `proximity_events` keeps its FK to `trip_reports` and is deferred to its own phase.

---

### 2b.1 — Unified Reports model

A **Report** is any user-submitted observation about a reach. Drive-by, paddle, hazard sighting, conditions note — same record. Lower bar than AW trip reports: a user driving home from work who notices a strainer can submit one in 30 seconds.

**Required:** reach, report_date, name, content
**Optional:** report_time, hazard_warning text, photos, paddled flag (`true` = author was on the water; default `false`)

CFS at observation time is auto-stamped from the reach's primary gauge nearest to `report_date` + `report_time` (or noon if time omitted). Flow band at observation time is computed from the reach's bands and stored.

**Schema (migration 000076):**

```sql
CREATE TABLE reports (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id        TEXT NOT NULL,
  slug            TEXT NOT NULL,
  reach_id        UUID NOT NULL REFERENCES reaches(id) ON DELETE CASCADE,
  name            TEXT NOT NULL,
  report_date     DATE NOT NULL,
  report_time     TIME,
  content         TEXT NOT NULL,
  hazard_warning  TEXT,
  paddled         BOOLEAN NOT NULL DEFAULT FALSE,
  flow_cfs        NUMERIC,
  flow_band       TEXT CHECK (flow_band IN ('low','running','high')),
  aw_synced_at    TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (owner_id, slug)
);
CREATE INDEX reports_reach_idx       ON reports (reach_id, report_date DESC);
CREATE INDEX reports_owner_idx       ON reports (owner_id);
CREATE INDEX reports_hazard_idx      ON reports (reach_id) WHERE hazard_warning IS NOT NULL;

CREATE TABLE report_photos (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  report_id   UUID NOT NULL REFERENCES reports(id) ON DELETE CASCADE,
  position    SMALLINT NOT NULL,
  storage_key TEXT NOT NULL,
  caption     TEXT,
  taken_at    TIMESTAMPTZ,
  exif_lat    DOUBLE PRECISION,
  exif_lng    DOUBLE PRECISION
);
CREATE INDEX report_photos_report_idx ON report_photos (report_id, position);
```

`storage_key` points to R2; upload pre-signed URLs issued by the API.

**Reports against `user_reaches` are blocked at the API layer** — community moderation surface stays bounded. User reach detail page hides the Report CTA.

**Rate limit + abuse:**
- 5 reports per user per hour (sliding window)
- Content text passed through a lightweight profanity / link-spam check before insert; soft-flag for admin queue if heuristic trips
- Photo upload: max 8 per report, 10 MB each, MIME sniffed server-side
- Edits within 24 hours of creation; after that, locked to preserve attestation. Owner can always delete.

**API:**

```
POST   /reaches/{slug}/reports
GET    /reaches/{slug}/reports?cursor=&limit=
GET    /reports/{slug}                      (public — owner-attributed view)
PATCH  /me/reports/{slug}
DELETE /me/reports/{slug}
GET    /me/reports
```

Public read; auth required for write/edit/delete. Slug is global (not per-owner) so report URLs are shareable: `/reports/{slug}`.

---

### 2b.2 — Submission UX + nav integration

**Header nav:** "Report" tab added beside Dashboard / Explore. Auth-gated; click while logged-out routes through sign-in.

**Submit flow:**
1. Reach picker (defaults to last-viewed reach if recent)
2. Date picker (defaults to today), optional time
3. Name field (one-line)
4. Content textarea
5. Optional hazard warning textarea (separate, surfaces with red badge in lists)
6. Optional "I paddled this" toggle
7. Photo uploader (drag/drop)
8. Submit → confirmation toast + redirect to the report detail page

Submit also reachable from a reach detail page via "Add report" button — pre-fills the reach.

**Report detail page (`/reports/{slug}`):**
- Owner attribution + avatar
- Reach link with current band color
- CFS / flow band at observation time
- Content + hazard callout + photo gallery
- Share button (2b.4)

**Per-reach reports section:**
- Below the gauge card on the reach detail page
- Paginated 5/10/25, default 5, sorted recency
- Hazard reports float to the top within page, prefixed with the warning badge
- Empty state: "Be the first to file a report for this reach."

**My Reports (avatar menu):**
- `/me/reports`
- Card grid (desktop multi-column, mobile single column)
- Filters: hazard-only, by reach, by date range
- Per-card actions: view, edit (within 24 h), delete

---

### 2b.3 — Sharing + AW cross-post

Each report has a Share button. Two surfaces: native social and AW cross-post.

**Social share:**
- Twitter / X, Facebook, SMS, Discord, "Copy link"
- Pre-formatted text: `{report.name} — {reach.name} @ {flow_cfs} cfs ({flow_band}). h2oflow.org/reports/{slug}`
- OG image generation pulled forward from Phase 3 for reports specifically: `/og/reports/{slug}.png` showing reach name, CFS, band color, first photo if present. Curated reach pages still get OG in Phase 3; reports need it now.

**AW cross-post:**

AW trip-report fields: title, run date, gauge, flow band (5 buckets: too-low / low / medium / high / too-high), rich-text content, photos.

H2OFlows has 3 bands. **Mapping is per-user, set once,** then reused on every cross-post:

| H2OFlows band | AW band (user picks) |
|---|---|
| below `running.min` | too-low *or* low |
| `running.min`…`running.max` | low *or* medium *or* high |
| above `running.max` | high *or* too-high |

Mapping prompt shown the first time a user clicks "Share to AW". Stored as a JSON object on a new `user_preferences` row. Editable later from settings.

If AW has a usable submission API → POST directly with token-bound auth and stamp `aw_synced_at`.
If not (most likely path) → open a deep-link to AW's web form with all fields URL-encoded, including the mapped flow band. User reviews + submits manually; we still stamp `aw_synced_at` optimistically with an "I posted it" confirmation.

Photos cross-posted only if AW endpoint supports multipart; otherwise the share copy notes "photos available at h2oflow.org/reports/{slug}".

---

### 2b.4 — Long-context report grounding (no RAG)

The existing AI assistant (`internal/ai`) ingests reports at query time by **stuffing all reach-scoped reports into the prompt** instead of retrieving via embeddings. Reach-bounded queries are naturally narrow — pilot reaches will see tens of reports, 1.0-era reaches unlikely to exceed a few hundred. Long context (Claude 200K+) absorbs that comfortably and prompt caching makes repeat queries to the same reach near-free.

**Why not RAG:** embedding pipeline + pgvector + reindex worker + chunking + retrieval tuning all add infrastructure for a problem we don't have at this scale. Reaches are the natural shard. Skip the retrieval layer entirely.

**Loader:**
- For a reach query, fetch all reports for that reach (most recent first), capped at last 24 months and ~500 reports max as a defensive ceiling
- Stamp each with author handle, date, CFS at observation, flow band, hazard flag
- Format as a structured prompt section the model can cite from verbatim

**Prompt caching:**
- The reach-reports block is cached per reach using Claude's prompt cache (5-minute TTL, refreshed on each query)
- Cache key = reach_slug + last report `updated_at` — invalidates automatically on new report or edit
- Cold-cache cost paid once per reach per ~5-minute window; subsequent queries to the same reach are cheap

**Prompt scaffolding (mandatory):**

> "The following are user-submitted reports about this reach. They are unverified and may be inaccurate, stale, or contradicted by current conditions. Cite each report by author + date when referencing. If a hazard is mentioned, surface it with a 'paddler caution' note even if uncertain about current state."

Response format requires inline citations: `[Jane D., 2026-04-12]` linking back to `/reports/{slug}`. The assistant never paraphrases a report as authoritative h2oflows data. Because the model sees the full text of every cited report, citation accuracy is inherent — no retrieval-quality failure mode.

**Hazard short-circuit:** if any loaded report has a non-null `hazard_warning` within the last 30 days, the assistant leads with the hazard summary regardless of whether the user asked about hazards. Implemented as a deterministic check on the loaded set before prompting, not as a model behavior.

**Scale ceiling:** if a single reach ever crosses ~1000 reports (no current path to that — even an extremely active reach would take years), revisit. Options at that point: (1) trim to last N by date before stuffing, (2) reintroduce retrieval as a pre-filter while keeping long context for the final prompt. Not a 1.0 concern.

---

### 2b.5 — Multiple tabbed dashboards

Single dashboard becomes multi-dashboard with tabs.

**Schema (migration 000077):**

```sql
CREATE TABLE user_dashboards (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id    TEXT NOT NULL,
  slug        TEXT NOT NULL,
  name        TEXT NOT NULL,
  position    INTEGER NOT NULL DEFAULT 0,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (owner_id, slug)
);
CREATE INDEX user_dashboards_owner_idx ON user_dashboards (owner_id, position);

ALTER TABLE user_watchlists
  ADD COLUMN dashboard_id UUID REFERENCES user_dashboards(id) ON DELETE CASCADE;
```

**Backfill:** for each distinct `owner_id` in `user_watchlists`, insert one `user_dashboards` row (slug `default`, name "My Dashboard", position 0) and update existing watchlist rows to point at it. After backfill, set `dashboard_id NOT NULL` in a follow-up migration once all writers are updated.

**UX:**
- Tab bar above the dashboard grid
- "+" tab → modal: name, optional description, save
- Three-dot per tab: rename, delete, reorder via drag
- Delete prompts confirm; cascades watchlist rows
- Mobile: tabs become a horizontal-scroll strip; long-press for the three-dot menu
- Empty new dashboard shows the same empty state as the existing single-dashboard build

**Add-to-dashboard flow** (reach card, gauge card, custom gauge card) gains a dashboard picker — defaults to the most recently used dashboard.

**API:**

```
GET    /me/dashboards
POST   /me/dashboards
PATCH  /me/dashboards/{slug}
DELETE /me/dashboards/{slug}
PATCH  /me/dashboards/reorder
```

Watchlist routes gain a `dashboard_id` query param + body field.

---

### 2b.6 — Theme picker + dark mode

Avatar menu addition. Pattern lifted from Nuxt UI's docs theme picker (`docs/app/components/theme-picker/ThemePicker.vue`, `useTheme.ts`) and the local reference at `~/projects/iansrecipes.com/frontend/components/theme-picker.vue` + `app.config.ts` + `assets/css/main.css`.

**Color palettes (initial set):**
- `h2oflows` (default — existing brand)
- `ocean`
- `river`
- `indigo-pink`
- `forest`

Each palette defined as primary/neutral pairs in `app.config.ts`; CSS custom properties live in `assets/css/main.css` keyed off a body data attribute.

**Persistence:**
- Pinia `useTheme` composable; persisted to localStorage (per `feedback_pinia_hydration.md` — never cookies)
- `data-theme` and `data-color-mode` attributes set on `<html>` before paint to avoid FOUC (Nuxt color-mode integration)

**Dark mode toggle** sits next to the palette swatches in the same menu. Three-state: light / dark / system.

No DB column for now; cross-device sync deferred until pilot demands it.

---

### 2b.7 — Hero report count

Landing hero already shows reach + river counts. Add reports.

- Source: `SELECT COUNT(*) FROM reports`
- Cached 60 s in API memory; no need for materialized view at pilot scale
- New stat block to the right of rivers; same typography
- Hide block until > 0 reports exist (keeps hero clean before any user contributes)

---

### Migration sequence (2b)

```
000076_reports.up.sql                   (2b.1: unified reports + report_photos)
000077_user_dashboards.up.sql           (2b.5: dashboards + watchlist FK)
000078_user_preferences.up.sql          (2b.3: aw_band_mapping + future prefs)
000079_user_dashboards_required.up.sql  (after backfill: NOT NULL dashboard_id)
```

Old tables (`trip_reports`, `hazards`, `reach_conditions`) remain through 2b. A later cleanup migration drops them once frontend cuts over to `/reports`. `proximity_events.promoted_to → trip_reports` FK is repointed to `reports` *or* the column dropped, depending on whether proximity work resumes.

---

### Deferred from 2b

- **Proximity events + passive telemetry** — backend route exists; resume when mobile PWA work begins (Phase 10 territory).
- **QR sharing** for custom gauges, reaches, and reports — nice-to-have for parking-lot communication; revisit post-1.0 once link/JSON sharing has usage data.
- **Discord bot ingestion** of report posts — folded into Phase 7.
- **Google Earth picture layer** — moved to Future ideas at the end of this roadmap.

---

## Demo Pack (0.3.0)

*Pre-pilot feature extraction + new build to make the app demo-ready for the six pilot contacts. Each item below is either a polish-and-surface pass on an existing feature or a small new build. Lands as `0.3.0` once all four ship.*

### 3.1 — Basin Maps for dashboards

Dendritic tree view of a dashboard's reaches + gauges showing upstream→downstream topology. Distinct from the existing curated basin detail page (which is a fixed curated basin). This one renders per-user-dashboard from the watchlist contents.

- Pulls reaches + custom gauges + user reaches from active dashboard
- Resolves geometric relationships via existing reach lng/lat ordering (Colorado: lng ascending = downstream)
- Tree layout: tributaries branch off mainstems, custom gauges shown at confluence points
- Tap a node → opens reach/gauge detail
- Same theming/dark-mode integration as the rest of the app
- The visual hook of the demo — Nik / Tim / Matt see this and the "build your own" pitch lands

### 3.2 — AW trip-report HTTP preview window

Generate the AW trip-report submission form structure (gathered by inspecting AW's submit form as a logged-in member). On "submit", show the constructed HTTP request + body in a modal instead of posting. User can copy the body and paste it into AW manually.

- Unblocks demos to Owen + Greg (AW tech team + AW Stream Team) without needing prior AW board approval to actually POST
- Sets the table for Phase 5 outbound AW integration once approval is in hand — same payload, different submit handler
- Form schema lives in `/me/preferences` `aw_band_mapping` adjacent (already shipped 2b.3); add `aw_form_schema` JSONB if needed

### 3.3 — Share custom gauges + user reaches via link / JSON

Generate a portable share payload for a custom gauge or user reach so a recipient can clone it into their own dashboard.

- Two transports: signed link (`/share/{token}` → recipient logs in → "Add to dashboard?") and a copyable JSON snippet (recipient pastes into an Add → Import dialog)
- Recipient gets a clone, not a reference — payload includes formula (custom gauge) or geometry (user reach) + display metadata
- No DB-level sharing; per private-content rule (see `feedback_user_content_private.md`)
- QR generation pulled out — deferred (see "Deferred from 2b")

### 3.4 — SEO blocking until 1.0

Block search engines from indexing curated reach + report pages until 1.0 ships. Pilot is private validation, not a public launch — we don't want half-finished pages cached in Google.

- `robots.txt` disallow all
- `<meta name="robots" content="noindex, nofollow">` on layout default
- Pulled in the 1.0 release as part of the launch checklist
- Trivial; ships with whichever PR is first to merge after this section starts

---

## Repository restructure (pre-pilot)

*Split the monorepo into a GitHub org with three independent repos before pilot outreach. Cleaner deployment story per surface, separate release cadence per layer, easier to hand a single repo to a contributor (e.g. AW collaboration) without exposing the rest.*

GitHub org names cannot contain dots, so the org is `h2oflows` even though the production domain is `h2oflow.org`. (Domain currently registered as `h2oflow.org`, plural-domain `h2oflows.org` purchase TBD — flagged for the user to confirm.)

### Target topology

| Repo | Contents | Deploy target | Stack |
|---|---|---|---|
| **`h2oflows/api`** | Go backend (Chi, pgx v5, PostGIS) + migrations + `gauge-core` rolled in as `internal/gaugecore` + poller + AI handlers | TBD (Fly.io / Render / Railway — pick at split time) | Go 1.x, single module, no `go.work` |
| **`h2oflows/web`** | Nuxt 4 frontend, MapLibre, uPlot, Pinia, Nuxt UI Pro | **Netlify** | Nuxt 4 |
| **`h2oflows/docs`** | Pilot documentation site (per-feature walkthrough pages) | Netlify or Cloudflare Pages | Nuxt 4 + Nuxt UI + Nuxt Content, scaffolded from a Nuxt docs template (Docus or Nuxt UI Pro `docs` starter) so it matches the look of Nuxt module sites |

### What moves where

- `apps/api/**` → `h2oflows/api/` (root)
- `apps/api/migrations/**` → `h2oflows/api/migrations/` (unchanged structure)
- `packages/gauge-core/**` → `h2oflows/api/internal/gaugecore/` — only API consumes it; flatten to remove the workspace dep
- `apps/web/**` → `h2oflows/web/` (root)
- `apps/docs/**` (created during 2b for pilot) → `h2oflows/docs/` (root)
- `ROADMAP.md`, `ARCHITECTURE.md`, `NewFeatures.md` history — keep in `h2oflows/api` as the canonical planning home; symlink or duplicate `CLAUDE.md` per repo with repo-scoped guidance
- `.claude/memory/` stays local-only (gitignored everywhere)

### Split mechanics

1. Tag current monorepo HEAD as `pre-split` for forensic reference
2. Use `git filter-repo --path apps/api --path packages/gauge-core --path-rename apps/api:.` (and similar per repo) to preserve commit history per surface
3. Push each filtered branch to its new GitHub repo
4. Open a tracking issue in each new repo capturing residual cleanup
5. Archive (do not delete) the original monorepo with a top-level README pointing to the three new repos

### Cross-repo concerns

- **API URL** injected into `h2oflows/web` build via `NUXT_PUBLIC_API_URL` (Netlify env var); preview deploys point at staging API
- **CORS** on API explicitly allow-lists web's Netlify origins (production + preview wildcard) and docs origin if docs embed any live data
- **Auth (Supabase)** config duplicated as env vars in web + docs builds; API verifies same project's JWTs
- **Shared types** — Go API has no TS consumer today. Phase 4 introduces OpenAPI 3.1; defer codegen until that lands. In the interim, web maintains hand-written types matching the API contract.
- **Migrations** stay co-located with API; CI runs `migrate up` against staging DB on merge to `main`
- **Cross-repo references** — docs links to web pages; web links to docs pages; both link to API status / health. Use environment-aware base URLs in each.

### Versioning across the split

Each repo gets its own semver (independent git tags, independent CHANGELOGs). The product version (`0.1.0`, `0.2.0`, `1.0.0` from the Pilot rollout cadence) is a meta version recorded in `h2oflows/api/RELEASES.md`, mapping each product release to the specific commit / tag in each repo at that moment:

```
0.2.0 (Phase 2b shipped)
  api: v0.2.0   (commit abcd123)
  web: v0.2.0   (commit ef45678)
  docs: v0.1.0  (commit 9012abc)
```

Repos can ship patches independently (`api@v0.2.1` for a poller fix without touching web). The next coordinated product bump lifts whichever repos changed.

### Order of operations

1. Phase 2b + Demo Pack (3.1–3.4) ship in the monorepo (avoid restructuring mid-flight)
2. Confirm domain + create `h2oflows` org
3. Filter-split monorepo into `h2oflows/api` + `h2oflows/web` in one sitting; freeze monorepo writes during the cut
4. Stand up `h2oflows/docs` fresh post-split (no history to preserve — scaffolded directly in its own repo from a Nuxt docs template)
5. Reconfigure CI/CD per repo
6. Update local dev docs in each `CLAUDE.md`
7. Tag `0.3.0` across all three repos as the first post-split product release (Demo Pack + split)
8. Build docs pages per feature in `h2oflows/docs`
9. Begin pilot outreach against the new topology

### Risks + mitigations

- **History loss on filter-repo** — verify with `git log --follow` on a few key files before pushing; keep `pre-split` tag indefinitely as fallback
- **Netlify rebuild churn** during cutover — set up the new web repo's Netlify project with a placeholder before DNS flip, then move the domain once builds verify green
- **Auth env drift** — check Supabase keys identical across web + docs + API (single source: a 1Password vault entry referenced by all three CI configs)
- **Out-of-sync deploys** during a coordinated bump — release checklist in `RELEASES.md` enforces order: API first, web second, docs last (frontend can degrade gracefully behind a stale API; reverse is messier)

---

## Pilot rollout (0.x)

*Validate Phases 2 + 2b with a small targeted group before public launch. Each pilot contact gets a tailored pitch + a feature-focused walkthrough in a docs site. Doubles as a mobile/device acid test.*

### Pilot group + tailored messaging

The app pivoted from a generic flow tracker to "build your own reaches and dashboards on top of curated content." Curated reaches stay; user reaches and custom gauges are the personal layer. Each contact below gets a distinct pitch.

| Contact | Role / context | Lead with | Docs to link |
|---|---|---|---|
| **Nik** | Whitewater kayak instructor; AW stream team contributor; long-time paddling partner | Custom gauges (gauge math is his world) | Custom gauge builder; user reach creation |
| **Owen** | AW tech team | Data schema standard for an AW pipeline; auto-share Reports → AW trip-report pre-fill (2b.3); public API contract | Reports + AW cross-post (2b.3); public API (Phase 4) |
| **Greg** | AW Stream Team Google Group; previously floated an alt whitewater DB; emailed direct | "Complementary, not competing" — H2OFlows as the load-shedding seam AW didn't want to host | Public API; reach + gauge data model framed as offload |
| **Tim Kunin** | Expert paddler since 2014; deep community presence | Custom reach creation + flow tracking + Reports | User reach flow; custom gauges; reports |
| **Matt Beaman** | Paddler, non-technical | Plain UX walkthrough — no jargon, will catch dreadful breakage | Dashboard + add reach + reports |
| **Jamie Knight** | PNW paddler, ex-CO; active community member | Reports + conditions across regions; flow tracking | Reports; basin / state navigation |

**Send strategy:**
- Nik gets a cold link — he'll just load up and explore
- Everyone else gets a personalized DM/email with: (1) why them specifically, (2) one or two features tailored to them, (3) direct links to the docs pages for those features, (4) ask for an acid-test pass on phone + laptop
- Drafts kept in a `pilot-outreach/` scratch dir (not committed) until ready to send

### Pitch differentiation

- **Nik / Tim / Matt / Jamie** — paddler users; pitch the personal-dashboard + custom-reach angle. They use the app, they file reports, they break the UX.
- **Owen** — AW-internal; pitch the data interop angle. Lead with the share-back-to-AW flow (2b.3) and the public API (Phase 4) as a way for AW to receive structured submissions without operating the public API themselves. Open the door to schema-standard collaboration.
- **Greg** — pitch H2OFlows as an *offload*, not a replacement. Frame the public API as the interop seam. Acknowledge his concern (AW server load from a public API) and demonstrate H2OFlows already shouldering it. Reframes my earlier pushback against an AW alternative — the alternative is a complement, not a fork.

### Pilot docs site

Stand up a small Nuxt docs site (separate package or `apps/docs`). Lightweight — for the pilot, not SEO marketing. **Scaffold from a Nuxt docs template** so it has the polished look of Nuxt module documentation sites — candidates:

- **Docus** (`nuxtlabs/docus`) — the classic Nuxt module docs aesthetic
- **Nuxt UI Pro `docs` starter** (`nuxt-ui-pro/docs`) — newer, matches Nuxt UI v3/v4 styling, what powers the Nuxt UI Pro docs themselves

Pick at scaffold time based on which one is current and best supported when the docs repo is cut. Default leaning: Nuxt UI Pro `docs` starter, since the web app is already on Nuxt UI Pro and the design language stays consistent across web + docs.

Per-feature pages:

- Dashboard + watchlist
- Add a curated reach to your dashboard
- Create a custom gauge
- Create a user reach (with map walkthrough)
- File a report
- Share a report (social + AW cross-post)
- Theme picker

Each pilot message links directly to the doc pages relevant to that contact — no scrolling a generic landing page.

Tentative deploy: `docs.h2oflow.org` or a subpath on the main app.

### Acid test

The pilot is also a mobile/device matrix shakedown:

- Each contact runs the app on phone + laptop (whatever they own — iOS / Android / macOS / Windows mix expected)
- Targeted scenarios:
  - dashboard hydration on cold load
  - map gestures on touch (reach map zoom/pan, marker tap targets)
  - custom gauge formula builder on small screens
  - reach creation map flow on phone (anchor pick, take-out pick, auto-trim preview)
  - photo upload on report from mobile camera
  - tabbed dashboard interaction on mobile (horizontal scroll + long-press menu)
  - theme picker + dark mode toggle persistence across reloads
- Feedback collected via a single channel (Discord DM or email — TBD) and tracked in a lightweight log
- Bugs filed, prioritized, and folded into 0.x patch releases
- Diverse mix of technical expertise + paddling experience expected to surface different bug classes

### 0.x → 1.0 release cadence

Semantic versioning. The pilot lives on 0.x:

- `0.1.0` — Phase 2 (2.1–2.6) shipped; pilot can demo all current features (custom gauges, user reaches, polling resilience, discovery UX)
- `0.2.0` — Phase 2b shipped (Reports, multi-dashboards, theme picker, hero report stat)
- `0.x.y` patches — bug fixes from acid-testing
- `0.x.0` minors — new feature increments below the 1.0 threshold
- `1.0.0` — public launch

**1.0 gate criteria:**
- Stable across the pilot device matrix
- Load-tested (public API + poller under simulated traffic)
- Pilot UX feedback addressed (or explicitly deferred with rationale)
- Sufficient curated reach catalog to give a non-pilot user a reason to land
- All critical hazards in the issue tracker resolved
- Phase 3 OG images live (curated reaches + reports)

Each release cuts a git tag, a `CHANGELOG.md` entry, and (when relevant) a short post for the pilot channel. 1.0 is the public launch, not a version bump — feature freeze the week prior, full load test, finalize OG images, social prep.

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

---

## Future ideas

*Speculative or low-priority concepts kept on the roadmap for memory but not scheduled. Revisit only when adjacent phases create the right conditions.*

- **Google Earth picture layer** — overlay user photos at their EXIF GPS points on a 3D terrain view. Design coupled to photos-on-map; only worth revisiting once Reports photo upload has enough data to know what EXIF metadata users actually attach. Originally scoped in 2b, moved here as not pilot-relevant.
- Other speculative ideas land here as they come up.
