// Package claude wraps the Anthropic API to generate git commit messages.
package claude

import (
	"context"
	"errors"
	"fmt"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const (
	// maxDiffBytes is the upper bound for diff content sent to the API.
	// Diffs exceeding this limit are truncated to avoid excessive token usage.
	maxDiffBytes = 100 * 1024

	// model is the Anthropic model used for commit message generation.
	model = anthropic.ModelClaudeOpus4_6

	// maxTokens is the maximum number of tokens the model may produce.
	maxTokens = 1024

	// systemPrompt instructs the model to produce Conventional Commits output only.
	systemPrompt = `You are a git commit message generator. Output only the commit message, following Conventional Commits format (e.g. "feat: add login endpoint"). Use a short subject line (under 72 chars), then a blank line, then bullet points for details if needed. No preamble, no markdown code fences, no explanation.`
)

// Client wraps the Anthropic API client and exposes commit-message generation.
type Client struct {
	api anthropic.Client
}

// NewClient returns a Client configured with the given API key.
// It does not make any network calls.
func NewClient(apiKey string) *Client {
	return &Client{
		api: anthropic.NewClient(option.WithAPIKey(apiKey)),
	}
}

// GenerateMessage calls the Anthropic Messages API with the provided git diff
// and returns a Conventional Commits-formatted commit message.
//
// It returns an error when diff is empty. Diffs larger than 100 KB are
// truncated before being sent to the API; a notice is printed to stdout.
func (c *Client) GenerateMessage(diff string) (string, error) {
	if diff == "" {
		return "", errors.New("diff is empty: nothing to generate a commit message for")
	}

	if len(diff) > maxDiffBytes {
		fmt.Printf("warning: diff is %d bytes, truncating to %d bytes\n", len(diff), maxDiffBytes)
		diff = diff[:maxDiffBytes]
	}

	msg, err := c.api.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     model,
		MaxTokens: maxTokens,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(diff)),
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
