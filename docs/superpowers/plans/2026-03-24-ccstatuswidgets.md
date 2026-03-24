# ccstatuswidgets Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go binary (`ccw`) that provides a customizable, plugin-ready status line for Claude Code CLI with 10 built-in widgets, concurrent execution, and a JSON config system.

**Architecture:** Go binary reads JSON from Claude Code via stdin, runs configured widgets concurrently with timeouts, and renders colored multi-line output to stdout. Widgets implement a simple interface. Config lives in `~/.ccstatuswidgets/config.json`.

**Tech Stack:** Go 1.21+, no external dependencies (stdlib only for phase 1)

**Spec:** `docs/superpowers/specs/2026-03-24-ccstatuswidgets-design.md`

---

### Task 1: Project Scaffolding

**Files:**
- Create: `cmd/ccw/main.go`
- Create: `go.mod`
- Create: `LICENSE`
- Create: `internal/protocol/protocol.go`

- [ ] **Step 1: Create the project directory and initialize Go module**

```bash
mkdir -p /Users/waruna/Skunkwork/ccstatuswidgets
cd /Users/waruna/Skunkwork/ccstatuswidgets
go mod init github.com/warunacds/ccstatuswidgets
```

- [ ] **Step 2: Create the protocol types**

Create `internal/protocol/protocol.go` with `StatusLineInput`, `WidgetOutput`, and all sub-structs from the spec. `RateLimits` fields use pointer types for optional sections.

```go
package protocol

type StatusLineInput struct {
    Model         ModelInfo    `json:"model"`
    Workspace     WorkspaceInfo `json:"workspace"`
    ContextWindow ContextInfo  `json:"context_window"`
    RateLimits    *RateLimits  `json:"rate_limits"`
    Cost          CostInfo     `json:"cost"`
    SessionID     string       `json:"session_id"`
    Version       string       `json:"version"`
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

type WidgetOutput struct {
    Text  string `json:"text"`
    Color string `json:"color"`
}
```

- [ ] **Step 3: Create minimal main.go**

Create `cmd/ccw/main.go` that reads stdin JSON, unmarshals to `StatusLineInput`, and prints a placeholder message.

```go
package main

import (
    "encoding/json"
    "fmt"
    "io"
    "os"
    "time"

    "github.com/warunacds/ccstatuswidgets/internal/protocol"
)

func main() {
    // Read stdin with 1-second timeout
    done := make(chan []byte, 1)
    go func() {
        data, _ := io.ReadAll(os.Stdin)
        done <- data
    }()

    var data []byte
    select {
    case data = <-done:
    case <-time.After(1 * time.Second):
        os.Exit(0)
    }

    if len(data) == 0 {
        os.Exit(0)
    }

    var input protocol.StatusLineInput
    if err := json.Unmarshal(data, &input); err != nil {
        os.Exit(0)
    }

    fmt.Print("ccw: ok")
}
```

- [ ] **Step 4: Add MIT LICENSE file**

- [ ] **Step 5: Build and verify**

```bash
go build -o ccw ./cmd/ccw
echo '{"model":{"display_name":"Opus"}}' | ./ccw
# Expected: ccw: ok
```

- [ ] **Step 6: Commit**

```bash
git init
git add .
git commit -m "feat: project scaffolding with protocol types and main entry point"
```

---

### Task 2: Widget Interface and Registry

**Files:**
- Create: `internal/widget/widget.go`
- Create: `internal/widget/registry.go`
- Create: `internal/widget/widget_test.go`
- Create: `internal/widget/registry_test.go`

- [ ] **Step 1: Write test for Widget interface compliance**

