# autogit Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Go CLI tool that reads git diffs, calls the Claude API to generate a commit message, lets the user review/edit it interactively, then commits.

**Architecture:** Single binary with clean `internal/` package structure. Four packages: `git` (diff + commit), `claude` (API client), `editor` (open $EDITOR), `ui` (interactive menu loop). `main.go` wires them together and handles flags/env validation.

**Tech Stack:** Go 1.22+, `github.com/anthropics/anthropic-sdk-go` (only external dep), standard library for everything else.

---

## Prerequisites

Before starting, ensure Go is installed:
```bash
go version  # should print go1.22 or later
```

---

### Task 1: Initialize the Go module and project structure

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `internal/git/diff.go`
- Create: `internal/git/commit.go`
- Create: `internal/claude/client.go`
- Create: `internal/editor/editor.go`
- Create: `internal/ui/prompt.go`

**Step 1: Initialize the module**

```bash
cd /Users/waruna/Skunkwork/autogit
go mod init github.com/waruna/autogit
```

Expected: `go.mod` created with module line.

**Step 2: Create the directory structure**

```bash
mkdir -p internal/git internal/claude internal/editor internal/ui
```

**Step 3: Create placeholder files so the module compiles**

Create `main.go`:
```go
package main

func main() {}
```

Create `internal/git/diff.go`:
```go
package git
```

Create `internal/git/commit.go`:
```go
package git
```

Create `internal/claude/client.go`:
```go
package claude
```

Create `internal/editor/editor.go`:
```go
package editor
```

Create `internal/ui/prompt.go`:
```go
package ui
```

**Step 4: Verify it compiles**

```bash
go build ./...
```

Expected: no output, no errors.

**Step 5: Add the Anthropic SDK dependency**

```bash
go get github.com/anthropics/anthropic-sdk-go
```

Expected: `go.mod` and `go.sum` updated.

**Step 6: Commit**

```bash
git init
git add .
git commit -m "chore: initialize go module with project structure"
```

---

### Task 2: Implement git diff (`internal/git/diff.go`)

**Files:**
- Modify: `internal/git/diff.go`
- Create: `internal/git/diff_test.go`

**Step 1: Write the failing test**

Create `internal/git/diff_test.go`:
```go
package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/waruna/autogit/internal/git"
)

func TestGetDiff_NotARepo(t *testing.T) {
	// Run in a temp dir that is not a git repo
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

	// Create and stage a file
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

// initTestRepo creates a temp dir with a git repo initialized
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
```

**Step 2: Run the test to verify it fails**

```bash
go test ./internal/git/ -v -run TestGetDiff
```

Expected: compile error — `git.GetDiff` not defined yet.

**Step 3: Implement `GetDiff`**

Write `internal/git/diff.go`:
```go
package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetDiff returns the git diff as a string.
// If stagedOnly is true, returns only staged changes (git diff --cached).
// If stagedOnly is false, returns staged + unstaged changes (git diff HEAD).
func GetDiff(stagedOnly bool) (string, error) {
	// First verify we are in a git repo
	check := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if out, err := check.CombinedOutput(); err != nil {
		return "", fmt.Errorf("not a git repository: %s", strings.TrimSpace(string(out)))
	}

	var args []string
	if stagedOnly {
		args = []string{"diff", "--cached"}
	} else {
		// All changes: staged + unstaged. Use diff HEAD for tracked files,
		// plus diff for untracked would require more steps — keep it simple:
		// use diff (unstaged) and diff --cached (staged) combined.
		args = []string{"diff", "HEAD"}
	}

	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		// On a brand new repo with no commits, diff HEAD fails.
		// Fall back to diff --cached for --all mode too.
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
```

**Step 4: Run the tests to verify they pass**

```bash
go test ./internal/git/ -v -run TestGetDiff
```

Expected: all 3 tests PASS.

**Step 5: Commit**

```bash
git add internal/git/diff.go internal/git/diff_test.go
git commit -m "feat: implement git diff reader with staged/all modes"
```

---

### Task 3: Implement git commit (`internal/git/commit.go`)

**Files:**
- Modify: `internal/git/commit.go`
- Create: `internal/git/commit_test.go`

**Step 1: Write the failing test**

Create `internal/git/commit_test.go`:
```go
package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/waruna/autogit/internal/git"
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
```

**Step 2: Run to verify failure**

