# MCP (Model Context Protocol) Integration

`bb` implements the [Model Context Protocol](https://modelcontextprotocol.io/) (MCP), allowing AI agents and LLM-powered development tools to interact with Bitbucket through `bb` as a tool provider.

## What is MCP?

MCP is an open protocol that standardizes how AI applications connect to external data sources and tools. By exposing `bb`'s capabilities as MCP tools, AI agents can programmatically create pull requests, trigger pipelines, manage issues, and more through natural language interactions.

## Quick Start

### 1. Start the MCP server

**Stdio (default)** — for use as a subprocess:

```sh
bb mcp serve
```

**HTTP** — for remote or networked access:

```sh
bb mcp serve --transport http
```

The stdio server communicates over stdin/stdout using JSON-RPC 2.0. The HTTP server uses Server-Sent Events (SSE) and listens on `localhost:8080/mcp` by default.

### 2. Configure your MCP client

Use the built-in installer for quick setup:

```sh
# Install for Claude Code (default)
bb mcp install

# Install for Claude Desktop
bb mcp install --client claude-desktop

# Install a remote HTTP server
bb mcp install --transport http --host myserver.example.com --port 8080 --token my-secret
```

Or configure manually — see [Client Configuration](#client-configuration) below.

## Transports

### Stdio (default)

The stdio transport runs `bb` as a subprocess. This is the simplest setup and is used by most MCP clients.

```sh
bb mcp serve
```

### HTTP with SSE

The HTTP transport exposes the MCP server over HTTP with Server-Sent Events, suitable for remote access, shared servers, or containerized deployments.

```sh
# Auto-generated bearer token (printed to stderr)
bb mcp serve --transport http

# Custom host and port
bb mcp serve --transport http --host 0.0.0.0 --port 9090

# Explicit token
bb mcp serve --transport http --token my-secret-token

# No authentication (not recommended)
bb mcp serve --transport http --no-auth
```

### HTTP with Per-User OAuth (recommended for shared servers)

The OAuth mode runs the MCP server as an **OAuth authorization server proxy**. Each user authenticates with their own Bitbucket account via the browser — no shared credentials, proper audit trail. Sessions are persisted to disk and survive restarts.

```sh
bb mcp serve --transport http --host 0.0.0.0 \
  --client-id YOUR_CONSUMER_KEY --client-secret YOUR_CONSUMER_SECRET \
  --external-url http://your-server:8080
```

MCP clients like Claude Code handle the OAuth flow automatically — users just open a browser to authorize.

**HTTP flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--host` | `localhost` | Listen address |
| `--port` | `8080` | Listen port |
| `--base-path` | `/mcp` | HTTP endpoint path |
| `--token` | (auto-generated) | Bearer token for authentication (shared token mode) |
| `--no-auth` | `false` | Disable authentication |
| `--client-id` | | Bitbucket OAuth consumer key (enables per-user OAuth mode) |
| `--client-secret` | | Bitbucket OAuth consumer secret |
| `--external-url` | | External URL for OAuth redirects (required for Docker) |

The auto-generated token is persisted across restarts so MCP clients stay authenticated.

## Docker Deployment

A Dockerfile is included for running the MCP server in a container.

### Build the image

```sh
docker build -t bb-mcp .
```

### Per-user OAuth mode (recommended for teams)

Run as a shared server where each user authenticates with their own Bitbucket account. The Dockerfile entrypoint defaults to HTTP transport, so you just pass the OAuth consumer credentials:

```sh
docker run -d --name bb-mcp \
  -p 8080:8080 -p 8817:8817 \
  bb-mcp \
  --client-id YOUR_CONSUMER_KEY \
  --client-secret YOUR_CONSUMER_SECRET \
  --external-url http://localhost:8080
```

Port 8817 is used for the OAuth callback (matching `bb auth login`). Sessions are persisted to the `/config` volume inside the container.

Users connect without needing `bb` installed:

```sh
claude mcp add --transport http --scope user bitbucket http://localhost:8080/mcp
```

Claude Code auto-discovers OAuth via `/.well-known/oauth-authorization-server` and handles the browser flow.

### Run with volume-mounted config

Mount your existing `bb` config directory for single-user deployments:

```sh
# macOS
docker run -d --rm -p 8080:8080 \
  -v "$HOME/Library/Application Support/bitbucket-cli:/config" \
  bb-mcp --no-auth

# Linux
docker run -d --rm -p 8080:8080 \
  -v "$HOME/.config/bitbucket-cli:/config" \
  bb-mcp --no-auth
```

### Run with OAuth env vars (private consumers only)

If your OAuth consumer is configured as **private** in Bitbucket workspace settings, you can use the `client_credentials` grant via environment variables:

```sh
docker run -d --rm -p 8080:8080 \
  -e BB_OAUTH_KEY=your-key -e BB_OAUTH_SECRET=your-secret \
  bb-mcp --no-auth
```

This obtains tokens automatically — no browser interaction required.

## Client Configuration

### Claude Code

**OAuth mode** (recommended — auto-discovers auth via the server):

```sh
claude mcp add --transport http --scope user bitbucket http://localhost:8080/mcp
```

**Shared token mode** (manual token):

```sh
claude mcp add-json bb '{"url":"http://localhost:8080/mcp","headers":{"Authorization":"Bearer YOUR_TOKEN"}}'
```

### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "bb": {
      "url": "http://localhost:8080/mcp"
    }
  }
}
```

### Cursor

Add to your Cursor MCP settings:

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

### Other MCP Clients

Any MCP-compatible client can connect via stdio:

```sh
bb mcp serve
```

Or via HTTP at `http://HOST:PORT/mcp` with a Bearer token.

## Managing MCP Registration

```sh
# Check registration status
bb mcp status

# Uninstall from Claude Code
bb mcp uninstall

# Uninstall from Claude Desktop
bb mcp uninstall --client claude-desktop
```

## Authentication

### HTTP transport authentication modes

| Mode | Flags | Description |
|------|-------|-------------|
| **Per-user OAuth** | `--client-id` + `--client-secret` | Each user authenticates with their own Bitbucket account via browser. Sessions persist across restarts. Recommended for shared/team servers. |
| **Shared bearer token** | `--token` or auto-generated | Single token shared by all clients. Simple but no per-user identity. |
| **No auth** | `--no-auth` | No authentication. Not recommended for production. |

### Bitbucket API authentication (resolved in order)

1. **Per-user OAuth session** — in OAuth mode, each request uses the authenticated user's Bitbucket token.
2. **`BB_OAUTH_KEY` + `BB_OAUTH_SECRET` env vars** — uses the OAuth 2.0 client_credentials grant. Requires a **private** OAuth consumer.
3. **Stored token** — from a previous `bb auth login`.

For local use, log in once:

```sh
bb auth login
```

## Available Tools

The MCP server exposes the following tools. AI agents can discover and invoke these tools automatically.

### Pull Requests

#### `pr_list`
List pull requests in a repository with optional state filtering.

**Parameters:**
- `repository` (required): Repository in format `workspace/repo-slug`
- `state` (optional): State filter - `OPEN`, `MERGED`, `DECLINED`, or `SUPERSEDED`
- `page` (optional): Page number (default: 1)

**Example:**
```
List all open pull requests in myworkspace/myrepo
```

#### `pr_view`
View detailed information about a specific pull request.

**Parameters:**
- `repository` (required): Repository in format `workspace/repo-slug`
- `pr_id` (required): Pull request ID

**Example:**
```
Show me details of PR #42 in myworkspace/myrepo
```

#### `pr_create`
Create a new pull request.

**Parameters:**
- `repository` (required): Repository in format `workspace/repo-slug`
- `title` (required): Pull request title
- `source` (required): Source branch name
- `description` (optional): Pull request description
- `destination` (optional): Destination branch (defaults to main branch)
- `close_branch` (optional): Close source branch after merge (default: false)

**Example:**
```
Create a pull request in myworkspace/myrepo from feature-branch to main with title "Add new feature"
```

### Issues

#### `issue_list`
List issues in a repository with optional state filtering.

**Parameters:**
- `repository` (required): Repository in format `workspace/repo-slug`
- `state` (optional): State filter - `new`, `open`, `resolved`, `on hold`, `invalid`, `duplicate`, `wontfix`, `closed`
- `page` (optional): Page number (default: 1)

**Example:**
```
List all open issues in myworkspace/myrepo
```

#### `issue_create`
Create a new issue.

**Parameters:**
- `repository` (required): Repository in format `workspace/repo-slug`
- `title` (required): Issue title
- `content` (optional): Issue description
- `kind` (optional): Issue kind - `bug`, `enhancement`, `proposal`, `task` (default: `bug`)
- `priority` (optional): Priority - `trivial`, `minor`, `major`, `critical`, `blocker` (default: `major`)

**Example:**
```
Create a bug issue in myworkspace/myrepo with title "Login fails on Safari" and critical priority
```

### Pipelines

#### `pipeline_list`
List CI/CD pipelines in a repository.

**Parameters:**
- `repository` (required): Repository in format `workspace/repo-slug`
- `page` (optional): Page number (default: 1)

**Example:**
```
Show me recent pipelines in myworkspace/myrepo
```

#### `pipeline_trigger`
Trigger a new CI/CD pipeline.

**Parameters:**
- `repository` (required): Repository in format `workspace/repo-slug`
- `branch` (optional): Branch to run pipeline on (default: `main`)
- `pattern` (optional): Custom pipeline pattern name
- `custom` (optional): Trigger a custom pipeline (default: false)

**Example:**
```
Trigger a pipeline on the develop branch in myworkspace/myrepo
```

## Usage Examples

Once configured, you can interact with Bitbucket through your AI agent using natural language:

### Creating a Pull Request
```
"Create a pull request in myworkspace/myrepo from feature/new-auth to main
with the title 'Add OAuth2 authentication' and description 'Implements OAuth2
login flow with token refresh'"
```

### Checking Pipeline Status
```
"Show me the recent pipeline runs for myworkspace/myrepo"
```

### Managing Issues
```
"Create a critical bug in myworkspace/myrepo titled 'Database connection timeout'
with description 'Users experiencing timeouts during peak hours'"
```

### Listing Open PRs
```
"What are the open pull requests in myworkspace/myrepo?"
```

## How It Works

1. **Tool Discovery**: When your MCP client connects to `bb mcp serve`, it receives a list of available tools and their schemas
2. **Natural Language Processing**: Your AI agent interprets your natural language request and maps it to the appropriate MCP tool
3. **Tool Invocation**: The AI agent calls the tool with the correct parameters
4. **Execution**: `bb` executes the Bitbucket API operation using your OAuth credentials
5. **Response**: The result is returned to the AI agent, which formats it in a user-friendly way

## Troubleshooting

### Authentication Errors

If you see authentication errors, ensure you're logged in:

```sh
bb auth status
```

If not authenticated, log in:

```sh
bb auth login
```

### Tool Not Found

Restart your MCP client after updating `bb` to the latest version to ensure tool definitions are current.

### Connection Issues (stdio)

The stdio MCP server communicates over stdin/stdout. Make sure your client is configured to execute `bb mcp serve` as a subprocess, not as a network service.

### Connection Issues (HTTP)

If using HTTP transport, verify:

1. The server is running: `curl http://localhost:8080/mcp`
2. The bearer token is correct in your client configuration
3. Firewall rules allow access to the configured port
4. For Docker, ensure ports are properly mapped (`-p 8080:8080`)

## Security Considerations

- The MCP server only exposes read and create operations — no delete operations are exposed
- **Stdio transport**: All communication is local (stdin/stdout) — no network services are exposed
- **OAuth mode**: Each user authenticates with their own Bitbucket account. Sessions are stored with `0600` permissions. The server never exposes Bitbucket tokens to clients — only session bearer tokens.
- **Shared token mode**: Bearer token uses constant-time comparison to prevent timing attacks. The token is persisted locally.
- **Docker**: OAuth consumer credentials are passed as flags or env vars — never stored in client configuration files. The `/config` volume stores sessions with restricted permissions.
- Avoid using `--no-auth` in production or on shared networks

## Protocol Details

`bb` implements MCP version 1.0 with the following capabilities:

- **Protocol**: JSON-RPC 2.0
- **Transports**: stdin/stdout (stdio) or HTTP with Server-Sent Events (SSE)
- **Tool Discovery**: Full schema support for all tools
- **Authentication**: Per-user OAuth proxy (RFC 8414) or shared bearer token
- **OAuth Discovery**: `/.well-known/oauth-authorization-server` for automatic client configuration
- **Dynamic Client Registration**: `POST /oauth/register` for MCP client onboarding
- **Error Handling**: Structured error responses with helpful messages

## Extending with Custom Tools

The MCP server currently exposes core Bitbucket operations. Additional tools can be added by extending the `internal/mcp/tools.go` registry. Future versions may support plugin-based tool extensions.

## Further Reading

- [Model Context Protocol Specification](https://modelcontextprotocol.io/)
- [Claude Desktop MCP Guide](https://docs.anthropic.com/claude/docs/model-context-protocol)
- [Bitbucket Cloud REST API](https://developer.atlassian.com/cloud/bitbucket/rest/)
