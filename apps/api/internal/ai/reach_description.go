package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// GenerateReachDescription asks Claude to write a 1-2 paragraph description
// for a whitewater reach. Claude is given the web_search tool so it can look
// up current information from paddling forums, American Whitewater, and
// guidebook sites rather than relying solely on training knowledge.
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

Use web search to find current, accurate information from American Whitewater (americanwhitewater.org), paddling forums, or guidebook sites. Search for the reach name and river name together.

Guidelines:
- Write in third person, present tense ("The run offers...", "Paddlers will find...")
- Describe character of the river, key rapids or features if known, typical flow season, and access notes if known
- If you found real information online, incorporate it naturally — do not mention that you searched
- If you genuinely can't find reliable information about this specific reach, write one sentence noting the description needs manual editing, then stop
- Do not invent rapid names, distances, or flow stats you're not confident about
- 100-250 words total`, sb.String())

	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5,
		MaxTokens: 600,
		Tools: []anthropic.ToolUnionParam{
			{OfWebSearchTool20250305: &anthropic.WebSearchTool20250305Param{
				MaxUses: anthropic.Int(3),
			}},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("claude: %w", err)
	}

	var parts []string
	for _, block := range msg.Content {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			parts = append(parts, strings.TrimSpace(block.Text))
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("claude returned empty response")
	}
	return strings.Join(parts, "\n\n"), nil
}
