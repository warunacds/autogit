package ui_test

import (
	"strings"
	"testing"

	"github.com/warunacds/autogit/internal/ui"
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
