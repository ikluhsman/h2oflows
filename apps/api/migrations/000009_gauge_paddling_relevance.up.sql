-- Add paddling_relevance score computed nightly by the AI batch scorer.
-- 0 = unknown/not evaluated, 1-100 = relevance to recreational boating.
-- Null means the gauge has not been scored yet.
ALTER TABLE gauges ADD COLUMN paddling_relevance SMALLINT CHECK (paddling_relevance BETWEEN 0 AND 100);

-- Seed seeded featured gauges at 100 — they were hand-curated for paddling.
UPDATE gauges SET paddling_relevance = 100 WHERE featured = TRUE;

CREATE INDEX gauges_paddling_relevance_idx ON gauges (paddling_relevance DESC NULLS LAST);
