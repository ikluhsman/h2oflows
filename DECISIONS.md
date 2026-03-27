# H2OFlow — Decision Log

> This document captures the *reasoning* behind key decisions made during
> initial project design. The what is in architecture.md. This is the why.
> Useful when revisiting a decision months later or onboarding contributors
> who want to understand the thinking, not just the outcome.

---

## Product decisions

### Why this is a data platform, not a social network

Early in planning the scope included social features — community forums, group chat, trip coordination social feeds. We pulled back from this deliberately.

The Colorado whitewater community already has Discord (Colorado Whitewater server), Facebook groups, and SMS threads that work well enough for social coordination. People are not going to abandon those. Trying to compete with them would split H2OFlow's focus and produce a worse social experience than tools people already use and trust.

The genuine gap is structured, queryable river data tied to real-time conditions. Nobody owns that in an open form. That's the thing worth building. The social platforms become *consumers* of H2OFlow data rather than competitors — the Discord bot integration is the expression of this: H2OFlow feeds data into the community's existing social spaces rather than trying to replace them.

**The test:** Would a paddler use this if none of their friends were on it? For a social network, no. For a gauge dashboard with good flow data, yes.

---

### Why streamflow is the wedge, not trip planning

The permit trip coordinator is the most ambitious and most differentiated feature on the roadmap. It's also the hardest to build and requires a critical mass of users to be useful (you need a roster to coordinate).

The gauge dashboard requires zero other users to be valuable. One paddler, three rivers, one graph. That works on day one with no community.

The build order reflects this: ship the gauge dashboard first, get real users, then layer trip planning on top of an existing user base. Building trip planning first and hoping users follow is the wrong order.

---

### Why the scope is Colorado-first

The primary developer is Colorado-based with an existing Colorado paddler network. The initial beta group of ~20-30 paddlers is all Colorado. The USGS and Colorado DWR integrations already exist as Prometheus exporters.

Starting with Colorado means the reach data can be personally verified by people who know the rivers. A Grand Canyon reach defined by someone who has run it 5 times is more trustworthy than one imported from a dataset nobody on the team can verify.

Expansion to other regions follows once the platform is stable and contributors from those regions show up. The gauge plugin architecture is designed for this — adding California's CDEC or Environment Canada is writing one adapter file.

---

### Why not build a native iOS/Android app immediately

Native apps require maintaining two codebases (or one React Native / Flutter codebase with its own tradeoffs), cost more to build, and require App Store review cycles that slow iteration during early development.

A well-built PWA covers 90% of what H2OFlow needs: gauge dashboards, trip planning, map views, offline caching, push notifications. The remaining 10% — camera integration for geotagged photos, deep OS-level location access for put-in detection — can wait until there's a real user base telling us those features are blocking them.

The path to app stores is Capacitor: the same Nuxt/Vue PWA wrapped in a thin native container. Same codebase, App Store presence, no separate native development. This gets deferred to post-v1 but the architecture doesn't preclude it.

**Update — trip tracking requires Capacitor sooner than planned**

Background GPS location access is the hard constraint. iOS gives PWAs essentially no background location. Android is marginally better but unreliable. A 3–4 hour paddle on Browns Canyon with the phone in a hip pack or dry bag will not produce a usable track from a PWA — iOS will terminate background tasks within minutes.

Trip tracking (GPS collection → on-device segmentation → user review → offline sync) is therefore a Capacitor feature, not a PWA feature. The Phase 1 gauge dashboard works fine as a PWA. The moment trip tracking is in scope, Capacitor wrapping is a prerequisite, not an optimization.

The Capacitor plugin ecosystem covers what's needed: `@capacitor/geolocation` with background mode, `@capacitor-community/background-geolocation` for continuous tracking, and the existing IndexedDB strategy for offline storage. The Nuxt/Vue codebase doesn't change — Capacitor adds the native layer on top.

*March 2026*

---

### Why the social gap analysis led to the Discord integration design

Rather than building community features into H2OFlow, we inverted the relationship: H2OFlow pushes data *into* the community spaces that already exist.

The Discord bot sends hazard warnings, flow alerts, and conditions digests to designated channels. Paddlers get H2OFlow data without leaving Discord. H2OFlow gets distribution without building a social layer.

The command interface (`!hflow hazard`, `!hflow conditions`) gives community members a way to contribute structured data back to H2OFlow from within Discord — closing the loop without requiring them to open a separate app.

