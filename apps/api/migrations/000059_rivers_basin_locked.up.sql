-- basin_locked = TRUE means a human has manually overridden the basin assignment.
-- The metadata sync (propagateRiverBasins) will not overwrite it.
ALTER TABLE rivers ADD COLUMN basin_locked BOOLEAN NOT NULL DEFAULT FALSE;
