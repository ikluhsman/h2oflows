-- Parking/meet-up location separate from the water access point.
-- These are often different places:
--   - parking_location: where vehicles park, people carpool and gear up
--   - location (existing): where boats physically enter or exit the water
-- Both are useful on the map — navigation apps route to parking_location,
-- the put-in marker shows where to carry the boat.
ALTER TABLE reach_access
    ADD COLUMN parking_location  GEOGRAPHY(POINT, 4326),
    ADD COLUMN parking_notes     TEXT,           -- "unpaved lot, 20 spaces, can be muddy"
    ADD COLUMN hike_to_water_min INT;            -- approx walk from parking to water, in minutes

CREATE INDEX reach_access_parking_idx ON reach_access USING GIST (parking_location)
    WHERE parking_location IS NOT NULL;
