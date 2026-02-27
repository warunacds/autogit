# autogit — Design Document

**Date:** 2026-02-27
**Status:** Approved

## Overview

`autogit` is a Go CLI tool that analyzes staged (or all) git changes, generates a commit message using the Claude API, lets the user review and edit it, then commits using the user's existing git config.

## User Flow

```
$ autogit              # analyzes staged changes (default)
$ autogit --all        # analyzes staged + unstaged changes

[autogit] Analyzing 3 changed files...
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

- **Accept (`a`):** Run `git commit -m "<message>"` immediately.
- **Edit (`e`):** Open `$EDITOR` (fallback: `nano`) with message in a temp file. On save+close, show updated message with menu again.
- **Inline edit:** If user types text at the prompt, use that as the message.
- **Regenerate (`r`):** Call Claude API again, show new message.
- **Quit (`q`):** Exit 0, nothing committed.

## Architecture

```
autogit/
├── main.go                  # entry point: parse flags, wire up, run
├── go.mod                   # module: github.com/waruna/autogit
├── go.sum
├── internal/
│   ├── git/
│   │   ├── diff.go          # git diff --cached or all changes → string
│   │   └── commit.go        # git commit -m "..."
│   ├── claude/
│   │   └── client.go        # Anthropic Messages API → commit message string
│   ├── editor/
│   │   └── editor.go        # open $EDITOR with temp file → edited string
│   └── ui/
│       └── prompt.go        # display message + menu, handle user input loop
└── docs/
    └── plans/
        └── 2026-02-27-autogit-design.md
```

### Data Flow

1. `main.go` — parse `--all` flag, check `ANTHROPIC_API_KEY` env var
2. `git.GetDiff(staged bool)` — run `git diff --cached` or `git diff HEAD`, return raw diff string
3. Validate: empty diff → error and exit
4. `claude.GenerateMessage(diff string)` — POST to Anthropic Messages API, return message string
5. `ui.Run(message string, regenerateFn func() string)` — interactive loop
6. If user picks `e`: call `editor.Open(message)` → return edited string
7. If user picks `a`: call `git.Commit(message)` → run `git commit -m "..."`

### Dependencies

- `github.com/anthropics/anthropic-sdk-go` — official Anthropic Go SDK (only external dep)

### Claude Prompt

- **System:** "You are a git commit message generator. Output only the commit message, following Conventional Commits format. No preamble, no markdown code fences, no explanation."
- **User:** The raw git diff
- **Temperature:** 0 (deterministic output)
- **Model:** `claude-opus-4-6` (latest capable model)

## Configuration

| Config | Source |
|--------|--------|
| API key | `ANTHROPIC_API_KEY` environment variable |
| Editor | `$EDITOR` env var, fallback to `nano` |
| Diff scope | `--all` flag (default: staged only) |

## Error Handling

| Scenario | Behavior |
|----------|----------|
| `ANTHROPIC_API_KEY` not set | Fatal: clear message telling user to export it |
| Not in a git repo | Fatal: "Not a git repository" |
| No staged changes (default mode) | Fatal: "No staged changes. Use `git add` first, or run `autogit --all`" |
| No changes at all (`--all`) | Fatal: "No changes detected" |
| Claude API error / rate limit | Fatal: show API error message |
| Diff too large | Truncate to ~100KB with a warning, still generate |
| `git commit` fails (pre-commit hook etc.) | Show git's stderr output, exit with non-zero code |
| User quits (`q`) | Exit 0 cleanly |

## Flags

| Flag | Description |
|------|-------------|
| `--all` | Include unstaged changes in addition to staged |
| `--help` | Show usage |

## Out of Scope

- Config file (`~/.autogit/config`) — env var only for now
- Subcommands — single binary, single purpose
- Sign-off or GPG signing — use user's existing git config
- Interactive staging — user must run `git add` before `autogit`
