-- Trip records created when a paddler taps "Run it" on a gauge.
-- A trip ties together: which gauge/reach was run, the flow at the time,
-- and the raw GPS track collected by the device.
--
-- auth: user_id is nullable for now — anonymous trips are accepted during beta.
-- The device_id is a random UUID generated on first app install, used to
-- associate anonymous trips without requiring a login.

CREATE TABLE trips (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),

    -- What was run
    gauge_id        UUID        REFERENCES gauges(id)  ON DELETE SET NULL,
    reach_id        UUID        REFERENCES reaches(id) ON DELETE SET NULL,

    -- Flow conditions at trip time
    start_cfs       NUMERIC(10,2),
    end_cfs         NUMERIC(10,2),

    -- Timing
    started_at      TIMESTAMPTZ NOT NULL,
    ended_at        TIMESTAMPTZ,
    duration_min    SMALLINT    GENERATED ALWAYS AS (
                        EXTRACT(EPOCH FROM (ended_at - started_at)) / 60
                    ) STORED,

    -- Derived geometry — simplified linestring for map rendering.
    -- Populated by the server after track_points are received.
    track           GEOGRAPHY(LINESTRING, 4326),
    distance_mi     NUMERIC(6,2),

    -- User-provided notes added after the trip
    notes           TEXT,

    -- Device that recorded the trip (anonymous identifier)
    device_id       TEXT,

    -- Data provenance
    -- 'device_gps' = recorded by the app on-device
    -- 'manual'     = entered by hand after the fact
    data_source     TEXT        NOT NULL DEFAULT 'device_gps'
                                CHECK (data_source IN ('device_gps','manual')),

    -- Consent: did the paddler opt in to sharing this track publicly?
    -- NULL = not yet asked, FALSE = private, TRUE = shared
    share_consent   BOOLEAN,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX trips_gauge_id_idx   ON trips (gauge_id);
CREATE INDEX trips_reach_id_idx   ON trips (reach_id);
CREATE INDEX trips_started_at_idx ON trips (started_at DESC);
CREATE INDEX trips_device_id_idx  ON trips (device_id);
CREATE INDEX trips_track_idx      ON trips USING GIST (track) WHERE track IS NOT NULL;

-- Raw GPS track points from the device.
-- Stored separately from the trip so the trip record stays lightweight.
-- Points are kept indefinitely — they're the raw material for AI track analysis
-- and future access point improvement.
CREATE TABLE trip_track_points (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id     UUID        NOT NULL REFERENCES trips(id) ON DELETE CASCADE,

    timestamp   TIMESTAMPTZ NOT NULL,
    lat         NUMERIC(10, 7) NOT NULL,
    lng         NUMERIC(10, 7) NOT NULL,
    accuracy_m  NUMERIC(7, 2),   -- horizontal accuracy in metres; null if unknown
    altitude_m  NUMERIC(8, 1),   -- metres above sea level; null if unavailable
    speed_mps   NUMERIC(6, 2),   -- metres per second; null if unavailable
    heading     NUMERIC(5, 1),   -- degrees from north; null if unavailable

    UNIQUE (trip_id, timestamp)
);

CREATE INDEX trip_track_points_trip_id_idx ON trip_track_points (trip_id, timestamp);
