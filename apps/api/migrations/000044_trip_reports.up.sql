CREATE TABLE trip_reports (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                 UUID,
    device_id               TEXT,
    reach_id                UUID NOT NULL REFERENCES reaches(id) ON DELETE CASCADE,
    title                   TEXT,
    body                    TEXT,
    flow_impression         TEXT CHECK (flow_impression IN ('too_low','good','high')),
    observed_at             TIMESTAMPTZ NOT NULL,
    cfs_at_time             NUMERIC(10,2),
    photos                  JSONB DEFAULT '[]'::jsonb,
    public_slug             TEXT UNIQUE,
    share_consent_h2oflows  BOOLEAN DEFAULT FALSE,
    published               BOOLEAN DEFAULT FALSE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_trip_reports_reach ON trip_reports (reach_id);
CREATE INDEX idx_trip_reports_slug  ON trip_reports (public_slug) WHERE public_slug IS NOT NULL;
