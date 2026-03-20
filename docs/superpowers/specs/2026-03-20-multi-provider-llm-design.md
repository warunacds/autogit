# Multi-Provider LLM Support for autogit

**Date:** 2026-03-20
**Status:** Approved

## Summary

Replace autogit's hardcoded Anthropic/Claude integration with a provider interface that supports any LLM — Claude, ChatGPT, Gemini, Ollama, LM Studio, or any OpenAI-compatible endpoint. Users select their provider via `~/.autogit.yaml` config file (created by `autogit init`), with CLI flags for overrides.

## Design Decisions

- **Two provider implementations cover all targets:** Anthropic (native SDK) and OpenAI-compatible (raw HTTP). Since Ollama, LM Studio, vLLM, LocalAI, and even Gemini all expose OpenAI-compatible endpoints, this covers the full spectrum without adding SDKs.
- **Config file for persistence, env vars for secrets:** `~/.autogit.yaml` stores provider, model, and base URL. API keys stay in environment variables only — never written to the config file.
- **CLI flags override config:** `--provider` and `--model` flags override config file values for one-off runs.
- **`autogit init` for interactive setup:** Guided prompts to select provider, model, and base URL, then writes the config file.
- **Backward compatible:** If no config file exists and no flags are set, defaults to `claude` provider — existing users keep working without changes.

## Architecture

### Package Structure

```
internal/
  config/          # NEW — config file loading/saving
    config.go
    config_test.go
  provider/        # NEW — Provider interface + factory
    provider.go
    common.go      # shared: diff truncation, validation, system prompt
    anthropic/     # Anthropic SDK implementation
      anthropic.go
      anthropic_test.go
    openai/        # OpenAI-compatible HTTP implementation
      openai.go
      openai_test.go
  git/             # UNCHANGED
  editor/          # UNCHANGED
  ui/              # UNCHANGED
  claude/          # DELETED — logic moves to provider/anthropic/
```

### Provider Interface

```go
// internal/provider/provider.go
package provider

type Provider interface {
    GenerateMessage(diff string) (string, error)
}

func New(cfg *config.Config) (Provider, error)
```

The factory function `New` reads the provider name from config, validates the required env var, and returns the appropriate implementation.

### Shared Logic — `internal/provider/common.go`

Both providers share:
- **Diff truncation:** Diffs > 100KB are truncated with a stderr warning
- **Empty diff validation:** Returns error if diff is empty
- **System prompt:** The Conventional Commits prompt used by both providers

```go
const MaxDiffBytes = 100 * 1024
const MaxTokens = 1024  // max completion tokens, used by both providers

const SystemPrompt = `You are a git commit message generator. Output only the commit message, following Conventional Commits format (e.g. "feat: add login endpoint"). Use a short subject line (under 72 chars), then a blank line, then bullet points for details if needed. No preamble, no markdown code fences, no explanation.`

func ValidateAndTruncateDiff(diff string) (string, error)
```

### Anthropic Provider — `internal/provider/anthropic/`

Migrated from `internal/claude/client.go`. Uses the existing `anthropic-sdk-go` dependency. Accepts model name from config (default: `claude-opus-4-6`, matching the current codebase).

```go
type Anthropic struct {
    api   anthropic.Client
    model string
}

func New(apiKey string, model string) *Anthropic
func (a *Anthropic) GenerateMessage(diff string) (string, error)
```

### OpenAI Provider — `internal/provider/openai/`

Uses `net/http` directly — no SDK dependency. POSTs to `{base_url}/chat/completions` with the standard OpenAI request format. The HTTP client is configured with a 30-second timeout to avoid hanging on unreachable endpoints. The request body includes `max_tokens` from the shared `common.MaxTokens` constant.

```go
type OpenAI struct {
    apiKey  string
    baseURL string
    model   string
    client  *http.Client  // initialized with 30s timeout
}

func New(apiKey string, baseURL string, model string) *OpenAI
func (o *OpenAI) GenerateMessage(diff string) (string, error)
```

Works with:
- **ChatGPT:** base_url=`https://api.openai.com/v1`, model=`gpt-4o`
- **Ollama:** base_url=`http://localhost:11434/v1`, model=`llama3` (no API key needed)
- **LM Studio:** base_url=`http://localhost:1234/v1`, model=`local-model`
- **Gemini:** base_url=`https://generativelanguage.googleapis.com/v1beta/openai`, model=`gemini-2.0-flash`

For local models where no API key is required, the `OPENAI_API_KEY` env var can be omitted. The env var check is skipped when `base_url` points to a local address (hostname matches `localhost`, `127.0.0.1`, or `::1`, determined via `net/url` parsing).

## Config File

### Location

`~/.autogit.yaml`

### Format

```yaml
provider: claude
claude:
  model: claude-opus-4-6
openai:
  base_url: https://api.openai.com/v1
  model: gpt-4o
```

### Config Package — `internal/config/`

```go
type Config struct {
    Provider string        `yaml:"provider"`
    Claude   ClaudeConfig  `yaml:"claude"`
    OpenAI   OpenAIConfig  `yaml:"openai"`
}

type ClaudeConfig struct {
    Model string `yaml:"model"`
}

type OpenAIConfig struct {
    BaseURL string `yaml:"base_url"`
    Model   string `yaml:"model"`
}

func Load() (*Config, error)           // reads ~/.autogit.yaml, returns defaults if missing
func Save(cfg *Config) error           // writes ~/.autogit.yaml
func DefaultConfig() *Config           // returns sensible defaults

// ApplyOverrides applies CLI flag overrides to the config.
// If provider is non-empty, it sets Config.Provider.
// If model is non-empty, it sets the model on the NOW-ACTIVE provider
// (i.e., after applying the provider override). This ensures
// `--provider openai --model llama3` sets OpenAI.Model, not Claude.Model.
func (c *Config) ApplyOverrides(provider, model string)
```

