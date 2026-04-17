ALTER TABLE gauges ADD COLUMN elevation_ft NUMERIC(8, 1);
COMMENT ON COLUMN gauges.elevation_ft IS 'Altitude of gauge datum in feet (USGS alt_va). Used for upstream-to-downstream sorting of reaches.';
