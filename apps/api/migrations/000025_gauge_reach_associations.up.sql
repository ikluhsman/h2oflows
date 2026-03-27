-- gauge_reach_associations replaces the 1:1 reach_id column with a proper
-- many-to-many join table. A single gauge (e.g. PLAGRACO at Grant) can now
-- serve as an upstream indicator for multiple reaches (Bailey, Foxton).
-- A gauge at a confluence can be the downstream indicator for several runs.
--
-- gauges.reach_id is kept as a convenience pointer to the gauge's primary
-- display reach (used for quick JOINs in search/card display). The association
-- table is the authoritative many-to-many record.

CREATE TABLE gauge_reach_associations (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    gauge_id     UUID        NOT NULL REFERENCES gauges(id)  ON DELETE CASCADE,
    reach_id     UUID        NOT NULL REFERENCES reaches(id) ON DELETE CASCADE,
    relationship TEXT        NOT NULL DEFAULT 'primary'
                             CHECK (relationship IN (
                                 'primary',
                                 'upstream_indicator',
                                 'downstream_indicator',
                                 'tributary'
                             )),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (gauge_id, reach_id)
);

CREATE INDEX gauge_reach_assoc_reach_idx ON gauge_reach_associations (reach_id);
CREATE INDEX gauge_reach_assoc_gauge_idx ON gauge_reach_associations (gauge_id);

-- Seed from existing gauges.reach_id / reach_relationship data so nothing is lost.
INSERT INTO gauge_reach_associations (gauge_id, reach_id, relationship)
SELECT id, reach_id, COALESCE(reach_relationship, 'primary')
FROM gauges
WHERE reach_id IS NOT NULL
ON CONFLICT DO NOTHING;
