-- Add basin and state context to gauges for disambiguation and UI grouping.
-- state_abbr: two-letter state code (CO, UT, NM, WY, etc.)
-- basin_name: major drainage basin (Colorado River Basin, Arkansas River Basin, etc.)
-- watershed_name is already present from migration 018 — populated from HUC4 subregion.
ALTER TABLE gauges
    ADD COLUMN state_abbr TEXT,
    ADD COLUMN basin_name TEXT;

CREATE INDEX gauges_state_idx  ON gauges (state_abbr) WHERE state_abbr IS NOT NULL;
CREATE INDEX gauges_basin_idx  ON gauges (basin_name)  WHERE basin_name IS NOT NULL;
