CREATE TABLE flow_ranges (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    gauge_id       UUID        NOT NULL REFERENCES gauges(id) ON DELETE CASCADE,
    label          TEXT        NOT NULL
                               CHECK (label IN ('too_low','minimum','fun','optimal','pushy','high','flood')),
    min_cfs        NUMERIC(10,2),
    max_cfs        NUMERIC(10,2),
    class_modifier NUMERIC(3,1), -- how difficulty shifts at this flow band
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- One row per label per gauge
    UNIQUE (gauge_id, label)
);

CREATE INDEX flow_ranges_gauge_id_idx ON flow_ranges (gauge_id);
