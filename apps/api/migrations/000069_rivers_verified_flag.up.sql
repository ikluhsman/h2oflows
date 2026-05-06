ALTER TABLE rivers ADD COLUMN verified BOOLEAN NOT NULL DEFAULT FALSE;
-- All existing curated rivers are considered verified.
UPDATE rivers SET verified = TRUE;
