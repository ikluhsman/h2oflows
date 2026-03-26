CREATE TABLE gauges (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    reach_id             UUID        REFERENCES reaches(id) ON DELETE SET NULL,
    external_id          TEXT        NOT NULL,
    source               TEXT        NOT NULL, -- usgs/dwr/cdec/manual/community
    name                 TEXT,
    location             GEOGRAPHY(POINT, 4326),
    param_code           TEXT        NOT NULL DEFAULT '00060',

    -- Lifecycle fields (see DECISIONS.md — gauge lifecycle model)
    status               TEXT        NOT NULL DEFAULT 'active'
                                     CHECK (status IN ('active','seasonal','inactive','retired','maintenance')),
    seasonal_start_mmdd  CHAR(5),    -- 'MM-DD', null if not seasonal
    seasonal_end_mmdd    CHAR(5),    -- 'MM-DD'
    successor_id         UUID        REFERENCES gauges(id) ON DELETE SET NULL,
    last_reading_at      TIMESTAMPTZ,
    consecutive_failures INT         NOT NULL DEFAULT 0,
    auto_managed         BOOLEAN     NOT NULL DEFAULT TRUE,
    notes                TEXT,

    -- Prominence fields (see DECISIONS.md — gauge prominence model)
    featured             BOOLEAN     NOT NULL DEFAULT FALSE,
    prominence_score     NUMERIC     NOT NULL DEFAULT 0,

    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- A gauge is uniquely identified by its external ID within a source
    UNIQUE (external_id, source)
);

-- Resolve the circular FK between reaches and gauges.
-- reaches.primary_gauge_id was created without a constraint; add it now.
ALTER TABLE reaches
    ADD CONSTRAINT reaches_primary_gauge_id_fkey
    FOREIGN KEY (primary_gauge_id)
    REFERENCES gauges(id)
    ON DELETE SET NULL;

-- Indexes to support the gauge search API
-- (/api/v1/gauges/search?bbox=...&q=...&source=...&limit=...)
CREATE INDEX gauges_location_idx      ON gauges USING GIST (location);
CREATE INDEX gauges_reach_id_idx      ON gauges (reach_id);
CREATE INDEX gauges_source_idx        ON gauges (source);
CREATE INDEX gauges_status_idx        ON gauges (status);
CREATE INDEX gauges_featured_idx      ON gauges (featured) WHERE featured = TRUE;
CREATE INDEX gauges_prominence_idx    ON gauges (prominence_score DESC);
CREATE INDEX gauges_external_id_idx   ON gauges (external_id);
CREATE INDEX gauges_name_trgm_idx     ON gauges USING GIN (name gin_trgm_ops);
