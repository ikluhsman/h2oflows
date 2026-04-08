ALTER TABLE reaches DROP COLUMN IF EXISTS class_hardest;
ALTER TABLE rapids
    DROP COLUMN IF EXISTS is_permanent_hazard,
    DROP COLUMN IF EXISTS hazard_type;
DROP INDEX IF EXISTS rapids_is_permanent_hazard_idx;