```go
// internal/widget/widget_test.go
package widget_test

import (
    "testing"

    "github.com/warunacds/ccstatuswidgets/internal/protocol"
    "github.com/warunacds/ccstatuswidgets/internal/widget"
)

type mockWidget struct{}

func (m *mockWidget) Name() string { return "mock" }
func (m *mockWidget) Render(input *protocol.StatusLineInput, cfg map[string]interface{}) (*protocol.WidgetOutput, error) {
    return &protocol.WidgetOutput{Text: "hello", Color: "green"}, nil
}

func TestWidgetInterface(t *testing.T) {
    var w widget.Widget = &mockWidget{}
    if w.Name() != "mock" {
        t.Fatalf("expected mock, got %s", w.Name())
    }
    out, err := w.Render(&protocol.StatusLineInput{}, nil)
    if err != nil {
        t.Fatal(err)
    }
    if out.Text != "hello" || out.Color != "green" {
        t.Fatalf("unexpected output: %+v", out)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/widget/ -v
# Expected: FAIL — widget package does not exist
```

- [ ] **Step 3: Create widget interface**

```go
// internal/widget/widget.go
package widget

import "github.com/warunacds/ccstatuswidgets/internal/protocol"

type Widget interface {
    Name() string
    Render(input *protocol.StatusLineInput, cfg map[string]interface{}) (*protocol.WidgetOutput, error)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/widget/ -v
# Expected: PASS
```

- [ ] **Step 5: Write registry tests**

```go
// internal/widget/registry_test.go
package widget_test

import (
    "testing"

    "github.com/warunacds/ccstatuswidgets/internal/widget"
)

func TestRegistryRegisterAndGet(t *testing.T) {
    r := widget.NewRegistry()
    r.Register(&mockWidget{})

    w, ok := r.Get("mock")
    if !ok {
        t.Fatal("widget not found")
    }
    if w.Name() != "mock" {
        t.Fatalf("expected mock, got %s", w.Name())
    }
}

func TestRegistryGetMissing(t *testing.T) {
    r := widget.NewRegistry()
    _, ok := r.Get("nonexistent")
    if ok {
        t.Fatal("expected not found")
    }
}

func TestRegistryNames(t *testing.T) {
    r := widget.NewRegistry()
    r.Register(&mockWidget{})
    names := r.Names()
    if len(names) != 1 || names[0] != "mock" {
        t.Fatalf("unexpected names: %v", names)
    }
}
```

- [ ] **Step 6: Implement registry**

```go
// internal/widget/registry.go
package widget

type Registry struct {
    widgets map[string]Widget
}

func NewRegistry() *Registry {
    return &Registry{widgets: make(map[string]Widget)}
}

func (r *Registry) Register(w Widget) {
    r.widgets[w.Name()] = w
}

func (r *Registry) Get(name string) (Widget, bool) {
    w, ok := r.widgets[name]
    return w, ok
}

func (r *Registry) Names() []string {
    names := make([]string, 0, len(r.widgets))
    for name := range r.widgets {
        names = append(names, name)
    }
    return names
}
```

- [ ] **Step 7: Run tests and commit**

```bash
go test ./internal/widget/ -v
# Expected: all PASS
git add . && git commit -m "feat: widget interface and registry"
```

---

### Task 3: Renderer

**Files:**
- Create: `internal/renderer/renderer.go`
- Create: `internal/renderer/renderer_test.go`

- [ ] **Step 1: Write renderer tests**

Test cases: single widget, multiple widgets on a line, multiple lines, empty lines skipped, color application, raw ANSI passthrough, last line no trailing newline.

- [ ] **Step 2: Run tests to verify they fail**

- [ ] **Step 3: Implement renderer**

The renderer takes `[][]WidgetResult` where `WidgetResult` contains `*WidgetOutput` and widget name. It joins widgets on each line with spaces, wraps text in ANSI color codes from the color map, skips empty lines, and writes `\n` between lines (no trailing newline on last line).

- [ ] **Step 4: Run tests to verify they pass**

- [ ] **Step 5: Commit**

```bash
git add . && git commit -m "feat: renderer with ANSI color support"
```

---

### Task 4: Cache System

**Files:**
- Create: `internal/cache/cache.go`
- Create: `internal/cache/cache_test.go`

- [ ] **Step 1: Write cache tests**

Test cases: Set then Get returns data, Get on expired key returns false, Get on missing key returns false, Set overwrites existing, concurrent access safety.

- [ ] **Step 2: Run tests to verify they fail**

- [ ] **Step 3: Implement file-based TTL cache**

