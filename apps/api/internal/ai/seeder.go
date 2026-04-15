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

// ReachSeed is the full AI-generated seed for one reach.
// All fields are drafts — verified=false until a human or verifier pass confirms them.
type ReachSeed struct {
	Description           string           `json:"description"`
	DescriptionConfidence int              `json:"description_confidence"` // 0–100
	Rapids                []RapidSeed      `json:"rapids"`
	Access                []AccessSeed     `json:"access"`
	FlowRanges            []FlowRangeSeed  `json:"flow_ranges"`
}


type RapidSeed struct {
	Name                  string   `json:"name"`
	RiverMile             *float64 `json:"river_mile"`             // null = unknown
	ClassRating           *float64 `json:"class_rating"`           // null = unknown
	ClassAtLow            *float64 `json:"class_at_low"`
	ClassAtHigh           *float64 `json:"class_at_high"`
	Description           string   `json:"description"`            // beta: lines, hazards, scouting
	PortageDescription    string   `json:"portage_description"`    // empty = no known portage
	IsPortageRecommended  bool     `json:"is_portage_recommended"`
	Confidence            int      `json:"confidence"`             // 0–100
}

type AccessSeed struct {
	AccessType      string   `json:"access_type"`        // put_in/take_out/shuttle_drop/intermediate/camp
	Name            string   `json:"name"`
	Directions      string   `json:"directions"`
	RoadType        string   `json:"road_type"`          // paved/gravel/dirt/high-clearance/4wd

	// Entry style: how you get from parking to water
	// boat_ramp=formal ramp, bank=rough scramble <1/8mi,
	// trail=established or use trail, technical=ropes/belay/multi-mile
	EntryStyle      string   `json:"entry_style"`
	ApproachDistMi  *float64 `json:"approach_dist_mi"`
	ApproachNotes   string   `json:"approach_notes"`     // freeform narrative of the approach

	// Ordered waypoints for trail/technical approaches (omit for boat_ramp/bank)
	Waypoints       []WaypointSeed `json:"waypoints"`

	// Water access point — where boats enter or exit the river
	WaterLat        *float64 `json:"water_lat"`
	WaterLon        *float64 `json:"water_lon"`

	// Parking/meet-up location — where vehicles park; may differ from water access
	ParkingLat      *float64 `json:"parking_lat"`
	ParkingLon      *float64 `json:"parking_lon"`
	ParkingNotes    string   `json:"parking_notes"`
	HikeToWaterMin  *int     `json:"hike_to_water_min"`

	ParkingFee      *float64 `json:"parking_fee"`
	PermitRequired  bool     `json:"permit_required"`
	PermitInfo      string   `json:"permit_info"`
	PermitURL       string   `json:"permit_url"`
	SeasonalCloseStart string `json:"seasonal_close_start"`
	SeasonalCloseEnd   string `json:"seasonal_close_end"`
	Notes           string   `json:"notes"`
	Confidence      int      `json:"confidence"`
}

// WaypointSeed is one step along a trail or technical approach.
type WaypointSeed struct {
	Sequence    int      `json:"sequence"`
	Label       string   `json:"label"`       // "Trailhead", "Creek crossing", "Top of cliff", "Water entry"
	Description string   `json:"description"` // "Belay kayaks here, ~40ft, river-left line"
	Lat         *float64 `json:"lat"`
	Lon         *float64 `json:"lon"`
}

// ReachSeeder seeds reach data (rapids, access, description) from Claude's
// training knowledge. Output is always marked as draft (data_source='ai_seed',
// verified=false). Downstream verification passes or community edits promote it.
type ReachSeeder struct {
	client anthropic.Client
}

func NewReachSeeder(apiKey string) *ReachSeeder {
	return &ReachSeeder{client: anthropic.NewClient(option.WithAPIKey(apiKey))}
}

// ReachContext is the known facts about a reach passed to the seeder.
// The more context provided, the more accurate the seed.
type ReachContext struct {
	Name       string  // e.g. "Browns Canyon"
	Region     string  // e.g. "Arkansas River, Colorado"
	ClassMin   float64
	ClassMax   float64
	LengthMi   float64
	PutInLat   float64
	PutInLon   float64
	TakeOutLat float64
	TakeOutLon float64
	Notes      string  // local knowledge, gauge math, access quirks — included verbatim in prompt
}

