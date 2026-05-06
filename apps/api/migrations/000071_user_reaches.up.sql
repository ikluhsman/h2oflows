CREATE TABLE user_reaches (
  id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id          TEXT        NOT NULL,
  slug              TEXT        NOT NULL,
  name              TEXT        NOT NULL,
  river_id          UUID        REFERENCES rivers(id) ON DELETE SET NULL,
  river_name        TEXT,
  put_in            GEOGRAPHY(POINT, 4326)      NOT NULL,
  take_out          GEOGRAPHY(POINT, 4326)      NOT NULL,
  centerline        GEOGRAPHY(LINESTRING, 4326),
  up_comid          TEXT,
  down_comid        TEXT,
  primary_gauge_id  UUID        REFERENCES gauges(id) ON DELETE SET NULL,
  note              TEXT,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (owner_id, slug)
);

CREATE INDEX user_reaches_owner_idx  ON user_reaches (owner_id);
CREATE INDEX user_reaches_river_idx  ON user_reaches (river_id);
CREATE INDEX user_reaches_gauge_idx  ON user_reaches (primary_gauge_id);

CREATE TABLE user_reach_flow_ranges (
  id             UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
  user_reach_id  UUID    NOT NULL REFERENCES user_reaches(id) ON DELETE CASCADE,
  label          TEXT    NOT NULL CHECK (label IN ('low','running','high')),
  min_value      NUMERIC,
  max_value      NUMERIC,
  UNIQUE (user_reach_id, label)
);
