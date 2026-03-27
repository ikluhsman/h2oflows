-- Describes how a gauge relates to the reach it is bound to.
-- A gauge can be the direct measurement for a run, or an indirect indicator
-- that requires local knowledge to interpret (e.g. an upstream gauge where
-- the run is the canyon below, not the measurement site itself).
--
-- primary           — gauge is on or immediately adjacent to the run section.
--                    This is the authoritative reading for the reach.
-- upstream_indicator — gauge is upstream; flow will arrive at the run after
--                    some travel time. Ranges are calibrated for the offset
--                    but the UI should indicate this is indirect. "Gray area."
-- downstream_indicator — gauge is below the take-out; useful for big-picture
--                    trend but not a direct run measurement.
-- tributary         — measures a feeder stream that significantly affects the
--                    main run (e.g. a major side creek joining above the put-in).
ALTER TABLE gauges
    ADD COLUMN reach_relationship TEXT DEFAULT 'primary'
        CHECK (reach_relationship IN ('primary','upstream_indicator','downstream_indicator','tributary'));

-- Seed relationship for existing bound gauges — all are assumed primary until
-- a maintainer or community member indicates otherwise.
UPDATE gauges SET reach_relationship = 'primary' WHERE reach_id IS NOT NULL;

CREATE INDEX gauges_reach_relationship_idx ON gauges (reach_id, reach_relationship)
    WHERE reach_id IS NOT NULL;
