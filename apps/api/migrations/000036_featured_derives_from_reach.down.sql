-- No-op. The previous state of the `featured` column cannot be reconstructed
-- from the current row state — manual `featured = TRUE` flags on unlinked
-- gauges were not preserved before the up migration ran. Restoring the
-- previous values would require a backup. The column itself is unchanged.
SELECT 1;
