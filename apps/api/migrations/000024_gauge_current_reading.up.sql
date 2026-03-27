-- Denormalized current reading on gauges for fast search results without a join.
-- The poller updates these whenever a new reading is written.
ALTER TABLE gauges
    ADD COLUMN current_cfs   NUMERIC,
    ADD COLUMN flow_status   TEXT NOT NULL DEFAULT 'unknown'
        CHECK (flow_status IN ('runnable', 'caution', 'low', 'flood', 'unknown'));

-- Backfill current_cfs from the latest reading for each gauge.
UPDATE gauges g
SET current_cfs = r.value
FROM (
    SELECT DISTINCT ON (gauge_id) gauge_id, value
    FROM gauge_readings
    ORDER BY gauge_id, timestamp DESC
) r
WHERE g.id = r.gauge_id;
