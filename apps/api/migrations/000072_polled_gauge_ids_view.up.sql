-- Union view that drives the poller's gauge selection.
-- Replaces the direct gauge_reach_associations EXISTS check in poller.go.
-- NOTE: reverting this migration requires also reverting the Go binary — the
-- poller references this view by name.
CREATE VIEW polled_gauge_ids AS
  SELECT DISTINCT gauge_id FROM gauge_reach_associations
  UNION
  SELECT DISTINCT gauge_id FROM custom_gauge_inputs
  UNION
  SELECT DISTINCT primary_gauge_id AS gauge_id
    FROM user_reaches
   WHERE primary_gauge_id IS NOT NULL;
