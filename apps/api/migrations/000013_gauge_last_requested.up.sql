-- Tracks when a gauge was last actively requested by a user (dashboard add,
-- detail page view, watchlist). The poller uses this to implement demand-driven
-- polling: only gauges that are featured OR recently requested get polled.
-- Non-featured gauges with no recent requests quietly drop out of the poll window.
ALTER TABLE gauges ADD COLUMN last_requested_at TIMESTAMPTZ;

CREATE INDEX gauges_last_requested_idx ON gauges (last_requested_at DESC NULLS LAST)
    WHERE featured = FALSE;
