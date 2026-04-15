-- Revert to 5-tier flow band system. LOSSY: the 'running' band was created by
-- merging low_runnable+runnable — on the way back, we rename it to 'runnable'
-- only (the low_runnable subdivision is not recoverable).

ALTER TABLE flow_ranges DROP CONSTRAINT flow_ranges_label_check;

UPDATE flow_ranges SET label = 'below_recommended' WHERE label = 'too_low';
UPDATE flow_ranges SET label = 'runnable'          WHERE label = 'running';
UPDATE flow_ranges SET label = 'high_runnable'     WHERE label = 'high';
UPDATE flow_ranges SET label = 'above_recommended' WHERE label = 'very_high';

ALTER TABLE flow_ranges ADD CONSTRAINT flow_ranges_label_check
  CHECK (label = ANY (ARRAY[
    'below_recommended',
    'low_runnable',
    'runnable',
    'high_runnable',
    'above_recommended'
  ]));
