package anthropic

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/warunacds/autogit/internal/provider/shared"
)

type Anthropic struct {
	api   sdk.Client
	model string
}

func New(apiKey string, model string) *Anthropic {
	return &Anthropic{
		api:   sdk.NewClient(option.WithAPIKey(apiKey)),
		model: model,
	}
}

func (a *Anthropic) GenerateMessage(diff string) (string, error) {
	diff, err := shared.ValidateAndTruncateDiff(diff)
	if err != nil {
		return "", err
	}

	msg, err := a.api.Messages.New(context.Background(), sdk.MessageNewParams{
		Model:     sdk.Model(a.model),
		MaxTokens: shared.MaxTokens,
		System: []sdk.TextBlockParam{
			{Text: shared.SystemPrompt},
		},
		Messages: []sdk.MessageParam{
			sdk.NewUserMessage(sdk.NewTextBlock(diff)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("anthropic API call failed: %w", err)
	}

	if len(msg.Content) == 0 {
		return "", errors.New("anthropic API returned an empty response")
	}

	first := msg.Content[0]
	if first.Type != "text" {
		return "", fmt.Errorf("anthropic API returned unexpected content block type: %q", first.Type)
	}
	return first.Text, nil
}
