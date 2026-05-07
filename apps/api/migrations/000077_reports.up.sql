CREATE TABLE reports (
  id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id       TEXT        NOT NULL,
  slug           TEXT        NOT NULL,
  reach_id       UUID        NOT NULL REFERENCES reaches(id) ON DELETE CASCADE,
  name           TEXT        NOT NULL,
  report_date    DATE        NOT NULL,
  report_time    TIME,
  content        TEXT        NOT NULL,
  hazard_warning TEXT,
  paddled        BOOLEAN     NOT NULL DEFAULT FALSE,
  flow_cfs       NUMERIC,
  flow_band      TEXT        CHECK (flow_band IN ('low', 'running', 'high')),
  aw_synced_at   TIMESTAMPTZ,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (owner_id, slug)
);
CREATE INDEX reports_reach_idx  ON reports (reach_id, report_date DESC);
CREATE INDEX reports_owner_idx  ON reports (owner_id);
CREATE INDEX reports_hazard_idx ON reports (reach_id) WHERE hazard_warning IS NOT NULL;

CREATE TABLE report_photos (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  report_id   UUID        NOT NULL REFERENCES reports(id) ON DELETE CASCADE,
  position    SMALLINT    NOT NULL,
  storage_key TEXT        NOT NULL,
  caption     TEXT,
  taken_at    TIMESTAMPTZ,
  exif_lat    DOUBLE PRECISION,
  exif_lng    DOUBLE PRECISION
);
CREATE INDEX report_photos_report_idx ON report_photos (report_id, position);
