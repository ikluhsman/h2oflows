CREATE TABLE user_profiles (
  owner_id    TEXT PRIMARY KEY,
  handle      TEXT NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (handle)
);
CREATE INDEX user_profiles_handle_idx ON user_profiles (handle);
