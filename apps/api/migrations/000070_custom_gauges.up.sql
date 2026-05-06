CREATE TABLE custom_gauges (
  id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id       TEXT        NOT NULL,
  slug           TEXT        NOT NULL,
  name           TEXT        NOT NULL,
  description    TEXT,
  note           TEXT,
  unit           TEXT        NOT NULL DEFAULT 'cfs',
  last_value_cfs NUMERIC,
  last_value_at  TIMESTAMPTZ,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (owner_id, slug)
);

CREATE INDEX custom_gauges_owner_idx ON custom_gauges (owner_id);

CREATE TABLE custom_gauge_inputs (
  custom_gauge_id UUID     NOT NULL REFERENCES custom_gauges(id) ON DELETE CASCADE,
  position        SMALLINT NOT NULL,
  gauge_id        UUID     NOT NULL REFERENCES gauges(id) ON DELETE RESTRICT,
  sign            SMALLINT NOT NULL CHECK (sign IN (-1, 1)),
  PRIMARY KEY (custom_gauge_id, position)
);

CREATE INDEX custom_gauge_inputs_gauge_idx ON custom_gauge_inputs (gauge_id);
