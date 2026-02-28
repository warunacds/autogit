package claude_test

import (
	"testing"

	"github.com/warunacds/autogit/internal/claude"
)

func TestGenerateMessage_EmptyDiff(t *testing.T) {
	client := claude.NewClient("fake-key")
	_, err := client.GenerateMessage("")
	if err == nil {
		t.Fatal("expected error for empty diff, got nil")
	}
}
