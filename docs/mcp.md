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

**HTTP flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--host` | `localhost` | Listen address |
| `--port` | `8080` | Listen port |
| `--base-path` | `/mcp` | HTTP endpoint path |
| `--token` | (auto-generated) | Bearer token for authentication |
| `--no-auth` | `false` | Disable authentication |

The auto-generated token is persisted across restarts so MCP clients stay authenticated.

## Docker Deployment

A Dockerfile is included for running the MCP server in a container.

### Build the image

```sh
docker build -t bb-mcp .
```

### Run with stdio

```sh
docker run -i --rm bb-mcp mcp serve
```

### Run with HTTP and OAuth

```sh
docker run -it --rm \
  -p 8817:8817 -p 8080:8080 \
  bb-mcp mcp serve \
    --transport http --host 0.0.0.0 --no-auth \
    --client-id YOUR_OAUTH_KEY --client-secret YOUR_OAUTH_SECRET
```

The `--client-id` and `--client-secret` flags trigger the OAuth browser flow at startup. Port 8817 is the OAuth callback port and must be exposed to the host for the browser flow to work.

## Client Configuration

### Claude Code

Use the installer:

```sh
bb mcp install
```

Or install manually:

```sh
claude mcp add-json bb '{"command":"docker","args":["run","-i","--rm","bb-mcp","mcp","serve"]}'
```

For a remote HTTP server:

```sh
claude mcp add-json bb '{"url":"http://localhost:8080/mcp","headers":{"Authorization":"Bearer YOUR_TOKEN"}}'
```

### Claude Desktop

Use the installer:

```sh
bb mcp install --client claude-desktop
```

Or add manually to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

**Docker (stdio):**

```json
{
  "mcpServers": {
    "bb": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "bb-mcp", "mcp", "serve"]
    }
  }
}
```

**Remote HTTP:**

```json
{
  "mcpServers": {
    "bb": {
      "url": "http://localhost:8080/mcp",
      "headers": {
        "Authorization": "Bearer YOUR_TOKEN"
      }
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

The MCP server uses `bb`'s existing OAuth authentication. Before using MCP tools, ensure you're logged in:

```sh
bb auth login
```

The MCP server will use your stored OAuth credentials automatically. No additional authentication setup is required.

For Docker deployments, pass OAuth credentials at startup:

```sh
bb mcp serve --client-id YOUR_KEY --client-secret YOUR_SECRET
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

- The MCP server uses your existing `bb` OAuth credentials stored in your configuration directory
- All API operations are subject to the same permissions as your Bitbucket OAuth consumer
- The MCP server only exposes read and create operations - no delete operations are exposed
- **Stdio transport**: All communication is local (stdin/stdout) - no network services are exposed
- **HTTP transport**: Bearer token authentication is enabled by default. The token is persisted locally and uses constant-time comparison to prevent timing attacks
- Avoid using `--no-auth` in production or on shared networks

## Protocol Details

`bb` implements MCP version 1.0 with the following capabilities:

- **Protocol**: JSON-RPC 2.0
- **Transports**: stdin/stdout (stdio) or HTTP with Server-Sent Events (SSE)
- **Tool Discovery**: Full schema support for all tools
- **Authentication**: Transparent OAuth2 integration with existing `bb auth` system
- **HTTP Auth**: Bearer token (auto-generated or user-provided)
- **Error Handling**: Structured error responses with helpful messages

## Extending with Custom Tools

The MCP server currently exposes core Bitbucket operations. Additional tools can be added by extending the `internal/mcp/tools.go` registry. Future versions may support plugin-based tool extensions.

## Further Reading

- [Model Context Protocol Specification](https://modelcontextprotocol.io/)
- [Claude Desktop MCP Guide](https://docs.anthropic.com/claude/docs/model-context-protocol)
- [Bitbucket Cloud REST API](https://developer.atlassian.com/cloud/bitbucket/rest/)
