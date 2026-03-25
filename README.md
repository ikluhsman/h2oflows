# H2OFlow

A streamflow data platform for whitewater paddlers. Live gauge dashboards, community flow ranges, reach registry, and a free API — all in one place.

See [ARCHITECTURE.md](ARCHITECTURE.md) for full technical design and [PROJECT.md](PROJECT.md) for project context and roadmap.

---

## Stack

| Layer | Technology |
|---|---|
| Backend | Go, Chi, PostgreSQL + PostGIS, Redis |
| Frontend | Nuxt 4, Nuxt UI Pro, uPlot, MapLibre GL |
| Infrastructure | Docker, Traefik |

---

## Running locally

### Dependencies only (recommended for development)

Runs PostgreSQL/PostGIS and Redis in containers. Build and run the Go API and Nuxt frontend natively.

```sh
docker compose up
```

### Environment

```sh
cp .env.example .env
# edit .env with your local values
```

---

## Project structure

```
apps/
  api/              Go backend
  web/              Nuxt 4 frontend
  discord-bot/      Discord webhook + slash command service
packages/
  gauge-core/       Gauge source adapter interface + USGS/DWR implementations
  river-data/       Reach schema, seed data, OSM import tools
infra/
  traefik/          Reverse proxy config
docs/
  api.md            Public API reference
  contributing.md
```
