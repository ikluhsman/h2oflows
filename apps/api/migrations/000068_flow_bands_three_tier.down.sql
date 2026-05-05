-- Restore 4-band schema (too_low / running / high / very_high) with craft_type.
-- Reads legacy_band_data JSONB to reconstruct original rows.

BEGIN;

-- 1. Re-add old columns.
ALTER TABLE flow_ranges
  ADD COLUMN craft_type TEXT NOT NULL DEFAULT 'general',
  ADD COLUMN min_cfs    NUMERIC,
  ADD COLUMN max_cfs    NUMERIC;

-- 2. Expand legacy_band_data arrays into a staging table.
CREATE TEMP TABLE _flow_bands_legacy AS
SELECT
  fr.reach_id,
  fr.gauge_id,
  fr.data_source,
  fr.verified,
  (elem->>'label')::text        AS label,
  (elem->>'min_value')::numeric AS min_cfs,
  (elem->>'max_value')::numeric AS max_cfs,
  COALESCE(elem->>'craft_type', 'general') AS craft_type
FROM flow_ranges fr,
  jsonb_array_elements(fr.legacy_band_data) AS elem
WHERE fr.legacy_band_data IS NOT NULL;

DELETE FROM flow_ranges;

INSERT INTO flow_ranges (reach_id, gauge_id, label, min_cfs, max_cfs, craft_type, data_source, verified)
SELECT reach_id, gauge_id, label, min_cfs, max_cfs, craft_type, data_source, verified
FROM _flow_bands_legacy;

DROP TABLE _flow_bands_legacy;

-- 3. Drop new columns.
ALTER TABLE flow_ranges DROP COLUMN min_value;
ALTER TABLE flow_ranges DROP COLUMN max_value;
ALTER TABLE flow_ranges DROP COLUMN legacy_band_data;

-- 4. Swap constraints back.
ALTER TABLE flow_ranges DROP CONSTRAINT flow_ranges_label_check;
ALTER TABLE flow_ranges DROP CONSTRAINT flow_ranges_reach_id_label_key;

ALTER TABLE flow_ranges ADD CONSTRAINT flow_ranges_label_check
  CHECK (label = ANY (ARRAY['too_low'::text, 'running'::text, 'high'::text, 'very_high'::text]));
ALTER TABLE flow_ranges ADD CONSTRAINT flow_ranges_reach_id_label_craft_key
  UNIQUE (reach_id, label, craft_type);
ALTER TABLE flow_ranges ADD CONSTRAINT flow_ranges_craft_type_check
  CHECK (craft_type = ANY (ARRAY['general'::text, 'kayak'::text, 'raft'::text, 'sup'::text, 'packraft'::text, 'canoe'::text]));

COMMIT;
