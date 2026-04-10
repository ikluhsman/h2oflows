-- Extend flow_ranges to support 5-tier system:
--   below_recommended, low_runnable, runnable, high_runnable, above_recommended
--
-- Also makes gauge_id nullable so KML-imported ranges (which are reach-level
-- descriptions, not tied to a specific gauge) can be inserted without a gauge.

-- 1. Extend label constraint to include new tiers.
ALTER TABLE flow_ranges DROP CONSTRAINT flow_ranges_label_check;
ALTER TABLE flow_ranges ADD CONSTRAINT flow_ranges_label_check
  CHECK (label = ANY (ARRAY[
    'below_recommended',
    'low_runnable',
    'runnable',
    'high_runnable',
    'above_recommended'
  ]));

-- 2. Make gauge_id nullable — KML-sourced ranges have no gauge reference.
ALTER TABLE flow_ranges ALTER COLUMN gauge_id DROP NOT NULL;
