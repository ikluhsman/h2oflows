ALTER TABLE user_reaches
  ADD COLUMN custom_gauge_id UUID REFERENCES custom_gauges(id) ON DELETE SET NULL;

-- Allow zero or one gauge — never both at once.
ALTER TABLE user_reaches
  ADD CONSTRAINT user_reaches_one_gauge_chk
  CHECK (NOT (primary_gauge_id IS NOT NULL AND custom_gauge_id IS NOT NULL));

CREATE INDEX user_reaches_custom_gauge_idx
  ON user_reaches (custom_gauge_id)
  WHERE custom_gauge_id IS NOT NULL;
