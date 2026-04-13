CREATE TABLE contributions (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                 UUID,
    device_id               TEXT,
    reach_id                UUID NOT NULL REFERENCES reaches(id) ON DELETE CASCADE,
    contribution_type       TEXT NOT NULL CHECK (contribution_type IN ('flow_update','hazard_alert','general')),
    flow_impression         TEXT CHECK (flow_impression IN ('too_low','good','high')),
    body                    TEXT,
    observed_at             TIMESTAMPTZ NOT NULL,
    cfs_at_time             NUMERIC(10,2),
    share_consent_h2oflows  BOOLEAN DEFAULT FALSE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_contributions_reach ON contributions (reach_id);
