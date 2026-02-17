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

## Authentication

`bb` supports two authentication methods:

### Interactive login

```sh
bb auth login
```

You'll be prompted to choose between:

1. **App Password** - username + app password (simplest)
2. **OAuth 2.0** - browser-based authorization

### Non-interactive / CI

```sh
# App password from stdin
echo "$APP_PASSWORD" | bb auth login --with-token --username myuser

# App password via environment variables
BB_USERNAME=myuser BB_TOKEN=password bb auth login --with-token

# OAuth browser flow
bb auth login --web --client-id KEY --client-secret SECRET
```

### Other auth commands

```sh
bb auth status       # Show current auth state
bb auth token        # Print token to stdout (for piping)
bb auth refresh      # Refresh OAuth access token
bb auth logout       # Remove stored credentials
```

## Commands

| Command       | Description                        |
|---------------|------------------------------------|
| `bb auth`     | Authenticate with Bitbucket        |
| `bb repo`     | Manage repositories                |
| `bb pr`       | Manage pull requests               |
| `bb pipeline` | Manage pipelines (CI/CD)           |
| `bb issue`    | Manage issues (issue tracker)      |
| `bb branch`   | Manage branches and tags           |
| `bb snippet`  | Manage snippets                    |
| `bb workspace`| Manage workspaces and projects     |
| `bb user`     | Manage user account and settings   |
| `bb config`   | Manage CLI configuration           |

### Pull requests

```sh
bb pr list myworkspace/myrepo
bb pr list myworkspace/myrepo --state OPEN --json
bb pr view myworkspace/myrepo 42
bb pr create myworkspace/myrepo --title "Feature" --source feature-branch
bb pr merge myworkspace/myrepo 42 --strategy squash
bb pr approve myworkspace/myrepo 42
bb pr comment myworkspace/myrepo 42 --body "Looks good!"
bb pr comment myworkspace/myrepo 42 --body "Fix this" --file src/main.go --line 42
bb pr comments myworkspace/myrepo 42
bb pr diff myworkspace/myrepo 42
bb pr activity myworkspace/myrepo 42
```

The `bb pr comment` command supports inline comments on specific files and lines in a PR diff using `--file/-f` and `--line/-l` flags. Both flags must be provided together.

### Repositories

```sh
bb repo list myworkspace
bb repo view myworkspace/myrepo
bb repo create myworkspace --name myrepo --private
bb repo commits myworkspace/myrepo
bb repo diff myworkspace/myrepo main..feature
bb repo fork myworkspace/myrepo
```

### Pipelines

```sh
bb pipeline list myworkspace/myrepo
bb pipeline view myworkspace/myrepo <uuid>
bb pipeline run myworkspace/myrepo --branch main
bb pipeline stop myworkspace/myrepo <uuid>
bb pipeline steps myworkspace/myrepo <uuid>
bb pipeline log myworkspace/myrepo <pipeline-uuid> <step-uuid>
```

### Branches and tags

```sh
bb branch list myworkspace/myrepo
bb branch create myworkspace/myrepo --name feature --target main
bb branch delete myworkspace/myrepo feature
bb branch tags myworkspace/myrepo
bb branch create-tag myworkspace/myrepo --name v1.0 --target main
```

### Issues

```sh
bb issue list myworkspace/myrepo
bb issue view myworkspace/myrepo 1
bb issue create myworkspace/myrepo --title "Bug" --priority critical
bb issue comment myworkspace/myrepo 1 --body "Fixed in #42"
```

## Configuration

```sh
bb config show                              # View current config
bb config set-workspace myworkspace         # Set default workspace
bb config set-format json                   # Set default output format (table, json)
```

Configuration and credentials are stored in `$XDG_CONFIG_HOME/bitbucket-cli/` (or `~/.config/bitbucket-cli/` on Linux, `~/Library/Application Support/bitbucket-cli/` on macOS, `%AppData%/bitbucket-cli/` on Windows).

## Output formats

Most list and view commands support a `--json` flag for machine-readable output:

```sh
bb pr list myworkspace/myrepo --json
bb auth status --json
```

The default output format is a human-readable table.

## Environment variables

| Variable           | Description                                      |
|--------------------|--------------------------------------------------|
| `BB_USERNAME`      | Bitbucket username (for app password auth)       |
| `BB_TOKEN`         | Bitbucket app password                           |
| `BB_HTTP_TIMEOUT`  | HTTP client timeout in seconds (default: 30)     |

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