---

### Why we chose a trust ladder for Discord rather than full AI parsing immediately

The whitewater community is small and trust is earned slowly. One bad auto-post of incorrect hazard information — a strainer that doesn't exist, a condition report for the wrong river — could damage the platform's reputation in a community where that reputation is everything.

The trust ladder (explicit commands → keyword nudges → AI extraction with confirmation → auto-post) builds trust incrementally. Phase 1 is completely deterministic and predictable. The community learns to trust the bot before the bot gets smarter. Phase 4 (auto-post without confirmation) may never be the right call and that's fine.

---

## Technical decisions

### Why Go for the backend

The primary developer came in with Python experience (existing Prometheus exporters) and limited Go experience. Go was chosen over staying in Python for several reasons:

**Self-hosting story.** A single Go binary with no runtime dependencies is dramatically easier to self-host than a Python application with virtualenvs, dependency management, and runtime version pinning. The target audience includes technically sophisticated paddlers who will self-host — this matters.

**Concurrency model.** The gauge poller needs to hit multiple APIs on independent schedules, handle timeouts gracefully, and push alerts without blocking. Go's goroutines make this natural. Python's async story is more complex and the existing exporter code was synchronous.

**Learning opportunity.** The primary developer explicitly wanted to learn Go. Porting the existing USGS/DWR logic to Go was identified as the ideal first Go project — familiar domain, known APIs, clear expected output. This removes the "blank canvas" anxiety of learning a new language.

**Contributor ecosystem.** Go has strong adoption in the infrastructure/tooling community that overlaps with the kind of technically sophisticated paddlers who might contribute to a FOSS project like this.

The tradeoff is a steeper initial learning curve. This was judged acceptable given the 6-month timeline and the learning project framing.

---

### Why PostgreSQL + PostGIS over alternatives

River data is fundamentally geographic. Put-in and take-out coordinates, reach centerlines, rapid locations, gauge positions — all of it is spatial data that benefits from proper GIS support.

PostGIS enables queries that aren't otherwise practical: "reaches within 50 miles of Colorado Springs", "gauges near this coordinate", "draw a river-following line between photo points using reach centerline geometry." These aren't edge cases — they're core features.

SQLite was considered for simplicity but ruled out because PostGIS support on SQLite is limited and the spatial query requirements are real. MongoDB was not seriously considered — the data model is relational and benefits from foreign key constraints.

---

### Why Nuxt 4 + Nuxt UI Pro over alternatives

The primary developer knows Vue and Nuxt well. Framework familiarity matters more than framework preference in a time-constrained solo project. Switching to SvelteKit or React would have been a meaningful productivity hit during the period when shipping matters most.

Nuxt 4 specifically adds:
- Better SSR performance (critical for SEO on reach pages)
- Improved TypeScript support
- Better PWA tooling via @vite-pwa/nuxt

Nuxt UI Pro was chosen over raw Tailwind + a component library because the dashboard layouts, stat cards, and data tables it provides are exactly the components H2OFlow needs most. Building those from scratch on raw Tailwind would take weeks. The per-project license cost is justified by the time saved.

**Why SSR matters here:** The primary discovery path for new users is organic search. Someone googles "arkansas river numbers gauge conditions" and lands on a reach page. That page needs to be server-rendered and indexable. A pure SPA would lose all of that.

---

### Why uPlot for gauge graphs over Chart.js

The multi-gauge aggregate view potentially renders thousands of data points per gauge across a date range. Chart.js degrades visibly at that scale. uPlot is purpose-built for time-series data and renders 100k+ points without performance issues.

The tradeoff is that uPlot has no official Vue wrapper — it's used imperatively in `onMounted` hooks. This is a minor ergonomic cost worth paying for the performance characteristics.

---

### Why MapLibre GL over Google Maps or Leaflet

**Google Maps:** Requires an API key, has usage-based pricing that could become significant at scale, and is not open source. A FOSS project that depends on a commercial mapping API is fragile.

**Leaflet:** Open source and free but raster-tile-based. Vector tiles render more sharply, especially at high zoom levels on retina displays, and are more performant on mobile. For a map-heavy app on mobile devices, the difference is noticeable.

**MapLibre GL:** FOSS fork of Mapbox GL JS after Mapbox changed their license. Vector tiles, no API key required for the renderer (you supply your own tile source), strong community, native Vue wrapper available. The correct choice for a FOSS project that takes geographic data seriously.

---

