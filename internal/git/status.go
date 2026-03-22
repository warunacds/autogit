package git

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// FileStatus represents the type of change on a file.
type FileStatus int

const (
	StatusModified  FileStatus = iota
	StatusAdded
	StatusDeleted
	StatusRenamed
	StatusUntracked
)

// ChangedFile represents a single file with changes.
type ChangedFile struct {
	Path   string
	Status FileStatus
	Staged bool // true if the file has changes in the index
}

// StatusLabel returns a single-character label for display.
func (s FileStatus) StatusLabel() string {
	switch s {
	case StatusModified:
		return "M"
	case StatusAdded:
		return "A"
	case StatusDeleted:
		return "D"
	case StatusRenamed:
		return "R"
	case StatusUntracked:
		return "?"
	default:
		return "?"
	}
}

// GetChangedFiles returns all files with changes (staged, unstaged, and untracked).
func GetChangedFiles() ([]ChangedFile, error) {
	cmd := exec.Command("git", "status", "--porcelain=v1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git status failed:\n%s", string(out))
	}

	output := strings.TrimSpace(string(out))
	if output == "" {
		return nil, nil
	}

	var files []ChangedFile
	seen := make(map[string]bool)

	for _, line := range strings.Split(output, "\n") {
		if len(line) < 3 {
			continue
		}

		x := line[0] // index status
		y := line[1] // worktree status
		path := strings.TrimSpace(line[3:])

		// Handle renames: "R  old -> new"
		if idx := strings.Index(path, " -> "); idx != -1 {
			path = path[idx+4:]
		}

		if seen[path] {
			continue
		}
		seen[path] = true

		// Untracked
		if x == '?' && y == '?' {
			files = append(files, ChangedFile{
				Path:   path,
				Status: StatusUntracked,
				Staged: false,
			})
			continue
		}

		// Determine status and whether it's staged
		staged := x != ' ' && x != '?'
		status := parseStatus(x, y)

		files = append(files, ChangedFile{
			Path:   path,
			Status: status,
			Staged: staged,
		})
	}

	// Sort: staged first, then by path
	sort.Slice(files, func(i, j int) bool {
		if files[i].Staged != files[j].Staged {
			return files[i].Staged
		}
		return files[i].Path < files[j].Path
	})

	return files, nil
}

func parseStatus(x, y byte) FileStatus {
	// Prefer the index status if staged, otherwise use worktree status
	ch := y
	if x != ' ' && x != '?' {
		ch = x
	}

	switch ch {
	case 'M':
		return StatusModified
	case 'A':
		return StatusAdded
	case 'D':
		return StatusDeleted
	case 'R':
		return StatusRenamed
	default:
		return StatusModified
	}
}
