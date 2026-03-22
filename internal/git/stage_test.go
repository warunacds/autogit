package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/warunacds/autogit/internal/git"
)

func TestStageFiles_SingleFile(t *testing.T) {
	dir := initTestRepo(t)
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("content\n"), 0644)

	err := git.StageFiles([]string{"a.txt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the file is staged
	out, _ := exec.Command("git", "-C", dir, "status", "--porcelain=v1").Output()
	line := strings.TrimSpace(string(out))
	if !strings.HasPrefix(line, "A") {
		t.Errorf("expected staged added file, got %q", line)
	}
}

func TestStageFiles_MultipleFiles(t *testing.T) {
	dir := initTestRepo(t)
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a\n"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b\n"), 0644)

	err := git.StageFiles([]string{"a.txt", "b.txt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify both files are staged
	out, _ := exec.Command("git", "-C", dir, "status", "--porcelain=v1").Output()
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 staged files, got %d: %q", len(lines), string(out))
	}
	for _, line := range lines {
		if !strings.HasPrefix(line, "A") {
			t.Errorf("expected staged added file, got %q", line)
		}
	}
}

func TestStageFiles_EmptyPaths(t *testing.T) {
	dir := initTestRepo(t)
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	err := git.StageFiles([]string{})
	if err == nil {
		t.Fatal("expected error for empty paths, got nil")
	}
}

func TestUnstageAll_WithStagedFiles(t *testing.T) {
	dir := initTestRepo(t)
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	// Create a commit first so that git reset HEAD works
	os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0644)
	exec.Command("git", "-C", dir, "add", "base.txt").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "initial").Run()

	// Stage a new file
	os.WriteFile(filepath.Join(dir, "staged.txt"), []byte("staged\n"), 0644)
	exec.Command("git", "-C", dir, "add", "staged.txt").Run()

	// Verify the file is staged before unstaging
	out, _ := exec.Command("git", "-C", dir, "diff", "--cached", "--name-only").Output()
	if !strings.Contains(string(out), "staged.txt") {
		t.Fatal("expected staged.txt to be staged before unstage")
	}

	err := git.UnstageAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify no files are staged
	out, _ = exec.Command("git", "-C", dir, "diff", "--cached", "--name-only").Output()
	if strings.TrimSpace(string(out)) != "" {
		t.Errorf("expected no staged files after UnstageAll, got %q", string(out))
	}
}

func TestUnstageAll_NoCommitsYet(t *testing.T) {
	dir := initTestRepo(t)
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	// Stage a file in a repo with no commits
	os.WriteFile(filepath.Join(dir, "first.txt"), []byte("first\n"), 0644)
	exec.Command("git", "-C", dir, "add", "first.txt").Run()

	err := git.UnstageAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the file is now untracked, not staged
	out, _ := exec.Command("git", "-C", dir, "status", "--porcelain=v1").Output()
	line := strings.TrimSpace(string(out))
	if !strings.HasPrefix(line, "??") {
		t.Errorf("expected untracked file after unstage in empty repo, got %q", line)
	}
}
