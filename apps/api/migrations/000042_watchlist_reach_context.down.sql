ALTER TABLE user_watchlists DROP CONSTRAINT user_watchlists_user_gauge_reach_key;
ALTER TABLE user_watchlists ADD CONSTRAINT user_watchlists_user_id_gauge_id_key
  UNIQUE (user_id, gauge_id);
ALTER TABLE user_watchlists DROP COLUMN reach_slug;
