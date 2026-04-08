-- class_hardest: the hardest single rapid on the reach, which paddlers may portage.
-- Supports the III-IV(V) AW notation where class_min/class_max represent the
-- "standard" range most paddlers encounter and class_hardest represents the
-- hardest feature that can be scouted and portaged.
-- When class_hardest IS NOT NULL AND class_hardest > class_max, the UI renders
-- the rating as e.g. "III-IV (V)" rather than just "III-IV".
ALTER TABLE reaches ADD COLUMN IF NOT EXISTS class_hardest NUMERIC(3,1);

-- is_permanent_hazard: marks a rapid row as a fixed infrastructure hazard rather
-- than a natural whitewater feature. Low-head dams, rebar, concrete debris,
-- bridge pilings, and similar man-made hazards are entered here with hazard_type
-- to distinguish them from standard rapids in the river features UI.
ALTER TABLE rapids
    ADD COLUMN IF NOT EXISTS is_permanent_hazard BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS hazard_type          TEXT;

-- Index makes filtering hazards from rapids fast on reach pages.
CREATE INDEX IF NOT EXISTS rapids_is_permanent_hazard_idx
    ON rapids (reach_id) WHERE is_permanent_hazard = TRUE;
