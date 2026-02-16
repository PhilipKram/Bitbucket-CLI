# bb - Bitbucket CLI

A command-line tool for interacting with Bitbucket Cloud. Manage repositories, pull requests, pipelines, issues, branches, snippets, workspaces, and more from your terminal.

## Installation

### Quick install (macOS / Linux)

```sh
curl -fsSL https://raw.githubusercontent.com/PhilipKram/Bitbucket-CLI/main/install.sh | sh
```

### Homebrew (macOS / Linux)

> **Note:** Homebrew installation requires a published release. If you get a
> "repository not found" error, use the quick install or build from source
> instead.

```sh
brew install PhilipKram/tap/bb
```

### From source

Requires Go 1.24+.

```sh
go install github.com/PhilipKram/bitbucket-cli@latest
```

### Binary releases

Download pre-built binaries for macOS, Linux, and Windows from the [Releases](https://github.com/PhilipKram/Bitbucket-CLI/releases) page.

## Shell Completion

`bb` supports shell completion for Bash, Zsh, Fish, and PowerShell. Completions include command names, flags, and dynamic suggestions for workspaces, repositories, branches, and pull requests.

### Bash

#### Current session
```sh
source <(bb completion bash)
```

#### Permanent installation
```sh
# Linux:
bb completion bash > /etc/bash_completion.d/bb

# macOS (with bash-completion@2 from Homebrew):
bb completion bash > $(brew --prefix)/etc/bash_completion.d/bb
```

### Zsh

#### Current session
```sh
source <(bb completion zsh)
```

#### Permanent installation
```sh
bb completion zsh > "${fpath[1]}/_bb"
```

Then start a new shell session.

### Fish

#### Current session
```sh
bb completion fish | source
```

#### Permanent installation
```sh
bb completion fish > ~/.config/fish/completions/bb.fish
```

### PowerShell

#### Current session
```powershell
bb completion powershell | Out-String | Invoke-Expression
```

#### Permanent installation
```powershell
# Add to your PowerShell profile:
bb completion powershell >> $PROFILE
```

Or save to a file and source it from your profile:
```powershell
bb completion powershell > bb.ps1
# Then add ". /path/to/bb.ps1" to your $PROFILE
```

## Authentication

`bb` uses OAuth 2.0 for authentication. You'll need an OAuth consumer from Bitbucket.

### Setting up an OAuth consumer

1. Go to **Bitbucket** > **Workspace settings** > **OAuth consumers** > **Add consumer**
2. Fill in:
   - **Name**: `bb` (or any name you like)
   - **Callback URL**: `http://localhost`
   - **Permissions**: select the scopes you need (e.g. Repositories, Pull requests, Pipelines, Issues)
3. Click **Save**
4. Copy the **Key** (client ID) and **Secret** (client secret)

### Interactive login

```sh
bb auth login
```

You'll be prompted for your OAuth consumer key and secret (or saved credentials will be used), then a browser window will open for authorization.

### Non-interactive / CI

```sh
# Provide an OAuth access token from stdin
echo "$OAUTH_TOKEN" | bb auth login --with-token

# OAuth browser flow with explicit credentials
bb auth login --web --client-id KEY --client-secret SECRET

# OAuth with saved credentials (re-authenticate)
bb auth login --web
```

### Other auth commands

```sh
bb auth status       # Show current auth state
bb auth token        # Print token to stdout (for piping)
bb auth refresh      # Refresh OAuth access token
bb auth logout       # Remove stored credentials
```

## Commands

| Command         | Description                        |
|-----------------|------------------------------------|
| `bb auth`       | Authenticate with Bitbucket        |
| `bb repo`       | Manage repositories                |
| `bb pr`         | Manage pull requests               |
| `bb pipeline`   | Manage pipelines (CI/CD)           |
| `bb issue`      | Manage issues (issue tracker)      |
| `bb branch`     | Manage branches and tags           |
| `bb snippet`    | Manage snippets                    |
| `bb workspace`  | Manage workspaces and projects     |
| `bb user`       | Manage user account and settings   |
| `bb config`     | Manage CLI configuration           |
| `bb completion` | Generate shell completion scripts  |
| `bb mcp`        | Model Context Protocol server      |
| `bb upgrade`    | Self-update to the latest version  |

## MCP (Model Context Protocol)

`bb` supports the Model Context Protocol, allowing AI agents and LLM-powered development tools to interact with Bitbucket through `bb` as a tool provider.

### Start the MCP server

```sh
bb mcp serve
```

### Configure your AI client

**Claude Desktop** - Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "bitbucket": {
      "command": "bb",
      "args": ["mcp", "serve"]
    }
  }
}
```

### What you can do

Once configured, AI agents can:

- Create and manage pull requests
- Trigger and monitor CI/CD pipelines
- Create and list issues
- All using natural language!

**Example interactions:**
- "Create a pull request in myworkspace/myrepo from feature-branch to main"
- "Show me recent pipeline runs"
- "List all open pull requests"

For full documentation, configuration examples, and available tools, see [docs/mcp.md](docs/mcp.md).

### Pull requests

```sh
bb pr list myworkspace/myrepo
bb pr list myworkspace/myrepo --state OPEN --json
bb pr view myworkspace/myrepo 42
bb pr create myworkspace/myrepo --title "Feature" --source feature-branch
bb pr create myworkspace/myrepo --title "Feature" --source dev --no-default-reviewers
bb pr edit myworkspace/myrepo 42 --title "Updated title"
bb pr merge myworkspace/myrepo 42 --strategy squash
bb pr approve myworkspace/myrepo 42
bb pr unapprove myworkspace/myrepo 42
bb pr decline myworkspace/myrepo 42
bb pr comment myworkspace/myrepo 42 --body "Looks good!"
bb pr comment myworkspace/myrepo 42 --body "Fix this" --file src/main.go --line 42
bb pr comments myworkspace/myrepo 42
bb pr diff myworkspace/myrepo 42
bb pr activity myworkspace/myrepo 42
```

`bb pr create` automatically fetches and adds the repository's default reviewers. Use `--no-default-reviewers` to skip this. The `bb pr comment` command supports inline comments on specific files and lines using `--file/-f` and `--line/-l` flags (both must be provided together). The `bb pr list` output includes a reviewers column.

### Repositories

```sh
bb repo list myworkspace
bb repo view myworkspace/myrepo
bb repo create myworkspace --name myrepo --private
bb repo clone myworkspace/myrepo
bb repo clone myworkspace/myrepo --protocol ssh
bb repo clone myworkspace/myrepo ./my-directory
bb repo commits myworkspace/myrepo
bb repo diff myworkspace/myrepo main..feature
bb repo fork myworkspace/myrepo
bb repo delete myworkspace/myrepo
```

The `bb repo clone` command supports HTTPS (default, with automatic token injection) and SSH protocols via `--protocol/-p`. It automatically sets `bb.workspace` in the cloned repo's local git config.

### Pipelines

```sh
bb pipeline list myworkspace/myrepo
bb pipeline view myworkspace/myrepo <uuid>
bb pipeline run myworkspace/myrepo --branch main
bb pipeline run myworkspace/myrepo --custom --pattern deploy
bb pipeline stop myworkspace/myrepo <uuid>
bb pipeline steps myworkspace/myrepo <uuid>
bb pipeline log myworkspace/myrepo <pipeline-uuid> <step-uuid>
bb pipeline watch myworkspace/myrepo                    # Watch latest pipeline
bb pipeline watch myworkspace/myrepo --build 187        # Watch specific build
```

The `bb pipeline watch` command monitors pipeline status in real-time with colored output and auto-exits with an appropriate exit code when the pipeline completes. Use `--interval/-i` to set the polling interval.

### Branches and tags

```sh
bb branch list myworkspace/myrepo
bb branch create myworkspace/myrepo --name feature --target main
bb branch delete myworkspace/myrepo feature
bb branch tags myworkspace/myrepo
bb branch create-tag myworkspace/myrepo --name v1.0 --target main
bb branch tag-delete myworkspace/myrepo v1.0
bb branch restrictions myworkspace/myrepo
```

### Issues

```sh
bb issue list myworkspace/myrepo
bb issue view myworkspace/myrepo 1
bb issue create myworkspace/myrepo --title "Bug" --priority critical
bb issue edit myworkspace/myrepo 1 --title "Updated title"
bb issue delete myworkspace/myrepo 1
bb issue comment myworkspace/myrepo 1 --body "Fixed in #42"
bb issue comments myworkspace/myrepo 1
bb issue vote myworkspace/myrepo 1
bb issue watch myworkspace/myrepo 1
```

### Workspaces

```sh
bb workspace list
bb workspace view myworkspace
bb workspace members myworkspace
bb workspace projects myworkspace
bb workspace project-create myworkspace --name "My Project"
bb workspace permissions myworkspace
```

### Snippets

```sh
bb snippet list myworkspace
bb snippet view myworkspace <snippet-id>
bb snippet create myworkspace --title "My Snippet" --file script.sh
bb snippet delete myworkspace <snippet-id>
```

### User

```sh
bb user me                          # Show current authenticated user
bb user view <username>             # View a user's profile
bb user emails                      # List your email addresses
bb user ssh-keys                    # List your SSH keys
bb user ssh-key-add --label "My Key" --key "ssh-rsa ..."
```

### Upgrading

```sh
bb upgrade              # Self-update to the latest release
bb upgrade --force      # Force upgrade (bypass install method check)
```

If you installed via Homebrew or `go install`, `bb upgrade` will suggest the appropriate command instead. Use `--force` to override.

## Configuration

```sh
bb config view                                 # View current config
bb config set-default-workspace myworkspace    # Set default workspace
bb config set-format json                      # Set default output format (table, json)
```

Configuration and credentials are stored in `$XDG_CONFIG_HOME/bitbucket-cli/` (or `~/.config/bitbucket-cli/` on Linux, `~/Library/Application Support/bitbucket-cli/` on macOS, `%AppData%/bitbucket-cli/` on Windows).

## Output formats

Most list and view commands support a `--json` flag for machine-readable output:

```sh
bb pr list myworkspace/myrepo --json
bb auth status --json
```

The default output format is a human-readable table.

## Update notifications

`bb` automatically checks for new releases in the background (once every 24 hours). When a newer version is available, a notice is printed after the command output:

```
Update available: v0.0.7 → v0.0.8
Run `bb upgrade` to update
```

The check adds no latency to commands and is skipped when output is piped.

## Environment variables

| Variable           | Description                                      |
|--------------------|--------------------------------------------------|
| `BB_HTTP_TIMEOUT`  | HTTP client timeout in seconds (default: 30)     |
| `VISUAL`           | Preferred editor for composing comments           |
| `EDITOR`           | Fallback editor if `VISUAL` is not set            |

## Development

### Build

```sh
go build -o bb .
```

### Test

```sh
go test ./...
```

### Release

Releases are automated with [GoReleaser](https://goreleaser.com/) via GitHub Actions. Tag a version to trigger a release:

```sh
git tag v0.1.0
git push origin v0.1.0
```

This builds binaries for all platforms and publishes the Homebrew formula automatically.

## License

[MIT](LICENSE)
