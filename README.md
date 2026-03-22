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

[a] Accept  [A] Accept and Push  [e] Edit in $EDITOR  [r] Regenerate  [q] Quit
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
| `A` | Accept the message, commit, and push |
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
6. Optionally pushes to the remote with `--push` / `-p` or interactively with `A`

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