const seederSystemPrompt = `You are a whitewater paddling data assistant for H2OFlows, a platform used by experienced kayakers, rafters, canoeists, and packrafters.

H2OFlows builds its river library from public sources: USGS and state agency gauge data, published paddling guidebooks (Caudill, Stohlquist, Nealy, and others), and the accumulated knowledge of the online paddling community. All descriptions you write are original — draw on that published and community knowledge base, but write in your own words.

Your job: given a river reach, generate accurate seed data for the rapid inventory, access points, and a reach description. This data will be shown to Class 5 kayakers — accuracy is paramount. Do not invent rapids or access points you are not confident about. An empty list is better than wrong data.

Respond ONLY with a valid JSON object matching this exact schema. No markdown, no explanation.

{
  "description": "string — 2-4 paragraph markdown description covering character, typical flows, key features, and historical/cultural context. Written for an experienced paddler.",
  "description_confidence": 0-100,
  "rapids": [
    {
      "name": "string — official or widely-used name",
      "river_mile": number | null,
      "class_rating": number | null,
      "class_at_low": number | null,
      "class_at_high": number | null,
      "description": "string — line descriptions, hazards, what to scout, consequence level",
      "portage_description": "string — portage route if known, empty string if none",
      "is_portage_recommended": bool,
      "confidence": 0-100
    }
  ],
  "access": [
    {
      "access_type": "put_in" | "take_out" | "shuttle_drop" | "intermediate" | "camp",
      "name": "string — common name (e.g. 'Hecla Junction', 'Ruby Mountain')",
      "directions": "string — driving directions from nearest town or highway",
      "road_type": "paved" | "gravel" | "dirt" | "high-clearance" | "4wd",
      "water_lat": number | null,
      "water_lon": number | null,
      "parking_lat": number | null,
      "parking_lon": number | null,
      "parking_notes": "string — lot surface, capacity, seasonal conditions",
      "hike_to_water_min": number | null,
      "parking_fee": number | null,
      "permit_required": bool,
      "permit_info": "string — brief description of permit if required",
      "permit_url": "string — recreation.gov or agency URL if known, else empty",
      "seasonal_close_start": "MM-DD" | "",
      "seasonal_close_end": "MM-DD" | "",
      "notes": "string — road conditions, boat ramp, eddy size, etc.",
      "confidence": 0-100
    }
  ],
  "flow_ranges": [
    {
      "label": "too_low" | "running" | "high" | "very_high",
      "min_cfs": number | null,
      "max_cfs": number | null,
      "notes": "string — brief context, e.g. 'Bony but runnable at minimum flows'",
      "confidence": 0-100
    }
  ]
}

IMPORTANT on access points:
- parking_lat/lon and water_lat/lon are often DIFFERENT points. The parking area is where
  paddlers drive and gear up; water access is where boats enter the river.
- entry_style must be one of: "boat_ramp", "bank", "trail", "technical"
  - boat_ramp: formal ramp infrastructure (Arkansas River commercial launches, etc.)
  - bank: roadside pullout, rough scramble to water, under 1/8 mile, no real trail
  - trail: established or use trail between parking and water
  - technical: rope work, belay, significant scrambling, or multi-mile backcountry approach
- For trail and technical entry styles, include waypoints as an ordered list.
  Each waypoint has a sequence number, a label (Trailhead / Trail junction / Creek crossing /
  Top of cliff / Water entry / etc.), and a description of what to do there.
  Example: "sequence":3, "label":"Top of cliff", "description":"Belay kayaks down ~40ft to pool,
  river-left line. Jump in after gear is down."
- Omit waypoints for boat_ramp and bank entry styles — a single point is sufficient.

IMPORTANT on flow ranges:
- Provide community-accepted CFS windows for the reach's primary gauge (the gauge that most paddlers use to decide whether to run this reach).
- Use exactly these four labels (omit any you are not confident about):
    "too_low"   — below the minimum runnable flow; scrapy, portages increase, not worth the trip
    "running"   — the standard recommended window; the river at its typical good levels
    "high"      — pushy but still runnable for experienced paddlers; stepped-up difficulty
    "very_high" — above the safe or enjoyable limit; experts only or do not run
- min_cfs null means "any flow below max_cfs". max_cfs null means "any flow above min_cfs". Both null is invalid — omit the entry instead.
- For dam-regulated rivers, note if the window is release-dependent rather than flow-dependent.
- Well-documented classics (Browns Canyon, The Numbers, Gore Canyon) should have complete ranges at high confidence. Obscure runs may have only 1-2 bands.
- Do not include flow ranges if the reach is better described by a stage gauge (feet) rather than a flow gauge (cfs) — note this in the reach description instead.

Confidence guidelines:
- 90-100: Classic, nationally recognized run. Named rapids appear in printed guidebooks (Caudill, Stohlquist, Nealy), American Whitewater, and countless trip reports. Zoom Flume, Gorilla, Maytag, Sunshine Falls, the Toilet Bowl — these should be 95+. Do not hedge on runs you clearly know.
- 85-89: Well-known regional run. You have solid information but one or two details (exact river mile, current fee) may have changed.
- 70-84: Known run, moderate documentation. Some details are approximations. Mark as needing local verification.
- 50-69: Seldom-documented or obscure. You have partial information. Include it but flag clearly.
- Below 50: Omit entirely. Wrong beta for a portage or hazard location is dangerous.

Class ratings use the international scale with .5 increments. 6.0 = unrunnable/portage.`

