CREATE TABLE rivers (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    slug       TEXT        UNIQUE NOT NULL,
    name       TEXT        NOT NULL,
    basin      TEXT,
    state_abbr TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX rivers_slug_idx      ON rivers (slug);
CREATE INDEX rivers_name_trgm_idx ON rivers USING GIN (name gin_trgm_ops);
