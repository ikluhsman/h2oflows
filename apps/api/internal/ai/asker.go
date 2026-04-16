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

const askSystemPrompt = `You are a knowledgeable river guide assistant for H2OFlows, a whitewater paddling app.

You are answering a question specifically about %s.

Use only the context below — which comes from curated reach data — to answer. If the answer isn't in the context, say so clearly rather than guessing. Never invent rapid names, distances, or conditions that aren't mentioned.

Be concise and direct. Use river terminology naturally. Assume the reader is an experienced paddler.

Context:
---
%s`

const identifySystemPrompt = `You are a river assistant for H2OFlows, a whitewater paddling app.

Given a question from a paddler, identify which river reach(es) they are asking about.
Return ONLY a JSON object with this schema — no explanation, no markdown:

{
  "slugs": ["slug-1", "slug-2"],
  "question": "the question with any reach name removed or kept as-is"
}

The slugs must match known reaches from the list below. Return up to 3 slugs, sorted by relevance.
If the question clearly names a single reach, return only that one slug.
If the question is ambiguous or names a river that has multiple reaches (e.g. "Arkansas River"), return all matching slugs.
If no reach is identifiable, return an empty array.

Known reaches:
%s

Rules:
- Match common nicknames: "the numbers" → arkansas-the-numbers, "browns" → arkansas-browns-canyon, "gore" → colorado-gore-canyon, etc.
- Keep the question natural — don't rewrite it heavily`

// ReachAsker answers natural-language questions about a reach using RAG:
// 1. Embed the question via Voyage AI
// 2. Retrieve the top-K most similar chunks from reach_embeddings
// 3. Feed those chunks as context to Claude and return its answer
type ReachAsker struct {
	db       *pgxpool.Pool
	embedder *Embedder
	claude   anthropic.Client
}

func NewReachAsker(pool *pgxpool.Pool, voyageKey, anthropicKey string) *ReachAsker {
	return &ReachAsker{
		db:       pool,
		embedder: NewEmbedder(voyageKey),
		claude:   anthropic.NewClient(option.WithAPIKey(anthropicKey)),
	}
}

// IdentifyResult is returned by IdentifyReach.
type IdentifyResult struct {
	Slugs    []string `json:"slugs"`
	Question string   `json:"question"`
}

// IdentifyReach uses Claude to figure out which reach slug a question is about,
// given a list of known slugs. Returns an empty slug if no match.
func (a *ReachAsker) IdentifyReach(ctx context.Context, question string, slugs []string) (*IdentifyResult, error) {
	slugList := strings.Join(slugs, "\n")
	system := fmt.Sprintf(identifySystemPrompt, slugList)

	msg, err := a.claude.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5,
		MaxTokens: 256,
		System:    []anthropic.TextBlockParam{{Text: system}},
		Messages:  []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(question)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude identify: %w", err)
	}
	if len(msg.Content) == 0 {
		return nil, fmt.Errorf("claude identify: empty response")
	}
	raw := strings.TrimSpace(msg.Content[0].Text)
	// Strip markdown code fences Claude sometimes adds despite instructions
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var result IdentifyResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("claude identify: parse %q: %w", raw, err)
	}
	return &result, nil
}

// Answer loads the reach's full structured data (description, rapids, access,
// flow ranges) directly from the DB, then supplements with embedding-based
// semantic chunks. This ensures all reach data is available to the AI even
// if embeddings haven't been generated yet.
func (a *ReachAsker) Answer(ctx context.Context, reachID, reachName, question string) (string, error) {
	// 1. Load live structured data directly from DB — always available.
	r, err := loadEmbedReach(ctx, a.db, reachID)
	if err != nil {
		return "", fmt.Errorf("load reach data: %w", err)
	}
	liveChunks := buildEmbedChunks(r)

	var chunks []string
	for _, c := range liveChunks {
		chunks = append(chunks, c.text)
	}

	// 2. Supplement with embedding-based semantic search if available.
	vecs, embedErr := a.embedder.Embed(ctx, []string{question})
	if embedErr == nil && len(vecs) > 0 && vecs[0] != nil {
		rows, queryErr := a.db.Query(ctx, `
			SELECT chunk_type, content
			FROM reach_embeddings
			WHERE reach_id = $1
			ORDER BY embedding <=> $2::vector
			LIMIT 6
		`, reachID, FormatVector(vecs[0]))
		if queryErr == nil {
			defer rows.Close()
			seen := make(map[string]bool)
			for _, c := range chunks {
				seen[c] = true
			}
			for rows.Next() {
				var chunkType, content string
				if err := rows.Scan(&chunkType, &content); err == nil && !seen[content] {
					chunks = append(chunks, content)
				}
			}
		}
	}

	if len(chunks) == 0 {
		return "I don't have any data about this reach yet.", nil
	}

	// 3. Build the grounded prompt and call Claude.
	context := strings.Join(chunks, "\n\n---\n\n")
	systemText := fmt.Sprintf(askSystemPrompt, reachName, context)

	msg, err := a.claude.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5,
		MaxTokens: 512,
		System: []anthropic.TextBlockParam{
			{Text: systemText},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(question)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("claude: %w", err)
	}
	if len(msg.Content) == 0 {
		return "", fmt.Errorf("claude: empty response")
	}

	return msg.Content[0].Text, nil
}
