-- 000067 is a data-repair migration. The re-pointing of associations and
-- primary_gauge_id pointers cannot be safely inverted without risk of
-- data loss. Down is intentionally a no-op.
SELECT 1;
