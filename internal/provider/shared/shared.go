package shared

import (
	"errors"
	"fmt"
	"os"
)

const MaxDiffBytes = 100 * 1024
const MaxTokens = 1024
const SystemPrompt = `You are a git commit message generator. Output only the commit message, following Conventional Commits format (e.g. "feat: add login endpoint"). Use a short subject line (under 72 chars), then a blank line, then bullet points for details if needed. No preamble, no markdown code fences, no explanation.`

func ValidateAndTruncateDiff(diff string) (string, error) {
	if diff == "" {
		return "", errors.New("diff is empty: nothing to generate a commit message for")
	}
	if len(diff) > MaxDiffBytes {
		fmt.Fprintf(os.Stderr, "[autogit] Warning: diff is %d bytes, truncating to %d bytes\n", len(diff), MaxDiffBytes)
		diff = diff[:MaxDiffBytes]
	}
	return diff, nil
}
