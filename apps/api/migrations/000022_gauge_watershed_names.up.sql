-- Populate watershed_name for featured gauges seeded in 000008.
-- Groups gauges by the major river system they belong to for UI grouping.

UPDATE gauges SET watershed_name = 'Arkansas River' WHERE external_id IN (
    '07087050',  -- Arkansas R. Below Granite (The Numbers)
    '07091200',  -- Arkansas R. Near Nathrop (Browns Canyon)
    '07094500'   -- Arkansas R. at Parkdale (Royal Gorge)
) AND source = 'usgs';

UPDATE gauges SET watershed_name = 'Upper Colorado' WHERE external_id IN (
    '09058000',  -- Colorado R. Near Kremmling (Gore Canyon)
    '09070500',  -- Colorado R. Near Dotsero
    '09070000',  -- Eagle R. Below Gypsum
    '09066510',  -- Gore Creek at Mouth Near Minturn
    '09050700',  -- Blue R. Below Dillon
    '09057500',  -- Blue R. Below Green Mountain Reservoir
    '09050100',  -- Tenmile Creek at Frisco
    '09076300',  -- Roaring Fork Blw Maroon Creek
    '09085000'   -- Roaring Fork at Glenwood Springs
) AND source = 'usgs';

UPDATE gauges SET watershed_name = 'Yampa River' WHERE external_id IN (
    '09251000',  -- Yampa R. Near Maybell (Cross Mountain Gorge)
    '09260050'   -- Yampa R. at Deerlodge Park (Yampa Canyon)
) AND source = 'usgs';

UPDATE gauges SET watershed_name = 'Gunnison River' WHERE external_id IN (
    '09128000',  -- Gunnison R. Below Gunnison Tunnel (Black Canyon)
    '09152500',  -- Gunnison R. Near Grand Junction (Gunnison Gorge)
    '09114520',  -- Gunnison R. at Gunnison Whitewater Park
    '09151500'   -- Escalante Creek Near Delta
) AND source = 'usgs';

UPDATE gauges SET watershed_name = 'Clear Creek' WHERE external_id IN (
    '06716500',  -- Clear Creek Near Lawson (Canyon)
    '06719505'   -- Clear Creek at Golden
) AND source = 'usgs';

UPDATE gauges SET watershed_name = 'Animas River' WHERE external_id IN (
    '09361500',  -- Animas R. at Durango
    '09359020'   -- Animas R. Below Silverton
) AND source = 'usgs';

UPDATE gauges SET watershed_name = 'Rio Grande' WHERE external_id IN (
    '08276500',  -- Rio Grande Blw Taos Junction Bridge (Taos Box)
    '08217500',  -- Rio Grande at Wagon Wheel Gap
    '09342500',  -- San Juan R. at Pagosa Springs
    '09349800'   -- Piedra R. Near Arboles
) AND source = 'usgs';

UPDATE gauges SET watershed_name = 'North Platte' WHERE external_id IN (
    '06620000'   -- North Platte R. Near Northgate (Six Mile Gap)
) AND source = 'usgs';

UPDATE gauges SET watershed_name = 'South Platte' WHERE external_id IN (
    '06700000',  -- South Platte Abv Cheesman Lake (Cheesman Canyon)
    '06701900',  -- South Platte Blw Brush Creek
    '06710245',  -- South Platte at Union Ave
    '06710605',  -- Bear Creek Above Bear Creek Lake
    '06713000',  -- Cherry Creek Below Cherry Creek Lake
    '06713500',  -- Cherry Creek at Denver
    '06730200',  -- Boulder Creek at North 75th
    'PLAWATCO',  -- South Platte at Waterton
    'PLAGRACO',  -- North Fork South Platte at Grant
    'PLASPLCO',  -- South Platte at South Platte
    'PLAGEOCO',  -- South Platte Near Lake George
    'LNBC10CO',  -- Littleton Boat Chute No 10
    'PLADENCO'   -- South Platte at Denver
) AND source IN ('usgs', 'dwr');

UPDATE gauges SET watershed_name = 'St. Vrain / Front Range' WHERE external_id IN (
    'SVCLYOCO'   -- St. Vrain Creek at Lyons
) AND source = 'dwr';

UPDATE gauges SET watershed_name = 'Upper Colorado' WHERE external_id IN (
    'LAKATLCO',  -- Lake Creek Above Twin Lakes Reservoir
    'CCACCRCO'   -- Clear Creek Above Clear Creek Reservoir
) AND source = 'dwr';

UPDATE gauges SET watershed_name = 'Green River' WHERE external_id IN (
    '09234500',  -- Green R. Near Greendale, UT
    '09261000',  -- Green R. Near Jensen, UT
    '09315000'   -- Green R. at Green River, UT
) AND source = 'usgs';

UPDATE gauges SET watershed_name = 'Lower Colorado' WHERE external_id IN (
    '09163500',  -- Colorado R. Near CO-UT State Line
    '09180500',  -- Colorado R. Near Cisco, UT
    '09328960',  -- Colorado R. at Gypsum Canyon (Cataract)
    '09380000'   -- Colorado R. at Lees Ferry, AZ
) AND source = 'usgs';

UPDATE gauges SET watershed_name = 'Dolores / San Juan' WHERE external_id IN (
    '09166500',  -- Dolores R. at Dolores
    '09304800',  -- White R. Below Meeker
    'RIOMILCO'   -- Rio Grande at Thirty Mile Bridge
) AND source IN ('usgs', 'dwr');
