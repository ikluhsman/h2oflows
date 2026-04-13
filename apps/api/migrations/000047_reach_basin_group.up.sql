ALTER TABLE reaches ADD COLUMN basin_group TEXT;
CREATE INDEX idx_reaches_basin_group ON reaches (basin_group) WHERE basin_group IS NOT NULL;
