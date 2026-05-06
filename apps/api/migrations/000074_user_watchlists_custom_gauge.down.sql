-- WARNING: down migration will fail if any rows have custom_gauge_id set.
-- Delete or migrate those rows manually before running this down migration.
DROP INDEX IF EXISTS user_watchlists_custom_gauge_idx;
DROP INDEX IF EXISTS user_watchlists_user_custom_gauge_uk;
DROP INDEX IF EXISTS user_watchlists_user_real_gauge_uk;
ALTER TABLE user_watchlists DROP CONSTRAINT IF EXISTS user_watchlists_one_target_chk;
ALTER TABLE user_watchlists ALTER COLUMN gauge_id SET NOT NULL;
ALTER TABLE user_watchlists DROP COLUMN IF EXISTS custom_gauge_id;
ALTER TABLE user_watchlists ADD CONSTRAINT user_watchlists_user_gauge_reach_key
  UNIQUE (user_id, gauge_id, reach_slug);
