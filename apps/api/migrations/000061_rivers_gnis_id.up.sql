-- gnis_id is the USGS GNIS feature ID for the named stream — globally unique
-- per named waterbody across the NHD. It disambiguates rivers that share a
-- common name (e.g. Clear Creek/South Platte vs Clear Creek/Arkansas) and lets
-- us collapse all NHDPlus flowline segments for one river into a single row
-- regardless of how many HUC8 sub-basins the river crosses.
--
-- Nullable to preserve existing manually-entered rivers without NHD backing.
ALTER TABLE rivers
    ADD COLUMN gnis_id TEXT UNIQUE;

CREATE INDEX rivers_gnis_id_idx ON rivers (gnis_id) WHERE gnis_id IS NOT NULL;