### Why local-first architecture

The app needs to work at the put-in. Put-ins for Gates of Lodore, Westwater, the Numbers, many Colorado creeks — there is no cell signal. A trip planner that requires connectivity to function is useless at the moment it's most needed.

Local-first means trip data is written to the device first. The app is fully functional offline. Cloud sync happens when connectivity returns. This is also a privacy benefit — trip data doesn't leave the device unless the user explicitly chooses to sync or publish.

The technical implementation uses the browser's IndexedDB (via a library like Dexie.js) for local trip storage in the PWA, with a sync queue that handles conflict resolution when the same trip is edited across devices.

---

### Why the gauge plugin interface is a Go interface, not a config file

Early consideration was a config-driven approach where gauge sources were defined in YAML with URL templates and response mappings. This was rejected because:

1. USGS and DWR have meaningfully different response shapes and authentication patterns. A generic config mapper would need to be complex enough to basically be code anyway.
2. A Go interface is more explicit, more testable, and more welcoming to contributors. "Implement this 3-method interface" is a clear contribution target. "Edit this YAML schema" is less obvious.
3. Type safety. A misconfigured YAML adapter fails at runtime. A Go adapter that doesn't satisfy the interface fails at compile time.

The cost is that adding a new gauge source requires writing Go rather than editing a config file. This is judged acceptable given the contributor profile — technically sophisticated developers who are also paddlers.

---

### Why Apache 2.0 over GPL

GPL would require any derivative work (including commercial products built on H2OFlow) to also be open source. This sounds appealing but creates friction for potential contributors who work at companies with GPL policies, and for outfitters who might want to build commercial tools that consume the API.

Apache 2.0 lets anyone use the code, including commercially, as long as they preserve attribution. The community data (the reach registry, flow ranges, trip reports) is the real asset and doesn't have a software license — it's community-contributed and implicitly available under the API terms.

The goal is maximum adoption and contribution, not license enforcement. Apache 2.0 serves that better.

---

### Auth: Google / Apple / magic-link, not email/password

The original decision was email/password with JWT for simplicity and self-hosting purity. That decision was revisited when thinking concretely about the beta pilot audience.

The Front Range paddling community is 30-50 year old adults who will not remember a new password for a whitewater app they use a few times a month. Password reset flows create support burden before there are any support resources. And the privacy argument against social login is weakest for Google and Apple — the people signing up for the beta already trust those providers with far more sensitive data.

**The revised plan:** Google / Apple OAuth for the hosted instance. Magic-link (email, no password) for users who prefer not to use social login and as the fallback for self-hosted instances that can't register OAuth apps.

This removes friction at the exact moment it matters most — first login during the beta pilot. The self-hosting story is preserved via magic-link. Auth is deferred entirely until after the Phase 1 pilot completes; the dashboard is fully functional without an account.

*March 2026*

---

## Scope decisions (what we deliberately excluded)

### No raw video hosting

GoPro footage from a Grand Canyon trip is gigabytes. Hosting, transcoding, and serving video at any scale is expensive and complex. YouTube and Vimeo solve this well and are free for users.

The trip blog export embeds YouTube/Vimeo iframes. This gives the hosted trip page video content without H2OFlow paying for storage or transcoding. Users who want their video in the trip record link to their existing YouTube channel.

Raw video hosting may be revisited if it becomes a strong community request, but it's firmly back-burner.

### No commercial outfitter features at launch

Outfitter integration (packages, gear rental, meal service) is in the data model and planned for Phase 5. A commercial outfitter API with SLA guarantees and elevated rate limits is back-burner post-v1.

Reason: the core community use case (non-commercial paddlers planning their own trips) needs to be proven before building commercial tooling on top of it. Building the commercial tier before proving community adoption is premature optimization.

### No international gauge sources at launch

Environment Canada, Australian BoM, EU Copernicus, New Zealand NIWA all have accessible APIs. The gauge plugin architecture is designed to support them. They're not being built at launch because:

1. The initial community is US-centric (Colorado specifically)
2. Each international source has its own quirks and requires research
3. The plugin architecture means adding them later is just writing an adapter file

International sources become community contributions once the platform has international users who care about them.

### No permit application integration

Recreation.gov handles permit applications. H2OFlow tracks permit trips (planning, roster, logistics) but doesn't attempt to replace or integrate with the permit lottery system itself. That's a different problem, a complex API relationship, and outside the core value proposition.

---

