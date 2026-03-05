# CLAUDE.md - Project Guide for Claude Code

## Project Overview

`bb` is a CLI tool for Bitbucket Cloud, written in Go. It uses the Bitbucket Cloud REST API v2.0 and supports OAuth 2.0 authentication.

## Build & Test

```sh
go build -o bb .          # Build
go test ./...             # Run all tests
go test ./internal/api/   # Test a specific package
go vet ./...              # Lint
```

## Project Structure

```
cmd/                  # CLI command definitions (one package per command group)
  auth/               # bb auth
  repo/               # bb repo (aliases: repository)
  pr/                 # bb pr (aliases: pull-request)
  pipeline/           # bb pipeline (aliases: pipe, ci)
  issue/              # bb issue
  branch/             # bb branch
  completion/         # bb completion
  config/             # bb config
  mcp/                # bb mcp
  snippet/            # bb snippet
  user/               # bb user
  workspace/          # bb workspace (aliases: ws)
internal/             # Shared internal packages
  api/                # HTTP client, Bitbucket API methods, request/response types
  auth/               # OAuth flow, token storage, refresh
  buildinfo/          # Version info (injected at build time via ldflags)
  cmdutil/            # Shared command helpers (repo arg resolution, editor support)
  completion/         # Dynamic shell completion (workspaces, repos, branches, PRs)
  config/             # Config file management (XDG-based paths)
  errors/             # Structured error types and user-friendly messages
  git/                # Git operations (remote detection, clone)
  mcp/                # Model Context Protocol server (JSON-RPC 2.0 over stdin/stdout)
  output/             # Table and JSON output formatting
  update/             # Background update checker
docs/                 # Documentation and landing page
```

## Key Conventions

- **CLI framework**: [cobra](https://github.com/spf13/cobra) for commands and flags
- **No external HTTP library**: Uses `net/http` stdlib with a thin wrapper in `internal/api/`
- **Repository argument format**: `workspace/repo-slug` (positional arg on most commands)
- **Output**: Table format by default, `--json` flag for machine-readable output
- **Error handling**: Use `internal/errors` types (`APIError`, `AuthError`, etc.) with user-friendly messages
- **Config storage**: XDG-based paths via `internal/config.ConfigDir()`
- **Authentication**: OAuth 2.0 tokens stored in `token.json`, auto-refresh supported
- **Minimal dependencies**: Only cobra + stdlib. No third-party HTTP, JSON, or testing libraries.

## Adding a New Command

1. Create a new package under `cmd/<name>/`
2. Define a `NewCmd()` function returning `*cobra.Command`
3. Register it in `cmd/root.go` via `rootCmd.AddCommand()`
4. Use `internal/api` for API calls, `internal/output` for formatting
5. Add dynamic completions in `internal/completion/` if needed

## Adding a New MCP Tool

1. Add the tool definition in `internal/mcp/tools.go`
2. Register it in the tools registry with schema and handler
3. Update `docs/mcp.md` with the new tool documentation

## Git Workflow

- Main branch: `main`
- Releases: Tag with `vX.Y.Z`, GoReleaser builds via GitHub Actions
- Do not commit or push without explicit user confirmation
