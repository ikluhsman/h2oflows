DROP INDEX IF EXISTS user_reaches_custom_gauge_idx;
ALTER TABLE user_reaches DROP CONSTRAINT IF EXISTS user_reaches_one_gauge_chk;
ALTER TABLE user_reaches DROP COLUMN IF EXISTS custom_gauge_id;
