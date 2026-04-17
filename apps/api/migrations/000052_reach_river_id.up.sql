ALTER TABLE reaches ADD COLUMN river_id UUID REFERENCES rivers(id) ON DELETE SET NULL;
CREATE INDEX reaches_river_id_idx ON reaches (river_id) WHERE river_id IS NOT NULL;

-- Seed one river per distinct river_name from existing reach data.
-- Slug: lowercase, non-alphanumeric runs collapsed to hyphens, leading/trailing hyphens trimmed.
-- basin: take the most common basin_group for that river_name.
INSERT INTO rivers (slug, name, basin)
SELECT
    trim(both '-' from regexp_replace(lower(river_name), '[^a-z0-9]+', '-', 'g')) AS slug,
    river_name                                                                      AS name,
    MAX(basin_group)                                                                AS basin
FROM reaches
WHERE river_name IS NOT NULL AND river_name <> ''
GROUP BY river_name
ON CONFLICT (slug) DO NOTHING;

-- Link reaches to their newly-created rivers.
UPDATE reaches r
SET    river_id = rv.id
FROM   rivers rv
WHERE  r.river_name = rv.name
  AND  r.river_id IS NULL;
