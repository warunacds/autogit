# autogit

A CLI tool that generates git commit messages using Claude AI.

## Install

```bash
go install github.com/waruna/autogit@latest
```

Or build from source:

```bash
git clone https://github.com/waruna/autogit
cd autogit
go build -o autogit .
# Move to PATH:
mv autogit /usr/local/bin/autogit
```

## Usage

```bash
# Stage your changes first
git add .

# Generate and commit
autogit

# Or include unstaged changes too
autogit --all
```

## Requirements

- Go 1.22+
- An [Anthropic API key](https://console.anthropic.com/)
- `ANTHROPIC_API_KEY` set in your environment

```bash
export ANTHROPIC_API_KEY=sk-ant-...
```

## How it works

1. Reads your git diff (staged by default, or all with `--all`)
2. Sends the diff to Claude API to generate a Conventional Commits message
3. Lets you: **[a]** accept, **[e]** edit in `$EDITOR`, **[r]** regenerate, or **[q]** quit
4. Commits using your existing git config (name/email)