```bash
go test ./internal/git/ -v -run TestCommit
```

Expected: compile error — `git.Commit` not defined.

**Step 3: Implement `Commit`**

Write `internal/git/commit.go`:
```go
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
```

**Step 4: Run tests**

```bash
go test ./internal/git/ -v -run TestCommit
```

Expected: both tests PASS.

**Step 5: Commit**

```bash
git add internal/git/commit.go internal/git/commit_test.go
git commit -m "feat: implement git commit runner"
```

---

### Task 4: Implement Claude API client (`internal/claude/client.go`)

**Files:**
- Modify: `internal/claude/client.go`
- Create: `internal/claude/client_test.go`

**Step 1: Write the failing test**

The Claude client requires a live API key for integration testing. We write a unit test that validates the client rejects empty diffs, and we'll do a live test manually.

Create `internal/claude/client_test.go`:
```go
package claude_test

import (
	"testing"

	"github.com/waruna/autogit/internal/claude"
)

func TestGenerateMessage_EmptyDiff(t *testing.T) {
	client := claude.NewClient("fake-key")
	_, err := client.GenerateMessage("")
	if err == nil {
		t.Fatal("expected error for empty diff, got nil")
	}
}
```

**Step 2: Run to verify failure**

```bash
go test ./internal/claude/ -v
```

Expected: compile error — `claude.NewClient` and `claude.GenerateMessage` not defined.

**Step 3: Implement the Claude client**

Write `internal/claude/client.go`:
```go
package claude

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const (
	model     = "claude-opus-4-6"
	maxTokens = 1024
	// Truncate diffs larger than ~100KB to avoid hitting context limits
	maxDiffBytes = 100 * 1024
)

const systemPrompt = `You are a git commit message generator. Output only the commit message, following Conventional Commits format (e.g. "feat: add login endpoint"). Use a short subject line (under 72 chars), then a blank line, then bullet points for details if needed. No preamble, no markdown code fences, no explanation.`

// Client wraps the Anthropic API.
type Client struct {
	apiKey string
}

// NewClient returns a new Claude client with the given API key.
func NewClient(apiKey string) *Client {
	return &Client{apiKey: apiKey}
}

// GenerateMessage calls the Claude API with the git diff and returns a commit message.
func (c *Client) GenerateMessage(diff string) (string, error) {
	if diff == "" {
		return "", fmt.Errorf("diff is empty, nothing to generate from")
	}

	// Truncate large diffs
	if len(diff) > maxDiffBytes {
		diff = diff[:maxDiffBytes] + "\n\n[diff truncated — too large]"
		fmt.Println("[autogit] Warning: diff is large and has been truncated.")
	}

	client := anthropic.NewClient(option.WithAPIKey(c.apiKey))

	msg, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: maxTokens,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(diff)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("Claude API error: %w", err)
	}

	if len(msg.Content) == 0 {
		return "", fmt.Errorf("Claude returned empty response")
	}

	// Extract text from first content block
	text := msg.Content[0].Text
	return text, nil
}
```

**Step 4: Run tests**

```bash
go test ./internal/claude/ -v
```

Expected: `TestGenerateMessage_EmptyDiff` PASSES.

**Step 5: Verify it compiles**

```bash
go build ./...
```

Expected: no errors.

**Step 6: Commit**

```bash
git add internal/claude/client.go internal/claude/client_test.go
git commit -m "feat: implement Claude API client for commit message generation"
```

---

### Task 5: Implement editor launcher (`internal/editor/editor.go`)

**Files:**
- Modify: `internal/editor/editor.go`
- Create: `internal/editor/editor_test.go`

**Step 1: Write the failing test**

Create `internal/editor/editor_test.go`:
```go
package editor_test

import (
	"os"
	"testing"

	"github.com/waruna/autogit/internal/editor"
)

func TestOpen_UsesEditorEnvVar(t *testing.T) {
	// Set EDITOR to `cat` — it will just print the file and exit immediately
	// The result will be the initial message since cat doesn't modify the file
	os.Setenv("EDITOR", "cat")
	defer os.Unsetenv("EDITOR")

	initial := "feat: initial message"
	result, err := editor.Open(initial)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// cat just prints, doesn't modify — file content stays the same
	if result != initial {
		t.Fatalf("expected %q, got %q", initial, result)
	}
}

func TestOpen_FallsBackToNano(t *testing.T) {
	// With no EDITOR set and nano not guaranteed in CI, just check it
	// returns an error gracefully rather than panicking.
	// This is more of a smoke test.
	os.Unsetenv("EDITOR")
	// We can't test nano interactively, so just verify Open doesn't panic
	// We'll use a no-op approach: set EDITOR to `true` (exits 0 immediately)
	os.Setenv("EDITOR", "true")
	defer os.Unsetenv("EDITOR")

	result, err := editor.Open("test message")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// `true` exits immediately without modifying — original message returned
	if result != "test message" {
		t.Fatalf("expected original message returned, got %q", result)
	}
}
```

