-- Rivers with the same name but different basins need distinct slugs.
-- Update existing river slugs to incorporate the basin suffix when set.
-- e.g. "Clear Creek" with basin "South Platte" → "clear-creek-south-platte"
UPDATE rivers
SET slug =
    trim(both '-' from regexp_replace(lower(name), '[^a-z0-9]+', '-', 'g'))
    || '-' ||
    trim(both '-' from regexp_replace(lower(basin), '[^a-z0-9]+', '-', 'g'))
WHERE basin IS NOT NULL AND basin <> '';

-- Prevent future collisions: two rivers with the same name must have different basins.
-- Rivers with no basin still dedupe by slug (which encodes only the name).
CREATE UNIQUE INDEX rivers_name_basin_uniq
    ON rivers (lower(name), COALESCE(lower(basin), ''));
