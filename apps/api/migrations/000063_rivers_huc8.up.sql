-- huc8 stores the 8-digit USGS Hydrologic Unit Code for the watershed the river
-- primarily flows through. Derived from NHD reachcode (first 8 chars) via the
-- GNIS ID lookup pipeline. Used to group rivers by canonical watershed in the
-- admin UI and to resolve duplicate basin names (e.g. "South Platte" vs
-- "South Platte River" → both map to HUC8 1019xxxx).
ALTER TABLE rivers
    ADD COLUMN huc8 VARCHAR(8);

CREATE INDEX rivers_huc8_idx ON rivers (huc8) WHERE huc8 IS NOT NULL;
