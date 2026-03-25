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

### Why no social login (Google, GitHub OAuth)

Several reasons:

**Privacy.** The whitewater community skews toward people who are outdoorsy, privacy-conscious, and skeptical of big tech data collection. "Sign in with Google" sends a signal that conflicts with the FOSS/community-owned positioning.

**Dependency.** Social login creates a dependency on a third-party auth provider. If Google changes their OAuth terms or a provider goes down, login breaks.

**Simplicity.** Email/password with JWT is straightforward to implement, straightforward to self-host, and has no external dependencies. Self-hosted instances don't need to register OAuth apps.

The tradeoff is slightly more friction at signup. This is judged acceptable — the people who will use H2OFlow are motivated enough to create an account with an email address.

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

*This document should be updated when significant decisions are revisited or new decisions are made. Date and context every entry.*

*Initial decisions: March 2026*
