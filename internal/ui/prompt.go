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
	RegenerateFn   func() (string, error)      // called when user picks 'r'
	EditFn         func(string) (string, error) // called when user picks 'e'
	CommitFn       func(string) error           // called when user picks 'a'
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
