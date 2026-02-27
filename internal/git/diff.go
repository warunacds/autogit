package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetDiff returns the git diff as a string.
// If stagedOnly is true, returns only staged changes (git diff --cached).
// If stagedOnly is false, returns staged + unstaged changes; on a brand-new
// repository with no commits, it falls back to git diff --cached automatically.
func GetDiff(stagedOnly bool) (string, error) {
	check := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if out, err := check.CombinedOutput(); err != nil {
		return "", fmt.Errorf("not a git repository: %s", strings.TrimSpace(string(out)))
	}

	var args []string
	if stagedOnly {
		args = []string{"diff", "--cached"}
	} else {
		args = []string{"diff", "HEAD"}
	}

	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		// On a brand-new repo with no commits, diff HEAD fails.
		// Fall back to diff --cached so staged files are still visible.
		if !stagedOnly {
			cmd2 := exec.Command("git", "diff", "--cached")
			out2, err2 := cmd2.Output()
			if err2 != nil {
				return "", fmt.Errorf("git diff failed: %v", err2)
			}
			return string(out2), nil
		}
		return "", fmt.Errorf("git diff failed: %v", err)
	}

	return string(out), nil
}
