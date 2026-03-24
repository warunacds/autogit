# ccstatuswidgets — Design Spec

## Overview

**ccstatuswidgets** is an open-source Go binary that provides a customizable, plugin-ready status line for Claude Code CLI. It replaces ad-hoc shell scripts with a structured, extensible system where widgets are the unit of composition.

**Repository:** `github.com/warunacds/ccstatuswidgets`
**License:** MIT
**CLI binary:** `ccstatuswidgets` (aliased as `ccw`)

## Goals

1. Fast — render in <10ms for built-in widgets
2. Customizable — JSON config controls layout and widget settings
3. Extensible — plugin protocol for community widgets (Python SDK in phase 2)
4. Easy to install — Homebrew + curl script, `ccw init` wires everything up
5. Community-friendly — MIT license, clear contribution path

## Non-Goals (Phase 1)

- TUI config editor
- Powerline/theme support
- Windows support
- Python plugin SDK (phase 2)
- Community plugin registry (phase 2)

---

## Error Handling

### Widget Errors

When a widget's `Render()` returns an error:
- The engine silently omits the widget from the output (no error placeholder)
- If a cached result exists, the cached result is used instead
- Errors are logged to stderr (visible in debug mode via `CCW_DEBUG=1`)
- No distinction between transient and permanent errors at the engine level — widgets handle their own retry logic

### Stdin Handling

- The binary reads stdin with a 1-second timeout. If no data arrives, it exits with an empty output (Claude Code will simply show nothing)
- Malformed JSON: exit with empty output, log error to stderr
- Missing fields (e.g., `rate_limits` is null): widgets must handle nil/zero values gracefully. The `StatusLineInput` struct uses pointer types for optional sections (`*RateLimits`, etc.)

### Widget Fallbacks

- `git-branch`: omits itself when not in a git repository
- `effort`: reads from `~/.claude/settings.json`, then `~/.claude/settings.local.json`. Omits itself if neither exists or field is absent
- `memory`: uses `os.Getppid()` to get parent process. If the process cannot be read, omits itself
- `cost`: detects Max/Pro plan by checking if `RateLimits` is non-nil in the stdin JSON. If present, prefixes cost with "api eq."

---

## Architecture

### Approach

Go binary with concurrent widget execution. All widgets run in parallel with a 500ms timeout. Timed-out widgets fall back to cached results. External plugins (phase 2) communicate via JSON over stdin/stdout — same contract as built-in widgets.

### Project Structure

```
ccstatuswidgets/
├── cmd/
│   └── ccw/
│       └── main.go                 # entry point, CLI routing
├── internal/
│   ├── config/
│   │   ├── config.go               # load/save/validate config.json
│   │   └── defaults.go             # default config with all built-in widgets
│   ├── engine/
│   │   └── engine.go               # concurrent widget executor with timeouts
│   ├── protocol/
│   │   └── protocol.go             # StatusLineInput, WidgetOutput structs
│   ├── widget/
│   │   ├── widget.go               # Widget interface definition
│   │   └── registry.go             # widget registry (name -> factory)
│   ├── widgets/
│   │   ├── model.go                # model name widget
│   │   ├── effort.go               # effort level widget
│   │   ├── directory.go            # current directory widget
│   │   ├── gitbranch.go            # git branch widget
│   │   ├── context.go              # context bar + percentage
│   │   ├── usage.go                # 5h/7d usage bars + pace tracking
│   │   ├── lines.go                # lines added/removed
│   │   ├── cost.go                 # session cost (api eq. detection)
│   │   └── memory.go               # process memory
│   ├── plugin/
│   │   └── runner.go               # discovers + executes external plugins
│   ├── cache/
│   │   └── cache.go                # file-based TTL cache
│   ├── renderer/
│   │   └── renderer.go             # assembles lines, ANSI colors
│   └── cli/
│       ├── init.go                 # ccw init
│       ├── doctor.go               # ccw doctor
│       └── preview.go              # ccw preview
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

### Data Directory

```
~/.ccstatuswidgets/
├── config.json          # user configuration
├── plugins/             # community/custom plugins (phase 2)
└── cache/               # per-widget cache files (populated as needed)
```

---

## Widget Interface

```go
// Widget is the interface all widgets must implement
type Widget interface {
    // Name returns the widget identifier used in config
    Name() string

    // Render produces the widget output given statusline input and widget-specific config.
    // Built-in widgets define typed config structs and unmarshal from cfg internally.
    // External plugins receive cfg as JSON. The map[string]interface{} type is intentional
    // to support arbitrary plugin configs while keeping the interface uniform.
    Render(input *protocol.StatusLineInput, cfg map[string]interface{}) (*protocol.WidgetOutput, error)
}
```

### WidgetOutput

```go
type WidgetOutput struct {
    Text  string `json:"text"`   // display text, may contain raw ANSI
    Color string `json:"color"`  // "red", "green", "yellow", "cyan", "magenta", "white", "dim", "gray"
}
```

### StatusLineInput

```go
type StatusLineInput struct {
    Model         ModelInfo     `json:"model"`
    Workspace     WorkspaceInfo `json:"workspace"`
    ContextWindow ContextInfo   `json:"context_window"`
    RateLimits    RateLimits    `json:"rate_limits"`
    Cost          CostInfo      `json:"cost"`
    SessionID     string        `json:"session_id"`
    Version       string        `json:"version"`
}

