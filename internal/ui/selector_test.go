package ui

import "testing"

func TestSelectorState_MoveUp(t *testing.T) {
	entries := []FileEntry{
		{Path: "a.go", Label: "M", Selected: true},
		{Path: "b.go", Label: "A", Selected: true},
		{Path: "c.go", Label: "?", Selected: true},
	}
	s := newSelectorState(entries)

	// At top, moveUp is a no-op.
	s.moveUp()
	if s.cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", s.cursor)
	}

	s.moveDown()
	s.moveDown()
	if s.cursor != 2 {
		t.Errorf("cursor should be 2, got %d", s.cursor)
	}

	s.moveUp()
	if s.cursor != 1 {
		t.Errorf("cursor should be 1, got %d", s.cursor)
	}
}

func TestSelectorState_MoveDown(t *testing.T) {
	entries := []FileEntry{
		{Path: "a.go", Label: "M", Selected: true},
		{Path: "b.go", Label: "A", Selected: true},
	}
	s := newSelectorState(entries)

	s.moveDown()
	if s.cursor != 1 {
		t.Errorf("cursor should be 1, got %d", s.cursor)
	}

	// At bottom, moveDown is a no-op.
	s.moveDown()
	if s.cursor != 1 {
		t.Errorf("cursor should stay at 1, got %d", s.cursor)
	}
}

func TestSelectorState_Toggle(t *testing.T) {
	entries := []FileEntry{
		{Path: "a.go", Label: "M", Selected: true},
		{Path: "b.go", Label: "A", Selected: false},
	}
	s := newSelectorState(entries)

	s.toggle()
	if s.entries[0].Selected {
		t.Error("a.go should be deselected after toggle")
	}

	s.toggle()
	if !s.entries[0].Selected {
		t.Error("a.go should be selected after second toggle")
	}
}

func TestSelectorState_ToggleAtDifferentCursors(t *testing.T) {
	entries := []FileEntry{
		{Path: "a.go", Label: "M", Selected: false},
		{Path: "b.go", Label: "A", Selected: false},
	}
	s := newSelectorState(entries)

	s.toggle() // toggle a.go on
	s.moveDown()
	s.toggle() // toggle b.go on

	if !s.entries[0].Selected {
		t.Error("a.go should be selected")
	}
	if !s.entries[1].Selected {
		t.Error("b.go should be selected")
	}
}

func TestSelectorState_SelectAll(t *testing.T) {
	entries := []FileEntry{
		{Path: "a.go", Label: "M", Selected: false},
		{Path: "b.go", Label: "A", Selected: false},
		{Path: "c.go", Label: "D", Selected: true},
	}
	s := newSelectorState(entries)

	s.selectAll()
	for _, e := range s.entries {
		if !e.Selected {
			t.Errorf("%s should be selected", e.Path)
		}
	}
}

func TestSelectorState_SelectNone(t *testing.T) {
	entries := []FileEntry{
		{Path: "a.go", Label: "M", Selected: true},
		{Path: "b.go", Label: "A", Selected: true},
	}
	s := newSelectorState(entries)

	s.selectNone()
	for _, e := range s.entries {
		if e.Selected {
			t.Errorf("%s should not be selected", e.Path)
		}
	}
}

func TestSelectorState_SelectedPaths(t *testing.T) {
	entries := []FileEntry{
		{Path: "a.go", Label: "M", Selected: true},
		{Path: "b.go", Label: "A", Selected: false},
		{Path: "c.go", Label: "D", Selected: true},
	}
	s := newSelectorState(entries)

	paths := s.selectedPaths()
	if len(paths) != 2 {
		t.Fatalf("expected 2 selected paths, got %d", len(paths))
	}
	if paths[0] != "a.go" || paths[1] != "c.go" {
		t.Errorf("unexpected paths: %v", paths)
	}
}

func TestSelectorState_SelectedPathsEmpty(t *testing.T) {
	entries := []FileEntry{
		{Path: "a.go", Label: "M", Selected: false},
		{Path: "b.go", Label: "A", Selected: false},
	}
	s := newSelectorState(entries)

	paths := s.selectedPaths()
	if paths != nil {
		t.Errorf("expected nil for no selections, got %v", paths)
	}
}

func TestSelectorState_SelectedCount(t *testing.T) {
	entries := []FileEntry{
		{Path: "a.go", Label: "M", Selected: true},
		{Path: "b.go", Label: "A", Selected: false},
		{Path: "c.go", Label: "D", Selected: true},
	}
	s := newSelectorState(entries)

	if s.selectedCount() != 2 {
		t.Errorf("expected count 2, got %d", s.selectedCount())
	}
}

func TestSelectorState_SelectedCountZero(t *testing.T) {
	entries := []FileEntry{
		{Path: "a.go", Label: "M", Selected: false},
	}
	s := newSelectorState(entries)

	if s.selectedCount() != 0 {
		t.Errorf("expected count 0, got %d", s.selectedCount())
	}
}

func TestSelectorState_SingleEntry(t *testing.T) {
	entries := []FileEntry{
		{Path: "only.go", Label: "M", Selected: false},
	}
	s := newSelectorState(entries)

	// Navigation should be no-ops with a single entry.
	s.moveUp()
	if s.cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", s.cursor)
	}
	s.moveDown()
	if s.cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", s.cursor)
	}

	s.toggle()
	if !s.entries[0].Selected {
		t.Error("only.go should be selected after toggle")
	}
}
