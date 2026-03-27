-- Unfeatured the seeded gauges rather than deleting them —
-- by the time this runs the poller may have enriched them with location/readings.
UPDATE gauges
SET    featured = FALSE,
       prominence_score = CASE source WHEN 'usgs' THEN 100 WHEN 'dwr' THEN 80 ELSE 0 END
WHERE  (source = 'usgs' AND external_id IN (
    '09361500','07087050','07091200','07094500','06710605','09050700','09057500',
    '06716500','06719505','09070500','09070000','09066510','09114520','09128000',
    '09152500','09085000','06701900','06710245','06700000','09050100','06730200',
    '09058000','09076300','09251000','09328960','09315000','08276500','09234500',
    '09163500','09180500','09380000','06713000','06713500','09151500','09342500',
    '09349800','06620000','09260050','09261000','09304800','08217500','09359020',
    '09166500'
))
OR (source = 'dwr' AND external_id IN (
    'PLAWATCO','PLAGRACO','PLASPLCO','PLAGEOCO','LNBC10CO',
    'SVCLYOCO','PLADENCO','LAKATLCO','CCACCRCO','RIOMILCO'
));
