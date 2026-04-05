package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/h2oflow/h2oflow/apps/api/internal/ai"
)

// upsertReach inserts or updates the reach stub.
// Returns the reach UUID.
func upsertReach(ctx context.Context, pool *pgxpool.Pool, rd reachDef) (string, error) {
	var reachID string
	err := pool.QueryRow(ctx, `
		INSERT INTO reaches (slug, name, region, class_min, class_max, character, length_mi, aw_reach_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (slug) DO UPDATE SET
			name       = EXCLUDED.name,
			region     = EXCLUDED.region,
			class_min  = EXCLUDED.class_min,
			class_max  = EXCLUDED.class_max,
			character  = EXCLUDED.character,
			length_mi  = EXCLUDED.length_mi,
			aw_reach_id = COALESCE(EXCLUDED.aw_reach_id, reaches.aw_reach_id)
		RETURNING id
	`,
		rd.Slug, rd.Name, rd.Region,
		rd.ClassMin, rd.ClassMax,
		nullStr(rd.Character), nullFloat(rd.LengthMi),
		nullStr(rd.AWReachID),
	).Scan(&reachID)
	return reachID, err
}

// linkGauge links a gauge to a reach as its primary gauge.
//
// gauges.reach_id is a display shortcut — first-come-first-served, only set
// when the gauge has no existing reach. A gauge shared across multiple reaches
// (e.g. PLAGRACO for both Bailey and Foxton) will only update reach_id for the
// first reach it processes, but the association table row is always written so
// the gauge still appears in reach-name searches for every reach it's linked to.
func linkGauge(ctx context.Context, pool *pgxpool.Pool, reachID, gaugeExtID, gaugeSource string) error {
	// Gauge → reach display pointer (only if unclaimed)
	tag, err := pool.Exec(ctx, `
		UPDATE gauges SET reach_id = $1
		WHERE external_id = $2 AND source = $3
		  AND reach_id IS NULL
	`, reachID, gaugeExtID, gaugeSource)
	if err != nil {
		return fmt.Errorf("update gauge reach_id: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var existingReach string
		pool.QueryRow(ctx, `
			SELECT COALESCE(r.name, r.slug, '?')
			FROM gauges g JOIN reaches r ON r.id = g.reach_id
			WHERE g.external_id = $1 AND g.source = $2
		`, gaugeExtID, gaugeSource).Scan(&existingReach)
		fmt.Printf("  ○ gauge %s/%s display pointer already claimed by %q — association still recorded\n", gaugeSource, gaugeExtID, existingReach)
	}

	// Reach → gauge (primary_gauge_id) — always attempt, reach wins first claim
	_, err = pool.Exec(ctx, `
		UPDATE reaches SET primary_gauge_id = (
			SELECT id FROM gauges WHERE external_id = $2 AND source = $3 LIMIT 1
		)
		WHERE id = $1 AND primary_gauge_id IS NULL
	`, reachID, gaugeExtID, gaugeSource)
	if err != nil {
		return err
	}

	// Association table — always written regardless of display pointer status
	_, err = pool.Exec(ctx, `
		INSERT INTO gauge_reach_associations (gauge_id, reach_id, relationship)
		SELECT id, $2, 'primary' FROM gauges WHERE external_id = $1 AND source = $3
		ON CONFLICT (gauge_id, reach_id) DO NOTHING
	`, gaugeExtID, reachID, gaugeSource)
	return err
}

// linkRelatedGauge inserts a non-primary association into gauge_reach_associations.
// Does NOT touch gauges.reach_id (that stays as the primary display pointer)
// and does NOT set primary_gauge_id on the reach.
// This allows a gauge like PLAGRACO to be an upstream_indicator for both
// Bailey and Foxton without disturbing either reach's primary gauge.
func linkRelatedGauge(ctx context.Context, pool *pgxpool.Pool, reachID string, ga gaugeAssoc) error {
	tag, err := pool.Exec(ctx, `
		INSERT INTO gauge_reach_associations (gauge_id, reach_id, relationship)
		SELECT id, $2, $3 FROM gauges WHERE external_id = $1 AND source = $4
		ON CONFLICT (gauge_id, reach_id) DO UPDATE SET relationship = EXCLUDED.relationship
	`, ga.ExtID, reachID, ga.Relationship, ga.Source)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		fmt.Printf("  ○ gauge %s/%s not found in DB — skipping\n", ga.Source, ga.ExtID)
	}
	return nil
}

