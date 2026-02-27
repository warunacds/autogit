package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/waruna/autogit/internal/git"
)

func TestGetDiff_NotARepo(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	_, err := git.GetDiff(false)
	if err == nil {
		t.Fatal("expected error for non-git directory, got nil")
	}
}

func TestGetDiff_EmptyStaged(t *testing.T) {
	dir := initTestRepo(t)
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	diff, err := git.GetDiff(false) // staged only
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff != "" {
		t.Fatalf("expected empty diff, got: %q", diff)
	}
}

func TestGetDiff_WithStagedFile(t *testing.T) {
	dir := initTestRepo(t)
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello world\n"), 0644)
	exec.Command("git", "add", "hello.txt").Run()

	diff, err := git.GetDiff(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff == "" {
		t.Fatal("expected non-empty diff for staged file")
	}
}

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init", dir},
		{"git", "-C", dir, "config", "user.email", "test@test.com"},
		{"git", "-C", dir, "config", "user.name", "Test"},
	}
	for _, c := range cmds {
		if out, err := exec.Command(c[0], c[1:]...).CombinedOutput(); err != nil {
			t.Fatalf("setup cmd %v failed: %v\n%s", c, err, out)
		}
	}
	return dir
}
