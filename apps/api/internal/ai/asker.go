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
- Each line: slug | Official Name [aka Common Name] (Region) | rapids: ... | access: ...
- Match "aka" names and common nicknames: "browns canyon" → reach with aka "Browns Canyon"; "the numbers" → arkansas-the-numbers; "gore" → colorado-gore-canyon, etc.
- Match by rapid name or access point name (e.g. "pinball rapid" → the reach whose rapids include "Pinball").
- Reach names often reference put-in/take-out landmarks (e.g. "Fisherman's Bridge to Hecla Junction"); the common name (aka) is what paddlers call it.
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

// IdentifyReachContext carries the identifying information about a reach used
// for natural-language reach identification.
type IdentifyReachContext struct {
	Slug       string
	Name       string
	CommonName string
	Region     string
	Rapids     []string
	Access     []string
}

// IdentifyReach uses Claude to figure out which reach slug a question is about,
// given a list of known reaches with rapids and access point names.
func (a *ReachAsker) IdentifyReach(ctx context.Context, question string, reaches []IdentifyReachContext) (*IdentifyResult, error) {
	var sb strings.Builder
	for _, rc := range reaches {
		fmt.Fprintf(&sb, "%s | %s", rc.Slug, rc.Name)
		if rc.CommonName != "" && rc.CommonName != rc.Name {
			fmt.Fprintf(&sb, " aka %s", rc.CommonName)
		}
		if rc.Region != "" {
			fmt.Fprintf(&sb, " (%s)", rc.Region)
		}
		rapids := rc.Rapids
		if len(rapids) > 15 {
			rapids = rapids[:15]
		}
		if len(rapids) > 0 {
			fmt.Fprintf(&sb, " | rapids: %s", strings.Join(rapids, ", "))
		}
		if len(rc.Access) > 0 {
			fmt.Fprintf(&sb, " | access: %s", strings.Join(rc.Access, ", "))
		}
		sb.WriteByte('\n')
	}
	system := fmt.Sprintf(identifySystemPrompt, strings.TrimRight(sb.String(), "\n"))

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
// flow ranges) directly from the DB, supplements with embedding-based semantic
// chunks, and injects all community reports using long-context prompt stuffing
// (no retrieval). The reports block is prompt-cached per reach.
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

	// 3. Load community reports for long-context grounding.
	reachReports, _ := loadReachReports(ctx, a.db, reachID)

	// 4. Deterministic hazard short-circuit — prepended regardless of question.
	hazards := activeHazards(reachReports)
	hazardPreamble := buildHazardPreamble(hazards)

	// 5. Build system blocks. Reports block carries a prompt-cache breakpoint
	//    so repeat queries to the same reach (within 5 min) skip re-tokenisation.
	reachContext := strings.Join(chunks, "\n\n---\n\n")
	systemBlocks := []anthropic.TextBlockParam{
		{Text: fmt.Sprintf(askSystemPrompt, reachName, reachContext)},
	}
	if reportsBlock := buildReportsBlock(reachReports); reportsBlock != "" {
		systemBlocks = append(systemBlocks, anthropic.TextBlockParam{
			Text:         reportsBlock,
			CacheControl: anthropic.CacheControlEphemeralParam{},
		})
	}

	msg, err := a.claude.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5,
		MaxTokens: 512,
		System:    systemBlocks,
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

	return hazardPreamble + msg.Content[0].Text, nil
}
