package git

import (
	"fmt"
	"os/exec"
)

// Push runs git push to the default remote and branch.
// Returns an error with the command output if the push fails.
func Push() error {
	cmd := exec.Command("git", "push")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push failed:\n%s", string(out))
	}
	return nil
}
