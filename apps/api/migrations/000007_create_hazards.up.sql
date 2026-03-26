CREATE TABLE hazards (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    reach_id      UUID        NOT NULL REFERENCES reaches(id) ON DELETE CASCADE,
    location      GEOGRAPHY(POINT, 4326),
    hazard_type   TEXT        NOT NULL
                              CHECK (hazard_type IN ('strainer','sieve','undercut','low-head-dam','other')),
    description   TEXT        NOT NULL,
    cfs_at_report NUMERIC(10,2),
    reported_by   UUID,       -- FK to users added in Phase 3 migration
    active        BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX hazards_reach_id_idx  ON hazards (reach_id);
CREATE INDEX hazards_location_idx  ON hazards USING GIST (location);
CREATE INDEX hazards_active_idx    ON hazards (reach_id) WHERE active = TRUE;
