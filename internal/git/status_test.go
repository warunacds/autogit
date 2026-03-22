package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/warunacds/autogit/internal/git"
)

func TestGetChangedFiles_UntrackedOnly(t *testing.T) {
	dir := initTestRepo(t)
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	os.WriteFile(filepath.Join(dir, "new.txt"), []byte("hello\n"), 0644)

	files, err := git.GetChangedFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Path != "new.txt" {
		t.Errorf("expected path 'new.txt', got %q", files[0].Path)
	}
	if files[0].Status != git.StatusUntracked {
		t.Errorf("expected StatusUntracked, got %d", files[0].Status)
	}
	if files[0].Staged {
		t.Error("expected Staged=false for untracked file")
	}
}

func TestGetChangedFiles_StagedModified(t *testing.T) {
	dir := initTestRepo(t)
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	// Create and commit a file, then modify and stage it
	filePath := filepath.Join(dir, "file.txt")
	os.WriteFile(filePath, []byte("original\n"), 0644)
	exec.Command("git", "-C", dir, "add", "file.txt").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "initial").Run()

	os.WriteFile(filePath, []byte("modified\n"), 0644)
	exec.Command("git", "-C", dir, "add", "file.txt").Run()

	files, err := git.GetChangedFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Path != "file.txt" {
		t.Errorf("expected path 'file.txt', got %q", files[0].Path)
	}
	if files[0].Status != git.StatusModified {
		t.Errorf("expected StatusModified, got %d", files[0].Status)
	}
	if !files[0].Staged {
		t.Error("expected Staged=true for staged modified file")
	}
}

func TestGetChangedFiles_MixedStates(t *testing.T) {
	dir := initTestRepo(t)
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	// Create and commit two files
	os.WriteFile(filepath.Join(dir, "committed.txt"), []byte("v1\n"), 0644)
	os.WriteFile(filepath.Join(dir, "unstaged.txt"), []byte("v1\n"), 0644)
	exec.Command("git", "-C", dir, "add", "committed.txt", "unstaged.txt").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "initial").Run()

	// Staged modification of committed.txt
	os.WriteFile(filepath.Join(dir, "committed.txt"), []byte("v2\n"), 0644)
	exec.Command("git", "-C", dir, "add", "committed.txt").Run()

	// Unstaged modification of unstaged.txt (modify without staging)
	os.WriteFile(filepath.Join(dir, "unstaged.txt"), []byte("v2\n"), 0644)

	// Untracked file
	os.WriteFile(filepath.Join(dir, "untracked.txt"), []byte("new\n"), 0644)

	files, err := git.GetChangedFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}

	// Staged files should come first (sorted by path), then unstaged (sorted by path)
	if files[0].Path != "committed.txt" || !files[0].Staged {
		t.Errorf("expected first file to be staged 'committed.txt', got %q staged=%v", files[0].Path, files[0].Staged)
	}
	if files[1].Path != "unstaged.txt" || files[1].Staged {
		t.Errorf("expected second file to be unstaged 'unstaged.txt', got %q staged=%v", files[1].Path, files[1].Staged)
	}
	if files[2].Path != "untracked.txt" || files[2].Staged {
		t.Errorf("expected third file to be untracked 'untracked.txt', got %q staged=%v", files[2].Path, files[2].Staged)
	}
}

func TestGetChangedFiles_NoChanges(t *testing.T) {
	dir := initTestRepo(t)
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	files, err := git.GetChangedFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if files != nil {
		t.Fatalf("expected nil for no changes, got %v", files)
	}
}

func TestGetChangedFiles_DeletedFile(t *testing.T) {
	dir := initTestRepo(t)
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	// Create, commit, then delete and stage the deletion
	filePath := filepath.Join(dir, "doomed.txt")
	os.WriteFile(filePath, []byte("goodbye\n"), 0644)
	exec.Command("git", "-C", dir, "add", "doomed.txt").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "add doomed").Run()

	os.Remove(filePath)
	exec.Command("git", "-C", dir, "add", "doomed.txt").Run()

	files, err := git.GetChangedFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Status != git.StatusDeleted {
		t.Errorf("expected StatusDeleted, got %d", files[0].Status)
	}
	if !files[0].Staged {
		t.Error("expected Staged=true for staged deletion")
	}
}

func TestStatusLabel(t *testing.T) {
	tests := []struct {
		status git.FileStatus
		want   string
	}{
		{git.StatusModified, "M"},
		{git.StatusAdded, "A"},
		{git.StatusDeleted, "D"},
		{git.StatusRenamed, "R"},
		{git.StatusUntracked, "?"},
		{git.FileStatus(99), "?"},
	}
	for _, tt := range tests {
		got := tt.status.StatusLabel()
		if got != tt.want {
			t.Errorf("StatusLabel(%d) = %q, want %q", tt.status, got, tt.want)
		}
	}
}
