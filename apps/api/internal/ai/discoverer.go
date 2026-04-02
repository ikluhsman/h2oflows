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

// DiscoveredReach is one whitewater run returned by the state discoverer.
// Names are common geographic names (what the paddling community calls the run).
// Class ratings are normalized to our numeric schema — no Roman numeral strings.
type DiscoveredReach struct {
	CommonName   string  `json:"common_name"`   // "Browns Canyon", "Gore Canyon"
	River        string  `json:"river"`         // "Arkansas River"
	SectionDesc  string  `json:"section_desc"`  // "Arkansas River, CO — Buena Vista to Salida"
	ClassMin     float64 `json:"class_min"`     // 1.0–6.0, .5 = + modifier
	ClassMax     float64 `json:"class_max"`
	Character    string  `json:"character"`     // creeking|pool-drop|continuous|big-water|canyon|flatwater
	LengthMi     float64 `json:"length_mi"`     // 0 = unknown
	PutInLat     float64 `json:"put_in_lat"`
	PutInLon     float64 `json:"put_in_lon"`
	TakeOutLat   float64 `json:"take_out_lat"`
	TakeOutLon   float64 `json:"take_out_lon"`
	USGSGaugeID  string  `json:"usgs_gauge_id"` // 8-digit station ID if known, else ""
}

// ReachDiscoverer asks Claude to enumerate notable whitewater runs for a state.
// No third-party data sources are used — output is Claude's original knowledge.
type ReachDiscoverer struct {
	client anthropic.Client
}

func NewReachDiscoverer(apiKey string) *ReachDiscoverer {
	return &ReachDiscoverer{client: anthropic.NewClient(option.WithAPIKey(apiKey))}
}

const discovererSystemPrompt = `You are a whitewater paddling reference for H2OFlows, a river gauge monitoring platform for kayakers, rafters, and packrafters.

Task: given a US state, list the notable whitewater sections that intermediate-to-expert paddlers would want gauge data for.

Respond ONLY with a valid JSON array. No markdown fences, no explanation, no preamble.

[
  {
    "common_name":  "string — the name the paddling community uses (e.g. Browns Canyon, Gore Canyon, The Numbers)",
    "river":        "string — river name only (e.g. Arkansas River)",
    "section_desc": "string — location context for a reach stub (e.g. Arkansas River, Colorado — Buena Vista to Salida)",
    "class_min":    number,  // numeric 1.0–6.0; use .5 for + modifier (IV+ = 4.5, V- = 4.7)
    "class_max":    number,
    "character":    "creeking" | "pool-drop" | "continuous" | "big-water" | "canyon" | "flatwater",
    "length_mi":    number,  // approximate, 0 if unknown
    "put_in_lat":   number,  // decimal degrees, within ~1 mile is fine
    "put_in_lon":   number,
    "take_out_lat": number,
    "take_out_lon": number,
    "usgs_gauge_id": "string" // USGS 8-digit station ID for the most relevant gauge; empty string if unsure — do not guess
  }
]

Rules:
- Include Class II through V runs that have established put-in/take-out points and are known to the regional or national paddling community.
- Aim for 20–50 runs for a well-known whitewater state (Colorado, Idaho, California), fewer for quieter states.
- character must be exactly one of: creeking, pool-drop, continuous, big-water, canyon, flatwater
- class_min/class_max are numeric only: 1=I, 2=II, 3=III, 4=IV, 4.5=IV+, 5=V, 5.5=V+, 6=unrunnable
- Do not include runs you have little information about.
- Do not include purely flatwater sections or lakes.`

// DiscoverReaches calls Claude to enumerate well-known whitewater runs for the
// given state. Returns original data — no third-party API is called.
func (d *ReachDiscoverer) DiscoverReaches(ctx context.Context, stateAbbr string) ([]DiscoveredReach, error) {
	ctx, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()

	msg, err := d.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_6,
		MaxTokens: 8192,
		System: []anthropic.TextBlockParam{
			{Text: discovererSystemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(
				fmt.Sprintf("List notable whitewater runs in %s.", stateFullName(stateAbbr)),
			)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude: %w", err)
	}
	if len(msg.Content) == 0 {
		return nil, fmt.Errorf("claude: empty response")
	}

	raw := strings.TrimSpace(msg.Content[0].Text)
	// Strip markdown code fences if the model includes them anyway
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var reaches []DiscoveredReach
	if err := json.Unmarshal([]byte(raw), &reaches); err != nil {
		return nil, fmt.Errorf("parse response: %w\nraw (first 500): %.500s", err, raw)
	}

	// Sanitize character values in case the model drifts from the allowed set
	allowed := map[string]bool{
		"creeking": true, "pool-drop": true, "continuous": true,
		"big-water": true, "canyon": true, "flatwater": true,
	}
	for i := range reaches {
		c := strings.ToLower(strings.ReplaceAll(reaches[i].Character, " ", "-"))
		if !allowed[c] {
			reaches[i].Character = "continuous" // safe fallback
		} else {
			reaches[i].Character = c
		}
	}

	return reaches, nil
}

func stateFullName(abbr string) string {
	names := map[string]string{
		"CO": "Colorado", "UT": "Utah", "WY": "Wyoming",
		"NM": "New Mexico", "AZ": "Arizona", "ID": "Idaho",
		"MT": "Montana", "NV": "Nevada", "CA": "California",
		"OR": "Oregon", "WA": "Washington", "AK": "Alaska",
		"TX": "Texas", "NC": "North Carolina", "VA": "Virginia",
		"WV": "West Virginia", "TN": "Tennessee", "GA": "Georgia",
	}
	if n, ok := names[strings.ToUpper(abbr)]; ok {
		return n
	}
	return abbr
}
