package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/warunacds/autogit/internal/git"
)

func TestCommit_Success(t *testing.T) {
	dir := initTestRepo(t)
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	// Stage a file
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content\n"), 0644)
	exec.Command("git", "-C", dir, "add", "file.txt").Run()

	err := git.Commit("feat: test commit message")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify the commit exists
	out, _ := exec.Command("git", "-C", dir, "log", "--oneline", "-1").Output()
	msg := string(out)
	if msg == "" {
		t.Fatal("expected a commit to exist")
	}
}

func TestCommit_NothingStaged(t *testing.T) {
	dir := initTestRepo(t)
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	err := git.Commit("feat: nothing to commit")
	if err == nil {
		t.Fatal("expected error when nothing is staged")
	}
}
