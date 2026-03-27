ALTER TABLE flow_ranges
    DROP CONSTRAINT IF EXISTS flow_ranges_gauge_id_label_craft_key;

ALTER TABLE flow_ranges
    ADD CONSTRAINT flow_ranges_gauge_id_label_key UNIQUE (gauge_id, label);

ALTER TABLE flow_ranges
    DROP COLUMN IF EXISTS source_url,
    DROP COLUMN IF EXISTS craft_type,
    DROP COLUMN IF EXISTS ai_confidence,
    DROP COLUMN IF EXISTS data_source,
    DROP COLUMN IF EXISTS verified;
