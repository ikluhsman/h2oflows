CREATE TABLE gauge_readings (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    gauge_id    UUID        NOT NULL REFERENCES gauges(id) ON DELETE CASCADE,
    value       NUMERIC(12,3) NOT NULL,
    unit        TEXT        NOT NULL DEFAULT 'cfs',
    timestamp   TIMESTAMPTZ NOT NULL,
    qual_code   TEXT,
    provisional BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Prevent duplicate readings from the poller inserting the same measurement twice
    UNIQUE (gauge_id, timestamp)
);

-- Primary query pattern: latest N readings for a gauge
CREATE INDEX gauge_readings_gauge_timestamp_idx ON gauge_readings (gauge_id, timestamp DESC);

-- Time-range queries across all gauges (nightly scoring, history export)
CREATE INDEX gauge_readings_timestamp_idx ON gauge_readings (timestamp DESC);