## Things we considered and rejected

**Building on top of American Whitewater's data:** AW keeps their dataset proprietary and has no public API for bulk access. Even if they opened it, building a FOSS platform on a proprietary data foundation creates a dependency that conflicts with the project's principles. OSM whitewater data plus community contribution is the right foundation.

**React/Next.js instead of Vue/Nuxt:** The primary developer knows Vue well. Framework familiarity during a time-constrained build matters more than framework preference. No strong technical reason to switch.

**Supabase as the backend:** Considered as a way to reduce backend development time. Rejected because self-hosting Supabase is complex, it introduces significant third-party dependency, and writing the Go backend is a learning goal for the project. The Go + Chi + PostgreSQL stack is simpler to understand, simpler to self-host, and more welcoming to contributors.

**ActivityPub for federation between self-hosted instances:** Architecturally interesting but significantly complex to implement correctly. The federation use case (self-hosted instances contributing reach data back to the public commons) can be solved with a much simpler custom API. ActivityPub is not worth the complexity at this stage.

---

### Primary gauge designation per reach

A reach can have multiple gauges bound to it (upstream context, downstream confirmation, tributaries), but exactly one is the **primary gauge** — the canonical CFS number paddlers quote for that run. "Numbers is at 850" means the Nathrop gauge specifically, not Granite or Parkdale.

Implemented as `primary_gauge_id UUID REFERENCES gauges(id)` on the `reaches` table. This drives:
- The headline CFS number on the reach page
- Flow range band evaluation ("is this run on?")
- User flow alerts (fire against primary gauge only unless user explicitly watches another)
- The multi-gauge aggregate dashboard default view

Supporting gauges remain bound to the reach via `gauges.reach_id` and appear as contextual readings ("upstream at Granite: 1,200 cfs ↑"). They don't trigger alerts or determine run status.

Some reaches use an upstream or downstream gauge as primary because no gauge sits at the put-in. Flow ranges on that gauge are calibrated by community experience to account for the offset. The architecture handles this implicitly — `primary_gauge_id` points to whatever gauge the community agrees is the reference, regardless of location.

*March 2026*

---

### Gauge prominence model

Two independent axes determine how prominently a gauge appears in search results and discovery:

**Source tier** (derived from `source` column, not stored separately):
- Tier 1: `usgs` — satellite/radio telemetry, federally maintained
- Tier 2: `dwr`, `cdec`, `usbr` — telemetry, state/federal but smaller networks
- Tier 3: `community` — scraped HTTP resources (e.g. PoudreRockReport), valuable but fragile
- Tier 4: `manual` — human-entered readings, infrequent

**Community prominence** (nightly computed `prominence_score` on `gauges`):
```
source_tier_base              (usgs=100, dwr=80, scraped=50, manual=20)
+ featured × 200              (manually elevated; beats source tier)
+ dashboard_saves × 5
+ trip_report_refs × 10
+ reach_bound × 50
+ uptime_30d_pct × 1
```

The `featured` boolean allows a community maintainer to elevate a tier-3 scraped gauge above a tier-1 USGS gauge for a specific reach — e.g. the Poudre rock gauge is more useful to paddlers for that run than the nearest USGS site, even though it's less technically reliable. `featured` is the explicit override.

Search results default to `ORDER BY prominence_score DESC`. Established community gauges surface naturally; obscure or new gauges are available but don't clutter discovery.

*March 2026*

---

### Phone stays in the car — the pre-trip usage model

Most paddlers do not carry a smartphone on the water. They carry an InReach, a SPOT, or nothing at all. The phone goes in the drybox in the car or the shuttle vehicle.

This shapes the entire app usage model:

- **Primary use case: pre-trip planning** — at home the night before or in the parking lot before launch. The user checks gauge levels, reads conditions, reviews the rapid inventory, confirms the put-in location. This is when the app needs to be fast, informative, and confidence-building.
- **Secondary use case: at the put-in** — final CFS check, any new hazard reports, confirm the shuttle is set. Needs to work on marginal cell signal or offline from cached data.
- **Tertiary use case: post-trip logging** — back at the car, upload a photo, log a condition report, note a new strainer. This is how community data gets contributed.
- **Not a use case: active navigation while paddling** — don't design for this. A paddler with their phone out in a rapid is a paddler about to lose their phone.

The PWA offline behavior should optimize for the put-in scenario: cache the watchlisted reaches with their current readings and rapid inventory before the user loses signal driving into the canyon. The cached data should be clearly timestamped so the paddler knows how fresh it is.

