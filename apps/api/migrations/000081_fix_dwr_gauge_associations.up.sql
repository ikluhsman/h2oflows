-- Fix DWR gauge polling gaps:
--   1. Restore three gauges auto-deactivated after consecutive failures
--      while orphaned (no reach associations → not in polled_gauge_ids).
--   2. Add gauge_reach_associations for six orphaned DWR gauges.
--   3. Extend polled_gauge_ids to include user_watchlists.gauge_id so
--      user-watched gauges are polled even without a reach association.
--
-- All blocks are idempotent: re-running is a no-op.

-- ---------------------------------------------------------------------------
-- 1. Reactivate gauges that went inactive while orphaned.
-- ---------------------------------------------------------------------------
UPDATE gauges
SET    status                    = 'active',
       consecutive_poll_failures = 0,
       poll_health               = 'healthy'
WHERE  source       = 'dwr'
  AND  external_id  IN ('PLAGEOCO', 'RIOMILCO', 'PLADENCO')
  AND  status       = 'inactive'
  AND  auto_managed = TRUE;

-- ---------------------------------------------------------------------------
-- 2. Associate orphaned gauges with their reaches.
--
-- PLAGEOCO — South Platte at Lake George (bottom of Elevenmile Canyon)
-- PLACHACO — South Platte below Chatfield Reservoir
-- ARKPUECO — Arkansas at Pueblo (downstream of all classic canyon runs;
--            Pueblo Reservoir between gauge and runs → downstream_indicator)
-- SVCLYOCO, RIOMILCO, PLADENCO — no matching reaches yet; kept active
--            and will be polled if a user watchlists them (see fix 3).
-- ---------------------------------------------------------------------------

INSERT INTO gauge_reach_associations (gauge_id, reach_id, relationship)
SELECT g.id, r.id, v.rel
FROM   gauges g
CROSS  JOIN LATERAL (VALUES
    ('south-platte-river-elevenmile-canyon', 'primary'),
    ('south-platte-river-wildcat-canyon',    'upstream_indicator'),
    ('south-platte-river-deckers',           'upstream_indicator')
) AS v(slug, rel)
JOIN   reaches r ON r.slug = v.slug
WHERE  g.source = 'dwr' AND g.external_id = 'PLAGEOCO'
ON CONFLICT (gauge_id, reach_id) DO NOTHING;

INSERT INTO gauge_reach_associations (gauge_id, reach_id, relationship)
SELECT g.id, r.id, 'primary'
FROM   gauges g
JOIN   reaches r ON r.slug = 'south-platte-river-chatfield-gravel-ponds'
WHERE  g.source = 'dwr' AND g.external_id = 'PLACHACO'
ON CONFLICT (gauge_id, reach_id) DO NOTHING;

INSERT INTO gauge_reach_associations (gauge_id, reach_id, relationship)
SELECT g.id, r.id, 'downstream_indicator'
FROM   gauges g
JOIN   reaches r ON r.slug IN (
    'arkansas-the-numbers',
    'arkansas-fractions',
    'arkansas-river-milk-run',
    'arkansas-pine-creek',
    'arkansas-browns-canyon',
    'arkansas-salida-to-rincon',
    'arkansas-river-salida-town-run',
    'arkansas-river-rincon-to-pinnacle-rock',
    'arkansas-river-parkdale',
    'arkansas-river-royal-gorge'
)
WHERE  g.source = 'dwr' AND g.external_id = 'ARKPUECO'
ON CONFLICT (gauge_id, reach_id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- 3. Extend polled_gauge_ids to include directly-watchlisted gauges.
--    gauge_id is nullable in user_watchlists (row may be for a custom gauge).
-- ---------------------------------------------------------------------------
CREATE OR REPLACE VIEW polled_gauge_ids AS
  SELECT DISTINCT gauge_id FROM gauge_reach_associations
  UNION
  SELECT DISTINCT gauge_id FROM custom_gauge_inputs
  UNION
  SELECT DISTINCT primary_gauge_id AS gauge_id
    FROM user_reaches
   WHERE primary_gauge_id IS NOT NULL
  UNION
  SELECT DISTINCT gauge_id
    FROM user_watchlists
   WHERE gauge_id IS NOT NULL;
