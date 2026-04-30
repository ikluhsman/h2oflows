# h2oflows — AI Context Map

> **Stack:** chi, nuxt, go-net-http | none | vue | typescript
> **Microservices:** api, h2oflow-web, gauge-core

> 159 routes | 20 models | 42 components | 41 lib files | 26 env vars | 2 middleware
> **Token savings:** this file is ~0 tokens. Without it, AI exploration would cost ~0 tokens. **Saves ~0 tokens per conversation.**
> **Last scanned:** 2026-04-30 15:16 — re-run after significant changes

---

# Routes

## CRUD Resources

- **`/api/v1/reaches`** GET | GET/:id | DELETE/:id → Reache
- **`/api/v1/watchlist`** GET | POST | GET/:id | DELETE/:id → Watchlist
- **`/api/v1/admin/rivers`** GET | POST | GET/:id | PUT/:id | DELETE/:id → River
- **`/api/v1/admin/reaches`** GET | POST | GET/:id | PATCH/:id → Reache
- **`/api/v1/admin/users/roles`** GET | POST | GET/:id | DELETE/:id → Role
- **`/api/v1/trip-reports`** GET/:id | PATCH/:id | DELETE/:id → Trip-report
- **`/api/v1/trips`** GET | POST | GET/:id | PATCH/:id → Trip
- **`/reaches`** GET | GET/:id | DELETE/:id → Reache
- **`/watchlist`** GET | POST | GET/:id | DELETE/:id → Watchlist
- **`/admin/rivers`** GET | POST | GET/:id | PUT/:id | DELETE/:id → River
- **`/admin/reaches`** GET | POST | GET/:id | PATCH/:id → Reache
- **`/admin/users/roles`** GET | POST | GET/:id | DELETE/:id → Role
- **`/trip-reports`** GET/:id | PATCH/:id | DELETE/:id → Trip-report
- **`/trips`** GET | POST | GET/:id | PATCH/:id → Trip

## Other Routes

