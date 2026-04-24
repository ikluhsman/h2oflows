#!/usr/bin/env bash
# pull-prod-db.sh — Replace local dev database with a fresh dump from production.
#
# One-time setup (run once, then never again):
#   ssh-keygen -t ed25519 -f ~/.ssh/h2oflow_prod -N ""
#   ssh-copy-id -i ~/.ssh/h2oflow_prod.pub ubuntu@52.88.134.14
#
# Then just run:
#   ./scripts/pull-prod-db.sh

set -euo pipefail

PROD_HOST="ubuntu@52.88.134.14"
PROD_DIR="/home/ubuntu/h2oflows"          # project directory on the server
SSH_KEY="$HOME/.ssh/h2oflow_prod"
LOCAL_PGURL="postgres://h2oflow:h2oflow@localhost:5432"
DUMP="/tmp/h2oflow-prod-$(date +%Y%m%d-%H%M%S).dump"

# ── sanity checks ─────────────────────────────────────────────────────────────
if [[ ! -f "$SSH_KEY" ]]; then
  echo "ERROR: SSH key not found at $SSH_KEY"
  echo "Run: ssh-keygen -t ed25519 -f $SSH_KEY -N \"\""
  echo "     ssh-copy-id -i ${SSH_KEY}.pub $PROD_HOST"
  exit 1
fi

echo "▶ Dumping production database..."
ssh -i "$SSH_KEY" -o StrictHostKeyChecking=accept-new "$PROD_HOST" \
  "cd $PROD_DIR && docker compose -f docker-compose.prod.yml exec -T postgres \
   pg_dump -U h2oflow -Fc h2oflow" \
  > "$DUMP"

SIZE=$(du -sh "$DUMP" | cut -f1)
echo "  dump: $DUMP ($SIZE)"

echo "▶ Recreating local database..."
psql "${LOCAL_PGURL}/postgres?sslmode=disable" -c "DROP DATABASE IF EXISTS h2oflow;"
psql "${LOCAL_PGURL}/postgres?sslmode=disable" -c "CREATE DATABASE h2oflow OWNER h2oflow;"

echo "▶ Restoring (as postgres superuser for extensions)..."
# Must restore as superuser so PostGIS/pgvector extensions can be created.
sudo -u postgres pg_restore \
  -d "dbname=h2oflow" \
  --no-privileges \
  "$DUMP" 2>&1 \
  | grep -v "already exists\|COMMENT ON EXTENSION\|topology\.\|tiger\|spatial_ref_sys" \
  | grep "^pg_restore: error" \
  || true   # non-zero exit from grep when no errors is fine

echo "▶ Cleaning up..."
rm "$DUMP"

echo ""
echo "✓ Local database is now in sync with production."
psql "${LOCAL_PGURL}/h2oflow?sslmode=disable" -c "
  SELECT
    (SELECT count(*) FROM reaches)     AS reaches,
    (SELECT count(*) FROM gauges)      AS gauges,
    (SELECT count(*) FROM flow_ranges) AS flow_ranges,
    (SELECT count(*) FROM rivers)      AS rivers,
    (SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1) AS migration;
"
