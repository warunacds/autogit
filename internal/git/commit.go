package git

import (
	"fmt"
	"os/exec"
)

// Commit runs git commit with the provided message.
// Returns an error if the commit fails (e.g. nothing staged, pre-commit hook failure).
func Commit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit failed:\n%s", string(out))
	}
	return nil
}
