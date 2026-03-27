-- Refine reach_access with entry style and approach distance,
-- then add ordered waypoints for trail and technical approaches.

-- entry_style describes how you get from the parking area to the water:
--   boat_ramp  — formal launch infrastructure (ramp, dock, courtesy dock)
--   bank       — rough scramble, no real trail, typically < 1/8 mile
--   trail      — established or use trail; walk ranges from a few minutes to miles
--   technical  — rope work, belay, significant scrambling, or multi-mile approach
--                with route-finding; requires specific beta to navigate safely
ALTER TABLE reach_access
    ADD COLUMN entry_style        TEXT CHECK (entry_style IN ('boat_ramp','bank','trail','technical')),
    ADD COLUMN approach_dist_mi   NUMERIC(5,2), -- distance from parking to water entry
    ADD COLUMN approach_notes     TEXT;         -- freeform narrative: "walk 2.1mi, belay boats down the cliff, jump in"

-- Ordered waypoints along an access approach.
-- Used for trail and technical put-ins where a single point isn't enough —
-- you need "park here → trail junction at the creek → top of the cliff → water entry."
-- Each waypoint can be contributed independently as people pin points from the map,
-- upload photos with GPS, or trace from an InReach track.
CREATE TABLE access_waypoints (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    access_id       UUID        NOT NULL REFERENCES reach_access(id) ON DELETE CASCADE,
    sequence        SMALLINT    NOT NULL,       -- order along the approach, 1-based
    location        GEOGRAPHY(POINT, 4326),     -- null if GPS not yet known
    label           TEXT        NOT NULL,       -- "Trailhead", "Creek crossing", "Top of cliff", "Water entry"
    description     TEXT,                       -- "Belay kayaks here, ~40ft. Scramble down river-left."

    -- Provenance — GPS can come from a map pin, photo EXIF, InReach track export, etc.
    gps_source      TEXT CHECK (gps_source IN ('map_pin','photo_exif','inreach','spot','manual','import')),
    data_source     TEXT        NOT NULL DEFAULT 'community'
                                CHECK (data_source IN ('ai_seed','community','import','maintainer')),
    ai_confidence   SMALLINT    CHECK (ai_confidence BETWEEN 0 AND 100),
    verified        BOOLEAN     NOT NULL DEFAULT FALSE,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (access_id, sequence)
);

CREATE INDEX access_waypoints_access_id_idx ON access_waypoints (access_id, sequence);
CREATE INDEX access_waypoints_location_idx  ON access_waypoints USING GIST (location)
    WHERE location IS NOT NULL;
