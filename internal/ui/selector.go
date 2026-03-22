package ui

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

// FileEntry is a display item for the file selector.
type FileEntry struct {
	Path     string
	Label    string // single-char status: "M", "A", "D", "R", "?"
	Selected bool
}

// selectorState holds the non-IO state of the selector.
// It is separated from terminal I/O so the logic is fully testable.
type selectorState struct {
	entries []FileEntry
	cursor  int
}

func newSelectorState(entries []FileEntry) *selectorState {
	return &selectorState{
		entries: entries,
		cursor:  0,
	}
}

func (s *selectorState) moveUp() {
	if s.cursor > 0 {
		s.cursor--
	}
}

func (s *selectorState) moveDown() {
	if s.cursor < len(s.entries)-1 {
		s.cursor++
	}
}

func (s *selectorState) toggle() {
	s.entries[s.cursor].Selected = !s.entries[s.cursor].Selected
}

func (s *selectorState) selectAll() {
	for i := range s.entries {
		s.entries[i].Selected = true
	}
}

func (s *selectorState) selectNone() {
	for i := range s.entries {
		s.entries[i].Selected = false
	}
}

func (s *selectorState) selectedPaths() []string {
	var paths []string
	for _, e := range s.entries {
		if e.Selected {
			paths = append(paths, e.Path)
		}
	}
	return paths
}

func (s *selectorState) selectedCount() int {
	count := 0
	for _, e := range s.entries {
		if e.Selected {
			count++
		}
	}
	return count
}

// render writes the selector UI to w using ANSI escape codes.
// On the first render it prints all lines; on subsequent renders it moves
// the cursor back up and overwrites the previous output.
func (s *selectorState) render(w *os.File, firstRender bool) {
	totalLines := len(s.entries) + 3 // header + entries + blank + help
	if !firstRender {
		fmt.Fprintf(w, "\033[%dA", totalLines)
	}

	fmt.Fprintf(w, "\033[K\033[1mSelect files to stage (%d/%d selected):\033[0m\n",
		s.selectedCount(), len(s.entries))

	for i, e := range s.entries {
		cursor := "  "
		if i == s.cursor {
			cursor = "\033[36m> \033[0m"
		}

		checkbox := "\033[90m[ ]\033[0m"
		if e.Selected {
			checkbox = "\033[32m[x]\033[0m"
		}

		pathStyle := ""
		pathReset := ""
		if !e.Selected {
			pathStyle = "\033[90m"
			pathReset = "\033[0m"
		}

		fmt.Fprintf(w, "\033[K%s%s  %s  %s%s%s\n", cursor, checkbox, e.Label, pathStyle, e.Path, pathReset)
	}

	fmt.Fprintf(w, "\033[K\n")
	fmt.Fprintf(w, "\033[K  \033[90m↑/↓ navigate  space toggle  a all  n none  enter confirm  q quit\033[0m\n")
}

// RunSelector displays an interactive file selector and returns the paths
// the user selected. It returns ErrUserQuit if the user cancels.
func RunSelector(entries []FileEntry) ([]string, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("no changed files")
	}

	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return nil, fmt.Errorf("not a terminal, cannot show file selector")
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, fmt.Errorf("failed to set raw mode: %w", err)
	}

	restore := func() {
		term.Restore(fd, oldState)
	}
	defer restore()

	// Handle SIGINT/SIGTERM so the terminal is always restored.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-sigCh:
			restore()
			fmt.Fprintln(os.Stderr, "\n[autogit] Aborted.")
			os.Exit(0)
		case <-done:
			return
		}
	}()

	state := newSelectorState(entries)
	state.render(os.Stderr, true)

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return nil, fmt.Errorf("failed to read input: %w", err)
		}

		if n == 1 {
			switch buf[0] {
			case ' ':
				state.toggle()
			case 'a':
				state.selectAll()
			case 'n':
				state.selectNone()
			case '\r', '\n':
				selected := state.selectedPaths()
				if len(selected) == 0 {
					// Don't quit — warn and let user fix their selection
					continue
				}
				fmt.Fprintf(os.Stderr, "\n")
				return selected, nil
			case 'q', 3: // q or Ctrl+C
				fmt.Fprintf(os.Stderr, "\n")
				return nil, ErrUserQuit
			}
		} else if n == 3 && buf[0] == 27 && buf[1] == '[' {
			switch buf[2] {
			case 'A': // up
				state.moveUp()
			case 'B': // down
				state.moveDown()
			}
		}

		state.render(os.Stderr, false)
	}
}