Cache stores JSON files in `~/.ccstatuswidgets/cache/`. Each file contains `{"data": <WidgetOutput JSON>, "expires_at": <unix timestamp>}`. `Get` checks expiry. `Set` writes atomically (write to temp file, rename).

- [ ] **Step 4: Run tests to verify they pass**

- [ ] **Step 5: Commit**

```bash
git add . && git commit -m "feat: file-based TTL cache"
```

---

### Task 5: Config System

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/defaults.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write config tests**

Test cases: Load valid config, Load missing file returns defaults, Load invalid JSON returns error, Default config has expected widgets on expected lines, widget config overrides.

- [ ] **Step 2: Run tests to verify they fail**

- [ ] **Step 3: Implement config**

`Config` struct with `TimeoutMs int`, `Lines []LineConfig`, `Widgets map[string]map[string]interface{}`. `LineConfig` has `Widgets []string`. `Load(path)` reads JSON, `Default()` returns the default config, `Save(path)` writes JSON. `ConfigDir()` returns `~/.ccstatuswidgets`.

- [ ] **Step 4: Implement defaults**

Default config: line 1 = `["model", "effort", "directory", "git-branch", "context-bar", "usage-5h", "usage-7d"]`, line 2 = `["lines-changed", "cost", "memory"]`. Default timeout = 500ms. Default widget configs for bar lengths, show_percentage, etc.

- [ ] **Step 5: Run tests to verify they pass**

- [ ] **Step 6: Commit**

```bash
git add . && git commit -m "feat: config system with defaults"
```

---

### Task 6: Engine — Concurrent Widget Executor

**Files:**
- Create: `internal/engine/engine.go`
- Create: `internal/engine/engine_test.go`

- [ ] **Step 1: Write engine tests**

Test cases: runs widgets concurrently and collects results, timed-out widget falls back to cache, timed-out widget with no cache is omitted, erroring widget is omitted, erroring widget falls back to cache, results maintain line/widget order.

- [ ] **Step 2: Run tests to verify they fail**

- [ ] **Step 3: Implement engine**

`Engine` struct holds a `*Registry`, `*Cache`, and `timeout`. `Run()` iterates config lines, launches goroutines per widget, collects results via channels with `select` + timeout, falls back to cache on timeout/error. Returns `[][]WidgetResult` preserving order.

- [ ] **Step 4: Run tests to verify they pass**

- [ ] **Step 5: Commit**

```bash
git add . && git commit -m "feat: concurrent engine with timeout and cache fallback"
```

---

### Task 7: Built-in Widgets — Simple (model, effort, directory, git-branch)

**Files:**
- Create: `internal/widgets/model.go`
- Create: `internal/widgets/effort.go`
- Create: `internal/widgets/directory.go`
- Create: `internal/widgets/gitbranch.go`
- Create: `internal/widgets/model_test.go`
- Create: `internal/widgets/effort_test.go`
- Create: `internal/widgets/directory_test.go`
- Create: `internal/widgets/gitbranch_test.go`

- [ ] **Step 1: Write tests for model widget**

Test: returns display_name in magenta, returns nil when display_name is empty.

- [ ] **Step 2: Implement model widget**

Reads `input.Model.DisplayName`, returns `WidgetOutput{Text: name, Color: "magenta"}`.

- [ ] **Step 3: Write tests for effort widget**

Test: returns effort level from settings.json mock, returns nil when no settings file.

- [ ] **Step 4: Implement effort widget**

Reads `~/.claude/settings.json` then `~/.claude/settings.local.json`, extracts `effortLevel`. Returns `WidgetOutput{Text: "(high)", Color: "dim"}`.

- [ ] **Step 5: Write tests for directory widget**

Test: returns basename of workspace.current_dir in cyan, falls back to PWD.

- [ ] **Step 6: Implement directory widget**

Reads `input.Workspace.CurrentDir`, returns `WidgetOutput{Text: basename, Color: "cyan"}`.

- [ ] **Step 7: Write tests for git-branch widget**

Test: returns branch name in yellow, returns nil when not in git repo.

- [ ] **Step 8: Implement git-branch widget**

