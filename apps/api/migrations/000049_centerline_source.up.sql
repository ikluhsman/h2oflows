ALTER TABLE reaches
  ADD COLUMN IF NOT EXISTS centerline_source TEXT NOT NULL DEFAULT 'osm';
-- values: 'osm' (auto-fetched), 'kml' (imported from KMZ), 'manual' (admin-set)
