package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/warunacds/autogit/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg.Provider != "claude" {
		t.Fatalf("expected default provider 'claude', got %q", cfg.Provider)
	}
	if cfg.Claude.Model != "claude-opus-4-6" {
		t.Fatalf("expected default claude model 'claude-opus-4-6', got %q", cfg.Claude.Model)
	}
	if cfg.OpenAI.BaseURL != "https://api.openai.com/v1" {
		t.Fatalf("expected default openai base_url, got %q", cfg.OpenAI.BaseURL)
	}
	if cfg.OpenAI.Model != "gpt-4o" {
		t.Fatalf("expected default openai model 'gpt-4o', got %q", cfg.OpenAI.Model)
	}
}

func TestLoadReturnsDefaultsWhenFileMissing(t *testing.T) {
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", t.TempDir())

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Provider != "claude" {
		t.Fatalf("expected default provider, got %q", cfg.Provider)
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	cfg := &config.Config{
		Provider: "openai",
		Claude:   config.ClaudeConfig{Model: "claude-opus-4-6"},
		OpenAI:   config.OpenAIConfig{BaseURL: "http://localhost:11434/v1", Model: "llama3"},
	}

	if err := config.Save(cfg); err != nil {
		t.Fatalf("save error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, ".autogit.yaml")); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if loaded.Provider != "openai" {
		t.Fatalf("expected provider 'openai', got %q", loaded.Provider)
	}
	if loaded.OpenAI.Model != "llama3" {
		t.Fatalf("expected model 'llama3', got %q", loaded.OpenAI.Model)
	}
	if loaded.OpenAI.BaseURL != "http://localhost:11434/v1" {
		t.Fatalf("expected base_url 'http://localhost:11434/v1', got %q", loaded.OpenAI.BaseURL)
	}
}

func TestApplyOverrides_ProviderAndModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ApplyOverrides("openai", "llama3")

	if cfg.Provider != "openai" {
		t.Fatalf("expected provider 'openai', got %q", cfg.Provider)
	}
	if cfg.OpenAI.Model != "llama3" {
		t.Fatalf("expected openai model 'llama3', got %q", cfg.OpenAI.Model)
	}
	if cfg.Claude.Model != "claude-opus-4-6" {
		t.Fatalf("expected claude model unchanged, got %q", cfg.Claude.Model)
	}
}

func TestApplyOverrides_ModelOnlyAppliesToActiveProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ApplyOverrides("", "claude-haiku-4-5-20251001")

	if cfg.Claude.Model != "claude-haiku-4-5-20251001" {
		t.Fatalf("expected claude model overridden, got %q", cfg.Claude.Model)
	}
	if cfg.OpenAI.Model != "gpt-4o" {
		t.Fatalf("expected openai model unchanged, got %q", cfg.OpenAI.Model)
	}
}

func TestApplyOverrides_EmptyStringsNoOp(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ApplyOverrides("", "")
	if cfg.Provider != "claude" {
		t.Fatalf("expected provider unchanged, got %q", cfg.Provider)
	}
}

func TestLoadUnknownProvider(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	content := "provider: gemini\n"
	os.WriteFile(filepath.Join(tmpDir, ".autogit.yaml"), []byte(content), 0644)

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	os.WriteFile(filepath.Join(tmpDir, ".autogit.yaml"), []byte("{{invalid yaml"), 0644)

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadInvalidBaseURL(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	content := "provider: openai\nopenai:\n  base_url: no-scheme-url\n  model: gpt-4o\n"
	os.WriteFile(filepath.Join(tmpDir, ".autogit.yaml"), []byte(content), 0644)

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for base_url without scheme")
	}
}

func TestLoadStripsTrailingSlash(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	content := "provider: openai\nopenai:\n  base_url: https://api.openai.com/v1/\n  model: gpt-4o\n"
	os.WriteFile(filepath.Join(tmpDir, ".autogit.yaml"), []byte(content), 0644)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.OpenAI.BaseURL != "https://api.openai.com/v1" {
		t.Fatalf("expected trailing slash stripped, got %q", cfg.OpenAI.BaseURL)
	}
}
