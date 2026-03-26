-- reaches are created before gauges so gauges can FK to them.
-- primary_gauge_id FK is added in 000003 after the gauges table exists.
CREATE TABLE reaches (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    slug             TEXT        UNIQUE NOT NULL,
    name             TEXT        NOT NULL,
    put_in           GEOGRAPHY(POINT, 4326),
    take_out         GEOGRAPHY(POINT, 4326),
    centerline       GEOGRAPHY(LINESTRING, 4326),
    class_min        NUMERIC(3,1),
    class_max        NUMERIC(3,1),
    class_at_low     NUMERIC(3,1),
    class_at_high    NUMERIC(3,1),
    character        TEXT,       -- creeking/pool-drop/continuous/big-water/flatwater
    length_mi        NUMERIC(6,2),
    region           TEXT,
    primary_gauge_id UUID,       -- FK constraint added in 000003_create_gauges
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX reaches_slug_idx       ON reaches (slug);
CREATE INDEX reaches_region_idx     ON reaches (region);
CREATE INDEX reaches_put_in_idx     ON reaches USING GIST (put_in);
CREATE INDEX reaches_take_out_idx   ON reaches USING GIST (take_out);
CREATE INDEX reaches_name_trgm_idx  ON reaches USING GIN  (name gin_trgm_ops);
