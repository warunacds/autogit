package claudecode

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/warunacds/autogit/internal/provider/shared"
)

type ClaudeCode struct {
	model string
}

func New(model string) *ClaudeCode {
	return &ClaudeCode{model: model}
}

func (c *ClaudeCode) GenerateMessage(diff string) (string, error) {
	diff, err := shared.ValidateAndTruncateDiff(diff)
	if err != nil {
		return "", err
	}

	args := []string{"-p"}
	if c.model != "" {
		args = append(args, "--model", c.model)
	}

	prompt := shared.SystemPrompt + "\n\n" + diff

	cmd := exec.Command("claude", args...)
	cmd.Stdin = strings.NewReader(prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return "", fmt.Errorf("claude CLI failed: %s", errMsg)
		}
		return "", fmt.Errorf("claude CLI failed: %w", err)
	}

	result := strings.TrimSpace(stdout.String())
	if result == "" {
		return "", fmt.Errorf("claude CLI returned an empty response")
	}
	return result, nil
}

// Available reports whether the claude CLI is installed and on PATH.
func Available() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}
