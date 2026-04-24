DROP INDEX IF EXISTS gauges_comid_idx;
DROP INDEX IF EXISTS reaches_anchor_comid_idx;

ALTER TABLE gauges
    DROP COLUMN IF EXISTS comid;

ALTER TABLE reaches
    DROP COLUMN IF EXISTS totdasqkm,
    DROP COLUMN IF EXISTS reachcode,
    DROP COLUMN IF EXISTS take_out_comid,
    DROP COLUMN IF EXISTS put_in_comid,
    DROP COLUMN IF EXISTS anchor_comid;
