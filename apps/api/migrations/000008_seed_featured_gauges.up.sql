-- Seed curated, featured gauges from the blackbox exporter YAML files.
-- These are hand-picked whitewater gauges with elevated prominence scores.
-- Prominence = source tier (usgs=100, dwr=80) + curation bonus (200).
-- location is NULL here — the poller fills it in on first poll via DiscoverSites.
-- ON CONFLICT allows safe re-running (idempotent).

INSERT INTO gauges (external_id, source, name, status, featured, prominence_score)
VALUES
    -- USGS gauges (prominence = 300)
    ('09361500', 'usgs', 'Animas River at Durango, CO',                                          'active', TRUE, 300),
    ('07087050', 'usgs', 'Arkansas River Below Granite, CO',                                     'active', TRUE, 300),
    ('07091200', 'usgs', 'Arkansas River Near Nathrop, CO',                                      'active', TRUE, 300),
    ('07094500', 'usgs', 'Arkansas River at Parkdale, Co.',                                      'active', TRUE, 300),
    ('06710605', 'usgs', 'Bear Creek Above Bear Creek Lake Near Morrison, CO',                   'active', TRUE, 300),
    ('09050700', 'usgs', 'Blue River Below Dillon, Co.',                                         'active', TRUE, 300),
    ('09057500', 'usgs', 'Blue River Below Green Mountain Reservoir, CO',                        'active', TRUE, 300),
    ('06716500', 'usgs', 'Clear Creek Near Lawson, CO',                                          'active', TRUE, 300),
    ('06719505', 'usgs', 'Clear Creek at Golden, CO',                                            'active', TRUE, 300),
    ('09070500', 'usgs', 'Colorado River Near Dotsero, CO',                                      'active', TRUE, 300),
    ('09070000', 'usgs', 'Eagle River Below Gypsum, Co.',                                        'active', TRUE, 300),
    ('09066510', 'usgs', 'Gore Creek at Mouth Near Minturn, CO',                                 'active', TRUE, 300),
    ('09114520', 'usgs', 'Gunnison River at Gunnison Whitewater Park, CO',                       'active', TRUE, 300),
    ('09128000', 'usgs', 'Gunnison River Below Gunnison Tunnel, CO',                             'active', TRUE, 300),
    ('09152500', 'usgs', 'Gunnison River Near Grand Junction, Co.',                              'active', TRUE, 300),
    ('09085000', 'usgs', 'Roaring Fork River at Glenwood Springs, Co.',                          'active', TRUE, 300),
    ('06701900', 'usgs', 'South Platte River Blw Brush Crk Near Trumbull, CO',                  'active', TRUE, 300),
    ('06710245', 'usgs', 'South Platte River at Union Ave',                                      'active', TRUE, 300),
    ('06700000', 'usgs', 'South Platte River Above Cheesman Lake, Co.',                          'active', TRUE, 300),
    ('09050100', 'usgs', 'Tenmile Creek BL North Tenmile C, at Frisco, Co.',                     'active', TRUE, 300),
    ('06730200', 'usgs', 'Boulder Creek at North 75TH St. Near Boulder, CO',                     'active', TRUE, 300),
    ('09058000', 'usgs', 'Colorado River Near Kremmling, CO',                                    'active', TRUE, 300),
    ('09076300', 'usgs', 'Roaring Fork River Blw Maroon Creek NR Aspen, CO',                    'active', TRUE, 300),
    ('09251000', 'usgs', 'Yampa River Near Maybell, CO',                                         'active', TRUE, 300),
    ('09328960', 'usgs', 'Colorado River at Gypsum Canyon Near Hite, UT',                        'active', TRUE, 300),
    ('09315000', 'usgs', 'Green River at Green River, UT',                                       'active', TRUE, 300),
    ('08276500', 'usgs', 'Rio Grande Blw Taos Junction Bridge Near Taos, NM',                    'active', TRUE, 300),
    ('09234500', 'usgs', 'Green River Near Greendale, UT',                                       'active', TRUE, 300),
    ('09163500', 'usgs', 'Colorado River Near Colorado-utah State Line',                         'active', TRUE, 300),
    ('09180500', 'usgs', 'Colorado River Near Cisco, UT',                                        'active', TRUE, 300),
    ('09380000', 'usgs', 'Colorado River at Lees Ferry, AZ',                                     'active', TRUE, 300),
    ('06713000', 'usgs', 'CHERRY CREEK BELOW CHERRY CREEK LAKE, CO.',                           'active', TRUE, 300),
    ('06713500', 'usgs', 'Cherry Creek at Denver, Co.',                                          'active', TRUE, 300),
    ('09151500', 'usgs', 'Escalante Creek Near Delta, Co.',                                      'active', TRUE, 300),
    ('09342500', 'usgs', 'San Juan River at Pagosa Springs, CO',                                 'active', TRUE, 300),
    ('09349800', 'usgs', 'Piedra River Near Arboles, Co.',                                       'active', TRUE, 300),
    ('06620000', 'usgs', 'North Platte River Near Northgate, CO',                                'active', TRUE, 300),
    ('09260050', 'usgs', 'Yampa River at Deerlodge Park, CO',                                    'active', TRUE, 300),
    ('09261000', 'usgs', 'Green River Near Jensen, UT',                                          'active', TRUE, 300),
    ('09304800', 'usgs', 'White River Below Meeker, CO',                                         'active', TRUE, 300),
    ('08217500', 'usgs', 'Rio Grande at Wagon Wheel Gap',                                        'active', TRUE, 300),
    ('09359020', 'usgs', 'Animas River Below Silverton, CO',                                     'active', TRUE, 300),
    ('09166500', 'usgs', 'Dolores River at Dolores, Co.',                                        'active', TRUE, 300),

    -- DWR gauges — external_id is the DWR station ABBREV (prominence = 280)
    ('PLAWATCO', 'dwr', 'South Platte River at Waterton, Co.',                 'active', TRUE, 280),
    ('PLAGRACO', 'dwr', 'North Fork South Platte River at Grant',              'active', TRUE, 280),
    ('PLASPLCO', 'dwr', 'South Platte River at South Platte, CO',              'active', TRUE, 280),
    ('PLAGEOCO', 'dwr', 'South Platte River Near Lake George, Co.',            'active', TRUE, 280),
    ('LNBC10CO', 'dwr', 'LITTLETON BOAT CHUTE NO 10',                          'active', TRUE, 280),
    ('SVCLYOCO', 'dwr', 'SAINT VRAIN CREEK AT LYONS, CO',                     'active', TRUE, 280),
    ('PLADENCO', 'dwr', 'SOUTH PLATTE RIVER AT DENVER',                       'active', TRUE, 280),
    ('LAKATLCO', 'dwr', 'LAKE CREEK ABOVE TWIN LAKES RESERVOIR, CO.',         'active', TRUE, 280),
    ('CCACCRCO', 'dwr', 'CLEAR CREEK ABOVE CLEAR CREEK RESERVOIR, CO.',       'active', TRUE, 280),
    ('RIOMILCO', 'dwr', 'Rio Grande at Thirty Mile Bridge Near Creede',        'active', TRUE, 280)
ON CONFLICT (external_id, source) DO UPDATE
    SET featured         = TRUE,
        prominence_score = GREATEST(gauges.prominence_score, EXCLUDED.prominence_score);
