# Multi-Provider LLM Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace autogit's hardcoded Claude integration with a provider interface supporting Claude, ChatGPT, Ollama, LM Studio, and any OpenAI-compatible LLM.

**Architecture:** A `Provider` interface with two implementations (Anthropic SDK, OpenAI-compatible HTTP). Config loaded from `~/.autogit.yaml` with CLI flag overrides. Interactive `autogit init` for setup.

**Tech Stack:** Go 1.25.6, `gopkg.in/yaml.v3`, `anthropic-sdk-go`, `net/http` for OpenAI-compatible

**Spec:** `docs/superpowers/specs/2026-03-20-multi-provider-llm-design.md`

---

## File Map

| Action | Path | Responsibility |
|--------|------|----------------|
| Create | `internal/config/config.go` | Config struct, Load, Save, DefaultConfig, ApplyOverrides, validation |
| Create | `internal/config/config_test.go` | Tests for config loading, saving, overrides, validation |
| Create | `internal/provider/provider.go` | Provider interface + New factory function |
| Create | `internal/provider/common.go` | Shared constants (MaxDiffBytes, MaxTokens, SystemPrompt) + ValidateAndTruncateDiff |
| Create | `internal/provider/common_test.go` | Tests for ValidateAndTruncateDiff |
| Create | `internal/provider/anthropic/anthropic.go` | Anthropic provider (migrated from internal/claude/) |
| Create | `internal/provider/anthropic/anthropic_test.go` | Tests (migrated from internal/claude/) |
| Create | `internal/provider/openai/openai.go` | OpenAI-compatible HTTP provider |
| Create | `internal/provider/openai/openai_test.go` | Tests with httptest server |
| Create | `internal/initialize/initialize.go` | Interactive `autogit init` flow |
| Modify | `main.go` | New flags, init subcommand, config loading, provider factory |
| Modify | `README.md` | Multi-provider docs, init, provider setup guides |
| Delete | `internal/claude/client.go` | Logic moves to provider/anthropic |
| Delete | `internal/claude/client_test.go` | Tests move to provider/anthropic |

---

### Task 1: Add YAML dependency

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Add gopkg.in/yaml.v3**

```bash
cd /Users/waruna/Skunkwork/autogit && go get gopkg.in/yaml.v3
```

- [ ] **Step 2: Tidy modules**

```bash
go mod tidy
```

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add gopkg.in/yaml.v3 dependency for config file support"
```

---

### Task 2: Create config package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write tests for config package**

Create `internal/config/config_test.go`:

```go
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

	// Verify file was created
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
	// Claude model should be unchanged
	if cfg.Claude.Model != "claude-opus-4-6" {
		t.Fatalf("expected claude model unchanged, got %q", cfg.Claude.Model)
	}
}

func TestApplyOverrides_ModelOnlyAppliesToActiveProvider(t *testing.T) {
	cfg := config.DefaultConfig() // provider=claude
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/config/ -v
```

Expected: compilation error — package does not exist yet.

- [ ] **Step 3: Write config.go implementation**

Create `internal/config/config.go`:

```go
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const configFileName = ".autogit.yaml"

type Config struct {
	Provider string       `yaml:"provider"`
	Claude   ClaudeConfig `yaml:"claude"`
	OpenAI   OpenAIConfig `yaml:"openai"`
}

type ClaudeConfig struct {
	Model string `yaml:"model"`
}

type OpenAIConfig struct {
	BaseURL string `yaml:"base_url"`
	Model   string `yaml:"model"`
}

func DefaultConfig() *Config {
	return &Config{
		Provider: "claude",
		Claude:   ClaudeConfig{Model: "claude-opus-4-6"},
		OpenAI:   OpenAIConfig{BaseURL: "https://api.openai.com/v1", Model: "gpt-4o"},
	}
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, configFileName), nil
}

