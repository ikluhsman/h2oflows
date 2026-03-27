// Package ai provides Claude-backed enrichment for H2OFlow search and scoring.
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

// SearchEnrichment is the structured output of the search intent parser.
// It supplements — but never replaces — the caller's original query.
type SearchEnrichment struct {
	// Terms are additional search words to OR into the SQL text filter.
	// E.g. query "numbers" → ["arkansas", "nathrop", "the numbers"]
	Terms []string `json:"terms"`

	// HintIDs are external_ids the model is confident map to the query.
	// These rows are boosted to the top of results regardless of score.
	HintIDs []string `json:"hint_ids"`

	// Irrelevant is set when the query is clearly not about a boatable waterway
	// (e.g. "how do I reset my password") and the caller should skip DB search.
	Irrelevant bool `json:"irrelevant"`
}

// SearchEnricher enriches free-text gauge search queries with AI-derived terms
// and gauge hint IDs. It degrades gracefully — callers must handle a nil enricher
// and a non-nil error by falling back to plain text search.
type SearchEnricher struct {
	client anthropic.Client
}

// NewSearchEnricher returns an enricher backed by the Anthropic API.
// apiKey is typically from config (ANTHROPIC_API_KEY env var).
func NewSearchEnricher(apiKey string) *SearchEnricher {
	return &SearchEnricher{client: anthropic.NewClient(option.WithAPIKey(apiKey))}
}

const enrichSystemPrompt = `You are a search assistant for H2OFlow, a whitewater paddling data platform.

Your job: given a paddler's free-text search query, return a JSON object that helps the platform find the right river gauges.

The platform tracks moving-water gauges (rivers and creeks boatable by kayaks, rafts, canoes, SUPs). It does NOT track lakes, reservoirs, or major commercial shipping rivers.

Respond ONLY with a JSON object. No explanation. No markdown.

Schema:
{
  "terms": ["string"],    // additional search terms to OR into the SQL text filter
  "hint_ids": ["string"], // USGS site numbers or DWR ABBREVs you are confident match
  "irrelevant": false     // true only if the query has nothing to do with paddling/gauges
}

Rules:
- Resolve common whitewater nicknames: "the numbers" → Arkansas River near Nathrop (07091200), "gore" → Colorado River near Kremmling (09058000), "shoshone" → Colorado River near Dotsero (09070500), "lower ark" or "royal gorge" → Arkansas at Parkdale (07094500), "browns canyon" → Arkansas near Nathrop (07091200), etc.
- Include river name variations (abbreviations, common spellings, alternate names).
- hint_ids should only contain IDs you are highly confident about. An empty list is fine.
- terms should be lowercase, no duplicates, useful for ILIKE matching.`

// Enrich parses a raw search query and returns structured enrichment.
// Times out after 4 seconds to stay within a reasonable API response budget.
// Returns (nil, nil) when apiKey is empty — callers should handle this as "no enrichment".
func (e *SearchEnricher) Enrich(ctx context.Context, query string) (*SearchEnrichment, error) {
	if e == nil || strings.TrimSpace(query) == "" {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	msg, err := e.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5, // fast + cheap for this task
		MaxTokens: 256,
		System: []anthropic.TextBlockParam{
			{Text: enrichSystemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(fmt.Sprintf("Query: %s", query))),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude: %w", err)
	}

	if len(msg.Content) == 0 {
		return nil, fmt.Errorf("claude: empty response")
	}

	raw := msg.Content[0].Text
	var out SearchEnrichment
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("claude: parse response: %w", err)
	}
	return &out, nil
}
