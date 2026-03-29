-- Surf wave flag on rapids
ALTER TABLE rapids ADD COLUMN IF NOT EXISTS is_surf_wave BOOLEAN NOT NULL DEFAULT false;

-- Reach-to-reach directional relationships (upstream/downstream/tributary)
CREATE TABLE IF NOT EXISTS reach_relationships (
  from_reach_id UUID NOT NULL REFERENCES reaches(id) ON DELETE CASCADE,
  to_reach_id   UUID NOT NULL REFERENCES reaches(id) ON DELETE CASCADE,
  relationship  TEXT NOT NULL,
  PRIMARY KEY (from_reach_id, to_reach_id),
  CONSTRAINT reach_relationships_relationship_check
    CHECK (relationship IN ('upstream', 'downstream', 'tributary', 'continuation'))
);

CREATE INDEX IF NOT EXISTS reach_relationships_from_idx ON reach_relationships(from_reach_id);
CREATE INDEX IF NOT EXISTS reach_relationships_to_idx   ON reach_relationships(to_reach_id);
