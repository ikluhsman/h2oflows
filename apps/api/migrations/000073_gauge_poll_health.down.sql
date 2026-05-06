DROP INDEX IF EXISTS gauges_poll_health_idx;
ALTER TABLE gauges
  DROP COLUMN IF EXISTS poll_health,
  DROP COLUMN IF EXISTS last_poll_success_at,
  DROP COLUMN IF EXISTS last_poll_failure_at;
ALTER TABLE gauges RENAME COLUMN consecutive_poll_failures TO consecutive_failures;