*March 2026*

---

### River sections are chained — shared access points

The take-out for one river section is typically the put-in for the next section downstream. This is not an edge case — it's the fundamental structure of how rivers are divided into runs. The Numbers ends at Nathrop; the next section starts at Nathrop. Browns Canyon ends at Hecla Junction; the section below starts at Hecla Junction.

The `reach_access` table handles this with duplicate rows: the same physical location (parking lot, boat ramp) gets two `reach_access` rows — one as `take_out` for the section above, one as `put_in` for the section below. The physical coordinates are the same; the relationship to each reach is different.

This means the AI seeder, track analyzer, and UI all need to be aware that a point labeled "take_out" for the reach being viewed may also be a "put_in" for an adjacent reach. The trip tracking system uses this to avoid the most common confusion: someone parks at the take-out first to drop a shuttle vehicle, then drives to the put-in. Without knowing that the first stationary cluster is near a known take-out (not a put-in), the track segmentation would misidentify it.

*March 2026*

---

### Trip records — open for enrichment after the paddle

A trip record should not be locked when the paddler reaches the take-out. Post-trip workflows are a primary contribution path:
- GoPro footage comes off the camera at home, sometimes days after the trip
- Photos from other group members get shared later
- The paddler may want to log a condition report or note a new hazard they noticed
- Track analysis results (suggestions from the AI) arrive asynchronously after upload

The trip record stays in an "open" state — editable, enrichable — until the user explicitly archives it or a configurable TTL expires (e.g. 30 days after the trip date). Media, notes, and condition reports can all be added to an open trip regardless of when the trip occurred.

*March 2026*

---

### GPS track ingest — OwnTracks, GPX, and InReach (Phase 2)

H2OFlow does not run a location broker or tracking server. OwnTracks recorder (C binary, LMDB storage, MQTT/HTTP) is a mature, standalone service that solves live location sharing well. Building a competing implementation inside H2OFlow would be redundant and drag in infrastructure (Mosquitto, LMDB) the core use case doesn't need.

Instead, H2OFlow speaks the relevant protocols as an **ingestion target**:

**OwnTracks JSON format** (`{"_type":"location","lat":X,"lon":Y,"tst":N,...}`)
Accepted at `POST /api/v1/ingest/owntracks`. Users who already run OwnTracks recorder configure a Lua `otr_putrec()` hook to forward payloads here. The recorder handles MQTT, device management, and its own historical storage — H2OFlow only sees what it needs for reach/rapid improvement.

**GPX file upload**
The universal format. InReach, Garmin devices, phone apps (Gaia GPS, AllTrails, OsmAnd), and OwnTracks recorder's `ocat` utility all export GPX. Accepted at `POST /api/v1/ingest/gpx`. Most users will use this path.

**On-device first, server-assisted second — offline architecture:**

Colorado mountain rivers have no cell signal. The device must collect and process data entirely offline, then sync when the user drives back to civilization. The architecture reflects this:

1. **Collection** (native background GPS, Capacitor) — runs silently while the phone is in a pocket or dry bag. Raw track stored in IndexedDB — survives app kills and signal loss.
2. **On-device segmentation** (heuristics, no network needed) — when the trip ends or the app is opened, lightweight heuristics identify meaningful moments: speed transitions mark put-in/take-out, stationary clusters on the water are scouts or portages. No AI call, no connectivity required.
3. **User review** (offline) — map screen shows the identified points. User confirms, corrects, or dismisses. Nothing leaves the device without user action.
4. **Upload queue** — confirmed data is queued. Background sync fires when signal returns (service worker or Capacitor background task).
5. **Server AI analysis** (Claude, online) — full reasoning against existing reach data. Returns refined suggestions comparing the submitted track against the known markers.
6. **Final review** (optional, async) — server suggestions appear as a notification: "Your track suggests the put-in marker is 180m off. Update it?" User accepts or ignores.

**Two consent modes:**
- **Auto mode** (user opt-in): Record and submit verified data automatically when confident. Device still does local review; only high-confidence segments are auto-submitted. User receives a post-trip summary.
- **Review mode** (default): Full user control. Nothing submitted without explicit approval. Suitable for privacy-conscious users and obscure runs people consider "their" spot.

**What H2OFlow does with ingested tracks — AI-assisted, not static geometry:**

