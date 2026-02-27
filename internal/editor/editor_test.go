package editor_test

import (
	"os"
	"path/filepath"
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
	// Create a fake `nano` script that exits 0 without modifying the file
	fakeNano := filepath.Join(t.TempDir(), "nano")
	script := "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(fakeNano, []byte(script), 0755); err != nil {
		t.Fatalf("failed to create fake nano: %v", err)
	}

	// Prepend fake nano's dir to PATH so it takes precedence
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", filepath.Dir(fakeNano)+":"+origPath)
	defer os.Setenv("PATH", origPath)

	// Unset EDITOR so the fallback to nano is triggered
	os.Unsetenv("EDITOR")
	defer os.Setenv("EDITOR", os.Getenv("EDITOR")) // restore (may be empty, that's fine)

	result, err := editor.Open("test message")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "test message" {
		t.Fatalf("expected %q, got %q", "test message", result)
	}
}