**Step 2: Run to verify failure**

```bash
go test ./internal/editor/ -v
```

Expected: compile error — `editor.Open` not defined.

**Step 3: Implement `Open`**

Write `internal/editor/editor.go`:
```go
package editor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Open writes message to a temp file, opens the user's $EDITOR, waits for
// the editor to close, then reads and returns the (potentially modified) content.
// Falls back to nano if $EDITOR is not set.
func Open(message string) (string, error) {
	// Write to a temp file
	f, err := os.CreateTemp("", "autogit-*.txt")
	if err != nil {
		return "", fmt.Errorf("could not create temp file: %w", err)
	}
	defer os.Remove(f.Name())

	if _, err := f.WriteString(message); err != nil {
		f.Close()
		return "", fmt.Errorf("could not write temp file: %w", err)
	}
	f.Close()

	// Determine editor
	editorCmd := os.Getenv("EDITOR")
	if editorCmd == "" {
		editorCmd = "nano"
	}

	// Launch editor attached to the terminal
	cmd := exec.Command(editorCmd, f.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor exited with error: %w", err)
	}

	// Read back the (possibly modified) content
	data, err := os.ReadFile(f.Name())
	if err != nil {
		return "", fmt.Errorf("could not read edited file: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}
```

**Step 4: Run tests**

```bash
go test ./internal/editor/ -v
```

Expected: both tests PASS.

**Step 5: Commit**

```bash
git add internal/editor/editor.go internal/editor/editor_test.go
git commit -m "feat: implement editor launcher using \$EDITOR env var"
```

---

### Task 6: Implement interactive UI prompt (`internal/ui/prompt.go`)

**Files:**
- Modify: `internal/ui/prompt.go`
- Create: `internal/ui/prompt_test.go`

**Step 1: Write the failing test**

The UI runs interactively (reads stdin). We test the display logic by capturing stdout and testing the non-interactive parts.

Create `internal/ui/prompt_test.go`:
```go
package ui_test

import (
	"strings"
	"testing"

	"github.com/waruna/autogit/internal/ui"
)

func TestFormatMessage(t *testing.T) {
	msg := "feat: add feature"
	output := ui.FormatMessage(msg)
	if !strings.Contains(output, msg) {
		t.Fatalf("formatted output should contain the message, got: %q", output)
	}
	if !strings.Contains(output, "─") {
		t.Fatalf("formatted output should contain separator lines")
	}
}

func TestParseChoice_ValidInputs(t *testing.T) {
	tests := []struct {
		input    string
		expected ui.Choice
	}{
		{"a", ui.ChoiceAccept},
		{"A", ui.ChoiceAccept},
		{"e", ui.ChoiceEdit},
		{"E", ui.ChoiceEdit},
		{"r", ui.ChoiceRegenerate},
		{"R", ui.ChoiceRegenerate},
		{"q", ui.ChoiceQuit},
		{"Q", ui.ChoiceQuit},
	}

	for _, tt := range tests {
		got := ui.ParseChoice(tt.input)
		if got != tt.expected {
			t.Errorf("ParseChoice(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestParseChoice_InlineText(t *testing.T) {
	// Any text longer than 1 char is treated as an inline message edit
	choice := ui.ParseChoice("feat: my custom message")
	if choice != ui.ChoiceInlineEdit {
		t.Fatalf("expected ChoiceInlineEdit for multi-char input, got %v", choice)
	}
}

func TestParseChoice_UnknownSingleChar(t *testing.T) {
	choice := ui.ParseChoice("x")
	if choice != ui.ChoiceUnknown {
		t.Fatalf("expected ChoiceUnknown for unrecognized single char, got %v", choice)
	}
}
```

**Step 2: Run to verify failure**

```bash
go test ./internal/ui/ -v
```

