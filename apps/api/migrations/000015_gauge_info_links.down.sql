ALTER TABLE gauges
    DROP CONSTRAINT IF EXISTS info_links_valid,
    DROP COLUMN IF EXISTS gauge_notes,
    DROP COLUMN IF EXISTS info_links;
