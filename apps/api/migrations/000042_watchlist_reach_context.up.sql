-- Allow a gauge to appear multiple times on the same user's watchlist,
-- once per reach context. reach_slug = NULL means a standalone gauge add
-- with no specific reach selected.

ALTER TABLE user_watchlists ADD COLUMN reach_slug TEXT;

-- Replace the old single-gauge-per-user constraint with one that
-- allows the same gauge for different reach contexts.
ALTER TABLE user_watchlists DROP CONSTRAINT user_watchlists_user_id_gauge_id_key;
ALTER TABLE user_watchlists ADD CONSTRAINT user_watchlists_user_gauge_reach_key
  UNIQUE (user_id, gauge_id, reach_slug);
