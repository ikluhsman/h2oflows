package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/jackc/pgx/v5/pgxpool"
)

const describerSystemPrompt = `You are a river trip log assistant for H2OFlows, a whitewater paddling app.

Given the details of a paddling trip, write a short log entry for the paddler.

Return ONLY a JSON object — no explanation, no markdown fences:

{
  "title": "A short title (5-10 words)",
  "description": "A 2-3 sentence trip summary"
}

Guidelines:
- Title: specific and evocative — include the reach name and something notable (flow level, conditions, a key feature)
- Description: write in first-person past tense, as if the paddler is recording it. Mention flow conditions, character of the run, and anything notable from the reach context. Don't invent facts not supported by the context.
- Use natural paddler language (cfs, put-in, take-out, lines, features, etc.)
- If duration or distance is given, weave it in naturally, not as a stat dump

Reach context (use this to add colour — don't contradict it):
---
%s`

// TripDescriber generates an AI-written title and description for a completed trip.
// It uses RAG chunks from the trip's reach to ground the output in real river knowledge.
type TripDescriber struct {
	db     *pgxpool.Pool
	claude anthropic.Client
}

func NewTripDescriber(pool *pgxpool.Pool, anthropicKey string) *TripDescriber {
	return &TripDescriber{
		db:     pool,
		claude: anthropic.NewClient(option.WithAPIKey(anthropicKey)),
	}
}

// DescribeResult is returned by Describe.
type DescribeResult struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// TripDetails is the input to Describe.
type TripDetails struct {
	ReachID     string
	ReachName   string
	StartCFS    *float64
	EndCFS      *float64
	DurationMin *int32
	DistanceMi  *float64
}

// Describe generates a title and description for a trip using reach RAG context.
func (d *TripDescriber) Describe(ctx context.Context, trip TripDetails) (*DescribeResult, error) {
	// Fetch the top-5 most general reach chunks as context.
	// We don't embed the trip itself; use a representative query instead.
	var chunks []string
	if trip.ReachID != "" {
		rows, err := d.db.Query(ctx, `
			SELECT content
			FROM reach_embeddings
			WHERE reach_id = $1
			ORDER BY id
			LIMIT 5
		`, trip.ReachID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var c string
				if rows.Scan(&c) == nil {
					chunks = append(chunks, c)
				}
			}
		}
	}

	reachContext := strings.Join(chunks, "\n\n---\n\n")
	if reachContext == "" {
		reachContext = fmt.Sprintf("Reach: %s (no additional context available)", trip.ReachName)
	}

	// Build the user message with trip facts.
	var facts []string
	facts = append(facts, fmt.Sprintf("Reach: %s", trip.ReachName))
	if trip.StartCFS != nil {
		facts = append(facts, fmt.Sprintf("Flow at put-in: %.0f cfs", *trip.StartCFS))
	}
	if trip.EndCFS != nil {
		facts = append(facts, fmt.Sprintf("Flow at take-out: %.0f cfs", *trip.EndCFS))
	}
	if trip.DurationMin != nil {
		h := *trip.DurationMin / 60
		m := *trip.DurationMin % 60
		if h > 0 {
			facts = append(facts, fmt.Sprintf("Duration: %dh %dm", h, m))
		} else {
			facts = append(facts, fmt.Sprintf("Duration: %dm", m))
		}
	}
	if trip.DistanceMi != nil {
		facts = append(facts, fmt.Sprintf("Distance: %.1f miles", *trip.DistanceMi))
	}

	userMsg := "Trip details:\n" + strings.Join(facts, "\n") + "\n\nWrite a trip log entry."

	systemText := fmt.Sprintf(describerSystemPrompt, reachContext)

	msg, err := d.claude.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5,
		MaxTokens: 300,
		System:    []anthropic.TextBlockParam{{Text: systemText}},
		Messages:  []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userMsg)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude: %w", err)
	}
	if len(msg.Content) == 0 {
		return nil, fmt.Errorf("claude: empty response")
	}

	raw := strings.TrimSpace(msg.Content[0].Text)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var result DescribeResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("parse response %q: %w", raw, err)
	}
	return &result, nil
}