Naive geometric analysis ("this waypoint is 200m from the marked rapid — flag it") misses context. A GPS track from a paddling day has a narrative: drive in → walk approach trail → stationary cluster at water's edge (gearing up) → fast linear movement matching river current (paddling) → stationary cluster at take-out → walk back to vehicles. Static distance calculations cannot distinguish a scouting stop from a put-in, or a lunch break from the take-out.

Claude analyzes the full track narrative given the reach context:
- Segment the track into meaningful phases (approach / on-water / egress)
- Identify stationary clusters and reason about their likely purpose
- Compare against existing markers and suggest corrections with reasoning
- When multiple tracks from different users converge at the same point, weight that as stronger evidence

Example AI output: *"Three tracks from separate users all decelerate from ~3mph to walking pace and cluster within 25m of 38.7234°N, 106.1021°W before movement pattern shifts to river-speed downstream. This is almost certainly the actual put-in. The existing marker is 180m northeast, likely at the parking lot rather than the water's edge. Confidence: 88."*

The user sees the suggestion, the reasoning, and a map comparison. They accept or reject. Accepted suggestions go in as `data_source='community'`, `verified=true` (the AI reasoning + user confirmation counts as a verification pass). Rejected ones are logged so the AI can calibrate.

**Multi-track convergence** is particularly powerful for obscure runs with sparse data: the first track from someone who ran an unknown creek provides a rough centerline. The second track tightens it. By the fifth track the rapid locations are accurate without anyone doing explicit data entry.

Privacy: track ingest is always opt-in. Tracks associated with obscure or sensitive runs can be used for geometry improvement only, with the raw track discarded after analysis.

*March 2026*

*March 2026*

---

### GoPro GPS and media geolocation (Phase 2+)

Modern GoPros, iPhones, and action cameras embed GPS coordinates in photo and video EXIF metadata. This data can meaningfully improve the accuracy of rapid locations, put-in/take-out points, and access descriptions — essentially community-sourced surveying without asking anyone to survey.

The planned approach when media uploads land:
- Extract GPS from EXIF on upload (server-side, using a library like `github.com/rwcarlsen/goexif`)
- Attach coordinates to the uploaded media as `gps_location GEOGRAPHY(POINT)`
- If the media is tagged to a rapid and the GPS confidence is high, offer to update the rapid's location (with the `data_source='community'` provenance and `verified=false` until confirmed)
- Associate the photo with the flow level at the time of capture (`current_cfs` from the nearest gauge reading at the timestamp in the EXIF)

This creates a body of "rapid X at Y cfs" photos over time without requiring any structured data entry from the paddler — they just upload their GoPro footage and the platform extracts what it can.

GPS data from uploads should always be opt-in. Some users will not want their location data used or stored, particularly for obscure runs they consider "their" spot.

*March 2026*

---

### The Class 5 paddler test

Every data and UX decision should pass this test: **when a Class 5 kayaker opens the app for the first time, do they say "huh, that worked well"?**

A Class 5 paddler will immediately recognize wrong rapid names, inflated class ratings, or a put-in that dumps them a mile from the actual river. They will not give the app a second chance.

This drives several concrete decisions:
- Wrong data is worse than no data. AI-seeded rapids with confidence < 50 are dropped entirely at generation time, not stored as low-confidence drafts that might surface to a user.
- Confidence scores and `verified` status are surfaced in the API response and shown in the UI — not hidden. An experienced paddler would rather see "AI-estimated, not locally verified" than trust an unmarked number.
- The rapid inventory leans toward well-documented runs first. Gaps are honest gaps.
- Community corrections from people who have actually run the water take precedence over AI-generated data once verified.

*March 2026*

---

### Moving water recreation scope — what we track and why

H2OFlow is for moving water: rivers, creeks, and streams relevant to human-powered and motorized watercraft navigating current. "Moving water" is the qualifier — we're not tracking lakes, reservoirs, or tidal estuaries, and we're not tracking every gauge on every navigable river.

The practical filter: **would a paddler care about the flow here?** That includes kayakers, rafters, canoeists, SUPers, packrafters, and any other craft that runs moving water. It excludes:

- Major commercial shipping rivers (Mississippi, Missouri, Ohio) — flow there is weather and flood information, not a paddling decision
- Agricultural diversion points and irrigation canals — relevant to water rights, not recreation
- Urban stormwater monitoring — very real danger during flash floods but not a run-planning gauge
- Purely scientific monitoring sites on headwater tributaries too small to boat