// writeKnownRapid inserts a domain-expert rapid with data_source='manual' and verified=true.
// ON CONFLICT DO NOTHING — never overwrites an existing row for the same reach+name.
func writeKnownRapid(ctx context.Context, pool *pgxpool.Pool, reachID string, r knownRapid) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO rapids
			(reach_id, name, river_mile, class_rating, class_at_low, class_at_high,
			 description, portage_description, is_portage_recommended,
			 data_source, verified)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'maintainer',TRUE)
		ON CONFLICT (reach_id, name) DO NOTHING
	`,
		reachID,
		r.Name,
		r.RiverMile,
		r.ClassRating,
		r.ClassAtLow,
		r.ClassAtHigh,
		nullStr(r.Description),
		nullStr(r.PortageDescription),
		r.IsPortageRecommended,
	)
	return err
}

// writeKnownFlowRange inserts a domain-expert flow range with verified=true.
// Uses ON CONFLICT DO NOTHING — never overwrites an existing verified row.
func writeKnownFlowRange(ctx context.Context, pool *pgxpool.Pool, gaugeExtID, gaugeSource string, fr knownFlowRange) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO flow_ranges
			(gauge_id, label, min_cfs, max_cfs, craft_type, data_source, verified)
		SELECT g.id, $2, $3, $4, 'general', 'manual', TRUE
		FROM gauges g
		WHERE g.external_id = $1 AND g.source = $5
		ON CONFLICT (gauge_id, label, craft_type) DO NOTHING
	`, gaugeExtID, fr.Label, fr.MinCFS, fr.MaxCFS, gaugeSource)
	return err
}

// writeFlowRanges inserts AI-seeded flow ranges for a reach's primary gauge.
// Returns (total written, auto-verified count).
// ON CONFLICT DO NOTHING — never overwrites existing manual or verified rows.
func writeFlowRanges(ctx context.Context, pool *pgxpool.Pool, gaugeExtID, gaugeSource string, ranges []ai.FlowRangeSeed) (written, autoVerified int) {
	for _, fr := range ranges {
		craftType := fr.CraftType
		if craftType == "" {
			craftType = "general"
		}
		_, err := pool.Exec(ctx, `
			INSERT INTO flow_ranges
				(gauge_id, label, min_cfs, max_cfs, craft_type, data_source, ai_confidence, verified)
			SELECT g.id, $2, $3, $4, $5, 'ai_seed', $6, $7
			FROM gauges g
			WHERE g.external_id = $1 AND g.source = $8
			ON CONFLICT (gauge_id, label, craft_type) DO NOTHING
		`, gaugeExtID, fr.Label, fr.MinCFS, fr.MaxCFS, craftType, fr.Confidence, fr.AutoVerified(), gaugeSource)
		if err != nil {
			fmt.Printf("    ✗ flow range %s: %v\n", fr.Label, err)
			continue
		}
		written++
		if fr.AutoVerified() {
			autoVerified++
		}
	}
	return
}

// writeDescription updates the reach with its AI-generated description.
func writeDescription(ctx context.Context, pool *pgxpool.Pool, reachID string, seed *ai.ReachSeed) error {
	_, err := pool.Exec(ctx, `
		UPDATE reaches SET
			description               = $2,
			description_source        = 'ai_seed',
			description_ai_confidence = $3,
			description_verified      = $4,
			description_updated_at    = NOW()
		WHERE id = $1
	`,
		reachID,
		seed.Description,
		seed.DescriptionConfidence,
		seed.DescriptionAutoVerified(),
	)
	return err
}

// writeRapids inserts AI-seeded rapids. Returns (total written, auto-verified count).
// ON CONFLICT DO NOTHING — never overwrites existing community or maintainer data.
func writeRapids(ctx context.Context, pool *pgxpool.Pool, reachID string, rapids []ai.RapidSeed) (written, autoVerified int) {
	for _, r := range rapids {
		_, err := pool.Exec(ctx, `
			INSERT INTO rapids
				(reach_id, name, river_mile, class_rating, class_at_low, class_at_high,
				 description, portage_description, is_portage_recommended,
				 data_source, ai_confidence, verified)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'ai_seed',$10,$11)
			ON CONFLICT (reach_id, name) DO NOTHING
		`,
			reachID,
			r.Name,
			r.RiverMile,
			r.ClassRating,
			r.ClassAtLow,
			r.ClassAtHigh,
			nullStr(r.Description),
			nullStr(r.PortageDescription),
			r.IsPortageRecommended,
			r.Confidence,
			r.AutoVerified(),
		)
		if err != nil {
			fmt.Printf("    ✗ rapid %q: %v\n", r.Name, err)
			continue
		}
		written++
		if r.AutoVerified() {
			autoVerified++
		}
	}
	return
}

