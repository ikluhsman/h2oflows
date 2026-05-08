-- Enforce NOT NULL after backfill in 000079 has run.
ALTER TABLE user_watchlists ALTER COLUMN dashboard_id SET NOT NULL;
