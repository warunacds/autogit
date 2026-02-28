# autogit

A CLI tool that generates git commit messages using Claude AI.

Analyzes your staged git diff, calls the Claude API to suggest a [Conventional Commits](https://www.conventionalcommits.org/) message, then lets you accept, edit, regenerate, or abort — all from the terminal.

## Demo

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

## Requirements

- Go 1.22+
- An [Anthropic API key](https://console.anthropic.com/)

## Setup

**1. Get an Anthropic API key**

Sign up at [console.anthropic.com](https://console.anthropic.com/) and create an API key.

**2. Export the key in your shell**

Add this to your `~/.zshrc` or `~/.bashrc`:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
```

Then reload: `source ~/.zshrc`

**3. Install autogit**

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

**4. Verify it works**

```bash
autogit --help
```

## Usage

```bash
# Stage your changes
git add .

# Generate and commit
autogit

# Include unstaged changes too
autogit --all
```

### Interactive options

| Key | Action |
|-----|--------|
| `a` | Accept the message and commit |
| `e` | Open `$EDITOR` to edit the message |
| `r` | Regenerate — call Claude again for a new suggestion |
| `q` | Quit without committing |
| *(type anything)* | Replace the message inline and loop back |

## How it works

1. Reads your git diff (`git diff --cached` by default, or `git diff HEAD` with `--all`)
2. Sends the diff to the Claude API with a Conventional Commits prompt
3. Shows the generated message with an interactive menu
4. Commits via `git commit -m` using your existing git config (name/email)

Diffs larger than 100 KB are automatically truncated before sending to the API.

## Configuration

| Setting | How to set |
|---------|-----------|
| API key | `ANTHROPIC_API_KEY` environment variable |
| Editor | `EDITOR` environment variable (falls back to `nano`) |
| Diff scope | `--all` flag (default: staged only) |

## License

MIT
