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

Given a question from a paddler, identify which river reach they are asking about.
Return ONLY a JSON object with this schema — no explanation, no markdown:

{
  "slug": "the-reach-slug-or-empty-string",
  "question": "the question with any reach name removed or kept as-is"
}

The slug must match one of these known reaches:
%s

Rules:
- Match common nicknames: "the numbers" → arkansas-the-numbers, "browns" → arkansas-browns-canyon, "gore" → colorado-gore-canyon, etc.
- If no reach is identifiable, return slug as ""
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
	Slug     string `json:"slug"`
	Question string `json:"question"`
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

// Answer embeds the question, retrieves the top-8 relevant reach chunks,
// and returns Claude's answer grounded in that context.
func (a *ReachAsker) Answer(ctx context.Context, reachID, reachName, question string) (string, error) {
	// 1. Embed the question using the same model as the stored chunks.
	vecs, err := a.embedder.Embed(ctx, []string{question})
	if err != nil {
		return "", fmt.Errorf("embed question: %w", err)
	}
	if len(vecs) == 0 || vecs[0] == nil {
		return "", fmt.Errorf("embed question: empty response from Voyage")
	}
	queryVec := vecs[0]

	// 2. Retrieve the 8 most semantically similar chunks for this reach.
	rows, err := a.db.Query(ctx, `
		SELECT chunk_type, content
		FROM reach_embeddings
		WHERE reach_id = $1
		ORDER BY embedding <=> $2::vector
		LIMIT 8
	`, reachID, FormatVector(queryVec))
	if err != nil {
		return "", fmt.Errorf("retrieve chunks: %w", err)
	}
	defer rows.Close()

	var chunks []string
	for rows.Next() {
		var chunkType, content string
		if err := rows.Scan(&chunkType, &content); err != nil {
			return "", err
		}
		_ = chunkType // available for logging/debugging if needed
		chunks = append(chunks, content)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	if len(chunks) == 0 {
		return "I don't have enough data about this reach yet to answer that question.", nil
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