Runs `git symbolic-ref --short HEAD` in the workspace dir. Returns `WidgetOutput{Text: "(main)", Color: "yellow"}`. Returns nil output (omit) if not a git repo.

- [ ] **Step 9: Run all tests, commit**

```bash
go test ./internal/widgets/ -v
git add . && git commit -m "feat: model, effort, directory, git-branch widgets"
```

---

### Task 8: Built-in Widgets — Bars (context-bar, usage-5h, usage-7d)

**Files:**
- Create: `internal/widgets/context.go`
- Create: `internal/widgets/usage.go`
- Create: `internal/widgets/bars.go` (shared bar builder helper)
- Create: `internal/widgets/context_test.go`
- Create: `internal/widgets/usage_test.go`
- Create: `internal/widgets/bars_test.go`

- [ ] **Step 1: Write tests for bar builder**

Test: buildBar(30, 10) returns correct filled/empty chars, color thresholds (green <50, yellow <80, red >=80).

- [ ] **Step 2: Implement shared bar builder**

`BuildBar(percentage float64, length int) string` returns `█░` bar string. `BarColor(percentage float64) string` returns color name based on thresholds.

- [ ] **Step 3: Write tests for context-bar widget**

Test: renders bar + percentage, handles zero/nil values, respects bar_length and show_percentage config.

- [ ] **Step 4: Implement context-bar widget**

Reads `input.ContextWindow.UsedPercentage`, builds bar, returns with dynamic color.

- [ ] **Step 5: Write tests for usage widgets**

Test: renders 5h/7d bars, pace tracking calculation (+green when headroom, -red when burning fast), handles nil RateLimits, respects config options.

- [ ] **Step 6: Implement usage-5h and usage-7d widgets**

Both read from `input.RateLimits`. Calculate pace by comparing usage percentage to elapsed time percentage in the window. Return bar + percentage + pace + reset time.

- [ ] **Step 7: Run all tests, commit**

```bash
go test ./internal/widgets/ -v
git add . && git commit -m "feat: context-bar, usage-5h, usage-7d widgets with pace tracking"
```

---

### Task 9: Built-in Widgets — Stats (lines-changed, cost, memory)

**Files:**
- Create: `internal/widgets/lines.go`
- Create: `internal/widgets/cost.go`
- Create: `internal/widgets/memory.go`
- Create: `internal/widgets/lines_test.go`
- Create: `internal/widgets/cost_test.go`
- Create: `internal/widgets/memory_test.go`

- [ ] **Step 1: Write tests for lines-changed widget**

Test: renders +N green -N red, handles zero values, handles nil.

- [ ] **Step 2: Implement lines-changed widget**

Returns raw ANSI: green `+N` red `-N`.

- [ ] **Step 3: Write tests for cost widget**

Test: shows `api eq. $X.XX` when RateLimits present (Max plan), shows `$X.XX` when no rate limits (API key), omits when cost is 0.

- [ ] **Step 4: Implement cost widget**

Checks `input.RateLimits != nil` to detect Max/Pro. Formats cost with 2 decimal places.

- [ ] **Step 5: Write tests for memory widget**

Test: returns memory in MB format, handles process not found.

- [ ] **Step 6: Implement memory widget**

Uses `os.Getppid()` and reads `/proc/{pid}/status` (Linux) or `ps -o rss=` (macOS) to get RSS. Returns `{N}MB` in dim.

- [ ] **Step 7: Run all tests, commit**

```bash
go test ./internal/widgets/ -v
git add . && git commit -m "feat: lines-changed, cost, memory widgets"
```

---

### Task 10: Wire Everything — Main Pipeline

**Files:**
- Modify: `cmd/ccw/main.go`
- Create: `internal/widgets/register.go`

- [ ] **Step 1: Create register.go that registers all built-in widgets**

```go
package widgets

import "github.com/warunacds/ccstatuswidgets/internal/widget"

func RegisterAll(r *widget.Registry) {
    r.Register(&ModelWidget{})
    r.Register(&EffortWidget{})
    r.Register(&DirectoryWidget{})
    r.Register(&GitBranchWidget{})
    r.Register(&ContextBarWidget{})
    r.Register(&Usage5hWidget{})
    r.Register(&Usage7dWidget{})
    r.Register(&LinesChangedWidget{})
    r.Register(&CostWidget{})
    r.Register(&MemoryWidget{})
}
```

