-- Add unique constraints to prevent duplicate AI-seeded rapids and access points
-- on re-runs. Previously ON CONFLICT DO NOTHING had no constraint to match on
-- (only fires on PK collisions), so each seeder pass appended duplicates.

-- Remove duplicates before adding constraint: keep the row with the lowest id
-- (first inserted) for each (reach_id, name) pair in rapids.
DELETE FROM rapids
WHERE id NOT IN (
    SELECT DISTINCT ON (reach_id, name) id
    FROM rapids
    ORDER BY reach_id, name, id
);

ALTER TABLE rapids
    ADD CONSTRAINT rapids_reach_name_unique UNIQUE (reach_id, name);

-- For reach_access: unique on (reach_id, access_type, name).
-- name can be NULL so use COALESCE to treat NULLs as empty string in the dedup.
DELETE FROM reach_access
WHERE id NOT IN (
    SELECT DISTINCT ON (reach_id, access_type, COALESCE(name, '')) id
    FROM reach_access
    ORDER BY reach_id, access_type, COALESCE(name, ''), id
);

ALTER TABLE reach_access
    ADD CONSTRAINT reach_access_reach_type_name_unique UNIQUE (reach_id, access_type, name);
