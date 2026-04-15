-- Simplify flow_ranges to 4-band system:
--   too_low, running, high, very_high
--
-- Mapping from old 5-tier system:
--   below_recommended → too_low
--   low_runnable + runnable → running (merged, min of mins, max of maxes)
--   high_runnable → high
--   above_recommended → very_high

ALTER TABLE flow_ranges DROP CONSTRAINT flow_ranges_label_check;

-- Merge low_runnable + runnable rows per (reach_id, gauge_id) into a single
-- 'running' row spanning the union of both ranges. COALESCE handles NULLs:
-- we want MIN across non-null min_cfs, MAX across non-null max_cfs.
WITH merged AS (
  SELECT
    reach_id,
    gauge_id,
    MIN(min_cfs) AS min_cfs,
    MAX(max_cfs) AS max_cfs
  FROM flow_ranges
  WHERE label IN ('low_runnable', 'runnable')
  GROUP BY reach_id, gauge_id
),
deleted AS (
  DELETE FROM flow_ranges
  WHERE label IN ('low_runnable', 'runnable')
  RETURNING 1
)
INSERT INTO flow_ranges (reach_id, gauge_id, label, min_cfs, max_cfs)
SELECT reach_id, gauge_id, 'running', min_cfs, max_cfs FROM merged;

UPDATE flow_ranges SET label = 'too_low'   WHERE label = 'below_recommended';
UPDATE flow_ranges SET label = 'high'      WHERE label = 'high_runnable';
UPDATE flow_ranges SET label = 'very_high' WHERE label = 'above_recommended';

ALTER TABLE flow_ranges ADD CONSTRAINT flow_ranges_label_check
  CHECK (label = ANY (ARRAY[
    'too_low',
    'running',
    'high',
    'very_high'
  ]));