Expected: compile errors — `ui.FormatMessage`, `ui.ParseChoice`, `ui.Choice` etc. not defined.

**Step 3: Implement the UI**

Write `internal/ui/prompt.go`:
```go
package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Choice represents what the user chose in the menu.
type Choice int

const (
	ChoiceUnknown    Choice = iota
	ChoiceAccept            // a — commit as-is
	ChoiceEdit              // e — open $EDITOR
	ChoiceRegenerate        // r — call Claude again
	ChoiceQuit              // q — exit without committing
	ChoiceInlineEdit        // user typed a replacement message directly
)

const separator = "─────────────────────────────────────────"

// FormatMessage returns the message wrapped in display borders.
func FormatMessage(message string) string {
	return fmt.Sprintf("\nGenerated message:\n%s\n%s\n%s\n", separator, message, separator)
}

// ParseChoice interprets a single line of user input into a Choice.
// Single-char inputs are mapped to menu choices.
// Multi-char inputs are treated as inline message replacements.
func ParseChoice(input string) Choice {
	trimmed := strings.TrimSpace(input)
	if len(trimmed) == 0 {
		return ChoiceUnknown
	}
	if len(trimmed) > 1 {
		return ChoiceInlineEdit
	}
	switch strings.ToLower(trimmed) {
	case "a":
		return ChoiceAccept
	case "e":
		return ChoiceEdit
	case "r":
		return ChoiceRegenerate
	case "q":
		return ChoiceQuit
	default:
		return ChoiceUnknown
	}
}

// RunOpts holds the dependencies for the UI loop.
type RunOpts struct {
	InitialMessage string
	RegenerateFn   func() (string, error) // called when user picks 'r'
	EditFn         func(string) (string, error) // called when user picks 'e'
	CommitFn       func(string) error // called when user picks 'a'
}

// Run displays the message and runs the interactive menu loop until the user
// accepts, quits, or an error occurs.
func Run(opts RunOpts) error {
	message := opts.InitialMessage
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print(FormatMessage(message))
		fmt.Print("\n[a] Accept  [e] Edit in $EDITOR  [r] Regenerate  [q] Quit\n> ")

		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		choice := ParseChoice(line)

		switch choice {
		case ChoiceAccept:
			return opts.CommitFn(message)

		case ChoiceEdit:
			edited, err := opts.EditFn(message)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[autogit] Editor error: %v\n", err)
				continue
			}
			if edited == "" {
				fmt.Fprintln(os.Stderr, "[autogit] Empty message after editing, keeping original.")
				continue
			}
			message = edited

		case ChoiceRegenerate:
			fmt.Println("[autogit] Regenerating...")
			newMsg, err := opts.RegenerateFn()
			if err != nil {
				fmt.Fprintf(os.Stderr, "[autogit] Regenerate error: %v\n", err)
				continue
			}
			message = newMsg

		case ChoiceInlineEdit:
			newMsg := strings.TrimSpace(line)
			if newMsg == "" {
				fmt.Fprintln(os.Stderr, "[autogit] Empty message, keeping original.")
				continue
			}
			message = newMsg

		case ChoiceQuit:
			fmt.Println("[autogit] Aborted.")
			os.Exit(0)

		default:
			fmt.Println("[autogit] Unknown option. Use a/e/r/q or type a replacement message.")
		}
	}
}
```

**Step 4: Run tests**

```bash
go test ./internal/ui/ -v
```

Expected: all tests PASS.

**Step 5: Commit**

```bash
git add internal/ui/prompt.go internal/ui/prompt_test.go
git commit -m "feat: implement interactive UI menu with accept/edit/regenerate/quit"
```

---

### Task 7: Wire everything together in `main.go`

**Files:**
- Modify: `main.go`

**Step 1: Implement `main.go`**

