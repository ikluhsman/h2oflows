# Schema

### reaches
- id: uuid (pk)
- slug: text (unique)
- name: text (required)
- put_in: geography(point
- take_out: geography(point
- centerline: geography(linestring
- class_min: numeric(3
- class_max: numeric(3
- class_at_low: numeric(3
- class_at_high: numeric(3
- character: text
- region: text
- primary_gauge_id: uuid (fk)

### gauges
- id: uuid (pk)
- reach_id: uuid (fk)
- external_id: text (required, fk)
- source: text (required)
- location: geography(point
- param_code: text (required)

### flow_ranges
- id: uuid (pk)
- gauge_id: uuid (required, fk)
- label: text (required)
- min_cfs: numeric(10
- max_cfs: numeric(10
- class_modifier: numeric(3

### gauge_readings
- id: uuid (pk)
- gauge_id: uuid (required, fk)
- value: numeric(12
- unit: text (required)
- timestamp: timestamp(tz) (required)
- qual_code: text
- provisional: boolean (required)

### reach_conditions
- id: uuid (pk)
- reach_id: uuid (required, fk)
- source_type: text (required)
- summary: text (required)
- runnable: boolean
- reported_by: uuid
- expires_at: timestamp(tz) (required)

### hazards
- id: uuid (pk)
- reach_id: uuid (required, fk)
- location: geography(point
- hazard_type: text (required)
- description: text (required)
- cfs_at_report: numeric(10
- reported_by: uuid

### rapids
- id: uuid (pk)
- reach_id: uuid (required, fk)
- name: text (required)
- river_mile: numeric(5

### reach_access
- id: uuid (pk)
- reach_id: uuid (required, fk)
- access_type: text (required)
- name: text
- directions: text
- parking_spaces: integer
- permit_info: text

### access_waypoints
- id: uuid (pk)
- access_id: uuid (required, fk)
- sequence: smallint (required)
- photo: exif
- InReach: track export
- data_source: text (required)
- ai_confidence: smallint    check
- verified: boolean (required)

### trips
- id: uuid (pk)
- reach_id: uuid (fk)
- end_cfs: numeric(10
- ended_at: timestamp(tz)
- duration_min: smallint
- distance_mi: numeric(6

### trip_track_points
- id: uuid (pk)
- trip_id: uuid (required, fk)
- timestamp: timestamp(tz) (required)
- lat: numeric(10
- lng: numeric(10
- accuracy_m: numeric(7

### gauge_reach_associations
- id: uuid (pk)
- gauge_id: uuid (required, fk)
- reach_id: uuid (required, fk)
- relationship: text (required)
- tributary: ))

### reach_relationships
- from_reach_id: uuid (required, fk)
- to_reach_id: uuid (required, fk)
- relationship: text (required)

### reach_embeddings
- id: uuid (pk)
- reach_id: uuid (required, fk)
- access_id: uuid (fk)
- flow_ranges: ))

### user_watchlists
- id: uuid (pk)
- user_id: text (required, fk)
- gauge_id: uuid (required, fk)

### trip_reports
- id: uuid (pk)
- user_id: uuid (fk)
- device_id: text (fk)
- reach_id: uuid (required, fk)
- title: text
- body: text
- observed_at: timestamp(tz) (required)
- cfs_at_time: numeric(10
- photos: jsonb (default)
- public_slug: text (unique)
- share_consent_h2oflows: boolean (default)
- published: boolean (default)

### contributions
- id: uuid (pk)
- user_id: uuid (fk)
- device_id: text (fk)
- reach_id: uuid (required, fk)
- contribution_type: text (required)
- body: text
- observed_at: timestamp(tz) (required)
- cfs_at_time: numeric(10
- share_consent_h2oflows: boolean (default)

### proximity_events
- id: uuid (pk)
- device_id: text (required, fk)
- user_id: uuid (fk)
- reach_id: uuid (required, fk)
- event_type: text (required)
- location: geography(point
- detected_at: timestamp(tz) (required)
- promoted_to: uuid (fk)

### rivers
- id: uuid (pk)
- slug: text (unique)
- name: text (required)
- basin: text
- state_abbr: text

### user_roles
- id: uuid (pk)
- user_id: text (required, fk)
- role: text (required)
- river_id: uuid (fk)
