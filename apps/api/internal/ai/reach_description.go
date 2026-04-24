package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// GenerateReachDescription asks Claude to write a 1-2 paragraph description
// for a whitewater reach using its training knowledge. If Claude has no
// reliable information about the reach it says so — the admin can then edit
// or replace the text before saving.
func GenerateReachDescription(ctx context.Context, apiKey, name, riverName, commonName string, classMin, classMax *float64) (string, error) {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	var classPart string
	if classMin != nil && classMax != nil {
		classPart = fmt.Sprintf("Class %.0f–%.0f", *classMin, *classMax)
	} else if classMin != nil {
		classPart = fmt.Sprintf("Class %.0f", *classMin)
	}

	displayName := name
	if commonName != "" {
		displayName = commonName
	}

	var sb strings.Builder
	sb.WriteString("Reach: ")
	sb.WriteString(displayName)
	if riverName != "" {
		sb.WriteString(" on the ")
		sb.WriteString(riverName)
	}
	if classPart != "" {
		sb.WriteString(" (")
		sb.WriteString(classPart)
		sb.WriteString(")")
	}

	prompt := fmt.Sprintf(`Write a 1-2 paragraph description of this whitewater paddling reach for an app used by river runners:

%s

Guidelines:
- Write in third person, present tense ("The run offers...", "Paddlers will find...")
- Describe character of the river, key rapids or features if known, typical flow season, and access if known
- Keep it factual — if you don't have reliable information about this specific reach, write one sentence noting that the description needs manual editing, then stop
- Do not invent rapid names, distances, or flow stats you're not confident about
- 100-200 words total`, sb.String())

	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5,
		MaxTokens: 400,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("claude: %w", err)
	}
	if len(msg.Content) == 0 {
		return "", fmt.Errorf("claude returned empty response")
	}
	return strings.TrimSpace(msg.Content[0].Text), nil
}
