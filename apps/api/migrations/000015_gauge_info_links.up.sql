-- Interpretation notes and external links for indirect/gray-area gauges.
-- gauge_notes: freeform text explaining how to read this gauge for its reach.
--   e.g. "Gauge is 8mi upstream — add 2-3hrs travel time. Readings below 200cfs
--         typically mean the lower canyon is unrunnable regardless of this number."
-- info_links: labeled URLs to relevant resources — Discord channels, AW pages,
--   local club beta pages, outfitter condition lines, etc.
--   Schema: [{"label": "string", "url": "string"}, ...]
ALTER TABLE gauges
    ADD COLUMN gauge_notes TEXT,
    ADD COLUMN info_links  JSONB NOT NULL DEFAULT '[]'::jsonb;

-- Validate info_links entries have at least a url field.
ALTER TABLE gauges
    ADD CONSTRAINT info_links_valid CHECK (
        jsonb_typeof(info_links) = 'array'
    );