The `featured` flag and `prominence_score` enforce this implicitly — a gauge on a run-worthy section of the Arkansas accumulates dashboard saves, trip report references, and reach bindings. A monitoring station on a ditch that drains a parking lot does not. Discovery naturally surfaces the former over the latter.

This matters for the gauge discovery map: the default view should feel like a whitewater atlas, not a hydrology dashboard. Non-paddling gauges can exist in the DB (USGS imports everything), but they should sort to the bottom and not clutter the map until someone zooms into a specific area and explicitly asks for them.

*March 2026*

---

### Gauge poll tiers — trusted, demand, cold

Every gauge in the DB falls into one of three tiers based on community trust and activity. The tier is computed at query time and returned in the API response so the frontend can show the right icon without any additional logic.

**Trusted** (`featured = TRUE`) — always polled every 15 minutes unconditionally. These are the hand-curated gauges the paddling community relies on. The name is intentional: "trusted" means paddlers have decided this is the right gauge for this run, not just that it's popular. Icon: green checkmark or filled star.

**Demand** (`featured = FALSE`, `last_requested_at` within 7 days) — polled while someone is actively using the gauge. `last_requested_at` is touched whenever the API serves the gauge to a user. Falls back to cold after 7 days of inactivity. Icon: activity/wave indicator — something that communicates "live because someone is watching."

**Cold** (everything else) — exists in the DB, searchable, but the poller ignores it. Historical data is available by proxying to the source API on request. Activates into demand tier the moment a user requests it. Icon: neutral/grey — or no icon at all, just no indicator.

This keeps the poll set small. USGS maintains ~10,000 active gauges nationally; polling all of them unconditionally would be pointless. At peak activity H2OFlow might poll a few hundred. The trusted tier ensures the classic, well-known runs always have current data for the discovery experience even before any users have explicitly requested them.

*March 2026*

---

### Three AI layers in the search and discovery experience

Three distinct places where AI adds value, each with a different latency budget and scope:

**1. Search intent enrichment (real-time, per-query)**

When a paddler types "numbers" or "gore" or "lower ark" or "above golden", a plain SQL `ILIKE` search is brittle — it misses common nicknames ("the numbers" = Arkansas at Nathrop), abbreviations, and reach names that don't match any gauge's official name.

Claude interprets the raw query string before it hits the database:
- Identify river names, reach nicknames, put-in names, geographic references
- Resolve known aliases ("the numbers" → external_id 07091200, search terms "Arkansas River", "Nathrop")
- Return a structured enrichment: `{ terms: ["arkansas", "nathrop"], hint_ids: ["07091200"] }`
- The SQL query ORs the enriched terms alongside the original query and boosts hint_ids to the top

This is a low-stakes Claude call — a wrong interpretation just returns sub-optimal search results, not an error. If the Claude call fails or times out, we fall back to plain text search transparently.

**2. Gauge paddling relevance scoring (batch, nightly)**

The `prominence_score` formula captures community engagement signals but doesn't know whether a gauge is on a boatable run. A gauge on a class IV creek that only a handful of expert paddlers know about can have a low score because it has few dashboard saves — not because it's irrelevant.

A nightly batch process uses Claude to evaluate each gauge against the reach metadata, flow range data, and any available community context:
- Is this gauge on a named, documented run?
- Is the flow range meaningful for recreational boating (not just 0.1 cfs spring snowmelt)?
- Does the gauge sit at a whitewater feature (put-in, take-out, named rapid) or in an agricultural valley?

Output is a `paddling_relevance` score (0–100) stored on `gauges`. This feeds into the final `prominence_score` formula and gates map display at low zoom levels.

**3. Flow condition interpretation (on-demand, per reach)**

Given a gauge reading, the reach's flow ranges, historical context, and any recent community conditions reports — produce a natural-language summary suitable for a paddler making a go/no-go decision:

> "Brown's Canyon is running at 1,240 cfs — solidly in the optimal range (850–1,800). Up 300 cfs from yesterday, trending up slightly. Pushy intermediate water; most commercial raft-supported trips operate in this range. Expect Zoom Flume and the Numbers (the rapid, not the section) to be spicy."

This is a heavier call — invoked when a user opens a reach page, cached per-reach for ~30 minutes. It draws on flow ranges, historical percentile context, and conditions reports. The output is explicitly labeled as AI-generated and links to the underlying data.

*March 2026*

