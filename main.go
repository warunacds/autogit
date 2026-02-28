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
