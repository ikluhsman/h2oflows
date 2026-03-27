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

// FlowRangeSeed is one flow band for a gauge, as returned by the AI seeder.
// It maps directly onto a row in the flow_ranges table.
type FlowRangeSeed struct {
	// Label must be one of the DB CHECK values:
	//   too_low | minimum | fun | optimal | pushy | high | flood
	//
	// Mapping to user-visible flowStatus:
	//   too_low              → "Too Low"     (red)
	//   minimum / pushy      → "Caution"     (yellow)
	//   fun / optimal        → "Runnable"    (green)
	//   high / flood         → "Flood Stage" (red)
	Label     string   `json:"label"`
	MinCFS    *float64 `json:"min_cfs"`    // null = no lower bound (only valid for too_low)
	MaxCFS    *float64 `json:"max_cfs"`    // null = no upper bound (only valid for flood/high)
	CraftType string   `json:"craft_type"` // general | kayak | raft | sup | packraft | canoe
	ClassMod  *float64 `json:"class_modifier"` // difficulty shift at this band, e.g. +0.5
	SourceURL string   `json:"source_url"` // AW page or other cited source; empty if unknown
	Notes      string   `json:"notes"`      // freeform — e.g. "Bony but runnable at minimum"
	Confidence int      `json:"confidence"` // 0–100
}

// AutoVerified reports whether this seed's confidence meets the auto-verify threshold.
// Matches the same threshold used by ReachSeeder so the standards are consistent.
func (f FlowRangeSeed) AutoVerified() bool { return f.Confidence >= autoVerifyThreshold }

// FlowRangeContext is what the caller knows about a gauge+reach before seeding.
// The more context provided, the more accurate the output.
type FlowRangeContext struct {
	// Gauge fields
	GaugeName    string // e.g. "ARKANSAS RIVER AT PARKDALE, CO"
	ExternalID   string // e.g. "07091200"
	Source       string // "usgs" | "dwr"

	// Reach fields — required for meaningful results.
	// Without a reach association the AI cannot know what "runnable" means for paddlers.
	ReachName    string  // e.g. "The Numbers"
	ReachRegion  string  // e.g. "Arkansas River, Colorado"
	ClassMin     float64
	ClassMax     float64

	// Optional extras that improve accuracy
	LengthMi     float64
	AWReachID    string  // if known, allows constructing the AW URL directly
}

// FlowRangeSeeder queries Claude for paddling flow ranges for a specific gauge+reach.
//
// It uses Claude's training knowledge, which includes American Whitewater data,
// published guidebooks, and years of paddling trip reports. For well-documented
// classic runs (Numbers, Browns Canyon, Gore Canyon, Cache la Poudre) Claude's
// training data is highly accurate and produces 90+ confidence scores.
//
// Phase 2: wire a WebSearcher to do live AW page lookups before calling Claude.
// The seeder architecture supports this via the optional searcher field — when
// non-nil, it pre-fetches current AW data and appends it to the prompt so Claude
// can cite the exact page rather than training memory.
type FlowRangeSeeder struct {
	client   anthropic.Client
	searcher WebSearcher // optional — nil = training knowledge only
}

// WebSearcher is a narrow interface for plugging in a live web search provider
// (Brave Search API, SerpAPI, etc.). Implement and inject to enable ai_web mode.
//
// Phase 2 implementation note: before calling Claude, fetch:
//   searcher.Search(ctx, "site:americanwhitewater.org "+reachName+" "+region+" flow levels")
// Append the page content to the prompt so Claude can cite specific cfs numbers
// rather than training-data estimates. This flips data_source from "ai_seed" → "ai_web".
type WebSearcher interface {
	Search(ctx context.Context, query string) ([]WebSearchResult, error)
}

type WebSearchResult struct {
	URL     string
	Title   string
	Snippet string
}

func NewFlowRangeSeeder(apiKey string) *FlowRangeSeeder {
	return &FlowRangeSeeder{client: anthropic.NewClient(option.WithAPIKey(apiKey))}
}

// WithSearcher attaches a live web searcher. When set, SeedFlowRanges will
// pre-fetch AW pages and set data_source to "ai_web" on the results.
func (s *FlowRangeSeeder) WithSearcher(ws WebSearcher) *FlowRangeSeeder {
	s.searcher = ws
	return s
}

// DataSource reports whether this seeder produces "ai_seed" or "ai_web" results.
func (s *FlowRangeSeeder) DataSource() string {
	if s.searcher != nil {
		return "ai_web"
	}
	return "ai_seed"
}