- `GET` `/api/v1/gauges/search` params() [auth, db, cache, ai]
- `GET` `/api/v1/gauges/batch` params() [auth, db, cache, ai]
- `GET` `/api/v1/gauges/{id}/readings` params(id) [auth, db, cache, ai]
- `GET` `/api/v1/gauges/{id}/flow-ranges` params(id) [auth, db, cache, ai]
- `GET` `/api/v1/gauges/{id}/seasonal` params(id) [auth, db, cache, ai]
- `GET` `/api/v1/reaches/map/all` params() [auth, db, cache, ai]
- `GET` `/api/v1/reaches/map` params() [auth, db, cache, ai]
- `GET` `/api/v1/reaches/{slug}/conditions` params(slug) [auth, db, cache, ai]
- `GET` `/api/v1/reaches/{slug}/hazards` params(slug) [auth, db, cache, ai]
- `GET` `/api/v1/reaches/{slug}/flow-ranges` params(slug) [auth, db, cache, ai]
- `POST` `/api/v1/reaches/{slug}/ask` params(slug) [auth, db, cache, ai]
- `POST` `/api/v1/ask` params() [auth, db, cache, ai]
- `GET` `/api/v1/admin/slug-check` params() [auth, db, cache, ai]
- `PUT` `/api/v1/reaches/{slug}/flow-ranges` params(slug) [auth, db, cache, ai]
- `POST` `/api/v1/reaches/{slug}/fetch-centerline` params(slug) [auth, db, cache, ai]
- `DELETE` `/api/v1/reaches/{slug}/centerline` params(slug) [auth, db, cache, ai]
- `POST` `/api/v1/import/kmz` params() [auth, db, cache, ai]
- `GET` `/api/v1/admin/reaches/unassigned` params() [auth, db, cache, ai]
- `GET` `/api/v1/admin/reaches/grouped` params() [auth, db, cache, ai]
- `POST` `/api/v1/admin/reaches/{slug}/auto-river` params(slug) [auth, db, cache, ai]
- `PUT` `/api/v1/admin/reaches/{slug}/river` params(slug) [auth, db, cache, ai]
- `GET` `/api/v1/admin/rivers/{riverSlug}/auto-fill` params(riverSlug) [auth, db, cache, ai]
- `POST` `/api/v1/admin/rivers/{riverSlug}/reorder-reaches` params(riverSlug) [auth, db, cache, ai]
- `GET` `/api/v1/admin/rivers/gnis-lookup` params() [auth, db, cache, ai]
- `GET` `/api/v1/admin/nldi/watershed` params() [auth, db, cache, ai]
- `GET` `/api/v1/admin/nldi/upstream-tributaries` params() [auth, db, cache, ai]
- `GET` `/api/v1/admin/nldi/downstream` params() [auth, db, cache, ai]
- `GET` `/api/v1/admin/nldi/river-name` params() [auth, db, cache, ai]
- `GET` `/api/v1/admin/nldi/preview-centerline` params() [auth, db, cache, ai]
- `GET` `/api/v1/admin/nldi/nearby-gauges` params() [auth, db, cache, ai]
- `PUT` `/api/v1/admin/reaches/{slug}/primary-gauge` params(slug) [auth, db, cache, ai]
- `POST` `/api/v1/admin/reaches/{slug}/generate-description` params(slug) [auth, db, cache, ai]
- `PUT` `/api/v1/admin/reaches/{slug}/meta` params(slug) [auth, db, cache, ai]
- `POST` `/api/v1/admin/reaches/{slug}/nldi-centerline` params(slug) [auth, db, cache, ai]
- `POST` `/api/v1/admin/reaches/{slug}/nldi-centerline-by-comid` params(slug) [auth, db, cache, ai]
- `GET` `/api/v1/admin/me/roles` params() [auth, db, cache, ai]
- `POST` `/api/v1/reaches/{slug}/contributions` params(slug) [auth, db, cache, ai]
- `POST` `/api/v1/reaches/{slug}/trip-reports` params(slug) [auth, db, cache, ai]
- `GET` `/api/v1/reaches/{slug}/trip-reports` params(slug) [auth, db, cache, ai]
- `POST` `/api/v1/proximity-events` params() [auth, db, cache, ai]
- `POST` `/api/v1/trips/{id}/describe` params(id) [auth, db, cache, ai]
- `GET` `/healthz` params() [auth, db, cache, ai]
- `GET` `/gauges/search` params() [auth, db, cache, ai]
- `GET` `/gauges/batch` params() [auth, db, cache, ai]
- `GET` `/gauges/{id}/readings` params(id) [auth, db, cache, ai]
- `GET` `/gauges/{id}/flow-ranges` params(id) [auth, db, cache, ai]
- `GET` `/gauges/{id}/seasonal` params(id) [auth, db, cache, ai]
- `GET` `/reaches/map/all` params() [auth, db, cache, ai]
- `GET` `/reaches/map` params() [auth, db, cache, ai]
- `GET` `/reaches/{slug}/conditions` params(slug) [auth, db, cache, ai]
- `GET` `/reaches/{slug}/hazards` params(slug) [auth, db, cache, ai]
- `GET` `/reaches/{slug}/flow-ranges` params(slug) [auth, db, cache, ai]
- `POST` `/reaches/{slug}/ask` params(slug) [auth, db, cache, ai]
- `POST` `/ask` params() [auth, db, cache, ai]
- `GET` `/admin/slug-check` params() [auth, db, cache, ai]
- `PUT` `/reaches/{slug}/flow-ranges` params(slug) [auth, db, cache, ai]
- `POST` `/reaches/{slug}/fetch-centerline` params(slug) [auth, db, cache, ai]
- `DELETE` `/reaches/{slug}/centerline` params(slug) [auth, db, cache, ai]
- `POST` `/import/kmz` params() [auth, db, cache, ai]
- `GET` `/admin/reaches/unassigned` params() [auth, db, cache, ai]
- `GET` `/admin/reaches/grouped` params() [auth, db, cache, ai]
- `POST` `/admin/reaches/{slug}/auto-river` params(slug) [auth, db, cache, ai]
- `PUT` `/admin/reaches/{slug}/river` params(slug) [auth, db, cache, ai]
- `GET` `/admin/rivers/{riverSlug}/auto-fill` params(riverSlug) [auth, db, cache, ai]
- `POST` `/admin/rivers/{riverSlug}/reorder-reaches` params(riverSlug) [auth, db, cache, ai]
- `GET` `/admin/rivers/gnis-lookup` params() [auth, db, cache, ai]
- `GET` `/admin/nldi/watershed` params() [auth, db, cache, ai]
- `GET` `/admin/nldi/upstream-tributaries` params() [auth, db, cache, ai]
- `GET` `/admin/nldi/downstream` params() [auth, db, cache, ai]
- `GET` `/admin/nldi/river-name` params() [auth, db, cache, ai]
- `GET` `/admin/nldi/preview-centerline` params() [auth, db, cache, ai]
- `GET` `/admin/nldi/nearby-gauges` params() [auth, db, cache, ai]
- `PUT` `/admin/reaches/{slug}/primary-gauge` params(slug) [auth, db, cache, ai]
- `POST` `/admin/reaches/{slug}/generate-description` params(slug) [auth, db, cache, ai]
- `PUT` `/admin/reaches/{slug}/meta` params(slug) [auth, db, cache, ai]
- `POST` `/admin/reaches/{slug}/nldi-centerline` params(slug) [auth, db, cache, ai]
- `POST` `/admin/reaches/{slug}/nldi-centerline-by-comid` params(slug) [auth, db, cache, ai]
- `GET` `/admin/me/roles` params() [auth, db, cache, ai]
- `POST` `/reaches/{slug}/contributions` params(slug) [auth, db, cache, ai]
- `POST` `/reaches/{slug}/trip-reports` params(slug) [auth, db, cache, ai]
- `GET` `/reaches/{slug}/trip-reports` params(slug) [auth, db, cache, ai]
- `POST` `/proximity-events` params() [auth, db, cache, ai]
- `POST` `/trips/{id}/describe` params(id) [auth, db, cache, ai]
- `GET` `Authorization` params() [auth]
- `GET` `email` params() [auth, cache]
- `GET` `app_metadata` params() [auth, cache]
- `GET` `slug` params() [auth, db]
- `GET` `exclude` params() [auth, db]
- `GET` `gnis_id` params() [auth, db]
- `GET` `limit` params() [db, cache]
- `GET` `since` params() [db, cache]
- `GET` `craft` params() [db, cache]
- `GET` `q` params() [db, cache]
- `GET` `bbox` params() [db, cache]
- `GET` `lat` params() [db, cache]
- `GET` `lon` params() [db, cache]
- `GET` `radius_mi` params() [db, cache]
- `GET` `source` params() [db, cache]
- `GET` `ids` params() [db, cache]
- `GET` `distance` params() [auth, db, cache, ai]
- `GET` `comid` params() [auth, db, cache, ai]
- `GET` `lng` params() [auth, db, cache, ai]
- `GET` `up_comid` params() [auth, db, cache, ai]
- `GET` `down_comid` params() [auth, db, cache, ai]
- `GET` `start_lat` params() [auth, db, cache, ai]
- `GET` `end_lat` params() [auth, db, cache, ai]
- `GET` `start_lng` params() [auth, db, cache, ai]
- `GET` `end_lng` params() [auth, db, cache, ai]
- `GET` `river_name` params() [db, cache, payment, ai]
- `GET` `device_id` params() [db, queue, upload, ai]
- `GET` `reach_slug` params() [auth, db]