---

### DWR station identifiers — ABBREV as external_id

DWR telemetry stations have two identifiers: a human-readable abbreviation (e.g. `PLAWATCO`) and, for some stations, a numeric USGS-style site number (`06708000`). Where both exist, the numeric ID often refers to the *USGS* gauge co-located at or near the DWR station — it's not a native DWR identifier.

The DWR telemetry API (`telemetrytimeseriesraw`, `telemetrystations`) uses the ABBREV as the primary station key in all API calls. Storing the ABBREV as `external_id` is correct — it's the key used to fetch data.

The ABBREV is also well-known to experienced Colorado paddlers. "PLAWATCO" shows up in trip reports, Discord, and in the community's mental model of Colorado gauge coverage. Making it the `external_id` keeps it visible in the API response without requiring a separate `abbrev` field.

Co-located USGS stations (where the numeric ID exists) are tracked as separate `gauges` rows with `source='usgs'`. Both rows will typically bind to the same reach, with one designated `primary_gauge_id` based on which source the community trusts more for that run.

*March 2026*

---

---

### gauge_reach_associations — many-to-many, not a single foreign key

Early schema had `gauges.reach_id` as the only reach relationship: one gauge, one reach. This broke immediately on the N Fork South Platte corridor where a single upstream gauge (PLAGRACO at Grant) is the practical reference for both Bailey and Foxton — two distinct reaches separated by miles of canyon.

The fix is a proper many-to-many association table with a typed `relationship` column:

```
gauge_reach_associations (gauge_id, reach_id, relationship)
  relationship: primary | upstream_indicator | downstream_indicator | tributary
```

`gauges.reach_id` stays as a display pointer — first-come-first-served, used for the LEFT JOIN that returns a single reach name in the simple case. The association table is authoritative for search, grouping, and relationship labeling.

**Concrete effect on the dashboard:** PLAGRACO's section header reads "Bailey / Foxton" — its combined primary reach label is computed from `gauge_reach_associations` at query time, not from the single `gauges.reach_id`. A gauge serving N primary reaches shows up under one combined group, capped at 3 names with a `/ …` ellipsis for pathological cases.

**The rule of thumb:** `gauges.reach_id` is a UX shortcut for the single-reach common case. `gauge_reach_associations` is the truth.

*March 2026*

---

### AI seeder uses training knowledge, not live scraping

The initial instinct was to scrape American Whitewater pages for rapids, access, and flow ranges before generating content. This was rejected for several reasons:

**AW data is proprietary.** H2OFlow's positioning is explicitly that it does not build on AW's proprietary dataset. Scraping it to feed into our own database would contradict that principle even if technically legal.

**Training knowledge is good enough for well-documented runs.** Claude's training data includes published guidebooks (Caudill, Stohlquist, Nealy), AW trip reports indexed before the training cutoff, and years of community beta on classic runs. For Browns Canyon, Gore Canyon, the Numbers, and the other marquee Colorado runs, the AI-seeded data is accurate and confident (85–95+ confidence scores).

**Provenance is cleaner.** AI-seeded content is clearly labeled `data_source='ai_seed'` with a confidence score. Content that came from a live AW scrape would require a different provenance label and legal consideration. Training-derived content is original synthesis, not reproduction.

**Live search is Phase 2.** The `WebSearcher` interface in `FlowRangeSeeder` is already there — a hook to pre-fetch AW pages before the Claude call and flip `data_source` to `'ai_web'`. This gets wired in once the legal and attribution questions are worked out, or replaced with a different live data source.

*March 2026*

---

### Watchlist is the root context — derive everything silently

The dashboard has no explicit "watershed" or "region" selector. The user's watchlist is the root context for every computation: flow comparisons, section headers, the aggregate graph picker, the poll tier calculation.

This means the app must silently derive reach associations, watershed groupings, and gauge relationships from the watchlist state — never asking the user to re-declare what they already told us when they added a gauge.

**Practical implication:** when a gauge is added from search results, it arrives with its reach association already resolved. The section header on the dashboard groups it correctly without any user input. The aggregate graph defaults to the most-watched watershed. The poll tier elevates on first view.

The watchlist is persisted to localStorage (Pinia persist plugin) and refreshed via BatchGet on every mount. It is the single source of truth for the dashboard state.

*March 2026*

---

*This document should be updated when significant decisions are revisited or new decisions are made. Date and context every entry.*

*Initial decisions: March 2026*