func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func Save(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

func (c *Config) ApplyOverrides(provider, model string) {
	if provider != "" {
		c.Provider = provider
	}
	if model != "" {
		switch c.Provider {
		case "claude":
			c.Claude.Model = model
		case "openai":
			c.OpenAI.Model = model
		}
	}
}

func (c *Config) validate() error {
	c.OpenAI.BaseURL = strings.TrimRight(c.OpenAI.BaseURL, "/")

	if c.OpenAI.BaseURL != "" {
		u, err := url.Parse(c.OpenAI.BaseURL)
		if err != nil {
			return fmt.Errorf("invalid base_url %q: %w", c.OpenAI.BaseURL, err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("base_url must start with http:// or https://, got %q", c.OpenAI.BaseURL)
		}
	}

	return nil
}

// ConfigExists reports whether ~/.autogit.yaml exists.
func ConfigExists() bool {
	path, err := configPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// Path returns the absolute path to the config file.
func Path() string {
	path, err := configPath()
	if err != nil {
		return "~/" + configFileName
	}
	return path
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/config/ -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: add config package for ~/.autogit.yaml loading and saving"
```

---

### Task 3: Create shared provider logic (common.go)

**Files:**
- Create: `internal/provider/common.go`
- Create: `internal/provider/common_test.go`

- [ ] **Step 1: Write tests for ValidateAndTruncateDiff**

Create `internal/provider/common_test.go`:

```go
package provider_test

import (
	"testing"

	"github.com/warunacds/autogit/internal/provider"
)

func TestValidateAndTruncateDiff_Empty(t *testing.T) {
	_, err := provider.ValidateAndTruncateDiff("")
	if err == nil {
		t.Fatal("expected error for empty diff")
	}
}

func TestValidateAndTruncateDiff_Normal(t *testing.T) {
	diff := "some diff content"
	result, err := provider.ValidateAndTruncateDiff(diff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != diff {
		t.Fatalf("expected unchanged diff, got %q", result)
	}
}

func TestValidateAndTruncateDiff_Oversized(t *testing.T) {
	// Create a diff larger than 100KB
	big := make([]byte, 200*1024)
	for i := range big {
		big[i] = 'x'
	}
	result, err := provider.ValidateAndTruncateDiff(string(big))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != provider.MaxDiffBytes {
		t.Fatalf("expected truncated to %d bytes, got %d", provider.MaxDiffBytes, len(result))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/provider/ -v
```

Expected: compilation error — package does not exist yet.

- [ ] **Step 3: Write common.go**

Create `internal/provider/common.go`:

```go
package provider

import (
	"errors"
	"fmt"
	"os"
)

const MaxDiffBytes = 100 * 1024

const MaxTokens = 1024

const SystemPrompt = `You are a git commit message generator. Output only the commit message, following Conventional Commits format (e.g. "feat: add login endpoint"). Use a short subject line (under 72 chars), then a blank line, then bullet points for details if needed. No preamble, no markdown code fences, no explanation.`

func ValidateAndTruncateDiff(diff string) (string, error) {
	if diff == "" {
		return "", errors.New("diff is empty: nothing to generate a commit message for")
	}
	if len(diff) > MaxDiffBytes {
		fmt.Fprintf(os.Stderr, "[autogit] Warning: diff is %d bytes, truncating to %d bytes\n", len(diff), MaxDiffBytes)
		diff = diff[:MaxDiffBytes]
	}
	return diff, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/provider/ -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/common.go internal/provider/common_test.go
git commit -m "feat: add shared provider constants and diff validation"
```

---

### Task 4: Create Provider interface and factory

**Files:**
- Create: `internal/provider/provider.go`

- [ ] **Step 1: Write provider.go**

Create `internal/provider/provider.go`:

```go
package provider

import (
	"fmt"
	"net/url"
	"os"

	"github.com/warunacds/autogit/internal/config"
	"github.com/warunacds/autogit/internal/provider/anthropic"
	"github.com/warunacds/autogit/internal/provider/openai"
)

type Provider interface {
	GenerateMessage(diff string) (string, error)
}

func New(cfg *config.Config) (Provider, error) {
	switch cfg.Provider {
	case "claude":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY is not set.\n  Export it with: export ANTHROPIC_API_KEY=your-key-here")
		}
		return anthropic.New(apiKey, cfg.Claude.Model), nil

	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" && !isLocalURL(cfg.OpenAI.BaseURL) {
			return nil, fmt.Errorf("OPENAI_API_KEY is not set.\n  Export it with: export OPENAI_API_KEY=your-key-here")
		}
		return openai.New(apiKey, cfg.OpenAI.BaseURL, cfg.OpenAI.Model), nil

	default:
		return nil, fmt.Errorf("unknown provider %q, supported: claude, openai", cfg.Provider)
	}
}

func isLocalURL(baseURL string) bool {
	u, err := url.Parse(baseURL)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
```

Note: This file will not compile until Task 5 and Task 6 create the anthropic and openai packages. That's expected — we'll test the factory after those are in place.

- [ ] **Step 2: Commit**

```bash
git add internal/provider/provider.go
git commit -m "feat: add Provider interface and factory function"
```

---

### Task 5: Create Anthropic provider (migrate from internal/claude/)

**Files:**
- Create: `internal/provider/anthropic/anthropic.go`
- Create: `internal/provider/anthropic/anthropic_test.go`

- [ ] **Step 1: Write test for Anthropic provider**

Create `internal/provider/anthropic/anthropic_test.go`:

```go
package anthropic_test

import (
	"testing"

	"github.com/warunacds/autogit/internal/provider/anthropic"
)

func TestGenerateMessage_EmptyDiff(t *testing.T) {
	client := anthropic.New("fake-key", "claude-opus-4-6")
	_, err := client.GenerateMessage("")
	if err == nil {
		t.Fatal("expected error for empty diff, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/provider/anthropic/ -v
```

Expected: compilation error — package does not exist yet.

- [ ] **Step 3: Write anthropic.go**

Create `internal/provider/anthropic/anthropic.go`:

```go
package anthropic

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/warunacds/autogit/internal/provider"
)

type Anthropic struct {
	api   sdk.Client
	model string
}

func New(apiKey string, model string) *Anthropic {
	return &Anthropic{
		api:   sdk.NewClient(option.WithAPIKey(apiKey)),
		model: model,
	}
}

func (a *Anthropic) GenerateMessage(diff string) (string, error) {
	diff, err := provider.ValidateAndTruncateDiff(diff)
	if err != nil {
		return "", err
	}

	msg, err := a.api.Messages.New(context.Background(), sdk.MessageNewParams{
		Model:     a.model,
		MaxTokens: provider.MaxTokens,
		System: []sdk.TextBlockParam{
			{Text: provider.SystemPrompt},
		},
		Messages: []sdk.MessageParam{
			sdk.NewUserMessage(sdk.NewTextBlock(diff)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("anthropic API call failed: %w", err)
	}

	if len(msg.Content) == 0 {
		return "", errors.New("anthropic API returned an empty response")
	}

	first := msg.Content[0]
	if first.Type != "text" {
		return "", fmt.Errorf("anthropic API returned unexpected content block type: %q", first.Type)
	}
	return first.Text, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/provider/anthropic/ -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/anthropic/
git commit -m "feat: add Anthropic provider (migrated from internal/claude)"
```

---

### Task 6: Create OpenAI-compatible provider

**Files:**
- Create: `internal/provider/openai/openai.go`
- Create: `internal/provider/openai/openai_test.go`

- [ ] **Step 1: Write tests for OpenAI provider**

Create `internal/provider/openai/openai_test.go`:

```go
package openai_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/warunacds/autogit/internal/provider/openai"
)

func TestGenerateMessage_EmptyDiff(t *testing.T) {
	client := openai.New("fake-key", "http://localhost:1234/v1", "gpt-4o")
	_, err := client.GenerateMessage("")
	if err == nil {
		t.Fatal("expected error for empty diff")
	}
}

func TestGenerateMessage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request format
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected /chat/completions, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Bearer test-key, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Verify request body structure
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["model"] != "test-model" {
			t.Errorf("expected model 'test-model', got %v", body["model"])
		}
		if body["max_tokens"] != float64(1024) {
			t.Errorf("expected max_tokens 1024, got %v", body["max_tokens"])
		}

		// Return valid response
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"content": "feat: add new feature",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := openai.New("test-key", server.URL, "test-model")
	msg, err := client.GenerateMessage("diff content here")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != "feat: add new feature" {
		t.Fatalf("expected 'feat: add new feature', got %q", msg)
	}
}

func TestGenerateMessage_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": {"message": "server error"}}`))
	}))
	defer server.Close()

	client := openai.New("test-key", server.URL, "test-model")
	_, err := client.GenerateMessage("diff content")
	if err == nil {
		t.Fatal("expected error for API error response")
	}
}

func TestGenerateMessage_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := openai.New("test-key", server.URL, "test-model")
	_, err := client.GenerateMessage("diff content")
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}

func TestGenerateMessage_NoAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should still work — no auth header required for local
		if r.Header.Get("Authorization") != "" {
			t.Errorf("expected no Authorization header, got %s", r.Header.Get("Authorization"))
		}
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{"content": "feat: local model"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := openai.New("", server.URL, "llama3")
	msg, err := client.GenerateMessage("diff content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != "feat: local model" {
		t.Fatalf("expected 'feat: local model', got %q", msg)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/provider/openai/ -v
```

Expected: compilation error — package does not exist yet.

- [ ] **Step 3: Write openai.go**

Create `internal/provider/openai/openai.go`:

```go
package openai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/warunacds/autogit/internal/provider"
)

type OpenAI struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

func New(apiKey string, baseURL string, model string) *OpenAI {
	return &OpenAI{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

type chatRequest struct {
	Model     string        `json:"model"`
	Messages  []chatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (o *OpenAI) GenerateMessage(diff string) (string, error) {
	diff, err := provider.ValidateAndTruncateDiff(diff)
	if err != nil {
		return "", err
	}

	reqBody := chatRequest{
		Model: o.model,
		Messages: []chatMessage{
			{Role: "system", Content: provider.SystemPrompt},
			{Role: "user", Content: diff},
		},
		MaxTokens: provider.MaxTokens,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := o.baseURL + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if o.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.apiKey)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", errors.New("API returned an empty response (no choices)")
	}

	return chatResp.Choices[0].Message.Content, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/provider/openai/ -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/openai/
git commit -m "feat: add OpenAI-compatible provider with httptest coverage"
```

---

### Task 7: Test the provider factory

**Files:**
- Create: `internal/provider/provider_test.go`

- [ ] **Step 1: Write factory tests**

Create `internal/provider/provider_test.go`:

```go
package provider_test

import (
	"os"
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

func TestNew_UnknownProvider(t *testing.T) {
	cfg := &config.Config{Provider: "gemini"}
	_, err := provider.New(cfg)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}
```

- [ ] **Step 2: Run all provider tests**

```bash
go test ./internal/provider/... -v
```

Expected: all tests PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/provider/provider_test.go
git commit -m "test: add factory tests for provider.New"
```

---

### Task 8: Create autogit init command

**Files:**
- Create: `internal/initialize/initialize.go`

- [ ] **Step 1: Write initialize.go**

Create `internal/initialize/initialize.go`:

```go
package initialize

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/warunacds/autogit/internal/config"
)

func Run() error {
	reader := bufio.NewReader(os.Stdin)

	// Check if config already exists
	if config.ConfigExists() {
		cfg, err := config.Load()
		if err == nil {
			fmt.Printf("Config already exists at %s:\n", config.Path())
			fmt.Printf("  provider: %s\n", cfg.Provider)
			if cfg.Provider == "claude" {
				fmt.Printf("  model: %s\n", cfg.Claude.Model)
			} else {
				fmt.Printf("  base_url: %s\n", cfg.OpenAI.BaseURL)
				fmt.Printf("  model: %s\n", cfg.OpenAI.Model)
			}
			fmt.Print("\nOverwrite? [y/N] ")
			line, _ := reader.ReadString('\n')
			if strings.TrimSpace(strings.ToLower(line)) != "y" {
				fmt.Println("Aborted.")
				return nil
			}
		}
	}

	// Select provider
	fmt.Println("\nSelect a provider:")
	fmt.Println("  1) Claude (Anthropic)")
	fmt.Println("  2) OpenAI-compatible (ChatGPT, Ollama, LM Studio, Gemini, etc.)")
	fmt.Print("> ")
	line, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	choice := strings.TrimSpace(line)
	cfg := config.DefaultConfig()

	switch choice {
	case "1":
		cfg.Provider = "claude"

		fmt.Printf("\nModel name [%s]: ", cfg.Claude.Model)
		line, _ = reader.ReadString('\n')
		if model := strings.TrimSpace(line); model != "" {
			cfg.Claude.Model = model
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("\nConfig saved to %s\n", config.Path())
		fmt.Println("\nMake sure ANTHROPIC_API_KEY is set in your environment:")
		fmt.Println("  export ANTHROPIC_API_KEY=your-key-here")

	case "2":
		cfg.Provider = "openai"

		fmt.Printf("\nBase URL [%s]: ", cfg.OpenAI.BaseURL)
		line, _ = reader.ReadString('\n')
		if baseURL := strings.TrimSpace(line); baseURL != "" {
			cfg.OpenAI.BaseURL = strings.TrimRight(baseURL, "/")
		}

		fmt.Printf("\nModel name [%s]: ", cfg.OpenAI.Model)
		line, _ = reader.ReadString('\n')
		if model := strings.TrimSpace(line); model != "" {
			cfg.OpenAI.Model = model
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("\nConfig saved to %s\n", config.Path())
		fmt.Println("\nMake sure OPENAI_API_KEY is set in your environment:")
		fmt.Println("  export OPENAI_API_KEY=your-key-here")
		fmt.Println("(For local models like Ollama, you can skip this or set it to any value.)")

	default:
		return fmt.Errorf("invalid choice %q, please enter 1 or 2", choice)
	}

	return nil
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/initialize/
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/initialize/
git commit -m "feat: add interactive autogit init command"
```

---

### Task 9: Update main.go to use new architecture

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Rewrite main.go**

Replace the full contents of `main.go` with:

```go
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
```

- [ ] **Step 2: Verify it compiles**

```bash
go build -o autogit .
```

Expected: successful build.

- [ ] **Step 3: Run all tests**

```bash
go test ./... -v
```

Expected: all tests PASS (note: internal/claude tests will still pass at this point since we haven't deleted it yet).

- [ ] **Step 4: Commit**

```bash
git add main.go
git commit -m "feat: update main.go to use provider interface and config system"
```

---

### Task 10: Delete old internal/claude/ package

**Files:**
- Delete: `internal/claude/client.go`
- Delete: `internal/claude/client_test.go`

- [ ] **Step 1: Delete the old package**

```bash
rm -r internal/claude/
```

- [ ] **Step 2: Run all tests to confirm nothing breaks**

```bash
go test ./... -v
```

Expected: all tests PASS — main.go no longer imports `internal/claude`.

- [ ] **Step 3: Run go vet**

```bash
go vet ./...
```

Expected: no issues.

- [ ] **Step 4: Commit**

```bash
git add -A internal/claude/
git commit -m "refactor: remove internal/claude package (replaced by provider/anthropic)"
```

---

### Task 11: Update README

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Rewrite README.md**

Replace the full contents of `README.md` with:

```markdown
# autogit

A CLI tool that generates git commit messages using AI.

Analyzes your staged git diff, calls an AI model to suggest a [Conventional Commits](https://www.conventionalcommits.org/) message, then lets you accept, edit, regenerate, or abort — all from the terminal. Supports Claude, ChatGPT, Ollama, LM Studio, Gemini, and any OpenAI-compatible endpoint.

## Demo

```
$ autogit init

Select a provider:
  1) Claude (Anthropic)
  2) OpenAI-compatible (ChatGPT, Ollama, LM Studio, Gemini, etc.)
> 1

Model name [claude-opus-4-6]:
>

Config saved to ~/.autogit.yaml
```

```
$ git add .
$ autogit

[autogit] Analyzing changes...
[autogit] Generating commit message...

Generated message:
─────────────────────────────────────────
feat: add user authentication with JWT tokens

- Implement login/logout endpoints
- Add JWT middleware for protected routes
- Store refresh tokens in Redis
─────────────────────────────────────────

[a] Accept  [e] Edit in $EDITOR  [r] Regenerate  [q] Quit
>
```

With `--push` to automatically push after committing:

```
$ git add .
$ autogit --push

[autogit] Analyzing changes...
[autogit] Generating commit message...
...
[autogit] Committed successfully!
[autogit] Pushing...
[autogit] Pushed successfully!
```

## Requirements

- Go 1.22+
- An API key for your chosen provider (not needed for local models)

## Setup

**1. Install autogit**

```bash
go install github.com/warunacds/autogit@latest
```

Or build from source:

```bash
git clone https://github.com/warunacds/autogit
cd autogit
go build -o autogit .
sudo mv autogit /usr/local/bin/autogit
```

**2. Run the setup wizard**

```bash
autogit init
```

This creates `~/.autogit.yaml` with your provider, model, and endpoint settings.

**3. Set your API key**

Add the appropriate key to your `~/.zshrc` or `~/.bashrc`:

```bash
# For Claude
export ANTHROPIC_API_KEY=sk-ant-...

# For ChatGPT / OpenAI
export OPENAI_API_KEY=sk-...
```

Then reload: `source ~/.zshrc`

For local models (Ollama, LM Studio), no API key is needed.

**4. Verify it works**

```bash
autogit --help
```

## Updating

If you installed with `go install`:

```bash
go install github.com/warunacds/autogit@latest
```

If you built from source:

```bash
cd autogit
git pull
go build -o autogit .
sudo mv autogit /usr/local/bin/autogit
```

## Usage

```bash
# Stage your changes
git add .

# Generate and commit
autogit

# Include unstaged changes too
autogit --all

# Commit and push in one step
autogit --push
autogit -p          # shorthand

# Override provider or model for one run
autogit --provider openai --model gpt-4o-mini
```

### Interactive options

| Key | Action |
|-----|--------|
| `a` | Accept the message and commit |
| `e` | Open `$EDITOR` to edit the message |
| `r` | Regenerate — call the AI again for a new suggestion |
| `q` | Quit without committing |
| *(type anything)* | Replace the message inline and loop back |

## Providers

### Claude (Anthropic)

```bash
autogit init  # select 1) Claude
export ANTHROPIC_API_KEY=sk-ant-...
```

Config (`~/.autogit.yaml`):
```yaml
provider: claude
claude:
  model: claude-opus-4-6
```

### ChatGPT (OpenAI)

```bash
autogit init  # select 2) OpenAI-compatible
# Use default base URL: https://api.openai.com/v1
export OPENAI_API_KEY=sk-...
```

Config:
```yaml
provider: openai
openai:
  base_url: https://api.openai.com/v1
  model: gpt-4o
```

### Ollama (local)

```bash
autogit init  # select 2) OpenAI-compatible
# Set base URL to: http://localhost:11434/v1
# No API key needed
```

Config:
```yaml
provider: openai
openai:
  base_url: http://localhost:11434/v1
  model: llama3
```

### LM Studio (local)

```bash
autogit init  # select 2) OpenAI-compatible
# Set base URL to: http://localhost:1234/v1
# No API key needed
```

Config:
```yaml
provider: openai
openai:
  base_url: http://localhost:1234/v1
  model: local-model
```

### Gemini (Google)

```bash
autogit init  # select 2) OpenAI-compatible
# Set base URL to: https://generativelanguage.googleapis.com/v1beta/openai
export OPENAI_API_KEY=your-gemini-api-key
```

Config:
```yaml
provider: openai
openai:
  base_url: https://generativelanguage.googleapis.com/v1beta/openai
  model: gemini-2.0-flash
```

## How it works

1. Loads provider config from `~/.autogit.yaml` (with CLI flag overrides)
2. Reads your git diff (`git diff --cached` by default, or `git diff HEAD` with `--all`)
3. Sends the diff to the configured AI provider with a Conventional Commits prompt
4. Shows the generated message with an interactive menu
5. Commits via `git commit -m` using your existing git config (name/email)
6. Optionally pushes to the remote with `--push` / `-p`

Diffs larger than 100 KB are automatically truncated before sending to the API.

## Configuration

| Setting | How to set |
|---------|-----------|
| Provider | `autogit init` or `--provider` flag |
| Model | `autogit init` or `--model` flag |
| Claude API key | `ANTHROPIC_API_KEY` environment variable |
| OpenAI API key | `OPENAI_API_KEY` environment variable (not needed for local models) |
| Editor | `EDITOR` environment variable (falls back to `nano`) |
| Diff scope | `--all` flag (default: staged only) |
| Push after commit | `--push` / `-p` flag (default: off) |

## License

MIT
```

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: rewrite README for multi-provider support"
```

---

### Task 12: Final verification

- [ ] **Step 1: Run all tests**

```bash
go test ./... -v
```

Expected: all tests PASS.

- [ ] **Step 2: Run go vet**

```bash
go vet ./...
```

Expected: no issues.

- [ ] **Step 3: Build the binary**

```bash
go build -o autogit .
```

Expected: successful build.

- [ ] **Step 4: Verify help output**

```bash
./autogit --help
```

Expected: shows updated usage with `autogit init`, `--provider`, `--model` flags.

- [ ] **Step 5: Clean up build artifact**

```bash
rm autogit
```

- [ ] **Step 6: Push all commits**

```bash
git push
```
