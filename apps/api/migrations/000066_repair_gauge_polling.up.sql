-- (a) Backfill gauge_reach_associations for reaches whose primary gauge
--     was set after migration 000065 ran.
INSERT INTO gauge_reach_associations (gauge_id, reach_id, relationship)
SELECT r.primary_gauge_id, r.id, 'primary'
FROM   reaches r
WHERE  r.primary_gauge_id IS NOT NULL
ON CONFLICT (gauge_id, reach_id) DO NOTHING;

-- (b) Strip source prefix from external_id stored verbatim from NLDI.
UPDATE gauges g
SET    external_id = substring(g.external_id from 6)
WHERE  g.source = 'usgs'
  AND  g.external_id LIKE 'USGS-%'
  AND  NOT EXISTS (
      SELECT 1 FROM gauges g2
      WHERE  g2.source = 'usgs'
        AND  g2.external_id = substring(g.external_id from 6)
  );

UPDATE gauges g
SET    external_id = substring(g.external_id from 5)
WHERE  g.source = 'dwr'
  AND  g.external_id LIKE 'DWR-%'
  AND  NOT EXISTS (
      SELECT 1 FROM gauges g2
      WHERE  g2.source = 'dwr'
        AND  g2.external_id = substring(g.external_id from 5)
  );

-- (c) Recover gauges auto-deactivated by the prefix bug.
UPDATE gauges g
SET    status               = 'active',
       consecutive_failures = 0
WHERE  g.auto_managed = TRUE
  AND  g.status        = 'inactive'
  AND  EXISTS (
      SELECT 1 FROM gauge_reach_associations gra WHERE gra.gauge_id = g.id
  );
