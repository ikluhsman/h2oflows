CREATE TABLE reach_conditions (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    reach_id      UUID        NOT NULL REFERENCES reaches(id) ON DELETE CASCADE,
    source_type   TEXT        NOT NULL DEFAULT 'personal'
                              CHECK (source_type IN ('gauge','personal','word-of-mouth','discord','outfitter')),
    summary       TEXT        NOT NULL,
    runnable      BOOLEAN,
    reported_by   UUID,       -- FK to users added in Phase 3 migration
    cfs_at_report NUMERIC(10,2),
    expires_at    TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '7 days',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX reach_conditions_reach_id_idx  ON reach_conditions (reach_id);
CREATE INDEX reach_conditions_expires_at_idx ON reach_conditions (expires_at);

-- Index for active conditions queries (filter by expires_at in queries, not index predicate)
CREATE INDEX reach_conditions_active_idx
    ON reach_conditions (reach_id, expires_at DESC, created_at DESC);
