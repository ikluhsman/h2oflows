-- Allow the same gauge+reach on multiple dashboards by including dashboard_id
-- in the unique constraint. Without this, adding a gauge already on the primary
-- dashboard to a secondary dashboard silently fails (ON CONFLICT DO NOTHING),
-- causing the gauge to disappear on the secondary dashboard after page refresh.

DROP INDEX IF EXISTS user_watchlists_user_real_gauge_uk;
DROP INDEX IF EXISTS user_watchlists_user_custom_gauge_uk;

CREATE UNIQUE INDEX user_watchlists_user_real_gauge_uk
  ON user_watchlists (user_id, gauge_id, COALESCE(reach_slug, ''), dashboard_id)
  WHERE gauge_id IS NOT NULL;

CREATE UNIQUE INDEX user_watchlists_user_custom_gauge_uk
  ON user_watchlists (user_id, custom_gauge_id, COALESCE(reach_slug, ''), dashboard_id)
  WHERE custom_gauge_id IS NOT NULL;
