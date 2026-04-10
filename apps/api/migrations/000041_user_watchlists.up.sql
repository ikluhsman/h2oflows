CREATE TABLE user_watchlists (
  id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    TEXT        NOT NULL,
  gauge_id   UUID        NOT NULL REFERENCES gauges(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (user_id, gauge_id)
);

CREATE INDEX user_watchlists_user_id ON user_watchlists (user_id);
