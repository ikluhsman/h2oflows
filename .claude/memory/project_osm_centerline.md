---
name: OSM centerline fetch approach
description: How the FetchReachLine algorithm works and what pitfalls to avoid
type: project
---

FetchReachLine uses Overpass to chain OSM waterway ways into a reach centerline.

Key design decisions that were hard-won:
- Use MIN(lng) put-in and MAX(lng) take-out as chain anchors (not AVG) — averaging mixed upstream/downstream access points and caused trimChainEnd to cut the chain short
- Prefer waterway=river ways over waterway=stream to avoid chaining up tributaries (Buffalo Creek was getting picked up instead of North Fork South Platte)
- trimChainStart/trimChainEnd snap the line to the nearest OSM node to the put-in/take-out, so the line doesn't extend past access points
- transition:scale not transition:transform on marker elements — MapLibre repositions via transform, so transitioning it makes pins float/lag during pan/zoom
- Overpass falls back through 3 endpoints (overpass-api.de → overpass.kumi.systems → maps.mail.ru) on 429/5xx

**Why:** Colorado rivers flow roughly west→east so MIN(lng)=most upstream put-in, MAX(lng)=most downstream take-out is a valid heuristic.
**How to apply:** If a reach has multiple put-ins at different sections (like Foxton Boulder Garden vs Standard Foxton), the extreme coordinates pick the full reach extent automatically.
