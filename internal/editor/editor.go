package editor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Open writes message to a temp file, opens the user's $EDITOR, waits for
// the editor to close, then reads and returns the (possibly modified) content.
// Falls back to nano if $EDITOR is not set.
func Open(message string) (string, error) {
	f, err := os.CreateTemp("", "autogit-*.txt")
	if err != nil {
		return "", fmt.Errorf("could not create temp file: %w", err)
	}
	defer os.Remove(f.Name())

	if _, err := f.WriteString(message); err != nil {
		f.Close()
		return "", fmt.Errorf("could not write temp file: %w", err)
	}
	f.Close()

	editorCmd := os.Getenv("EDITOR")
	if editorCmd == "" {
		editorCmd = "nano"
	}

	cmd := exec.Command(editorCmd, f.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor exited with error: %w", err)
	}

	data, err := os.ReadFile(f.Name())
	if err != nil {
		return "", fmt.Errorf("could not read edited file: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}