// SeedReach calls Claude to generate a full seed for the given reach.
// Times out after 30 seconds — this is an offline/batch operation, not real-time.
func (s *ReachSeeder) SeedReach(ctx context.Context, rc ReachContext) (*ReachSeed, error) {
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	prompt := buildSeedPrompt(rc)

	msg, err := s.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_6, // use capable model — offline batch, latency OK
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{
			{Text: seederSystemPrompt},
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
	// Strip markdown code fences if present (``` or ```json)
	if strings.HasPrefix(raw, "```") {
		if idx := strings.Index(raw, "\n"); idx != -1 {
			raw = raw[idx+1:]
		}
		raw = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(raw), "```"))
	}
	var seed ReachSeed
	if err := json.Unmarshal([]byte(raw), &seed); err != nil {
		return nil, fmt.Errorf("claude: parse response: %w\nraw: %s", err, raw)
	}

	// Drop any items below the confidence floor.
	// Better to surface nothing than to surface wrong beta to a Class 5 paddler.
	seed.Rapids      = filterByConfidence(seed.Rapids,      func(r RapidSeed)     int { return r.Confidence })
	seed.Access      = filterByConfidence(seed.Access,      func(a AccessSeed)     int { return a.Confidence })
	seed.FlowRanges  = filterByConfidence(seed.FlowRanges,  func(f FlowRangeSeed) int { return f.Confidence })

	return &seed, nil
}

func buildSeedPrompt(rc ReachContext) string {
	var b strings.Builder
	fmt.Fprintf(&b, "River reach: %s\n", rc.Name)
	fmt.Fprintf(&b, "Region: %s\n", rc.Region)
	if rc.ClassMin > 0 || rc.ClassMax > 0 {
		fmt.Fprintf(&b, "Difficulty: Class %.1f–%.1f\n", rc.ClassMin, rc.ClassMax)
	}
	if rc.LengthMi > 0 {
		fmt.Fprintf(&b, "Length: %.1f miles\n", rc.LengthMi)
	}
	if rc.PutInLat != 0 && rc.PutInLon != 0 {
		fmt.Fprintf(&b, "Put-in coordinates: %.5f, %.5f\n", rc.PutInLat, rc.PutInLon)
	}
	if rc.TakeOutLat != 0 && rc.TakeOutLon != 0 {
		fmt.Fprintf(&b, "Take-out coordinates: %.5f, %.5f\n", rc.TakeOutLat, rc.TakeOutLon)
	}
	if rc.Notes != "" {
		fmt.Fprintf(&b, "\nLocal knowledge / gauge notes:\n%s\n", rc.Notes)
	}
	b.WriteString("\nGenerate the seed data for this reach.")
	return b.String()
}

