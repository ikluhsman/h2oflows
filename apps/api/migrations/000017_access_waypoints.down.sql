DROP TABLE IF EXISTS access_waypoints;
ALTER TABLE reach_access
    DROP COLUMN IF EXISTS entry_style,
    DROP COLUMN IF EXISTS approach_dist_mi,
    DROP COLUMN IF EXISTS approach_notes;
