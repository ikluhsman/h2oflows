-- Repair: prefixed gauge rows (USGS-XXXXXXXX) are dead duplicates of the
-- bare gauge rows (XXXXXXXX) that are already polling correctly.
-- Migration 000066 stripped prefixes where no bare duplicate existed, but
-- the conflict guard correctly skipped 9 cases where a bare row already
-- existed. Those prefixed rows have reach associations and primary_gauge_id
-- pointers that must be re-pointed to the bare counterparts so the frontend
-- queries the row that actually has readings.

-- Step 1: Move gauge_reach_associations from prefixed to bare.
-- Only where bare doesn't already have an association to the same reach
-- (avoids unique constraint violation on (gauge_id, reach_id)).
UPDATE gauge_reach_associations gra
SET    gauge_id = b.id
FROM   gauges p
JOIN   gauges b
       ON  b.source      = p.source
       AND b.external_id = substring(p.external_id FROM 6)
WHERE  p.source      = 'usgs'
  AND  p.external_id ILIKE 'USGS-%'
  AND  gra.gauge_id  = p.id
  AND  NOT EXISTS (
      SELECT 1 FROM gauge_reach_associations ex
      WHERE  ex.gauge_id = b.id AND ex.reach_id = gra.reach_id
  );

-- Step 2: Delete any remaining prefixed associations (cases where bare
-- already had the association — e.g. 09342500 which got one via migration
-- 000065 or an earlier seed path).
DELETE FROM gauge_reach_associations gra
USING  gauges p
JOIN   gauges b
       ON  b.source      = p.source
       AND b.external_id = substring(p.external_id FROM 6)
WHERE  p.source     = 'usgs'
  AND  p.external_id ILIKE 'USGS-%'
  AND  gra.gauge_id = p.id;

-- Step 3: Re-point reaches.primary_gauge_id from prefixed to bare.
UPDATE reaches r
SET    primary_gauge_id = b.id
FROM   gauges p
JOIN   gauges b
       ON  b.source      = p.source
       AND b.external_id = substring(p.external_id FROM 6)
WHERE  p.source         = 'usgs'
  AND  p.external_id    ILIKE 'USGS-%'
  AND  r.primary_gauge_id = p.id;

-- Step 4: Retire prefixed duplicates so they are never polled or surfaced.
UPDATE gauges
SET    status       = 'retired',
       auto_managed = FALSE
WHERE  source      = 'usgs'
  AND  external_id ILIKE 'USGS-%';

-- Step 5: Reactivate bare counterparts that went inactive while the
-- prefixed row was holding the associations (poller had nothing to do
-- for the bare row; it may have been deactivated by an earlier bug cycle).
UPDATE gauges b
SET    status               = 'active',
       consecutive_failures = 0
FROM   gauges p
WHERE  p.source      = 'usgs'
  AND  p.external_id ILIKE 'USGS-%'
  AND  b.source      = p.source
  AND  b.external_id = substring(p.external_id FROM 6)
  AND  b.auto_managed = TRUE
  AND  b.status       = 'inactive';
