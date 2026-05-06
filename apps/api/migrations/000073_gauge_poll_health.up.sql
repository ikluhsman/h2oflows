-- Rename existing counter for clarity, add health-tracking columns.
ALTER TABLE gauges RENAME COLUMN consecutive_failures TO consecutive_poll_failures;

ALTER TABLE gauges
  ADD COLUMN last_poll_failure_at TIMESTAMPTZ,
  ADD COLUMN last_poll_success_at TIMESTAMPTZ,
  ADD COLUMN poll_health TEXT NOT NULL DEFAULT 'healthy'
    CHECK (poll_health IN ('healthy','degraded','stale','unreachable'));

-- Seed from last_reading_at so gauges don't show "never polled" on day 1.
UPDATE gauges SET last_poll_success_at = last_reading_at WHERE last_reading_at IS NOT NULL;

CREATE INDEX gauges_poll_health_idx ON gauges (poll_health) WHERE poll_health <> 'healthy';