**Defaults when no config file exists:**
- `provider: claude`
- `claude.model: claude-opus-4-6`
- `openai.base_url: https://api.openai.com/v1`
- `openai.model: gpt-4o`

### Validation

- `base_url` is validated during both `Load()` and `init`: must have an `http://` or `https://` scheme, and trailing slashes are stripped.
- `provider` must be `claude` or `openai` — unknown values produce a clear error.

### YAML Dependency

Requires `gopkg.in/yaml.v3` — the standard Go YAML library. Added to `go.mod`.

## `autogit init` Command

### Detection

Before `flag.Parse()`, check if `len(os.Args) > 1 && os.Args[1] == "init"`. If so, run the init flow and exit. `init` must be the very first argument — flags cannot precede it. `autogit --provider openai init` is not supported; use `autogit init` only. This is enforced by checking `os.Args[1]` directly before any flag parsing.

### Interactive Flow

```
$ autogit init

Select a provider:
  1) Claude (Anthropic)
  2) OpenAI-compatible (ChatGPT, Ollama, LM Studio, Gemini, etc.)
> 1

Model name [claude-opus-4-6]:
>

Config saved to ~/.autogit.yaml

Make sure ANTHROPIC_API_KEY is set in your environment:
  export ANTHROPIC_API_KEY=your-key-here
```

For OpenAI-compatible:
```
$ autogit init

Select a provider:
  1) Claude (Anthropic)
  2) OpenAI-compatible (ChatGPT, Ollama, LM Studio, Gemini, etc.)
> 2

Base URL [https://api.openai.com/v1]:
> http://localhost:11434/v1

Model name [gpt-4o]:
> llama3

Config saved to ~/.autogit.yaml

Make sure OPENAI_API_KEY is set in your environment:
  export OPENAI_API_KEY=your-key-here
(For local models like Ollama, you can skip this or set it to any value.)
```

**If `~/.autogit.yaml` already exists:** Show current values and ask "Overwrite? [y/N]". Bare Enter defaults to No (abort without changes).

### Implementation Location

`internal/initialize/` package (avoided `init` to prevent confusion with Go's `init()` function convention). Contains `Run() error` which handles the interactive flow.

## CLI Changes

### New Flags

| Flag | Type | Description |
|------|------|-------------|
| `--provider` | string | Override provider from config (e.g. `claude`, `openai`) |
| `--model` | string | Override model name for the active provider |

### Updated Usage

```
Usage: autogit [flags]
       autogit init

Generates a commit message from staged git changes using AI.

Flags:
  -all          Include unstaged changes in addition to staged changes
  -push, -p     Run git push after a successful commit
  -provider     Override AI provider (claude, openai)
  -model        Override model name
```

### Updated Main Flow

1. Check for `init` subcommand → run `initialize.Run()` if present
2. `flag.Parse()`
3. `config.Load()` → load `~/.autogit.yaml` (defaults if missing)
4. `cfg.ApplyOverrides(providerFlag, modelFlag)` → apply CLI flags
5. `provider.New(cfg)` → get the right provider (validates env var here)
6. Get diff → `git.GetDiff(stagedOnly)`
7. Generate message → `provider.GenerateMessage(diff)`
8. UI loop → same as today, but `RegenerateFn` calls provider instead of claude client
9. Commit (and optionally push) → same as today

## Error Handling

| Scenario | Error Message |
|----------|---------------|
| Invalid YAML in config | `[autogit] Error: failed to parse ~/.autogit.yaml: <yaml error>` |
| Unknown provider | `[autogit] Error: unknown provider 'X', supported: claude, openai` |
| Missing ANTHROPIC_API_KEY | `[autogit] Error: ANTHROPIC_API_KEY is not set.\n  Export it with: export ANTHROPIC_API_KEY=your-key-here` |
| Missing OPENAI_API_KEY | `[autogit] Error: OPENAI_API_KEY is not set.\n  Export it with: export OPENAI_API_KEY=your-key-here` |
| OpenAI endpoint unreachable | `[autogit] Error: API call failed: <http error>` |
| API returns error response | `[autogit] Error: API returned status <code>: <body>` |

For local models (Ollama, LM Studio), `OPENAI_API_KEY` is not required — if base_url hostname matches `localhost`, `127.0.0.1`, or `::1`, skip the env var check.

## Testing

| Package | Tests |
|---------|-------|
| `internal/config/` | Load from file, load defaults when file missing, Save roundtrip, ApplyOverrides, invalid YAML error |
| `internal/provider/` | Factory returns correct type for each provider, unknown provider error, missing env var error |
| `internal/provider/anthropic/` | Empty diff error, truncation warning (migrated from existing `claude` tests) |
| `internal/provider/openai/` | Empty diff error, request format validation (httptest server), error response handling, successful generation |
| `internal/provider/common.go` | ValidateAndTruncateDiff: empty, normal, oversized |
| `internal/initialize/` | Not unit tested (interactive I/O) — documented for manual testing |

## README Updates

- Update tool description: "using AI" instead of "using Claude AI"
- Add `autogit init` to the demo section
- New "Providers" section with setup for each: Claude, ChatGPT, Ollama, LM Studio, Gemini
- Updated Configuration table with new flags and env vars
- Updated "How it works" to mention provider selection
- Keep Updating section as-is

## Migration Path

No breaking changes. Existing users with `ANTHROPIC_API_KEY` set and no config file will keep working — the default provider is `claude`. The old `internal/claude/` package is deleted, but that's an internal change with no public API impact.

## New Dependencies

- `gopkg.in/yaml.v3` — YAML parsing for config file
