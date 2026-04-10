-- Revert to 3-tier labels and non-nullable gauge_id.
DELETE FROM flow_ranges WHERE label IN ('low_runnable', 'high_runnable');

ALTER TABLE flow_ranges DROP CONSTRAINT flow_ranges_label_check;
ALTER TABLE flow_ranges ADD CONSTRAINT flow_ranges_label_check
  CHECK (label = ANY (ARRAY['below_recommended', 'runnable', 'above_recommended']));

ALTER TABLE flow_ranges ALTER COLUMN gauge_id SET NOT NULL;
