DROP INDEX IF EXISTS gauges_name_trgm_idx;
DROP INDEX IF EXISTS gauges_external_id_trgm_idx;
DROP INDEX IF EXISTS reaches_name_trgm_idx;
DROP EXTENSION IF EXISTS pg_trgm;
