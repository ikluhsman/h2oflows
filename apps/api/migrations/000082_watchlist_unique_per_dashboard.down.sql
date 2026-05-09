-- Revert to global uniqueness (one entry per gauge+reach across all dashboards).

DROP INDEX IF EXISTS user_watchlists_user_real_gauge_uk;
DROP INDEX IF EXISTS user_watchlists_user_custom_gauge_uk;

CREATE UNIQUE INDEX user_watchlists_user_real_gauge_uk
  ON user_watchlists (user_id, gauge_id, COALESCE(reach_slug, ''))
  WHERE gauge_id IS NOT NULL;

CREATE UNIQUE INDEX user_watchlists_user_custom_gauge_uk
  ON user_watchlists (user_id, custom_gauge_id, COALESCE(reach_slug, ''))
  WHERE custom_gauge_id IS NOT NULL;
