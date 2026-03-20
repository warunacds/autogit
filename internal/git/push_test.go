package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/warunacds/autogit/internal/git"
)

func TestPush_Success(t *testing.T) {
	// Create a bare repo to act as a remote
	bareDir := t.TempDir()
	if out, err := exec.Command("git", "init", "--bare", bareDir).CombinedOutput(); err != nil {
		t.Fatalf("failed to init bare repo: %v\n%s", err, out)
	}

	// Create a working repo and add the bare repo as origin
	dir := initTestRepo(t)
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	if out, err := exec.Command("git", "-C", dir, "remote", "add", "origin", bareDir).CombinedOutput(); err != nil {
		t.Fatalf("failed to add remote: %v\n%s", err, out)
	}

	// Create a commit so there's something to push
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content\n"), 0644)
	exec.Command("git", "-C", dir, "add", "file.txt").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "initial commit").Run()

	// Set upstream so plain "git push" works
	branchOut, _ := exec.Command("git", "-C", dir, "branch", "--show-current").Output()
	branch := strings.TrimSpace(string(branchOut))
	if out, err := exec.Command("git", "-C", dir, "push", "--set-upstream", "origin", branch).CombinedOutput(); err != nil {
		t.Fatalf("failed to set upstream: %v\n%s", err, out)
	}

	// Make another commit so there's something new to push
	os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("more content\n"), 0644)
	exec.Command("git", "-C", dir, "add", "file2.txt").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "second commit").Run()

	err := git.Push()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify the push landed in the bare repo
	out, _ := exec.Command("git", "-C", bareDir, "log", "--oneline", "-1").Output()
	if string(out) == "" {
		t.Fatal("expected commit to exist in remote after push")
	}
}

func TestPush_NoRemote(t *testing.T) {
	dir := initTestRepo(t)
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	// Create a commit so the repo isn't empty, but don't add a remote
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content\n"), 0644)
	exec.Command("git", "-C", dir, "add", "file.txt").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "initial commit").Run()

	err := git.Push()
	if err == nil {
		t.Fatal("expected error when pushing with no remote configured")
	}
}
