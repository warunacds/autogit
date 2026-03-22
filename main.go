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

// selectAndStageFiles shows the interactive file selector, then unstages
// everything and stages only the selected files.
func selectAndStageFiles() error {
	files, err := git.GetChangedFiles()
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no changes detected")
	}

	// Build selector entries — all pre-selected
	entries := make([]ui.FileEntry, len(files))
	for i, f := range files {
		entries[i] = ui.FileEntry{
			Path:     f.Path,
			Label:    f.Status.StatusLabel(),
			Selected: true,
		}
	}

	selected, err := ui.RunSelector(entries)
	if err != nil {
		return err
	}

	// Unstage everything, then stage only what was selected
	if err := git.UnstageAll(); err != nil {
		return err
	}
	return git.StageFiles(selected)
}

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

	// If --all, show file selector for all changed files.
	// If no --all and nothing staged, show file selector as fallback.
	if *allFlag {
		if err := selectAndStageFiles(); err != nil {
			if errors.Is(err, ui.ErrUserQuit) {
				os.Exit(0)
			}
			fmt.Fprintf(os.Stderr, "[autogit] Error: %v\n", err)
			os.Exit(1)
		}
	}

	// Get the staged diff
	fmt.Println("[autogit] Analyzing changes...")
	diff, err := git.GetDiff(true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[autogit] Error: %v\n", err)
		os.Exit(1)
	}

	// If nothing staged and --all was not used, try showing the file selector
	if diff == "" && !*allFlag {
		files, err := git.GetChangedFiles()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[autogit] Error: %v\n", err)
			os.Exit(1)
		}
		if len(files) == 0 {
			fmt.Fprintln(os.Stderr, "[autogit] No changes detected.")
			os.Exit(1)
		}
		if err := selectAndStageFiles(); err != nil {
			if errors.Is(err, ui.ErrUserQuit) {
				os.Exit(0)
			}
			fmt.Fprintf(os.Stderr, "[autogit] Error: %v\n", err)
			os.Exit(1)
		}
		diff, err = git.GetDiff(true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[autogit] Error: %v\n", err)
			os.Exit(1)
		}
		if diff == "" {
			fmt.Fprintln(os.Stderr, "[autogit] No changes detected.")
			os.Exit(1)
		}
	}

	// Generate message
	fmt.Println("[autogit] Generating commit message...")
	message, err := p.GenerateMessage(diff)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[autogit] Error: %v\n", err)
		os.Exit(1)
	}

	commitAndMaybePush := func(msg string, push bool) error {
		if err := git.Commit(msg); err != nil {
			return err
		}
		fmt.Println("[autogit] Committed successfully!")
		if push {
			fmt.Println("[autogit] Pushing...")
			if err := git.Push(); err != nil {
				return err
			}
			fmt.Println("[autogit] Pushed successfully!")
		}
		return nil
	}

	// Interactive UI loop
	err = ui.Run(ui.RunOpts{
		InitialMessage: message,
		RegenerateFn: func() (string, error) {
			return p.GenerateMessage(diff)
		},
		EditFn: editor.Open,
		CommitFn: func(msg string) error {
			return commitAndMaybePush(msg, pushFlag)
		},
		CommitAndPushFn: func(msg string) error {
			return commitAndMaybePush(msg, true)
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
