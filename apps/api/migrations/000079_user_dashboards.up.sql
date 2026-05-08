CREATE TABLE user_dashboards (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id    TEXT        NOT NULL,
  slug        TEXT        NOT NULL,
  name        TEXT        NOT NULL,
  position    INTEGER     NOT NULL DEFAULT 0,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (owner_id, slug)
);
CREATE INDEX user_dashboards_owner_idx ON user_dashboards (owner_id, position);

-- Add nullable FK first so existing rows don't violate NOT NULL yet.
ALTER TABLE user_watchlists
  ADD COLUMN dashboard_id UUID REFERENCES user_dashboards(id) ON DELETE CASCADE;

-- Backfill: create a default dashboard for every user who has watchlist rows.
INSERT INTO user_dashboards (owner_id, slug, name, position)
SELECT DISTINCT user_id, 'default', 'My Dashboard', 0
FROM user_watchlists
ON CONFLICT DO NOTHING;

-- Point every existing watchlist row at its owner's default dashboard.
UPDATE user_watchlists w
SET    dashboard_id = d.id
FROM   user_dashboards d
WHERE  d.owner_id = w.user_id AND d.slug = 'default';
