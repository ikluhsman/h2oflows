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
