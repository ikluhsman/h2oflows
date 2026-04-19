-- HUC4 1019 is the Cache La Poudre subregion, which drains into the South Platte
-- (not the North Platte). Clear the stale watershed_name so the next metadata
-- sync re-derives correctly using the fixed CanonicalBasin logic.

UPDATE gauges
SET watershed_name = NULL
WHERE huc8 LIKE '1019%'
  AND watershed_name = 'North Platte';

-- Clear river.basin values that were propagated from the wrong watershed_name.
-- Only clears rivers whose primary-gauge HUC8 starts with 1019 — rivers that
-- legitimately cover true North Platte territory (HUC4 1023) are untouched.
UPDATE rivers rv
SET    basin = NULL
WHERE  basin = 'North Platte'
  AND  id IN (
      SELECT DISTINCT re.river_id
      FROM   reaches re
      JOIN   gauges  g ON g.id = re.primary_gauge_id
      WHERE  g.huc8 LIKE '1019%'
  );
