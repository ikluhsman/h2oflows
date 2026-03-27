-- Access points for a reach: put-ins, take-outs, shuttle drops, camps.
-- A reach can have multiple access points of each type
-- (e.g., two take-out options, an intermediate egress).
CREATE TABLE reach_access (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    reach_id        UUID        NOT NULL REFERENCES reaches(id) ON DELETE CASCADE,

    access_type     TEXT        NOT NULL
                                CHECK (access_type IN ('put_in','take_out','shuttle_drop','intermediate','camp')),
    name            TEXT,                           -- e.g. "Hecla Junction", "Ruby Mountain"
    location        GEOGRAPHY(POINT, 4326),

    directions      TEXT,                           -- driving directions to this point
    road_type       TEXT        CHECK (road_type IN ('paved','gravel','dirt','high-clearance','4wd')),
    parking_spaces  INT,                            -- rough capacity; null = unknown

    -- Fees and permits
    parking_fee     NUMERIC(8,2),                   -- USD per day; 0 = free; null = unknown
    permit_required BOOLEAN     NOT NULL DEFAULT FALSE,
    permit_info     TEXT,                           -- brief description of permit requirements
    permit_url      TEXT,                           -- recreation.gov or agency link

    -- Seasonal road/access closures (e.g. forest service roads)
    seasonal_close_start    CHAR(5),                -- 'MM-DD'; null = no seasonal closure
    seasonal_close_end      CHAR(5),

    notes           TEXT,                           -- anything else (boat ramp condition, etc.)

    -- Data provenance
    data_source     TEXT        NOT NULL DEFAULT 'ai_seed'
                                CHECK (data_source IN ('ai_seed','community','import','maintainer')),
    ai_confidence   SMALLINT    CHECK (ai_confidence BETWEEN 0 AND 100),
    verified        BOOLEAN     NOT NULL DEFAULT FALSE,
    verified_at     TIMESTAMPTZ,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX reach_access_reach_id_idx  ON reach_access (reach_id);
CREATE INDEX reach_access_type_idx      ON reach_access (reach_id, access_type);
CREATE INDEX reach_access_location_idx  ON reach_access USING GIST (location);
