ALTER TABLE flow_ranges DROP CONSTRAINT flow_ranges_label_check;
ALTER TABLE flow_ranges ADD CONSTRAINT flow_ranges_label_check
  CHECK (label = ANY (ARRAY['too_low', 'minimum', 'fun', 'optimal', 'pushy', 'high', 'flood']));
