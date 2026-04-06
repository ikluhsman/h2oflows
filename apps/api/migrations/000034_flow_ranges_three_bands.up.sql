-- Simplify flow ranges from 7 bands to 3:
--   below_recommended (red), runnable (green), above_recommended (blue)

-- Update the check constraint
ALTER TABLE flow_ranges DROP CONSTRAINT flow_ranges_label_check;
ALTER TABLE flow_ranges ADD CONSTRAINT flow_ranges_label_check
  CHECK (label = ANY (ARRAY[
    'below_recommended', 'runnable', 'above_recommended',
    -- legacy labels kept temporarily for migration
    'too_low', 'minimum', 'fun', 'optimal', 'pushy', 'high', 'flood'
  ]));

-- Collapse existing 7 bands into 3 per gauge
CREATE TEMP TABLE new_ranges AS
SELECT gauge_id, 'below_recommended'::text AS label,
       MIN(min_cfs) AS min_cfs,
       MAX(CASE WHEN label = 'minimum' THEN max_cfs END) AS max_cfs
FROM flow_ranges WHERE label IN ('too_low', 'minimum')
GROUP BY gauge_id
UNION ALL
SELECT gauge_id, 'runnable',
       MIN(CASE WHEN label = 'fun' THEN min_cfs END),
       COALESCE(
         MAX(CASE WHEN label = 'pushy' THEN max_cfs END),
         MAX(CASE WHEN label = 'high' THEN min_cfs END)
       )
FROM flow_ranges WHERE label IN ('fun', 'optimal', 'pushy')
GROUP BY gauge_id
UNION ALL
SELECT gauge_id, 'above_recommended',
       COALESCE(
         MIN(CASE WHEN label = 'high' THEN min_cfs END),
         MIN(CASE WHEN label = 'flood' THEN min_cfs END)
       ),
       MAX(max_cfs)
FROM flow_ranges WHERE label IN ('high', 'flood')
GROUP BY gauge_id;

DELETE FROM flow_ranges;
INSERT INTO flow_ranges (gauge_id, label, min_cfs, max_cfs)
SELECT gauge_id, label, min_cfs, max_cfs FROM new_ranges;

DROP TABLE new_ranges;

-- Tighten constraint to only the 3 new labels
ALTER TABLE flow_ranges DROP CONSTRAINT flow_ranges_label_check;
ALTER TABLE flow_ranges ADD CONSTRAINT flow_ranges_label_check
  CHECK (label = ANY (ARRAY['below_recommended', 'runnable', 'above_recommended']));
