-- NHD/NLDI reference layer for reaches and gauges.
--
-- Context: USGS NLDI (Network-Linked Data Index) identifies every NHDPlus
-- flowline by a ComID and exposes upstream/downstream navigation, gauge
-- discovery, and basin geometry. Linking our reaches and gauges to ComIDs
-- enables:
--   - fetching authoritative centerline geometry from NHD
--   - discovering related gauges (upstream indicators, downstream confirms)
--   - drainage-area-weighted flow math for calculated gauges
--   - round-trip import/export with the federal NHD catalog
--
-- All fields are nullable — existing KML-imported reaches and community
-- gauges stay valid. We backfill opportunistically via a separate CLI tool.

ALTER TABLE reaches
    ADD COLUMN anchor_comid    TEXT,     -- NHD reach snapped from put-in; canonical NLDI identifier for this run
    ADD COLUMN put_in_comid    TEXT,     -- NHD reach containing the put-in point
    ADD COLUMN take_out_comid  TEXT,     -- NHD reach containing the take-out point
    ADD COLUMN reachcode       TEXT,     -- NHD reachcode at anchor (14-digit catalog identifier)
    ADD COLUMN totdasqkm       NUMERIC;  -- total drainage area at anchor (sq km)

CREATE INDEX reaches_anchor_comid_idx ON reaches (anchor_comid) WHERE anchor_comid IS NOT NULL;

ALTER TABLE gauges
    ADD COLUMN comid TEXT;               -- NHD reach the gauge sits on

CREATE INDEX gauges_comid_idx ON gauges (comid) WHERE comid IS NOT NULL;
