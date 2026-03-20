package anthropic_test

import (
	"testing"

	"github.com/warunacds/autogit/internal/provider/anthropic"
)

func TestGenerateMessage_EmptyDiff(t *testing.T) {
	client := anthropic.New("fake-key", "claude-opus-4-6")
	_, err := client.GenerateMessage("")
	if err == nil {
		t.Fatal("expected error for empty diff, got nil")
	}
}
