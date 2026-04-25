DROP INDEX IF EXISTS rivers_gnis_id_idx;
ALTER TABLE rivers DROP COLUMN IF EXISTS gnis_id;
