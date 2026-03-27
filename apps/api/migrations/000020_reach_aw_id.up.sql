-- American Whitewater reach ID for cross-referencing AW's public dataset.
-- Used by the FlowRangeSeeder to construct source_url and give Claude precise context.
-- Also useful for future AW data sync and deduplication.
ALTER TABLE reaches
    ADD COLUMN IF NOT EXISTS aw_reach_id TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS reaches_aw_reach_id_idx
    ON reaches (aw_reach_id)
    WHERE aw_reach_id IS NOT NULL;