// writeAccess inserts AI-seeded access points and their waypoints.
// Returns (total written, auto-verified count).
func writeAccess(ctx context.Context, pool *pgxpool.Pool, reachID string, access []ai.AccessSeed) (written, autoVerified int) {
	for _, a := range access {
		var accessID string
		err := pool.QueryRow(ctx, `
			INSERT INTO reach_access
				(reach_id, access_type, name, directions, road_type,
				 parking_fee, permit_required, permit_info, permit_url,
				 seasonal_close_start, seasonal_close_end, notes,
				 entry_style, approach_dist_mi, approach_notes,
				 data_source, ai_confidence, verified)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,'ai_seed',$16,$17)
			RETURNING id
		`,
			reachID,
			a.AccessType,
			nullStr(a.Name),
			nullStr(a.Directions),
			nullStr(a.RoadType),
			a.ParkingFee,
			a.PermitRequired,
			nullStr(a.PermitInfo),
			nullStr(a.PermitURL),
			nullStr(a.SeasonalCloseStart),
			nullStr(a.SeasonalCloseEnd),
			nullStr(a.Notes),
			nullStr(a.EntryStyle),
			a.ApproachDistMi,
			nullStr(a.ApproachNotes),
			a.Confidence,
			a.AutoVerified(),
		).Scan(&accessID)
		if err != nil {
			fmt.Printf("    ✗ access %q: %v\n", a.Name, err)
			continue
		}
		written++
		if a.AutoVerified() {
			autoVerified++
		}

		// Water and parking locations — stored as PostGIS GEOGRAPHY points.
		if a.WaterLat != nil && a.WaterLon != nil {
			pool.Exec(ctx, `
				UPDATE reach_access
				SET location = ST_MakePoint($2, $3)::geography
				WHERE id = $1
			`, accessID, *a.WaterLon, *a.WaterLat)
		}
		if a.ParkingLat != nil && a.ParkingLon != nil {
			pool.Exec(ctx, `
				UPDATE reach_access
				SET parking_location = ST_MakePoint($2, $3)::geography
				WHERE id = $1
			`, accessID, *a.ParkingLon, *a.ParkingLat)
		}

		// Hike-to-water time.
		if a.HikeToWaterMin != nil {
			pool.Exec(ctx, `
				UPDATE reach_access SET hike_to_water_min = $2 WHERE id = $1
			`, accessID, *a.HikeToWaterMin)
		}

		if len(a.Waypoints) > 0 {
			writeWaypoints(ctx, pool, accessID, a.Waypoints)
		}
	}
	return
}

// writeWaypoints inserts ordered approach waypoints for a trail or technical access point.
func writeWaypoints(ctx context.Context, pool *pgxpool.Pool, accessID string, waypoints []ai.WaypointSeed) {
	for _, w := range waypoints {
		_, err := pool.Exec(ctx, `
			INSERT INTO access_waypoints
				(access_id, sequence, label, description, ai_confidence,
				 gps_source, data_source, verified)
			VALUES ($1,$2,$3,$4,$5,'map_pin','ai_seed',FALSE)
			ON CONFLICT (access_id, sequence) DO NOTHING
		`,
			accessID,
			w.Sequence,
			w.Label,
			nullStr(w.Description),
			nil, // AI-generated waypoints don't get a confidence score — they're spatial guesses
		)
		if err != nil {
			fmt.Printf("      ✗ waypoint seq %d: %v\n", w.Sequence, err)
			continue
		}
		if w.Lat != nil && w.Lon != nil {
			pool.Exec(ctx, `
				UPDATE access_waypoints
				SET location = ST_MakePoint($2, $3)::geography
				WHERE access_id = $1 AND sequence = $4
			`, accessID, *w.Lon, *w.Lat, w.Sequence)
		}
	}
}

// ---- Helpers ----------------------------------------------------------------

// writeKnownAccess inserts a domain-expert access point with data_source='manual' and verified=true.
// ON CONFLICT DO NOTHING — never overwrites an existing row for the same reach+name.
func writeKnownAccess(ctx context.Context, pool *pgxpool.Pool, reachID string, a knownAccess) error {
	var accessID string
	err := pool.QueryRow(ctx, `
		INSERT INTO reach_access
			(reach_id, access_type, name, directions, data_source, ai_confidence, verified)
		VALUES ($1,$2,$3,$4,'maintainer',100,TRUE)
		ON CONFLICT (reach_id, access_type, name) DO NOTHING
		RETURNING id
	`, reachID, a.AccessType, a.Name, nullStr(a.Directions)).Scan(&accessID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // conflict — already exists, skip coordinate update
	}
	if err != nil {
		return err
	}
	if a.WaterLat != nil && a.WaterLon != nil {
		_, err = pool.Exec(ctx, `
			UPDATE reach_access
			SET location = ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography
			WHERE id = $1
		`, accessID, *a.WaterLon, *a.WaterLat)
		if err != nil {
			return fmt.Errorf("location: %w", err)
		}
	}
	if a.ParkingLat != nil && a.ParkingLon != nil {
		_, err = pool.Exec(ctx, `
			UPDATE reach_access
			SET parking_location = ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography
			WHERE id = $1
		`, accessID, *a.ParkingLon, *a.ParkingLat)
		if err != nil {
			return fmt.Errorf("parking_location: %w", err)
		}
	}
	return nil
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nullFloat(f float64) *float64 {
	if f == 0 {
		return nil
	}
	return &f
}
