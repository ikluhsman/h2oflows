-- Collapse the standalone "featured" trust flag onto the reach association.
--
-- Background: until now, gauges had a manual `featured` boolean used to mark
-- them as the curated, always-polled backbone of the app. In practice, the
-- only gauges that should be polled every cycle are gauges that belong to a
-- reach — that's what makes them load-bearing. The 34 unlinked gauges that
-- carried `featured = TRUE` from earlier seeding rounds were wasting poll
-- cycles without contributing to any reach page.
--
-- This migration normalises the column so it always equals
-- (reach_id IS NOT NULL). Application code switches to querying reach_id
-- directly going forward; the column itself is kept for backwards-compat
-- with handlers that surface "trusted" badges, but it can be dropped in a
-- follow-up once those are migrated.
UPDATE gauges
SET    featured = (reach_id IS NOT NULL);
