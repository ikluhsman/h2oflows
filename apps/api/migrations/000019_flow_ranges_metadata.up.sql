-- Add provenance and craft-type columns to flow_ranges.
-- source_url: the AW page or other source the AI cited (empty for manual entries).
-- craft_type: ranges can differ per craft; 'general' applies to all.
-- ai_confidence: 0–100, null for manually-entered rows.
-- data_source: 'manual' (human-entered) | 'ai_seed' (training knowledge) | 'ai_web' (live web search).
-- verified: true once a human or high-confidence AI pass has confirmed the values.

ALTER TABLE flow_ranges
    ADD COLUMN IF NOT EXISTS source_url    TEXT,
    ADD COLUMN IF NOT EXISTS craft_type    TEXT NOT NULL DEFAULT 'general'
                               CHECK (craft_type IN ('general','kayak','raft','sup','packraft','canoe')),
    ADD COLUMN IF NOT EXISTS ai_confidence SMALLINT CHECK (ai_confidence BETWEEN 0 AND 100),
    ADD COLUMN IF NOT EXISTS data_source   TEXT NOT NULL DEFAULT 'manual'
                               CHECK (data_source IN ('manual','ai_seed','ai_web')),
    ADD COLUMN IF NOT EXISTS verified      BOOLEAN NOT NULL DEFAULT FALSE;

-- Existing UNIQUE(gauge_id, label) is too narrow — one gauge can have separate
-- ranges for kayak vs raft (e.g. Browns Canyon minimum kayak=300cfs, raft=600cfs).
ALTER TABLE flow_ranges
    DROP CONSTRAINT IF EXISTS flow_ranges_gauge_id_label_key;

ALTER TABLE flow_ranges
    ADD CONSTRAINT flow_ranges_gauge_id_label_craft_key
    UNIQUE (gauge_id, label, craft_type);

-- High-confidence manual entries should be marked verified on insert.
-- Backfill existing rows (all manual, pre-AI) as verified.
UPDATE flow_ranges SET verified = TRUE WHERE data_source = 'manual';
