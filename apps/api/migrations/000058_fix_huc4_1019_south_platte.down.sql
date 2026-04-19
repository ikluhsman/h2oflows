-- Restore old (incorrect) North Platte assignment for HUC4 1019 gauges.
-- This is the down migration; running it re-introduces the bug.
UPDATE gauges
SET watershed_name = 'North Platte'
WHERE huc8 LIKE '1019%'
  AND watershed_name IS NULL;
