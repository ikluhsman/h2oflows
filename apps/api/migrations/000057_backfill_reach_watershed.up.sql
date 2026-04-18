-- Backfill reaches.watershed_name for reaches that currently have NULL.
-- Priority 1: derive from the reach's primary gauge (most accurate — gauge has HUC8).
UPDATE reaches r
SET watershed_name = g.watershed_name
FROM gauges g
WHERE r.primary_gauge_id = g.id
  AND r.watershed_name IS NULL
  AND g.watershed_name IS NOT NULL;

-- Priority 2: fall back to the river's basin for any reach still missing a watershed.
UPDATE reaches r
SET watershed_name = rv.basin
FROM rivers rv
WHERE r.river_id = rv.id
  AND r.watershed_name IS NULL
  AND rv.basin IS NOT NULL AND rv.basin <> '';
