ALTER TABLE reaches ADD COLUMN basin_group TEXT;
CREATE INDEX idx_reaches_basin_group ON reaches (basin_group) WHERE basin_group IS NOT NULL;

-- Re-populate from rivers.basin for existing data.
UPDATE reaches r
SET    basin_group = rv.basin
FROM   rivers rv
WHERE  r.river_id = rv.id
  AND  rv.basin IS NOT NULL;
