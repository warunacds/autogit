package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/warunacds/autogit/internal/config"
	"github.com/warunacds/autogit/internal/editor"
	"github.com/warunacds/autogit/internal/git"
	"github.com/warunacds/autogit/internal/initialize"
	"github.com/warunacds/autogit/internal/provider"
	"github.com/warunacds/autogit/internal/ui"
)

func main() {
	// Check for init subcommand before flag parsing
	if len(os.Args) > 1 && os.Args[1] == "init" {
		if err := initialize.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "[autogit] Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	allFlag := flag.Bool("all", false, "Include unstaged changes in addition to staged changes")
	var pushFlag bool
	flag.BoolVar(&pushFlag, "push", false, "Run git push after a successful commit")
	flag.BoolVar(&pushFlag, "p", false, "Run git push after a successful commit (shorthand)")
	providerFlag := flag.String("provider", "", "Override AI provider (claude, openai)")
	modelFlag := flag.String("model", "", "Override model name")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: autogit [flags]\n")
		fmt.Fprintf(os.Stderr, "       autogit init\n\n")
		fmt.Fprintf(os.Stderr, "Generates a commit message from staged git changes using AI.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[autogit] Error: %v\n", err)
		os.Exit(1)
	}
	cfg.ApplyOverrides(*providerFlag, *modelFlag)

	// Create provider
	p, err := provider.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[autogit] Error: %v\n", err)
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
	fmt.Println("[autogit] Generating commit message...")
	message, err := p.GenerateMessage(diff)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[autogit] Error: %v\n", err)
		os.Exit(1)
	}

	// Interactive UI loop
	err = ui.Run(ui.RunOpts{
		InitialMessage: message,
		RegenerateFn: func() (string, error) {
			return p.GenerateMessage(diff)
		},
		EditFn: editor.Open,
		CommitFn: func(msg string) error {
			if err := git.Commit(msg); err != nil {
				return err
			}
			fmt.Println("[autogit] Committed successfully!")
			if pushFlag {
				fmt.Println("[autogit] Pushing...")
				if err := git.Push(); err != nil {
					return err
				}
				fmt.Println("[autogit] Pushed successfully!")
			}
			return nil
		},
	})

	if err != nil {
		if errors.Is(err, ui.ErrUserQuit) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "[autogit] Error: %v\n", err)
		os.Exit(1)
	}
}
