ALTER TABLE gauges ADD COLUMN IF NOT EXISTS last_requested_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS gauges_last_requested_idx ON gauges (last_requested_at DESC NULLS LAST);
