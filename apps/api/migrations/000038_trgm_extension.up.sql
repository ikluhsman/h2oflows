-- pg_trgm enables trigram similarity matching for fuzzy gauge/reach search.
-- Required for compound-word tolerance ("Elevenmile" → "Eleven Mile") and
-- typo correction ("Grore Canyon" → "Gore Canyon").
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- GIN trigram indexes on the columns we fuzzy-search against.
-- These make similarity() and LIKE '%...%' fast at scale.
CREATE INDEX IF NOT EXISTS gauges_name_trgm_idx
    ON gauges USING GIN (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS gauges_external_id_trgm_idx
    ON gauges USING GIN (external_id gin_trgm_ops);
CREATE INDEX IF NOT EXISTS reaches_name_trgm_idx
    ON reaches USING GIN (name gin_trgm_ops);
