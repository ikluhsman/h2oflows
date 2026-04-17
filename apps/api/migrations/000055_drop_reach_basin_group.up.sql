-- Basin now lives on the rivers table (rivers.basin), seeded by migration 000052.
-- Drop the denormalized column from reaches.
DROP INDEX IF EXISTS idx_reaches_basin_group;
ALTER TABLE reaches DROP COLUMN IF EXISTS basin_group;
