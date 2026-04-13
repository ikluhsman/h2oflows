CREATE TABLE proximity_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id   TEXT NOT NULL,
    user_id     UUID,
    reach_id    UUID NOT NULL REFERENCES reaches(id) ON DELETE CASCADE,
    event_type  TEXT NOT NULL CHECK (event_type IN ('enter','exit','dwell')),
    location    GEOGRAPHY(POINT, 4326),
    detected_at TIMESTAMPTZ NOT NULL,
    promoted_to UUID REFERENCES trip_reports(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_proximity_device ON proximity_events (device_id);
CREATE INDEX idx_proximity_reach  ON proximity_events (reach_id);
