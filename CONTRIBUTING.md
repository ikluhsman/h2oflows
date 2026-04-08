# Contributing to H2OFlows

H2OFlows is an open data platform for the whitewater community. We welcome contributions of all kinds — code, river data, flow range corrections, bug reports, and documentation.

---

## Table of contents

- [Project orientation](#project-orientation)
- [Local development setup](#local-development-setup)
- [Environment variables](#environment-variables)
- [Running the stack](#running-the-stack)
- [Seeding reach data](#seeding-reach-data)
- [Making changes](#making-changes)
- [Contributing river data via KMZ](#contributing-river-data-via-kmz)
- [Code conventions](#code-conventions)
- [Pull request process](#pull-request-process)

---

## Project orientation

Read these first — they contain the decisions and reasoning behind the architecture:

- [PROJECT.md](PROJECT.md) — vision, target users, build phases
- [ARCHITECTURE.md](ARCHITECTURE.md) — tech stack, data model, guiding principles
- [DECISIONS.md](DECISIONS.md) — non-obvious decision log

The codebase is a monorepo:

```
apps/api/        Go backend (Chi, pgx v5, PostGIS, pgvector)
apps/web/        Nuxt 4 frontend (Nuxt UI Pro, MapLibre, uPlot)
packages/        Shared Go packages (gauge-core adapters)
docs/            Guides (KMZ import, etc.)
```

---

## Local development setup

### Prerequisites

| Tool | Version | Notes |
|---|---|---|
| Go | 1.23+ | `go version` |
| Node.js | 20+ | `node -v` |
| Docker + Compose | any recent | for PostgreSQL + Redis |

On Ubuntu/WSL:
```sh
# Go (check https://go.dev/dl/ for latest)
wget https://go.dev/dl/go1.23.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc

# Node via nvm
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash
nvm install 20

# Docker Desktop (Windows) talks to WSL automatically.
# On pure Linux: sudo apt install docker.io docker-compose-v2
```

### 1. Start the database and cache

```sh
docker compose up -d
```

This starts:
- **PostgreSQL 16 + PostGIS** on `localhost:5432`
- **Redis 7** on `localhost:6379`

Data is persisted in named Docker volumes (`postgres_data`, `redis_data`).

### 2. Configure the API

```sh
cp apps/api/.env.example apps/api/.env
```

Edit `apps/api/.env`. The minimum required fields:

| Variable | Where to get it |
|---|---|
| `DATABASE_URL` | Already set for the local Docker postgres — no change needed |
| `ANTHROPIC_API_KEY` | [console.anthropic.com](https://console.anthropic.com) — free tier available |
| `VOYAGE_API_KEY` | [dash.voyageai.com](https://dash.voyageai.com) — free tier available |
| `SUPABASE_URL` | See [Auth setup](#auth-setup) below |
| `SUPABASE_JWKS_URL` | See [Auth setup](#auth-setup) below |

The API will start without Supabase configured — auth middleware is disabled and all requests run as anonymous. This is fine for most development work.

### Auth setup (optional for most contributors)

H2OFlows uses Supabase for authentication. Create a free project at [supabase.com](https://supabase.com) and fill in:

```sh
SUPABASE_URL=https://<your-project-ref>.supabase.co
SUPABASE_JWKS_URL=https://<your-project-ref>.supabase.co/auth/v1/.well-known/jwks.json
```

Both values are in your Supabase dashboard under **Project Settings → API**.

You don't need to share a Supabase project with anyone — your own free project is completely isolated. The frontend `.env` needs matching values:

```sh
# apps/web/.env (create if it doesn't exist)
SUPABASE_URL=https://<your-project-ref>.supabase.co
SUPABASE_KEY=<your-anon-key>  # Project Settings → API → anon public
```

---

## Running the stack

### API

```sh
cd apps/api
set -a && source .env && set +a
go run ./cmd/server
```

The API:
- Runs migrations automatically on startup
- Listens on `http://localhost:8080`
- Hot-reload: kill and re-run; the binary is fast to compile

### Frontend

```sh
cd apps/web
npm install
npm run dev
```

The dev server runs on `http://localhost:3000` and proxies API calls to `:8080`.

---

## Seeding reach data

A fresh local database has no reach data. Seed it:

```sh
cd apps/api
set -a && source .env && set +a

# 1. Seed the 19 Front Range Colorado reaches with AI-generated content
go run ./cmd/seed-reaches

# 2. Seed flow ranges for all reach-linked gauges
go run ./cmd/seed-flow-ranges

# 3. Embed reach content into pgvector for the AI river assistant
go run ./cmd/embed-reaches

# 4. Import bulk USGS gauges for Colorado
go run ./cmd/seed-usgs-states
```

Steps 1–3 call the Anthropic and Voyage APIs, so you need those keys set. Step 4 is USGS only — no AI keys needed.

To backfill descriptions for reaches you've imported via KMZ:

```sh
go run ./cmd/seed-reach-descriptions          # all reaches missing a description
go run ./cmd/seed-reach-descriptions -slug my-reach-slug  # single reach
```

---

## Making changes

### Database migrations

Never edit existing migration files. Always add a new numbered pair:

```sh
# Next migration number — check migrations/ for the current highest
ls apps/api/migrations/ | tail -4

# Create new pair
touch apps/api/migrations/000039_my_change.up.sql
touch apps/api/migrations/000039_my_change.down.sql
```

The API applies pending migrations on startup. The down file is required even if it's a no-op.

### Go build

```sh
/usr/local/go/bin/go build ./apps/api/...
```

Run this before committing any Go changes. We don't have CI yet but this will catch compile errors.

### Frontend type checking

```sh
cd apps/web
npm run build   # catches TypeScript errors
```

---

## Contributing river data via KMZ

The fastest way to add reaches is to build a Google My Map and export it as KMZ. The importer handles rapids, put-ins, take-outs, parking, and permanent hazards.

See [docs/kmz-import-guide.md](docs/kmz-import-guide.md) for the full conventions.

**Quick version:**
1. Create a Google My Map with one folder per reach (folder name = reach name)
2. Add pins with prefixes: `Rapid: Name`, `Put-in: Name`, `Take-out: Name`, `Hazard: Name`
3. Export as KMZ (File → Download → KMZ)
4. Import:
   ```sh
   cd apps/api
   go run ./cmd/import-kml -file /path/to/export.kmz
   ```

If the reach doesn't exist in the DB yet, create a stub first via `seed-reaches` or add it manually.

Data quality notes:
- All AI-seeded content is tagged `data_source='ai_seed'` — treat it as a starting point, not ground truth
- Rapids and access points from KMZ imports are tagged `data_source='import', verified=true`
- Community-verified data takes precedence over AI drafts automatically

---

## Code conventions

- **Migrations**: sequential numbered files, never edit existing ones
- **Reach slugs**: the canonical identifier across the codebase (e.g. `arkansas-the-numbers`)
- **Flow difficulty**: stored as floats (`3.5`), rendered as Roman numerals (`III+`)
- **Class ratings**: `class_min`/`class_max` = standard range; `class_hardest` = portage-able hardest (III-IV(V) notation)
- **Coordinates**: PostGIS geography type; `ST_GeomFromGeoJSON($1)::geography` to store GeoJSON
- **pgx v5**: scan `text` columns into `string`, not `[]byte` — pgx won't base64-decode text for you
- **MapLibre markers**: use `transition:scale` not `transition:transform` — MapLibre repositions via transform
- **No `featured` column** for polling decisions — use `reach_id IS NOT NULL` as the trust signal

---

## Pull request process

1. Fork the repo and create a branch off `main`
2. Keep changes focused — one feature or fix per PR
3. Run `go build ./apps/api/...` and `npm run build` in `apps/web` before opening the PR
4. Describe what changed and why in the PR description
5. Data contributions (KMZ imports, flow range corrections) can be submitted as PRs against the seed data files or as issues if you don't want to touch code

We don't have a formal review SLA yet — this is a small team. Tag `@ikluhsman` and expect a response within a few days.

---

## Getting help

- Open an issue for bugs or questions
- The AI-generated reach data is imperfect by design — if you paddle a run and the data is wrong, open a PR or issue with corrections. That's exactly the contribution pipeline we're building.
