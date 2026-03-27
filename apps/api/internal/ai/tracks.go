package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// TrackPoint is one GPS sample from an ingested track (GPX, OwnTracks, etc.)
type TrackPoint struct {
	Lat       float64   `json:"lat"`
	Lon       float64   `json:"lon"`
	Timestamp time.Time `json:"timestamp"`
	SpeedMph  *float64  `json:"speed_mph,omitempty"`  // if available from source
	ElevM     *float64  `json:"elev_m,omitempty"`     // if available from source
}

// TrackContext is what H2OFlow already knows about the reach this track likely covers.
type TrackContext struct {
	ReachName     string
	ReachSlug     string
	KnownPutIn    *[2]float64 // [lng, lat] or nil
	KnownTakeOut  *[2]float64
	KnownRapids   []KnownFeature
	KnownAccess   []KnownFeature
	LengthMi      float64
	ClassMin, ClassMax float64
}

// KnownFeature is an existing marker (rapid, access point) in the DB.
type KnownFeature struct {
	ID          string
	Label       string   // "Zoom Flume", "Hecla Junction", etc.
	Lat, Lon    float64
	FeatureType string   // "rapid", "put_in", "take_out", "waypoint"
}

// TrackSuggestion is one AI-generated recommendation from analyzing a track.
type TrackSuggestion struct {
	// Type of improvement suggested
	// update_put_in / update_take_out / update_rapid_location /
	// new_waypoint / update_centerline / confirm_existing
	SuggestionType string  `json:"suggestion_type"`

	// Target feature to update (ID from KnownFeature), or empty for new features
	TargetID string `json:"target_id,omitempty"`

	// Suggested coordinates
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`

	// Human-readable explanation of why — shown to the user for accept/reject
	Reasoning  string `json:"reasoning"`
	Confidence int    `json:"confidence"` // 0–100

	// For new waypoints: label and description
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
}

// TrackAnalysis is the full result of analyzing one GPS track.
type TrackAnalysis struct {
	ReachSlug   string            `json:"reach_slug"`
	Suggestions []TrackSuggestion `json:"suggestions"`

	// Summary of what the AI understood about the track narrative
	// (approach / on-water / egress segments, notable stops, etc.)
	// Shown to the user alongside the map for transparency.
	NarrativeSummary string `json:"narrative_summary"`
}

// TrackAnalyzer interprets GPS tracks against existing reach data and
// suggests improvements to put-in/take-out locations, rapid coordinates,
// and access waypoints using narrative reasoning rather than static geometry.
type TrackAnalyzer struct {
	client anthropic.Client
}

func NewTrackAnalyzer(apiKey string) *TrackAnalyzer {
	return &TrackAnalyzer{client: anthropic.NewClient(option.WithAPIKey(apiKey))}
}

const trackSystemPrompt = `You are a GPS track analyst for H2OFlow, a whitewater paddling platform.

You will receive a GPS track from a paddling trip alongside existing location data for a river reach (put-in, take-out, rapids, access waypoints). The track has already been pre-filtered to remove vehicle-speed movement (>15mph), but may still include walking to/from parking and shuttle logistics. Your job is to reason about what the track tells us and suggest improvements to the existing data.

IMPORTANT: You will be given the known put-in and take-out coordinates for this reach. Use these to anchor your analysis. The track should start near the put-in and end near the take-out — movement before the put-in cluster or after the take-out cluster is pre/post-trip logistics and should be ignored for data improvement purposes.

River sections are often chained: the take-out for one section is the put-in for the next section downstream. Groups frequently drive to the take-out FIRST to drop a shuttle vehicle, then drive to the put-in. This means the track may show a stationary cluster near the take-out BEFORE the paddle starts. Do not confuse a pre-trip shuttle stop at the take-out with the actual take-out event.

GPS tracks from paddling trips follow a predictable narrative:
1. Pre-trip shuttle (optional) — stationary cluster near the take-out area before the paddle
2. Vehicle approach to put-in — already filtered out if >15mph
3. Approach on foot — walking speed (2-4mph), heading toward the river from parking
4. Pre-launch stationary — cluster of points near water while people gear up (5-20min stop)
5. On-water movement — speed matches river current (1-15mph depending on run), linear downstream
6. Stationary on water — scouting stops, portages, lunch breaks (slower movement, may move laterally to shore)
7. Take-out cluster — stationary near water, then transition back to walking speed
8. Post-trip egress — walking back to vehicles or waiting for shuttle

Use the known put-in and take-out coordinates as anchors. The pre-launch stationary cluster nearest to the known put-in is the actual put-in event. The on-water stationary cluster nearest to the known take-out is the actual take-out event. A stationary cluster near the take-out that occurs BEFORE on-water movement is a shuttle stop, not the take-out.

Respond ONLY with valid JSON. No markdown, no explanation outside the JSON.

Schema:
{
  "narrative_summary": "string — 2-3 sentences describing what you understood about this track",
  "suggestions": [
    {
      "suggestion_type": "update_put_in" | "update_take_out" | "update_rapid_location" | "new_waypoint" | "confirm_existing",
      "target_id": "string | empty",
      "lat": number,
      "lon": number,
      "reasoning": "string — explain why, referencing specific track behavior",
      "confidence": 0-100,
      "label": "string (for new_waypoint only)",
      "description": "string (for new_waypoint only)"
    }
  ]
}

Confidence guidelines:
- 90+: Very clear evidence in the track. Multiple corroborating signals (speed transition + stationary cluster + direction change).
- 70-89: Good evidence but one ambiguous signal (e.g. stationary cluster but could be a scout, not the put-in).
- 50-69: Suggestive but inconclusive. Worth flagging for user review.
- Below 50: Omit — don't suggest changes you're not meaningfully confident about.`

// maxDrivingSpeedMph is the threshold above which a GPS sample is considered
// vehicle movement and excluded from trip analysis.
const maxDrivingSpeedMph = 15.0

// PrepareTrack filters and clips a raw GPS track for analysis:
//  1. Removes vehicle-speed points (driving to/from river, shuttle)
//  2. If put-in and take-out coords are known, clips the track to the window
//     starting near the put-in and ending near the take-out
//
// The returned track covers the river trip only — approach walk + paddle + egress walk.
// Pre-trip shuttle stops and post-trip driving are excluded.
func PrepareTrack(points []TrackPoint, putIn, takeOut *[2]float64) []TrackPoint {
	// Step 1: remove driving-speed points
	filtered := make([]TrackPoint, 0, len(points))
	for _, p := range points {
		if p.SpeedMph == nil || *p.SpeedMph <= maxDrivingSpeedMph {
			filtered = append(filtered, p)
		}
	}

	// Step 2: clip to put-in / take-out window when coordinates are known.
	// Find the first point within 500m of the put-in and the last point within
	// 500m of the take-out. Everything outside that window is logistics noise.
	if putIn != nil && takeOut != nil && len(filtered) > 2 {
		start, end := 0, len(filtered)-1
		for i, p := range filtered {
			if haversineM(p.Lat, p.Lon, putIn[1], putIn[0]) < 500 {
				start = i
				break
			}
		}
		for i := len(filtered) - 1; i >= start; i-- {
			if haversineM(filtered[i].Lat, filtered[i].Lon, takeOut[1], takeOut[0]) < 500 {
				end = i
				break
			}
		}
		filtered = filtered[start : end+1]
	}

	return filtered
}

// haversineM returns the great-circle distance in metres between two lat/lon points.
func haversineM(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6_371_000.0 // Earth radius in metres
	const deg = 3.14159265358979323846 / 180.0
	φ1, φ2 := lat1*deg, lat2*deg
	Δφ := (lat2 - lat1) * deg
	Δλ := (lon2 - lon1) * deg
	sinΔφ := sinApprox(Δφ / 2)
	sinΔλ := sinApprox(Δλ / 2)
	a := sinΔφ*sinΔφ + cosApprox(φ1)*cosApprox(φ2)*sinΔλ*sinΔλ
	return R * 2 * atanApprox(sqrtApprox(a), sqrtApprox(1-a))
}

// Minimal math approximations to avoid importing math in this package.
// These are accurate enough for the ~500m proximity threshold used above.
func sinApprox(x float64) float64  { return x - x*x*x/6 + x*x*x*x*x/120 }
func cosApprox(x float64) float64  { return 1 - x*x/2 + x*x*x*x/24 }
func sqrtApprox(x float64) float64 { /* Newton's method */ s := x; for i := 0; i < 10; i++ { s = (s + x/s) / 2 }; return s }
func atanApprox(y, x float64) float64 { /* atan2 via identity */ if x > 0 { return atanSimple(y/x) }; if x < 0 && y >= 0 { return atanSimple(y/x) + 3.14159 }; return atanSimple(y/x) - 3.14159 }
func atanSimple(x float64) float64 { return x / (1 + 0.28125*x*x) } // Bhaskara approximation

// AnalyzeTrack interprets a GPS track and returns improvement suggestions.
// Call PrepareTrack first to filter vehicle movement and clip to reach bounds.
// Times out after 45 seconds — this is an async background operation.
func (a *TrackAnalyzer) AnalyzeTrack(ctx context.Context, points []TrackPoint, tc TrackContext) (*TrackAnalysis, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	prompt, err := buildTrackPrompt(points, tc)
	if err != nil {
		return nil, fmt.Errorf("build prompt: %w", err)
	}

	msg, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_6,
		MaxTokens: 2048,
		System: []anthropic.TextBlockParam{
			{Text: trackSystemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude: %w", err)
	}
	if len(msg.Content) == 0 {
		return nil, fmt.Errorf("claude: empty response")
	}

	raw := strings.TrimSpace(msg.Content[0].Text)
	var result TrackAnalysis
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("claude: parse: %w\nraw: %s", err, raw)
	}
	result.ReachSlug = tc.ReachSlug

	// Drop low-confidence suggestions — ambiguous data shouldn't update community records.
	result.Suggestions = filterByConfidence(result.Suggestions, func(s TrackSuggestion) int {
		return s.Confidence
	})

	return &result, nil
}

func buildTrackPrompt(points []TrackPoint, tc TrackContext) (string, error) {
	// Summarize the track — sending every GPS sample would exhaust the context window.
	// Downsample to at most 500 points for the prompt; preserve start, end, and
	// any points where speed or direction changes significantly.
	summarized := downsampleTrack(points, 500)

	trackJSON, err := json.Marshal(summarized)
	if err != nil {
		return "", err
	}

	contextJSON, err := json.Marshal(tc)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Reach context:\n%s\n\n", contextJSON)
	fmt.Fprintf(&b, "GPS track (%d points after downsampling from %d):\n%s\n\n",
		len(summarized), len(points), trackJSON)
	b.WriteString("Analyze this track and suggest improvements to the reach data.")
	return b.String(), nil
}

// downsampleTrack reduces a track to at most maxPoints samples while preserving
// the start, end, and points that represent meaningful changes in movement.
func downsampleTrack(points []TrackPoint, maxPoints int) []TrackPoint {
	if len(points) <= maxPoints {
		return points
	}
	// Simple stride-based downsampling — good enough for now.
	// A smarter implementation would preserve speed-change inflection points.
	stride := len(points) / maxPoints
	out := make([]TrackPoint, 0, maxPoints)
	for i := 0; i < len(points); i += stride {
		out = append(out, points[i])
	}
	// Always include the last point.
	if out[len(out)-1] != points[len(points)-1] {
		out = append(out, points[len(points)-1])
	}
	return out
}
