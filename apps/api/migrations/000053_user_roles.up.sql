-- Application-level role assignments.
-- site_admin: full access including role management (backed by Supabase app_metadata too)
-- data_admin: can edit reach/river data; optionally scoped to a specific river
CREATE TABLE user_roles (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    TEXT        NOT NULL,
    role       TEXT        NOT NULL CHECK (role IN ('site_admin', 'data_admin')),
    river_id   UUID        REFERENCES rivers(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Partial unique indexes handle NULL river_id correctly (NULL ≠ NULL in UNIQUE).
-- Global role: one (user, role) pair when river_id is NULL.
CREATE UNIQUE INDEX user_roles_global_uniq ON user_roles (user_id, role) WHERE river_id IS NULL;
-- Scoped role: one (user, role, river) pair when river_id is set.
CREATE UNIQUE INDEX user_roles_scoped_uniq ON user_roles (user_id, role, river_id) WHERE river_id IS NOT NULL;

CREATE INDEX user_roles_user_id_idx  ON user_roles (user_id);
CREATE INDEX user_roles_river_id_idx ON user_roles (river_id) WHERE river_id IS NOT NULL;