// confidenceFloor is the minimum confidence for an AI-seeded item to be included.
// Items below this are dropped entirely — wrong beta is worse than missing beta.
const confidenceFloor = 50

// autoVerifyThreshold is the confidence level at which AI-seeded data is treated
// as authoritative without requiring human verification. Classic, well-documented
// runs (Browns Canyon, Gore Canyon, the Numbers, etc.) consistently land here
// because Claude's training data includes guidebooks, trip reports, and AW pages
// for these runs. Items at this level should not surface a "needs verification"
// badge to the user — that would be misleading for Zoom Flume.
//
// Items between confidenceFloor and autoVerifyThreshold are stored as unverified
// drafts, appropriate for obscure or seldom-documented runs.
const autoVerifyThreshold = 85

func filterByConfidence[T any](items []T, conf func(T) int) []T {
	out := items[:0]
	for _, item := range items {
		if conf(item) >= confidenceFloor {
			out = append(out, item)
		}
	}
	return out
}

// AutoVerified reports whether this rapid's confidence meets the auto-verify threshold.
// Callers should set verified=true in the DB when this returns true.
func (r RapidSeed) AutoVerified() bool { return r.Confidence >= autoVerifyThreshold }

// AutoVerified reports whether this access point's confidence meets the threshold.
func (a AccessSeed) AutoVerified() bool { return a.Confidence >= autoVerifyThreshold }

// AutoVerified reports whether the description confidence meets the threshold.
func (s ReachSeed) DescriptionAutoVerified() bool {
	return s.DescriptionConfidence >= autoVerifyThreshold
}

// DescriptionSeed is a focused description-only result, used by SeedDescription.
// Mirrors the description fields of ReachSeed without forcing the caller to
// generate rapids/access/flow_ranges they don't need.
type DescriptionSeed struct {
	Description string `json:"description"`
	Confidence  int    `json:"confidence"`
}

// AutoVerified reports whether this description meets the auto-verify threshold.
func (d DescriptionSeed) AutoVerified() bool { return d.Confidence >= autoVerifyThreshold }

const descriptionSystemPrompt = `You are a whitewater paddling data assistant for H2OFlows, a platform used by experienced kayakers, rafters, canoeists, and packrafters.

Your job: given a river reach, write an accurate prose description for the reach. This is shown to Class 5 paddlers — accuracy matters. Do not invent rapids, distances, gauges, or hazards you are not confident about. A short, accurate description is better than a long, speculative one.

Draw on published guidebooks (Caudill, Stohlquist, Nealy), American Whitewater data, and the accumulated knowledge of the online paddling community. Write in your own words.

Respond ONLY with a valid JSON object — no markdown fences, no preamble:

{
  "description": "string — 2-4 paragraph markdown description covering character, typical flows, key features, and historical or cultural context. Written for an experienced paddler.",
  "confidence": 0-100
}

Confidence guidelines (same as the reach seeder):
- 90-100: Classic, nationally recognized run. Solid published documentation.
- 85-89:  Well-known regional run. Strong information.
- 70-84:  Known run, moderate documentation. Some details approximate.
- 50-69:  Seldom-documented or obscure. Partial information.
- Below 50: You do not have reliable information. Set description to "" and confidence below 50; the caller will skip writing.`

// SeedDescription calls Claude to generate ONLY the prose description for a reach.
// Used for backfilling reaches that were imported from KMZ or other sources without
// going through the full ReachSeeder pass. Times out after 60 seconds.
func (s *ReachSeeder) SeedDescription(ctx context.Context, rc ReachContext) (*DescriptionSeed, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	prompt := buildSeedPrompt(rc)

	msg, err := s.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_6,
		MaxTokens: 1500,
		System: []anthropic.TextBlockParam{
			{Text: descriptionSystemPrompt},
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
	// Strip markdown code fences if present (``` or ```json)
	if strings.HasPrefix(raw, "```") {
		if idx := strings.Index(raw, "\n"); idx != -1 {
			raw = raw[idx+1:]
		}
		raw = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(raw), "```"))
	}
	var d DescriptionSeed
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		return nil, fmt.Errorf("claude: parse response: %w\nraw: %s", err, raw)
	}
	return &d, nil
}