- [ ] **Step 2: Update main.go to wire the full pipeline**

Main flow: read stdin -> parse JSON -> load config -> create registry -> register all widgets -> create engine -> run engine -> render output -> print to stdout.

- [ ] **Step 3: Build and test end-to-end**

```bash
go build -o ccw ./cmd/ccw
echo '{"model":{"display_name":"Opus 4.6"},"workspace":{"current_dir":"/Users/waruna/project"},"context_window":{"used_percentage":15}}' | ./ccw
# Expected: colored status line output
```

- [ ] **Step 4: Commit**

```bash
git add . && git commit -m "feat: wire main pipeline — stdin to rendered output"
```

---

### Task 11: CLI Commands (init, doctor, preview, version)

**Files:**
- Create: `internal/cli/init.go`
- Create: `internal/cli/doctor.go`
- Create: `internal/cli/preview.go`
- Create: `internal/cli/cli_test.go`
- Modify: `cmd/ccw/main.go`

- [ ] **Step 1: Write tests for init command**

Test: creates config dir, writes default config, patches Claude Code settings.json.

- [ ] **Step 2: Implement init command**

Creates `~/.ccstatuswidgets/`, `~/.ccstatuswidgets/cache/`, `~/.ccstatuswidgets/plugins/`. Writes default `config.json`. Reads `~/.claude/settings.json`, adds/updates `statusLine` field, writes back. Prints success.

- [ ] **Step 3: Write tests for doctor command**

Test: reports found/missing dependencies.

- [ ] **Step 4: Implement doctor command**

Checks: ccw version, config.json exists + valid, Claude settings wired, git available, python3 available.

- [ ] **Step 5: Implement preview command**

Creates a sample `StatusLineInput` with realistic data (model: Opus 4.6, context: 35%, usage: 22%, etc.) and runs it through the full pipeline. Prints the rendered output.

- [ ] **Step 6: Add version command**

Prints `ccstatuswidgets v0.1.0`. Version embedded at build time via `-ldflags`.

- [ ] **Step 7: Update main.go with CLI routing**

Parse `os.Args`: no args = stdin mode, `init`/`doctor`/`preview`/`config`/`version` = CLI mode.

- [ ] **Step 8: Run all tests, commit**

```bash
go test ./... -v
git add . && git commit -m "feat: CLI commands — init, doctor, preview, version"
```

---

### Task 12: Install Script

**Files:**
- Create: `install.sh`

- [ ] **Step 1: Write install script**

Detects OS (darwin/linux) and arch (amd64/arm64). Downloads binary from GitHub releases. Places in `/usr/local/bin/ccw`. Creates `ccstatuswidgets` symlink. Prints instructions.

- [ ] **Step 2: Test locally**

```bash
chmod +x install.sh
# Read through script, verify logic
```

- [ ] **Step 3: Commit**

```bash
git add . && git commit -m "feat: install script for curl-based installation"
```

---

### Task 13: README and Polish

**Files:**
- Create: `README.md`
- Create: `.goreleaser.yml` (for GitHub releases)

- [ ] **Step 1: Write README**

Sections: what it is, screenshot/example output, quick start (install + init), configuration, built-in widgets table, CLI commands, contributing, license. Keep it concise.

- [ ] **Step 2: Create .goreleaser.yml**

GoReleaser config for building cross-platform binaries and creating GitHub releases. Builds for darwin/amd64, darwin/arm64, linux/amd64, linux/arm64.

- [ ] **Step 3: Final full test run**

```bash
go test ./... -v -race
go vet ./...
go build -o ccw ./cmd/ccw
echo '{"model":{"display_name":"Opus 4.6"}}' | ./ccw
```

- [ ] **Step 4: Commit and tag**

```bash
git add . && git commit -m "docs: README and release configuration"
git tag v0.1.0
```
