-- Reverse 000081_fix_dwr_gauge_associations.

-- 3'. Restore polled_gauge_ids to 3-way union (no watchlists).
CREATE OR REPLACE VIEW polled_gauge_ids AS
  SELECT DISTINCT gauge_id FROM gauge_reach_associations
  UNION
  SELECT DISTINCT gauge_id FROM custom_gauge_inputs
  UNION
  SELECT DISTINCT primary_gauge_id AS gauge_id
    FROM user_reaches
   WHERE primary_gauge_id IS NOT NULL;

-- 2'. Remove the exact (gauge, reach) pairs this migration inserted.
DELETE FROM gauge_reach_associations gra
USING  gauges g, reaches r
WHERE  gra.gauge_id = g.id
  AND  gra.reach_id = r.id
  AND  g.source     = 'dwr'
  AND  (
       (g.external_id = 'PLAGEOCO' AND r.slug IN (
           'south-platte-river-elevenmile-canyon',
           'south-platte-river-wildcat-canyon',
           'south-platte-river-deckers'))
    OR (g.external_id = 'PLACHACO' AND r.slug =
           'south-platte-river-chatfield-gravel-ponds')
    OR (g.external_id = 'ARKPUECO' AND r.slug IN (
           'arkansas-the-numbers',
           'arkansas-fractions',
           'arkansas-river-milk-run',
           'arkansas-pine-creek',
           'arkansas-browns-canyon',
           'arkansas-salida-to-rincon',
           'arkansas-river-salida-town-run',
           'arkansas-river-rincon-to-pinnacle-rock',
           'arkansas-river-parkdale',
           'arkansas-river-royal-gorge'))
  );

-- 1'. Return reactivated gauges to auto-deactivated state.
UPDATE gauges
SET    status                    = 'inactive',
       consecutive_poll_failures = 5,
       poll_health               = 'unreachable'
WHERE  source       = 'dwr'
  AND  external_id  IN ('PLAGEOCO', 'RIOMILCO', 'PLADENCO')
  AND  auto_managed = TRUE;
