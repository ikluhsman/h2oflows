-- state_abbr: US state the reach's put-in (start_point) falls in.
-- Auto-populated by the API via TIGERweb Census lookup on reach save.
ALTER TABLE reaches ADD COLUMN state_abbr VARCHAR(2);

-- river_order: admin-set integer for upstream→downstream sort within a river.
-- NULL means unordered; sorted NULLS LAST in all queries.
ALTER TABLE reaches ADD COLUMN river_order SMALLINT;

CREATE INDEX reaches_state_abbr_idx ON reaches (state_abbr) WHERE state_abbr IS NOT NULL;