const flowRangeSystemPrompt = `You are a whitewater paddling data assistant for H2OFlow. Your job is to produce accurate river flow range data for a given gauge and reach combination.

Flow ranges define the cfs (cubic feet per second) bands at which a whitewater river section is paddleable and at what difficulty level. This data is shown to experienced paddlers — accuracy matters. A Class 5 kayaker who sees wrong flow data will distrust the entire platform.

You will output a JSON array of flow range objects. Each object represents one flow band for the gauge.

LABELS — use EXACTLY these values (they map to the database CHECK constraint):
  "too_low"  — below minimum runnable. Boat dragging, exposed rocks. Do not run.
  "minimum"  — barely runnable. Scratchy, bony, or pushy depending on character. Proceed with caution.
  "fun"      — solid, enjoyable flows. The run is in. Most paddlers enjoy this range.
  "optimal"  — prime flows. Classic level. The run shines here.
  "pushy"    — high, fast, pushy. Runnable but difficulty has increased. Know your limits.
  "high"     — very high flows. Significantly elevated class rating, some features may be washed out or dangerous.
  "flood"    — flood stage. Do not run. Strainer hazard, undefined hydraulics, dangerous.

CRAFT TYPES:
  "general"  — applies to all craft, or you do not have craft-specific info (most common)
  "kayak"    — whitewater kayak (creek boat, playboat, river runner)
  "raft"     — commercial or private raft (4-person+)
  "sup"      — stand-up paddleboard

Use "general" unless you have specific per-craft minimum knowledge (e.g., commercial raft minimums are often set by permit conditions and differ from kayak minimums).

CONFIDENCE guidelines:
  90–100: The cfs numbers appear in American Whitewater, published guidebooks (Caudill, Stohlquist),
          or are so widely cited in trip reports that there is no ambiguity.
          The Numbers optimal range, Browns Canyon minimum, Gore Canyon flood — these are 95+.
          Do not hedge on runs you clearly know.
  85–89:  Well-documented run, strong information, one detail (exact threshold) may have shifted.
  70–84:  Known run, moderate documentation. Include it but note uncertainty.
  50–69:  Seldom-documented. Partial information only.
  Below 50: Omit entirely. Wrong flow data is dangerous.

SOURCE URLS:
  If the data comes from an American Whitewater page, set source_url to the AW reach URL.
  Format: https://www.americanwhitewater.org/content/River/detail/id/[reach_id]/
  If you know the AW reach ID for a classic run, include it. If not, leave source_url empty.
  Do not fabricate URLs.

IMPORTANT:
  - min_cfs and max_cfs define the band: the label applies when current_cfs >= min_cfs AND current_cfs < max_cfs.
  - "too_low" has no lower bound — set min_cfs to null.
  - "flood" has no upper bound — set max_cfs to null.
  - Bands should be contiguous but may have small gaps if the transition is genuinely ambiguous.
  - class_modifier: how much the difficulty shifts at this band relative to the listed class rating.
    E.g. if the reach is Class IV at optimal and becomes Class IV+ at pushy flows, set class_modifier to +0.5.
    Leave null if you do not have reliable information.

Respond ONLY with a valid JSON array. No markdown fences, no explanation, no preamble.`

// SeedFlowRanges calls Claude to generate flow range seeds for the given gauge+reach.
// Returns only items at or above confidenceFloor — items below are silently dropped.
// Times out after 45 seconds — this is an offline batch operation.
func (s *FlowRangeSeeder) SeedFlowRanges(ctx context.Context, fc FlowRangeContext) ([]FlowRangeSeed, error) {
	if fc.ReachName == "" {
		return nil, fmt.Errorf("flowranges: ReachName is required — gauge must have an associated reach")
	}

	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	prompt, err := s.buildPrompt(ctx, fc)
	if err != nil {
		return nil, err
	}

	msg, err := s.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_6,
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: flowRangeSystemPrompt},
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
	var seeds []FlowRangeSeed
	if err := json.Unmarshal([]byte(raw), &seeds); err != nil {
		return nil, fmt.Errorf("claude: parse response: %w\nraw: %s", err, raw)
	}

	return filterByConfidence(seeds, func(f FlowRangeSeed) int { return f.Confidence }), nil
}

// buildPrompt constructs the user-turn message. When a WebSearcher is available
// it pre-fetches AW data and appends the page snippets so Claude can cite live data.
func (s *FlowRangeSeeder) buildPrompt(ctx context.Context, fc FlowRangeContext) (string, error) {
	var b strings.Builder

	fmt.Fprintf(&b, "Gauge: %s\n", fc.GaugeName)
	fmt.Fprintf(&b, "Gauge ID: %s (%s)\n", fc.ExternalID, strings.ToUpper(fc.Source))
	fmt.Fprintf(&b, "River reach: %s\n", fc.ReachName)
	fmt.Fprintf(&b, "Region: %s\n", fc.ReachRegion)
	if fc.ClassMin > 0 || fc.ClassMax > 0 {
		fmt.Fprintf(&b, "Difficulty: Class %.1f–%.1f\n", fc.ClassMin, fc.ClassMax)
	}
	if fc.LengthMi > 0 {
		fmt.Fprintf(&b, "Length: %.1f miles\n", fc.LengthMi)
	}
	if fc.AWReachID != "" {
		fmt.Fprintf(&b, "American Whitewater reach ID: %s\n", fc.AWReachID)
	}

	// When a live searcher is available, fetch AW data and append it.
	if s.searcher != nil {
		query := fmt.Sprintf("site:americanwhitewater.org %s %s flow levels cfs", fc.ReachName, fc.ReachRegion)
		results, err := s.searcher.Search(ctx, query)
		if err == nil && len(results) > 0 {
			b.WriteString("\n--- Live search results from americanwhitewater.org ---\n")
			for _, r := range results {
				fmt.Fprintf(&b, "URL: %s\nTitle: %s\nSnippet: %s\n\n", r.URL, r.Title, r.Snippet)
			}
			b.WriteString("--- End search results ---\n")
			b.WriteString("\nUsing the search results above, produce accurate flow ranges and set source_url to the AW page URL you used.\n")
		}
	} else {
		b.WriteString("\nUsing your training knowledge (American Whitewater data, guidebooks, trip reports), produce accurate flow ranges for this reach. Set source_url to the AW reach URL if you know it.\n")
	}

	return b.String(), nil
}
