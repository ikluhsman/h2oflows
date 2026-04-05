ALTER TABLE reaches
  DROP COLUMN IF EXISTS river_name,
  DROP COLUMN IF EXISTS common_name,
  DROP COLUMN IF EXISTS put_in_name,
  DROP COLUMN IF EXISTS take_out_name;
