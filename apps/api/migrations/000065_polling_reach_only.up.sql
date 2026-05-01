-- Backfill gauge_reach_associations from reaches.primary_gauge_id so that
-- every reach's primary gauge is included in the poll set under the new rule.
-- ON CONFLICT DO NOTHING is safe to run multiple times.
INSERT INTO gauge_reach_associations (gauge_id, reach_id, relationship)
SELECT r.primary_gauge_id, r.id, 'primary'
FROM   reaches r
WHERE  r.primary_gauge_id IS NOT NULL
ON CONFLICT (gauge_id, reach_id) DO NOTHING;

-- Drop demand-poll bookkeeping column — polling is now purely reach-driven.
DROP INDEX IF EXISTS gauges_last_requested_idx;
ALTER TABLE gauges DROP COLUMN IF EXISTS last_requested_at;
