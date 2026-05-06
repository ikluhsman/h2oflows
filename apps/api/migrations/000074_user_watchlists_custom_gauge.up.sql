-- Step 1: add custom_gauge_id column
ALTER TABLE user_watchlists
  ADD COLUMN custom_gauge_id UUID REFERENCES custom_gauges(id) ON DELETE CASCADE;

-- Step 2: drop old unique constraint before making gauge_id nullable
ALTER TABLE user_watchlists
  DROP CONSTRAINT user_watchlists_user_gauge_reach_key;

-- Step 3: make gauge_id nullable (existing rows retain their non-null values)
ALTER TABLE user_watchlists
  ALTER COLUMN gauge_id DROP NOT NULL;

-- Step 4: enforce exactly one of gauge_id / custom_gauge_id is set
ALTER TABLE user_watchlists
  ADD CONSTRAINT user_watchlists_one_target_chk
  CHECK ((gauge_id IS NOT NULL) <> (custom_gauge_id IS NOT NULL));

-- Step 5: replace old unique constraint with two partial unique indexes
-- COALESCE handles NULL reach_slug (standalone gauge adds) consistently.
CREATE UNIQUE INDEX user_watchlists_user_real_gauge_uk
  ON user_watchlists (user_id, gauge_id, COALESCE(reach_slug, ''))
  WHERE gauge_id IS NOT NULL;

CREATE UNIQUE INDEX user_watchlists_user_custom_gauge_uk
  ON user_watchlists (user_id, custom_gauge_id, COALESCE(reach_slug, ''))
  WHERE custom_gauge_id IS NOT NULL;

-- Step 6: index for custom_gauge_id lookups
CREATE INDEX user_watchlists_custom_gauge_idx
  ON user_watchlists (custom_gauge_id)
  WHERE custom_gauge_id IS NOT NULL;
