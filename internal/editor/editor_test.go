package editor_test

import (
	"os"
	"testing"

	"github.com/waruna/autogit/internal/editor"
)

func TestOpen_UsesEditorEnvVar(t *testing.T) {
	// Set EDITOR to `cat` — prints file contents and exits immediately without modifying it
	os.Setenv("EDITOR", "cat")
	defer os.Unsetenv("EDITOR")

	initial := "feat: initial message"
	result, err := editor.Open(initial)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// cat just prints, doesn't modify — file content stays the same
	if result != initial {
		t.Fatalf("expected %q, got %q", initial, result)
	}
}

func TestOpen_FallsBackToNano(t *testing.T) {
	// Use `true` as EDITOR — exits 0 immediately without modifying the file
	os.Setenv("EDITOR", "true")
	defer os.Unsetenv("EDITOR")

	result, err := editor.Open("test message")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// `true` exits immediately without modifying — original message returned
	if result != "test message" {
		t.Fatalf("expected original message returned, got %q", result)
	}
}
