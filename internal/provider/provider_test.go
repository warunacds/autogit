package provider_test

import (
	"os"
	"strings"
	"testing"

	"github.com/warunacds/autogit/internal/config"
	"github.com/warunacds/autogit/internal/provider"
)

func TestNew_Claude(t *testing.T) {
	os.Setenv("ANTHROPIC_API_KEY", "fake-key")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	cfg := config.DefaultConfig()
	p, err := provider.New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestNew_Claude_MissingKey(t *testing.T) {
	os.Unsetenv("ANTHROPIC_API_KEY")

	cfg := config.DefaultConfig()
	_, err := provider.New(cfg)
	if err == nil {
		t.Fatal("expected error for missing ANTHROPIC_API_KEY")
	}
}

func TestNew_OpenAI(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "fake-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	cfg := &config.Config{
		Provider: "openai",
		OpenAI:   config.OpenAIConfig{BaseURL: "https://api.openai.com/v1", Model: "gpt-4o"},
	}
	p, err := provider.New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestNew_OpenAI_LocalNoKey(t *testing.T) {
	os.Unsetenv("OPENAI_API_KEY")

	cfg := &config.Config{
		Provider: "openai",
		OpenAI:   config.OpenAIConfig{BaseURL: "http://localhost:11434/v1", Model: "llama3"},
	}
	p, err := provider.New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider for local URL without key")
	}
}

func TestNew_OpenAI_RemoteMissingKey(t *testing.T) {
	os.Unsetenv("OPENAI_API_KEY")

	cfg := &config.Config{
		Provider: "openai",
		OpenAI:   config.OpenAIConfig{BaseURL: "https://api.openai.com/v1", Model: "gpt-4o"},
	}
	_, err := provider.New(cfg)
	if err == nil {
		t.Fatal("expected error for missing OPENAI_API_KEY with remote URL")
	}
}

func TestNew_ClaudeCode(t *testing.T) {
	cfg := &config.Config{
		Provider:   "claudecode",
		ClaudeCode: config.ClaudeCodeConfig{Model: ""},
	}
	// This test only passes if `claude` is on PATH. If not, we expect
	// the "not found" error rather than an unknown-provider error.
	p, err := provider.New(cfg)
	if err != nil {
		if !strings.Contains(err.Error(), "claude CLI not found") {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestNew_UnknownProvider(t *testing.T) {
	cfg := &config.Config{Provider: "gemini"}
	_, err := provider.New(cfg)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}
