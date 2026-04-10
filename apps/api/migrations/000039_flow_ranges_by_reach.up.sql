-- Move flow_ranges from per-gauge to per-reach.
--
-- A gauge may be the primary gauge for multiple reaches (e.g. "Grant" gauge
-- serves both Bailey and Foxton).  With gauge-scoped ranges, both reaches
-- were forced to share the same optimal CFS windows — wrong because each
-- reach has a different character at the same reading.
--
-- Migration strategy:
--   1. Add reach_id column (nullable during data fill).
--   2. Assign reach_id = alphabetically-first reach for each gauge's rows.
--   3. Drop old unique constraint so we can insert duplicates.
--   4. Duplicate rows for every additional reach that shares the same gauge.
--   5. Tighten: NOT NULL + new unique constraint on (reach_id, label, craft_type).
--   6. Add index; keep gauge_id for reference (CFS scale stays gauge-specific).

-- 1. Add column
ALTER TABLE flow_ranges
  ADD COLUMN reach_id UUID REFERENCES reaches(id) ON DELETE CASCADE;

-- 2. Populate reach_id from the reach that uses this gauge as primary.
--    ORDER BY slug is deterministic so reruns are idempotent.
UPDATE flow_ranges fr
SET reach_id = (
  SELECT r.id
  FROM reaches r
  WHERE r.primary_gauge_id = fr.gauge_id
  ORDER BY r.slug
  LIMIT 1
);

-- Discard any rows whose gauge is not a primary gauge for any reach
-- (shouldn't exist per pre-migration check, but be safe).
DELETE FROM flow_ranges WHERE reach_id IS NULL;

-- 3. Drop old constraint before inserting duplicate (gauge_id, label, craft_type) combos.
ALTER TABLE flow_ranges
  DROP CONSTRAINT flow_ranges_gauge_id_label_craft_key;

-- 4. For gauges that are primary for more than one reach, clone existing rows
--    for every reach beyond the first.
INSERT INTO flow_ranges
  (gauge_id, reach_id, label, min_cfs, max_cfs, craft_type,
   class_modifier, source_url, data_source, ai_confidence, verified)
SELECT
  fr.gauge_id,
  extra.id          AS reach_id,
  fr.label, fr.min_cfs, fr.max_cfs, fr.craft_type,
  fr.class_modifier, fr.source_url, fr.data_source, fr.ai_confidence, fr.verified
FROM flow_ranges fr
JOIN reaches extra
  ON  extra.primary_gauge_id = fr.gauge_id
  AND extra.id               != fr.reach_id;

-- 5. Make reach_id mandatory and add new unique constraint.
ALTER TABLE flow_ranges
  ALTER COLUMN reach_id SET NOT NULL;

ALTER TABLE flow_ranges
  ADD CONSTRAINT flow_ranges_reach_id_label_craft_key
  UNIQUE (reach_id, label, craft_type);

-- 6. Index for fast reach lookup.
CREATE INDEX flow_ranges_reach_id_idx ON flow_ranges (reach_id);