type ModelInfo struct {
    ID          string `json:"id"`
    DisplayName string `json:"display_name"`
}

type WorkspaceInfo struct {
    CurrentDir string `json:"current_dir"`
    ProjectDir string `json:"project_dir"`
}

type ContextInfo struct {
    UsedPercentage      float64 `json:"used_percentage"`
    RemainingPercentage float64 `json:"remaining_percentage"`
    TotalInputTokens    int     `json:"total_input_tokens"`
    TotalOutputTokens   int     `json:"total_output_tokens"`
    ContextWindowSize   int     `json:"context_window_size"`
}

type RateLimits struct {
    FiveHour *RateLimit `json:"five_hour"`
    SevenDay *RateLimit `json:"seven_day"`
}

type RateLimit struct {
    UsedPercentage float64 `json:"used_percentage"`
    ResetsAt       int64   `json:"resets_at"`
}

type CostInfo struct {
    TotalCostUSD      float64 `json:"total_cost_usd"`
    TotalLinesAdded   int     `json:"total_lines_added"`
    TotalLinesRemoved int     `json:"total_lines_removed"`
}
```

### External Plugin Protocol (Phase 2)

External plugins are standalone executables in `~/.ccstatuswidgets/plugins/`. They receive `StatusLineInput` as JSON on stdin and write `WidgetOutput` as JSON to stdout. Same contract, different execution model.

Python plugin SDK example:

```python
from ccstatuswidgets import widget

@widget(name="weather", cache_ttl="30m")
def weather(config):
    return {"text": "+31°C Sunny", "color": "yellow"}
```

---

## Engine

The engine orchestrates concurrent widget execution:

```go
type Engine struct {
    widgets  []widget.Widget
    cache    *cache.Cache
    timeout  time.Duration  // default 500ms, configurable
}

func (e *Engine) Run(input *protocol.StatusLineInput, cfg *config.Config) [][]WidgetResult
```

### Execution Model

1. Parse config to determine which widgets go on which lines
2. For each line, launch all widgets concurrently as goroutines
3. Each widget has `timeout` to complete (default 500ms)
4. If a widget completes: use its result, update cache
5. If a widget times out: use last cached result (or omit if no cache)
6. Collect results per line, pass to renderer

### Cache Integration

- Widgets that only read stdin JSON (model, context, usage, git): no cache needed, instant
- Widgets with network calls or expensive operations: use cache with per-widget TTL
- Cache is file-based in `~/.ccstatuswidgets/cache/`

---

## Cache System

```go
type Cache struct {
    dir string  // ~/.ccstatuswidgets/cache/
}

func (c *Cache) Get(key string) ([]byte, bool)
func (c *Cache) Set(key string, value []byte, ttl time.Duration)
```

Cache files are JSON. The `data` field stores the full `WidgetOutput` struct:

```json
{
  "data": {"text": "+31°C Sunny", "color": "yellow"},
  "expires_at": 1774300000
}
```

---

## Renderer

```go
type Renderer struct{}

