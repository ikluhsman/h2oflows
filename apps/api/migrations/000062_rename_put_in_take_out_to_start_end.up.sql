-- Rename reaches.put_in / take_out geometry columns and comid columns to
-- start_point / end_point / start_comid / end_comid.
--
-- These columns represent the NLDI-authoring anchor points (where the
-- centerline trim begins and ends), distinct from KML access-point pins
-- stored in reach_access.  The new names match the UI copy ("Reach Start"
-- / "Reach End") introduced in the unified reach editor.
--
-- reach_access.access_type enum values ('put_in' / 'take_out') and the
-- reaches.put_in_name / reaches.take_out_name display-name columns are
-- NOT touched here — those are KML/display concerns handled separately.

ALTER TABLE reaches
    RENAME COLUMN put_in       TO start_point;

ALTER TABLE reaches
    RENAME COLUMN take_out     TO end_point;

ALTER TABLE reaches
    RENAME COLUMN put_in_comid  TO start_comid;

ALTER TABLE reaches
    RENAME COLUMN take_out_comid TO end_comid;