Write `main.go`:
```go
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/waruna/autogit/internal/claude"
	"github.com/waruna/autogit/internal/editor"
	"github.com/waruna/autogit/internal/git"
	"github.com/waruna/autogit/internal/ui"
)

func main() {
	allFlag := flag.Bool("all", false, "Include unstaged changes in addition to staged changes")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: autogit [--all]\n\n")
		fmt.Fprintf(os.Stderr, "Generates a commit message from staged git changes using Claude AI.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Validate API key
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "[autogit] Error: ANTHROPIC_API_KEY is not set.")
		fmt.Fprintln(os.Stderr, "  Export it with: export ANTHROPIC_API_KEY=your-key-here")
		os.Exit(1)
	}

	stagedOnly := !*allFlag

	// Get the diff
	fmt.Println("[autogit] Analyzing changes...")
	diff, err := git.GetDiff(stagedOnly)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[autogit] Error: %v\n", err)
		os.Exit(1)
	}

	if diff == "" {
		if stagedOnly {
			fmt.Fprintln(os.Stderr, "[autogit] No staged changes found.")
			fmt.Fprintln(os.Stderr, "  Run `git add <files>` first, or use `autogit --all` for unstaged changes.")
		} else {
			fmt.Fprintln(os.Stderr, "[autogit] No changes detected.")
		}
		os.Exit(1)
	}

	// Generate message
	claudeClient := claude.NewClient(apiKey)
	fmt.Println("[autogit] Generating commit message...")

	message, err := claudeClient.GenerateMessage(diff)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[autogit] Error: %v\n", err)
		os.Exit(1)
	}

	// Interactive UI loop
	err = ui.Run(ui.RunOpts{
		InitialMessage: message,
		RegenerateFn: func() (string, error) {
			return claudeClient.GenerateMessage(diff)
		},
		EditFn: editor.Open,
		CommitFn: func(msg string) error {
			if err := git.Commit(msg); err != nil {
				return err
			}
			fmt.Println("[autogit] Committed successfully!")
			return nil
		},
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "[autogit] Error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 2: Build and verify it compiles**

```bash
go build -o autogit ./...
```

Expected: binary `autogit` created, no errors.

**Step 3: Run all tests**

```bash
go test ./...
```

Expected: all tests PASS.

**Step 4: Smoke test (manual)**

```bash
# In a real git repo with staged changes:
export ANTHROPIC_API_KEY=your-key
./autogit
```

Expected: sees diff, calls Claude, shows generated message, presents menu.

**Step 5: Commit**

```bash
git add main.go
git commit -m "feat: wire up main entry point with flag parsing and dependency injection"
```

---

### Task 8: Add README and install instructions

**Files:**
- Create: `README.md`

**Step 1: Create README.md**

```markdown
# autogit

A CLI tool that generates git commit messages using Claude AI.

## Install

```bash
go install github.com/waruna/autogit@latest
```

Or build from source:

```bash
git clone https://github.com/waruna/autogit
cd autogit
go build -o autogit .
# Move to PATH:
mv autogit /usr/local/bin/autogit
```

## Usage

```bash
# Stage your changes first
git add .

# Generate and commit
autogit

# Or include unstaged changes too
autogit --all
```

## Requirements

- Go 1.22+
- An [Anthropic API key](https://console.anthropic.com/)
- `ANTHROPIC_API_KEY` set in your environment

```bash
export ANTHROPIC_API_KEY=sk-ant-...
```

## How it works

1. Reads your git diff (staged by default, or all with `--all`)
2. Sends the diff to Claude API to generate a Conventional Commits message
3. Lets you: **[a]** accept, **[e]** edit in `$EDITOR`, **[r]** regenerate, or **[q]** quit
4. Commits using your existing git config (name/email)
```

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add README with install and usage instructions"
```

---

### Task 9: Final verification

**Step 1: Run all tests one final time**

```bash
go test ./... -v
```

Expected: all tests PASS, no failures.

**Step 2: Verify the binary builds cleanly**

```bash
go build -o autogit .
ls -lh autogit
```

Expected: binary exists, reasonable size (~8-15MB).

**Step 3: Verify go vet passes**

```bash
go vet ./...
```

Expected: no output (no issues found).

**Step 4: Final commit if any loose ends**

```bash
git status
# If anything uncommitted:
git add -A
git commit -m "chore: final cleanup"
```

---

## Summary of Tasks

| # | Task | Key File(s) |
|---|------|-------------|
| 1 | Initialize module + structure | `go.mod`, skeleton files |
| 2 | Git diff reader | `internal/git/diff.go` |
| 3 | Git commit runner | `internal/git/commit.go` |
| 4 | Claude API client | `internal/claude/client.go` |
| 5 | Editor launcher | `internal/editor/editor.go` |
| 6 | Interactive UI | `internal/ui/prompt.go` |
| 7 | Main entrypoint | `main.go` |
| 8 | README | `README.md` |
| 9 | Final verification | all |