func (r *Renderer) Render(lines [][]WidgetResult) string
```

### Behavior

- Joins widget outputs on each line with a space separator
- Applies ANSI color codes based on `WidgetOutput.Color`
- Widgets returning raw ANSI in their text are passed through untouched
- Empty lines (all widgets omitted/timed out) are skipped
- Each line is written to stdout with `\n`, except the last line which omits the trailing newline

### Color Map

| Color     | ANSI Code       |
|-----------|-----------------|
| red       | `\033[0;31m`    |
| green     | `\033[0;32m`    |
| yellow    | `\033[0;33m`    |
| blue      | `\033[0;34m`    |
| magenta   | `\033[0;35m`    |
| cyan      | `\033[0;36m`    |
| white     | `\033[0;37m`    |
| dim       | `\033[2m`       |
| gray      | `\033[0;90m`    |

---

## Configuration

### Location

`~/.ccstatuswidgets/config.json`

### Schema

```json
{
  "timeout_ms": 500,
  "lines": [
    {
      "widgets": ["model", "effort", "directory", "git-branch", "context-bar", "usage-5h", "usage-7d"]
    },
    {
      "widgets": ["lines-changed", "cost", "memory"]
    }
  ],
  "widgets": {
    "context-bar": {
      "bar_length": 10,
      "show_percentage": true
    },
    "usage-5h": {
      "bar_length": 10,
      "show_percentage": true,
      "show_pace": true
    },
    "usage-7d": {
      "bar_length": 10,
      "show_percentage": true,
      "show_pace": true
    },
    "cost": {
      "detect_max_plan": true
    }
  }
}
```

### Defaults

`ccw init` creates a config with model, effort, directory, git-branch, context-bar, usage-5h, usage-7d on line 1 and lines-changed, cost, memory on line 2.

---

## Built-in Widgets (Phase 1)

| Widget        | Description                                      | Reads     | Cache |
|---------------|--------------------------------------------------|-----------|-------|
| `model`       | Model display name (magenta)                     | stdin     | no    |
| `effort`      | Effort level from settings.json (dim)            | file      | no    |
| `directory`   | Current directory basename (cyan)                | stdin     | no    |
| `git-branch`  | Current git branch (yellow)                      | git cmd   | no    |
| `context-bar` | Context usage bar + % (green/yellow/red)         | stdin     | no    |
| `usage-5h`    | 5-hour rate limit bar + % + pace (color-coded)   | stdin     | no    |
| `usage-7d`    | 7-day rate limit bar + % + pace (color-coded)    | stdin     | no    |
| `lines-changed` | Lines added/removed, git-style (+green/-red)   | stdin     | no    |
| `cost`        | Session cost, detects Max plan (api eq. prefix)  | stdin     | no    |
| `memory`      | Process memory via $PPID (dim)                   | ps cmd    | no    |

---

## CLI Commands

| Command            | Description                                          |
|--------------------|------------------------------------------------------|
| `ccw`              | Main mode: reads stdin JSON, renders status line     |
| `ccw init`         | Creates config.json, patches Claude Code settings    |
| `ccw doctor`       | Checks dependencies and configuration                |
| `ccw preview`      | Renders status line with sample data in terminal     |
| `ccw config edit`  | Opens config.json in $EDITOR                         |
| `ccw version`      | Prints version                                       |

### `ccw init`

1. Creates `~/.ccstatuswidgets/` directory structure
2. Writes default `config.json`
3. Patches `~/.claude/settings.json` to set `statusLine.command` to `ccw` binary path
4. Prints success message with next steps

### `ccw doctor`

Checks:
- ccw binary found and version
- `~/.ccstatuswidgets/config.json` exists and valid
- `~/.claude/settings.json` has statusLine pointing to ccw
- Python3 available (for phase 2 plugins)
- git available
- jq available (for external plugins)

### `ccw preview`

Renders the status line in terminal using sample/mock StatusLineInput data. Useful for testing config changes without needing Claude Code running.

---

## Installation

### Homebrew (recommended)

```bash
brew tap warunacds/ccstatuswidgets
brew install ccstatuswidgets
ccw init
```

### Install Script

```bash
curl -sSL https://raw.githubusercontent.com/warunacds/ccstatuswidgets/main/install.sh | sh
ccw init
```

The install script:
1. Detects OS and architecture
2. Downloads the correct binary from GitHub releases
3. Places binary as `/usr/local/bin/ccw` (or `~/.local/bin/ccw`). `ccw` is the canonical binary name.
4. Creates symlink `ccstatuswidgets` -> `ccw` (so both names work)
5. Prints instructions to run `ccw init`

---

## Phase 2 Roadmap

- Python plugin SDK: `pip install ccstatuswidgets`
- Plugin CLI: `ccw plugin list/add/install/remove`
- First-party plugins: weather, now-playing, flight, cricket, stocks, hackernews, moon, pomodoro
- Plugin registry (GitHub-based)
- Contributing guide for plugin authors
- `ccw track`, `ccw pomo`, `ccw hn` shortcut commands