---

# Schema

### reaches
- id: uuid (pk)
- slug: text (unique)
- name: text (required)
- put_in: geography(point
- take_out: geography(point
- centerline: geography(linestring
- class_min: numeric(3
- class_max: numeric(3
- class_at_low: numeric(3
- class_at_high: numeric(3
- character: text
- region: text
- primary_gauge_id: uuid (fk)

### gauges
- id: uuid (pk)
- reach_id: uuid (fk)
- external_id: text (required, fk)
- source: text (required)
- location: geography(point
- param_code: text (required)

### flow_ranges
- id: uuid (pk)
- gauge_id: uuid (required, fk)
- label: text (required)
- min_cfs: numeric(10
- max_cfs: numeric(10
- class_modifier: numeric(3

### gauge_readings
- id: uuid (pk)
- gauge_id: uuid (required, fk)
- value: numeric(12
- unit: text (required)
- timestamp: timestamp(tz) (required)
- qual_code: text
- provisional: boolean (required)

### reach_conditions
- id: uuid (pk)
- reach_id: uuid (required, fk)
- source_type: text (required)
- summary: text (required)
- runnable: boolean
- reported_by: uuid
- expires_at: timestamp(tz) (required)

### hazards
- id: uuid (pk)
- reach_id: uuid (required, fk)
- location: geography(point
- hazard_type: text (required)
- description: text (required)
- cfs_at_report: numeric(10
- reported_by: uuid

### rapids
- id: uuid (pk)
- reach_id: uuid (required, fk)
- name: text (required)
- river_mile: numeric(5

### reach_access
- id: uuid (pk)
- reach_id: uuid (required, fk)
- access_type: text (required)
- name: text
- directions: text
- parking_spaces: integer
- permit_info: text

### access_waypoints
- id: uuid (pk)
- access_id: uuid (required, fk)
- sequence: smallint (required)
- photo: exif
- InReach: track export
- data_source: text (required)
- ai_confidence: smallint    check
- verified: boolean (required)

### trips
- id: uuid (pk)
- reach_id: uuid (fk)
- end_cfs: numeric(10
- ended_at: timestamp(tz)
- duration_min: smallint
- distance_mi: numeric(6

### trip_track_points
- id: uuid (pk)
- trip_id: uuid (required, fk)
- timestamp: timestamp(tz) (required)
- lat: numeric(10
- lng: numeric(10
- accuracy_m: numeric(7

### gauge_reach_associations
- id: uuid (pk)
- gauge_id: uuid (required, fk)
- reach_id: uuid (required, fk)
- relationship: text (required)
- tributary: ))

### reach_relationships
- from_reach_id: uuid (required, fk)
- to_reach_id: uuid (required, fk)
- relationship: text (required)

### reach_embeddings
- id: uuid (pk)
- reach_id: uuid (required, fk)
- access_id: uuid (fk)
- flow_ranges: ))

### user_watchlists
- id: uuid (pk)
- user_id: text (required, fk)
- gauge_id: uuid (required, fk)

### trip_reports
- id: uuid (pk)
- user_id: uuid (fk)
- device_id: text (fk)
- reach_id: uuid (required, fk)
- title: text
- body: text
- observed_at: timestamp(tz) (required)
- cfs_at_time: numeric(10
- photos: jsonb (default)
- public_slug: text (unique)
- share_consent_h2oflows: boolean (default)
- published: boolean (default)

### contributions
- id: uuid (pk)
- user_id: uuid (fk)
- device_id: text (fk)
- reach_id: uuid (required, fk)
- contribution_type: text (required)
- body: text
- observed_at: timestamp(tz) (required)
- cfs_at_time: numeric(10
- share_consent_h2oflows: boolean (default)

### proximity_events
- id: uuid (pk)
- device_id: text (required, fk)
- user_id: uuid (fk)
- reach_id: uuid (required, fk)
- event_type: text (required)
- location: geography(point
- detected_at: timestamp(tz) (required)
- promoted_to: uuid (fk)

### rivers
- id: uuid (pk)
- slug: text (unique)
- name: text (required)
- basin: text
- state_abbr: text

### user_roles
- id: uuid (pk)
- user_id: text (required, fk)
- role: text (required)
- river_id: uuid (fk)

---

# Components

- **app** [client] — `apps/web/app/app.vue`
- **AppHeader** [client] — `apps/web/app/components/AppHeader.vue`
- **MobileTabBar** [client] — `apps/web/app/components/MobileTabBar.vue`
- **SiteDisclaimerBanner** [client] — `apps/web/app/components/SiteDisclaimerBanner.vue`
- **KmlImportPanel** [client] — `apps/web/app/components/admin/KmlImportPanel.vue`
- **ReachAuthor** [client] — `apps/web/app/components/admin/ReachAuthor.vue`
- **ReachEditor** [client] — props: slug, rivers — `apps/web/app/components/admin/ReachEditor.vue`
- **AggregateGraph** [client] — props: gauges — `apps/web/app/components/gauge/AggregateGraph.vue`
- **GaugeCard** [client] — props: gauge, hideReachSubtitle, density, sharedWith — `apps/web/app/components/gauge/GaugeCard.vue`
- **GaugeDetailModal** [client] — props: gauge, mode — `apps/web/app/components/gauge/GaugeDetailModal.vue`
- **GaugeGraph** [client] — props: gaugeId, reachSlug, currentCfs, noRanges, color — `apps/web/app/components/gauge/GaugeGraph.vue`
- **GaugeReachGroup** [client] — props: leadGauge, reachItems, density, hideRiverName, hideGaugeHeader — `apps/web/app/components/gauge/GaugeReachGroup.vue`
- **GaugeSearchMiniMap** [client] — props: gauges, highlightId — `apps/web/app/components/gauge/GaugeSearchMiniMap.vue`
- **GaugeSearchModal** [client] — `apps/web/app/components/gauge/GaugeSearchModal.vue`
- **GaugeSeasonalBanner** [client] — props: gaugeId, currentCfs — `apps/web/app/components/gauge/GaugeSeasonalBanner.vue`
- **GaugeSparkline** [client] — props: gaugeId, flowStatus, flowBandLabel, reachSlug, compact, color — `apps/web/app/components/gauge/GaugeSparkline.vue`
- **TrendArrow** [client] — props: gaugeId — `apps/web/app/components/gauge/TrendArrow.vue`
- **IconPlay** [client] — `apps/web/app/components/icons/IconPlay.vue`
- **IconRecord** [client] — `apps/web/app/components/icons/IconRecord.vue`
- **IconStop** [client] — `apps/web/app/components/icons/IconStop.vue`
- **DashboardMap** [client] — props: gauges — `apps/web/app/components/map/DashboardMap.vue`
- **NHDExplorerMap.client** [client] — props: upstreamFlowlines, downstreamFlowlines, upstreamGauges, snapLat, snapLng, pickMode, putInPin, takeOutPin, comidSelectMode, comidSelectSlot — `apps/web/app/components/map/NHDExplorerMap.client.vue`
- **ReachMap** [client] — props: name, classMax, centerline, rapids, access, slug, riverName, gaugeLng, gaugeLat, gauges — `apps/web/app/components/map/ReachMap.vue`
- **ReachesMap** [client] — props: hoveredSlug — `apps/web/app/components/map/ReachesMap.vue`
- **DashboardReachGroup** [client] — props: reaches, density — `apps/web/app/components/reach/DashboardReachGroup.vue`
- **DashboardReachRow** [client] — props: gauge, view, hideRiverName — `apps/web/app/components/reach/DashboardReachRow.vue`
- **ReachBrowseRow** [client] — props: reach — `apps/web/app/components/reach/ReachBrowseRow.vue`
- **ShareModal** [client] — props: reachSlug, reachName, currentCfs, flowStatus — `apps/web/app/components/reach/ShareModal.vue`
- **TripCard** [client] — props: trip — `apps/web/app/components/trip/TripCard.vue`
- **TripDetailModal** [client] — props: tripId — `apps/web/app/components/trip/TripDetailModal.vue`
- **admin** [client] — `apps/web/app/pages/admin.vue`
- **confirm** [client] — `apps/web/app/pages/confirm.vue`
- **dashboard** [client] — `apps/web/app/pages/dashboard.vue`
- **explore** [client] — `apps/web/app/pages/explore.vue`
- **index** [client] — `apps/web/app/pages/index.vue`
- **login** [client] — `apps/web/app/pages/login.vue`
- **map** [client] — `apps/web/app/pages/map.vue`
- **edit** [client] — `apps/web/app/pages/reaches/[slug]/edit.vue`
- **index** [client] — `apps/web/app/pages/reaches/[slug]/index.vue`
- **rivers** [client] — `apps/web/app/pages/rivers.vue`
- **[slug]** [client] — `apps/web/app/pages/trips/[slug].vue`
- **trips** [client] — `apps/web/app/pages/trips.vue`

---

# Libraries

- `apps/api/internal/ai/asker.go`
  - function NewReachAsker: (pool *pgxpool.Pool, voyageKey, anthropicKey string) *ReachAsker
  - class ReachAsker
  - class IdentifyResult
- `apps/api/internal/ai/describer.go`
  - function NewTripDescriber: (pool *pgxpool.Pool, anthropicKey string) *TripDescriber
  - class TripDescriber
  - class DescribeResult
  - class TripDetails
- `apps/api/internal/ai/discoverer.go`
  - function NewReachDiscoverer: (apiKey string) *ReachDiscoverer
  - class DiscoveredReach
  - class ReachDiscoverer
- `apps/api/internal/ai/embedder.go`
  - function NewEmbedder: (apiKey string) *Embedder
  - function FormatVector: (v []float32) string
  - class Embedder
- `apps/api/internal/ai/embedreach.go` — function EmbedReachesAll: (ctx context.Context, pool *pgxpool.Pool, embedder *Embedder, reembed bool) (embedded, skipped int, err error), function EmbedReaches: (ctx context.Context, pool *pgxpool.Pool, embedder *Embedder, ids []string, rateLimit bool) (embedded, skipped int, err error)
- `apps/api/internal/ai/flowranges.go`
  - function NewFlowRangeSeeder: (apiKey string) *FlowRangeSeeder
  - class FlowRangeSeed
  - class FlowRangeContext
  - class FlowRangeSeeder
  - class WebSearchResult
  - interface WebSearcher
- `apps/api/internal/ai/reach_description.go` — function GenerateReachDescription: (ctx context.Context, apiKey, name, riverName, commonName string, classMin, classMax *float64) (string, error)
- `apps/api/internal/ai/search.go`
  - function NewSearchEnricher: (apiKey string) *SearchEnricher
  - class SearchEnrichment
  - class SearchEnricher
- `apps/api/internal/ai/seeder.go`
  - function NewReachSeeder: (apiKey string) *ReachSeeder
  - class ReachSeed
  - class RapidSeed
  - class AccessSeed
  - class WaypointSeed
  - class ReachSeeder
  - _...2 more_
- `apps/api/internal/ai/tracks.go`
  - function NewTrackAnalyzer: (apiKey string) *TrackAnalyzer
  - function PrepareTrack: (points []TrackPoint, putIn, takeOut *[2]float64) []TrackPoint
  - class TrackPoint
  - class TrackContext
  - class KnownFeature
  - class TrackSuggestion
  - _...2 more_
- `apps/api/internal/alerts/alerts.go` — function New: (db *pgxpool.Pool) *Dispatcher, class Dispatcher
- `apps/api/internal/auth/context.go`
  - function WithUser: (ctx context.Context, userID, email, role string) context.Context
  - function WithAppRoles: (ctx context.Context, roles []string) context.Context
  - function AppRolesFromContext: (ctx context.Context) []string
  - function UserIDFromContext: (ctx context.Context) (string, bool)
  - function EmailFromContext: (ctx context.Context) (string, bool)
  - function IsSiteAdminFromContext: (ctx context.Context) bool
  - _...2 more_
- `apps/api/internal/auth/middleware.go`
  - function Optional: (v *Verifier) func(http.Handler) http.Handler
  - function Required: (v *Verifier) func(http.Handler) http.Handler
  - function RequireAdmin: (next http.Handler) http.Handler
  - function LoadAppRoles: (querier func(r *http.Request, userID string) ([]string, error)) func(http.Handler) http.Handler
  - function RequireDataAdmin: (next http.Handler) http.Handler
- `apps/api/internal/auth/verifier.go`
  - function NewVerifier: (ctx context.Context, jwksURL string) (*Verifier, error)
  - class Verifier
  - class Claims
- `apps/api/internal/config/config.go`
  - function Load: () Config
  - class Config
  - class PollIntervals
- `apps/api/internal/db/db.go` — function Connect: (ctx context.Context, dsn string) (*pgxpool.Pool, error)
- `apps/api/internal/elevation/elevation.go` — function QueryElevation: (ctx context.Context, lng, lat float64) (float64, error)
- `apps/api/internal/handlers/admin.go` — function NewAdminHandler: (db *pgxpool.Pool) *AdminHandler, class AdminHandler
- `apps/api/internal/handlers/contributions.go` — function NewContributionHandler: (db *pgxpool.Pool) *ContributionHandler, class ContributionHandler
- `apps/api/internal/handlers/gauges.go` — function NewGaugeHandler: (db *pgxpool.Pool, enricher *ai.SearchEnricher, poller toucher) *GaugeHandler, class GaugeHandler
- `apps/api/internal/handlers/import.go` — class Import
- `apps/api/internal/handlers/nldi.go` — function NewNLDIHandler: (db *pgxpool.Pool) *NLDIHandler, class NLDIHandler
- `apps/api/internal/handlers/reaches.go` — function NewReachHandler: (db *pgxpool.Pool, asker *ai.ReachAsker) *ReachHandler, class ReachHandler
- `apps/api/internal/handlers/respond.go`
  - class FeatureCollection
  - class Feature
  - class PointGeometry
- `apps/api/internal/handlers/trips.go` — function NewTripHandler: (db *pgxpool.Pool, describer *ai.TripDescriber) *TripHandler, class TripHandler
- `apps/api/internal/handlers/watchlist.go` — function NewWatchlistHandler: (db *pgxpool.Pool) *WatchlistHandler, class WatchlistHandler
- `apps/api/internal/kmlimport/kmlimport.go`
  - function ParseKMLBytes: (data []byte) (*KMLDoc, error)
  - function New: (pool *pgxpool.Pool, dryRun bool) *Importer
  - function Slugify: (s string) string
  - function SplitPrefixWithHint: (name, description, folderHint string) (prefix, rest string)
  - function SplitPrefix: (name string) (prefix, rest string)
  - function ParseCoords: (raw string) (lon, lat float64, ok bool)
  - _...13 more_
- `apps/api/internal/kmlimport/nldi.go` — function FetchCenterlinePreview: (ctx context.Context, upComID, downComID string) (string, error), function SnapReachComIDs: (ctx context.Context, pool *pgxpool.Pool, slug string) error
- `apps/api/internal/models/gauge.go`
  - class Gauge
  - class GaugeReading
  - class FlowRange
- `apps/api/internal/models/reach.go`
  - class Reach
  - class ReachCondition
  - class Hazard
- `apps/api/internal/nldi/client.go`
  - function New: () *Client
  - function NewWithBase: (base string, hc *http.Client) *Client
  - class Client
- `apps/api/internal/nldi/geo.go`
  - function DWRNearby: (ctx context.Context, lat, lng float64, distanceKm int) ([]DWRStation, error)
  - function StateAt: (ctx context.Context, lat, lng float64) (abbr string, err error)
  - function BasinAt: (ctx context.Context, lat, lng float64) (BasinInfo, error)
  - class DWRStation
  - class BasinInfo
- `apps/api/internal/nldi/mainstem.go` — function MergeMainstem: (features []Feature, targetComID string) ([]Coord, error), function ToGeoJSONLineString: (coords []Coord) string
- `apps/api/internal/nldi/nhd.go`
  - function FirstCoord: (g Geometry) []float64
  - function NHDStreamNameAt: (ctx context.Context, lat, lng float64) (name, gnisID string, err error)
  - function NHDCoordByGNISID: (ctx context.Context, gnisID string) (*GNISLookupResult, error)
  - class GNISLookupResult
- `apps/api/internal/nldi/types.go`
  - class Feature
  - class Geometry
  - class FeatureProps
  - class Collection
  - class SnapResult
- `apps/api/internal/osm/osm.go`
  - function FetchReachLine: (ctx context.Context, minLon, minLat, maxLon, maxLat, startLng, startLat, endLng, endLat float64, preferredName string, intermediatePoints []coord) (string, error)
  - function FetchRiverLine: (ctx context.Context, minLon, minLat, maxLon, maxLat float64) (string, error)
  - function OverpassQuery: (ctx context.Context, query string) ([]byte, error)
- `apps/api/internal/poller/poller.go` — function New: (db *pgxpool.Pool) *Poller, class Poller
- `packages/gauge-core/dwr.go` — function NewDWRSource: () *DWRSource, class DWRSource
- `packages/gauge-core/huc.go`
  - function HUCNames: (huc8 string) (basinName, watershedName string)
  - function CanonicalBasin: (huc8 string) string
  - function CanonicalBasinFromDWRDivision: (div int) string
- `packages/gauge-core/interface.go`
  - class LatLng
  - class BoundingBox
  - class Reading
  - class SiteMetadata
  - class DiscoverOptions
  - interface GaugeSource
  - _...1 more_
- `packages/gauge-core/usgs.go` — function NewUSGSSource: (apiKey string) *USGSSource, class USGSSource

---

# Config

## Environment Variables

- `ANTHROPIC_API_KEY` (has default) — apps/api/.env.example
- `APP_DOMAIN` (has default) — apps/api/.env.example
- `APP_NAME` (has default) — apps/api/.env.example
- `APP_PORT` (has default) — apps/api/.env.example
- `DATABASE_URL` (has default) — apps/api/.env.example
- `DRY_RUN` **required** — apps/api/cmd/seed-flow-ranges/main.go
- `DWR_POLL_INTERVAL` (has default) — apps/api/.env.example
- `FULL` **required** — apps/api/cmd/seed-state-reaches/main.go
- `JWT_SECRET` (has default) — .env.example
- `MIGRATIONS_PATH` (has default) — apps/api/.env
- `NUXT_PUBLIC_API_BASE` (has default) — apps/web/.env.local
- `NUXT_UI_PRO_LICENSE` (has default) — apps/web/.env
- `POSTGRES_DB` (has default) — .env.example
- `POSTGRES_PASSWORD` (has default) — .env.example
- `POSTGRES_USER` (has default) — .env.example
- `REDIS_URL` (has default) — .env.example
- `REEMBED` **required** — apps/api/cmd/embed-reaches/main.go
- `RESEED` **required** — apps/api/cmd/seed-reach-descriptions/main.go
- `SMOKE_SLUG` **required** — apps/api/internal/kmlimport/smoke_test.go
- `SUPABASE_JWKS_URL` (has default) — apps/api/.env.example
- `SUPABASE_KEY` (has default) — apps/web/.env
- `SUPABASE_SERVICE_KEY` **required** — apps/api/.env.example
- `SUPABASE_URL` (has default) — apps/web/.env
- `USGS_API_KEY` **required** — apps/api/.env.example
- `USGS_POLL_INTERVAL` (has default) — apps/api/.env.example
- `VOYAGE_API_KEY` (has default) — apps/api/.env.example

## Config Files

- `.env.example`
- `apps/api/.env.example`
- `docker-compose.yml`

---

# Middleware

## auth
- middleware — `apps/api/internal/auth/middleware.go`
- home-redirect.global — `apps/web/app/middleware/home-redirect.global.ts`

---

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

---

_Generated by [codesight](https://github.com/Houseofmvps/codesight) — see your codebase clearly_