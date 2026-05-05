-- Collapse flow_ranges from 4 bands (too_low / running / high / very_high)
-- to 3 bands (low / running / high) and drop the craft_type column.
--
-- Mapping (4-tier → 3-tier, aggregated per reach across all craft types):
--   low.max_value     = too_low.max_cfs
--   running.min_value = MIN(running.min_cfs across craft types)
--   running.max_value = COALESCE(very_high.min_cfs, high.max_cfs, running.max_cfs)
--   high.min_value    = running.max_value (boundary mirror)
--
-- legacy_band_data JSONB retained for rollback window; dropped after Phase 2 ships.

BEGIN;

-- 1. Add new columns alongside old ones.
ALTER TABLE flow_ranges
  ADD COLUMN legacy_band_data JSONB,
  ADD COLUMN min_value        NUMERIC,
  ADD COLUMN max_value        NUMERIC;

-- 2. Snapshot current data; copy values to new columns.
UPDATE flow_ranges SET
  legacy_band_data = jsonb_build_object(
    'label',      label,
    'min_cfs',    min_cfs,
    'max_cfs',    max_cfs,
    'craft_type', craft_type
  ),
  min_value = min_cfs,
  max_value = max_cfs;

-- 3. Compute 3-band values per reach (aggregate across craft types).
CREATE TEMP TABLE _flow_bands_new AS
SELECT
  reach_id,
  COALESCE(
    (array_agg(gauge_id) FILTER (WHERE label = 'running'))[1],
    (array_agg(gauge_id ORDER BY label))[1]
  )                                                              AS gauge_id,
  (array_agg(data_source ORDER BY CASE data_source
    WHEN 'manual'  THEN 1
    WHEN 'ai_web'  THEN 2
    WHEN 'ai_seed' THEN 3
    ELSE 4 END))[1]                                            AS data_source,
  BOOL_OR(verified)                                            AS verified,
  jsonb_agg(jsonb_build_object(
    'label',      label,
    'min_value',  min_value,
    'max_value',  max_value,
    'craft_type', craft_type
  ))                                                            AS legacy,
  MAX(CASE WHEN label = 'too_low'   THEN max_value END)        AS low_max,
  MIN(CASE WHEN label = 'running'   THEN min_value END)        AS running_min,
  COALESCE(
    MIN(CASE WHEN label = 'very_high' THEN min_value END),
    MAX(CASE WHEN label = 'high'      THEN max_value END),
    MAX(CASE WHEN label = 'running'   THEN max_value END)
  )                                                            AS running_max
FROM flow_ranges
GROUP BY reach_id;

-- 4. Drop constraints that would block inserts with new labels.
ALTER TABLE flow_ranges DROP CONSTRAINT flow_ranges_label_check;
ALTER TABLE flow_ranges DROP CONSTRAINT flow_ranges_craft_type_check;
ALTER TABLE flow_ranges DROP CONSTRAINT flow_ranges_reach_id_label_craft_key;

-- 5. Replace all rows with 3-band rows.
DELETE FROM flow_ranges;

INSERT INTO flow_ranges (reach_id, gauge_id, label, min_value, max_value, data_source, verified, legacy_band_data)
SELECT reach_id, gauge_id, 'low', NULL, low_max, data_source, verified, legacy
FROM _flow_bands_new
WHERE low_max IS NOT NULL;

INSERT INTO flow_ranges (reach_id, gauge_id, label, min_value, max_value, data_source, verified, legacy_band_data)
SELECT reach_id, gauge_id, 'running', running_min, running_max, data_source, verified, legacy
FROM _flow_bands_new
WHERE running_min IS NOT NULL OR running_max IS NOT NULL;

INSERT INTO flow_ranges (reach_id, gauge_id, label, min_value, max_value, data_source, verified, legacy_band_data)
SELECT reach_id, gauge_id, 'high', running_max, NULL, data_source, verified, legacy
FROM _flow_bands_new
WHERE running_max IS NOT NULL;

DROP TABLE _flow_bands_new;

-- 6. Drop old columns.
ALTER TABLE flow_ranges DROP COLUMN min_cfs;
ALTER TABLE flow_ranges DROP COLUMN max_cfs;
ALTER TABLE flow_ranges DROP COLUMN craft_type;

-- 7. Add new label CHECK and unique constraint.
ALTER TABLE flow_ranges ADD CONSTRAINT flow_ranges_label_check
  CHECK (label IN ('low', 'running', 'high'));
ALTER TABLE flow_ranges ADD CONSTRAINT flow_ranges_reach_id_label_key
  UNIQUE (reach_id, label);

COMMIT;
