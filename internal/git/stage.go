package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// StageFiles runs git add on the given file paths.
func StageFiles(paths []string) error {
	if len(paths) == 0 {
		return fmt.Errorf("no files to stage")
	}
	args := append([]string{"add", "--"}, paths...)
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git add failed:\n%s", string(out))
	}
	return nil
}

// UnstageAll runs git reset HEAD to unstage all files.
// For repos with no commits, it uses git rm --cached instead.
func UnstageAll() error {
	// Check if there are any commits
	check := exec.Command("git", "rev-parse", "HEAD")
	if err := check.Run(); err != nil {
		// No commits yet — use git rm --cached
		cmd := exec.Command("git", "rm", "--cached", "-r", ".")
		out, err := cmd.CombinedOutput()
		if err != nil {
			// If nothing is staged, git rm will fail — that's ok
			if strings.Contains(string(out), "did not match") {
				return nil
			}
			return fmt.Errorf("git rm --cached failed:\n%s", string(out))
		}
		return nil
	}

	cmd := exec.Command("git", "reset", "HEAD")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git reset HEAD failed:\n%s", string(out))
	}
	return nil
}
