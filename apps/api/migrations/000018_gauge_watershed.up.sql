-- Watershed context for gauge grouping in aggregate graphs.
-- huc8 is the USGS 8-digit Hydrologic Unit Code — precise enough for
-- grouping gauges on the same river system without being overly granular.
-- watershed_name is a human-readable label for the UI ("Arkansas River",
-- "South Platte", "Upper Colorado", etc.)
ALTER TABLE gauges
    ADD COLUMN huc8           TEXT,           -- e.g. '14020005' (USGS returns up to 12 digits)
    ADD COLUMN watershed_name TEXT;           -- e.g. 'Arkansas River'

ALTER TABLE reaches
    ADD COLUMN huc8           TEXT,
    ADD COLUMN watershed_name TEXT;

CREATE INDEX gauges_huc8_idx  ON gauges (huc8) WHERE huc8 IS NOT NULL;
CREATE INDEX reaches_huc8_idx ON reaches (huc8) WHERE huc8 IS NOT NULL;
