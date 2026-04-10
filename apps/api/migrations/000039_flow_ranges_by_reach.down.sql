-- Reverse: collapse per-reach rows back to per-gauge (keeps only the first reach's rows).
-- NOTE: rows added for extra reaches are lost; this is a lossy rollback.

ALTER TABLE flow_ranges DROP CONSTRAINT flow_ranges_reach_id_label_craft_key;
DROP INDEX flow_ranges_reach_id_idx;

-- Keep only the alphabetically-first reach row per (gauge_id, label, craft_type)
DELETE FROM flow_ranges fr
WHERE id NOT IN (
  SELECT DISTINCT ON (gauge_id, label, craft_type) id
  FROM flow_ranges
  ORDER BY gauge_id, label, craft_type, (SELECT r.slug FROM reaches r WHERE r.id = reach_id)
);

ALTER TABLE flow_ranges DROP COLUMN reach_id;

ALTER TABLE flow_ranges
  ADD CONSTRAINT flow_ranges_gauge_id_label_craft_key
  UNIQUE (gauge_id, label, craft_type);
