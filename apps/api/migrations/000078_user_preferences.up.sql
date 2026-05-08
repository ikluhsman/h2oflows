CREATE TABLE user_preferences (
  owner_id        TEXT        PRIMARY KEY,
  aw_band_mapping JSONB,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
