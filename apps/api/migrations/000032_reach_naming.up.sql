-- Add structured reach naming fields following AW's place-to-place convention.
-- Display format: "{river_name} · {put_in_name} to {take_out_name} ({common_name})"
--
-- river_name   — tributary/river name, e.g. "South Platte, North Fork"
-- common_name  — local nickname, e.g. "Foxton" (backfilled from existing name)
-- put_in_name  — short geographic anchor, e.g. "Buffalo Creek"
-- take_out_name — short geographic anchor, e.g. "South Platte"

ALTER TABLE reaches
  ADD COLUMN river_name    TEXT,
  ADD COLUMN common_name   TEXT,
  ADD COLUMN put_in_name   TEXT,
  ADD COLUMN take_out_name TEXT;

-- Backfill common_name from existing name so nothing breaks.
UPDATE reaches SET common_name = name;
