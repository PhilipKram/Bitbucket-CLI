package mcp

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	authPkg "github.com/PhilipKram/bitbucket-cli/internal/auth"
	"github.com/PhilipKram/bitbucket-cli/internal/buildinfo"
	"github.com/PhilipKram/bitbucket-cli/internal/config"
	"github.com/PhilipKram/bitbucket-cli/internal/mcp"
)

func NewCmdMCP() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Model Context Protocol server",
	}

	cmd.AddCommand(newCmdServe())
	cmd.AddCommand(newCmdInstall())
	cmd.AddCommand(newCmdUninstall())
	cmd.AddCommand(newCmdStatus())

	return cmd
}

// createMCPServer creates a fully configured MCP server with tools, resources, and prompts.
func createMCPServer() (*mcp.Server, error) {
	server := mcp.NewServer(
		"bb-mcp",
		buildinfo.Version,
		"Bitbucket CLI MCP server - exposes bb commands as MCP tools",
	)

	registry := mcp.NewToolRegistry()
	if err := mcp.RegisterDefaultTools(registry); err != nil {
		return nil, err
	}

	server.SetRegistry(registry)
	mcp.RegisterDefaultResources(server)
	mcp.RegisterDefaultPrompts(server)

	return server, nil
}

func newCmdServe() *cobra.Command {
	var (
		transport    string
		port         int
		host         string
		token        string
		noAuth       bool
		basePath     string
		clientID     string
		clientSecret string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start MCP server",
		Long: `Start an MCP (Model Context Protocol) server.

Supports two transports:
  stdio  - Standard I/O (default), for use as a subprocess or Docker container
  http   - HTTP with Server-Sent Events, for remote/networked access

Bitbucket authentication:
  --client-id / --client-secret  OAuth credentials for Bitbucket API access.
  When provided, the server runs the OAuth browser flow at startup to
  obtain a token before serving MCP requests. This is designed for Docker
  containers where the callback port (8817) is exposed to the host.

HTTP authentication modes:
  --token / auto-generated  Single shared bearer token (default)
  --no-auth                 Disable authentication (not recommended)`,
		Example: `  # Start over stdio (default, used by Docker)
  $ bb mcp serve

  # Start with Bitbucket OAuth (Docker)
  $ docker run -it --rm -p 8817:8817 -p 8080:8080 bb-mcp \
      mcp serve --transport http --host 0.0.0.0 --no-auth \
      --client-id KEY --client-secret SECRET

  # Start as HTTP server with auto-generated token
  $ bb mcp serve --transport http

  # Start on custom host and port
  $ bb mcp serve --transport http --host 0.0.0.0 --port 9090`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If OAuth credentials provided, authenticate first
			if clientID != "" && clientSecret != "" {
				if err := authenticateBitbucket(clientID, clientSecret); err != nil {
					return fmt.Errorf("Bitbucket authentication failed: %w", err)
				}
			}

			switch transport {
			case "stdio":
				server, err := createMCPServer()
				if err != nil {
					return err
				}
				return server.Start()
			case "http":
				server, err := createMCPServer()
				if err != nil {
					return err
				}
				return serveHTTP(server, host, port, token, noAuth, basePath)
			default:
				return fmt.Errorf("unsupported transport: %s (supported: stdio, http)", transport)
			}
		},
	}

	cmd.Flags().StringVar(&transport, "transport", "stdio", "Transport: stdio or http")
	cmd.Flags().IntVar(&port, "port", 8080, "HTTP listen port")
	cmd.Flags().StringVar(&host, "host", "localhost", "HTTP listen address")
	cmd.Flags().StringVar(&token, "token", "", "Bearer token for HTTP auth (auto-generated if empty)")
	cmd.Flags().BoolVar(&noAuth, "no-auth", false, "Disable authentication for HTTP transport")
	cmd.Flags().StringVar(&basePath, "base-path", "/mcp", "HTTP endpoint path")
	cmd.Flags().StringVar(&clientID, "client-id", "", "Bitbucket OAuth consumer key")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "Bitbucket OAuth consumer secret")

	return cmd
}

// authenticateBitbucket runs the OAuth 2.0 flow to obtain a Bitbucket access token.
// It saves both the OAuth credentials and the resulting token to the config directory
// so that api.NewClient() can pick them up.
func authenticateBitbucket(clientID, clientSecret string) error {
	// Save OAuth credentials so token refresh works later
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	cfg.OAuthKey = clientID
	cfg.OAuthSecret = clientSecret
	if err := config.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save OAuth credentials: %w", err)
	}

	// Run the OAuth browser flow
	fmt.Fprintln(os.Stderr, "Starting Bitbucket OAuth authentication...")
	fmt.Fprintln(os.Stderr, "Open the URL below in your browser to authorize:")
	fmt.Fprintln(os.Stderr)

	token, err := authPkg.Login(clientID, clientSecret)
	if err != nil {
		return err
	}

	if err := config.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Fprintln(os.Stderr, "Bitbucket authentication successful.")
	return nil
}

// serveHTTP starts the MCP server over HTTP with optional bearer token authentication.
func serveHTTP(server *mcp.Server, host string, port int, token string, noAuth bool, basePath string) error {
	// Handle authentication token — reuse a persisted token when possible
	// so that MCP clients stay authenticated across restarts.
	if !noAuth && token == "" {
		if saved := loadMCPToken(); saved != "" {
			token = saved
		} else {
			var err error
			token, err = generateToken()
			if err != nil {
				return fmt.Errorf("generating auth token: %w", err)
			}
			if err := saveMCPToken(token); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not persist token: %v\n", err)
			}
		}
	}

	if noAuth {
		fmt.Fprintln(os.Stderr, "WARNING: running without authentication — anyone with network access can use this server")
	}

	handler := mcp.NewHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.HTTPHandlerOptions{})

	mux := http.NewServeMux()
	var h http.Handler = handler
	if !noAuth {
		h = bearerAuthMiddleware(handler, token)
	}
	mux.Handle(basePath, h)

	addr := fmt.Sprintf("%s:%d", host, port)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Print server info
	url := fmt.Sprintf("http://%s%s", addr, basePath)
	fmt.Fprintf(os.Stderr, "bb MCP server running on %s\n", url)
	if !noAuth {
		fmt.Fprintf(os.Stderr, "Auth token: %s\n", token)
	}

	// Graceful shutdown on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("HTTP server error: %w", err)
		}
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, "\nShutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown error: %w", err)
		}
	}

	return nil
}

// bearerAuthMiddleware wraps an http.Handler to require a valid Bearer token.
func bearerAuthMiddleware(next http.Handler, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		provided := strings.TrimPrefix(auth, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// generateToken returns a cryptographically random 64-character hex string.
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// mcpTokenPath returns the path to the persisted MCP bearer token file.
func mcpTokenPath() string {
	dir, err := config.ConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "mcp_token")
}

// loadMCPToken reads a previously saved MCP bearer token from disk.
func loadMCPToken() string {
	path := mcpTokenPath()
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// saveMCPToken persists the MCP bearer token to disk so it survives restarts.
func saveMCPToken(token string) error {
	path := mcpTokenPath()
	if path == "" {
		return fmt.Errorf("could not determine config directory")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	return os.WriteFile(path, []byte(token+"\n"), 0o600)
}

// DefaultDockerImage is the default Docker image used for MCP installation.
const DefaultDockerImage = "bb-mcp"

func newCmdInstall() *cobra.Command {
	var (
		scope      string
		client     string
		transport  string
		host       string
		port       int
		basePath   string
		token      string
		dockerImage string
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Register bb as an MCP server in an AI client",
		Long: `Register bb as an MCP server in an AI client configuration.

Supported clients:
  claude-code     - Claude Code CLI (default)
  claude-desktop  - Claude Desktop application

Supports two transport modes:
  stdio  - Docker container via stdio (default)
  http   - Remote HTTP server

For stdio transport, the MCP server runs as a Docker container.
For http transport, the server must already be running.

Supported scopes (claude-code only):
  user    - User-level configuration (default)
  local   - Local project configuration
  project - Project-level configuration`,
		Example: `  # Install for Claude Code using Docker (default)
  $ bb mcp install

  # Install with a custom Docker image
  $ bb mcp install --docker-image ghcr.io/myorg/bb-mcp:latest

  # Install for Claude Desktop using Docker
  $ bb mcp install --client claude-desktop

  # Install a remote HTTP MCP server
  $ bb mcp install --transport http --host myserver.example.com --port 8080 --token my-secret`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var configJSON string
			var err error

			switch transport {
			case "stdio":
				configJSON, err = mcpDockerConfigJSON(dockerImage)
				if err != nil {
					return err
				}
			case "http":
				// Use persisted token if none provided explicitly
				if token == "" {
					token = loadMCPToken()
				}
				configJSON, err = mcpRemoteConfigJSON(host, port, basePath, token)
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported transport: %s (supported: stdio, http)", transport)
			}

			switch client {
			case "claude-code":
				return installClaudeCode(scope, configJSON)
			case "claude-desktop":
				if transport == "http" {
					return installClaudeDesktopRemote(host, port, basePath, token)
				}
				return installClaudeDesktopDocker(dockerImage)
			default:
				return fmt.Errorf("unsupported client: %s (supported: claude-code, claude-desktop)", client)
			}
		},
	}

	cmd.Flags().StringVar(&scope, "scope", "user", "Configuration scope: user, local, or project (claude-code only)")
	cmd.Flags().StringVar(&client, "client", "claude-code", "AI client: claude-code or claude-desktop")
	cmd.Flags().StringVar(&transport, "transport", "stdio", "Transport: stdio or http")
	cmd.Flags().StringVar(&dockerImage, "docker-image", DefaultDockerImage, "Docker image for the MCP server")
	cmd.Flags().StringVar(&host, "host", "localhost", "Remote MCP server host (http transport)")
	cmd.Flags().IntVar(&port, "port", 8080, "Remote MCP server port (http transport)")
	cmd.Flags().StringVar(&basePath, "base-path", "/mcp", "Remote MCP server endpoint path (http transport)")
	cmd.Flags().StringVar(&token, "token", "", "Bearer token for remote MCP server (http transport)")

	return cmd
}

func newCmdUninstall() *cobra.Command {
	var scope string
	var client string

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove bb MCP server from an AI client",
		Long: `Remove bb as an MCP server from an AI client configuration.

Supported clients:
  claude-code     - Claude Code CLI (default)
  claude-desktop  - Claude Desktop application`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstall(client, scope)
		},
	}

	cmd.Flags().StringVar(&scope, "scope", "user", "Configuration scope: user, local, or project (claude-code only)")
	cmd.Flags().StringVar(&client, "client", "claude-code", "AI client: claude-code or claude-desktop")

	return cmd
}

func newCmdStatus() *cobra.Command {
	var client string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check if bb is registered as an MCP server",
		Long: `Check if bb is registered as an MCP server in an AI client.

Currently only supports claude-code.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(client)
		},
	}

	cmd.Flags().StringVar(&client, "client", "claude-code", "AI client: claude-code")

	return cmd
}

// mcpDockerConfigJSON builds a Docker-based MCP server configuration JSON for stdio transport.
func mcpDockerConfigJSON(image string) (string, error) {
	cfg := map[string]interface{}{
		"command": "docker",
		"args":    []string{"run", "-i", "--rm", image, "mcp", "serve"},
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal MCP config: %w", err)
	}
	return string(data), nil
}

// mcpRemoteConfigJSON builds a remote HTTP MCP server configuration JSON.
func mcpRemoteConfigJSON(host string, port int, basePath, token string) (string, error) {
	url := fmt.Sprintf("http://%s:%d%s", host, port, basePath)
	cfg := map[string]interface{}{
		"url": url,
	}
	if token != "" {
		cfg["headers"] = map[string]string{
			"Authorization": "Bearer " + token,
		}
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal remote MCP config: %w", err)
	}
	return string(data), nil
}

// claudeDesktopConfigPath returns the platform-specific path for the Claude Desktop config file.
func claudeDesktopConfigPath() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"), nil
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		return filepath.Join(appData, "Claude", "claude_desktop_config.json"), nil
	case "linux":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		return filepath.Join(home, ".config", "Claude", "claude_desktop_config.json"), nil
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// findClaude looks up the claude binary in PATH.
func findClaude() (string, error) {
	path, err := exec.LookPath("claude")
	if err != nil {
		return "", fmt.Errorf("claude CLI not found in PATH: %w", err)
	}
	return path, nil
}

func installClaudeCode(scope, configJSON string) error {
	claudePath, err := findClaude()
	if err != nil {
		return err
	}

	//nolint:gosec // Arguments are constructed from trusted sources
	cmd := exec.Command(claudePath, "mcp", "add-json", "--scope", scope, "bb", configJSON)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to register bb in Claude Code: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Successfully registered bb as an MCP server in Claude Code (scope: %s)\n", scope)
	return nil
}

// installClaudeDesktopDocker registers a Docker-based bb MCP server in Claude Desktop.
func installClaudeDesktopDocker(image string) error {
	configPath, err := claudeDesktopConfigPath()
	if err != nil {
		return err
	}

	var cfg map[string]interface{}
	data, err := os.ReadFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		cfg = make(map[string]interface{})
	} else {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	mcpServers, ok := cfg["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
	}

	mcpServers["bb"] = map[string]interface{}{
		"command": "docker",
		"args":    []string{"run", "-i", "--rm", image, "mcp", "serve"},
	}
	cfg["mcpServers"] = mcpServers

	output, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(configPath, output, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Successfully registered bb as an MCP server in Claude Desktop\n")
	fmt.Fprintf(os.Stderr, "Config file: %s\n", configPath)
	return nil
}

// installClaudeDesktopRemote registers a remote bb MCP server in Claude Desktop.
func installClaudeDesktopRemote(host string, port int, basePath, token string) error {
	configPath, err := claudeDesktopConfigPath()
	if err != nil {
		return err
	}

	var cfg map[string]interface{}
	data, err := os.ReadFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		cfg = make(map[string]interface{})
	} else {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	mcpServers, ok := cfg["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
	}

	url := fmt.Sprintf("http://%s:%d%s", host, port, basePath)
	entry := map[string]interface{}{
		"url": url,
	}
	if token != "" {
		entry["headers"] = map[string]string{
			"Authorization": "Bearer " + token,
		}
	}
	mcpServers["bb"] = entry
	cfg["mcpServers"] = mcpServers

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	output, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, output, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Successfully registered bb remote MCP server in Claude Desktop\n")
	fmt.Fprintf(os.Stderr, "Config file: %s\n", configPath)
	return nil
}

func runUninstall(client, scope string) error {
	switch client {
	case "claude-code":
		return uninstallClaudeCode(scope)
	case "claude-desktop":
		return uninstallClaudeDesktop()
	default:
		return fmt.Errorf("unsupported client: %s (supported: claude-code, claude-desktop)", client)
	}
}

func uninstallClaudeCode(scope string) error {
	claudePath, err := findClaude()
	if err != nil {
		return err
	}

	//nolint:gosec // Arguments are constructed from trusted sources
	cmd := exec.Command(claudePath, "mcp", "remove", "bb", "--scope", scope)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove bb from Claude Code: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Successfully removed bb MCP server from Claude Code (scope: %s)\n", scope)
	return nil
}

func uninstallClaudeDesktop() error {
	configPath, err := claudeDesktopConfigPath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config file not found: %s", configPath)
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	mcpServers, ok := cfg["mcpServers"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("bb is not registered as an MCP server in Claude Desktop")
	}

	if _, exists := mcpServers["bb"]; !exists {
		return fmt.Errorf("bb is not registered as an MCP server in Claude Desktop")
	}

	delete(mcpServers, "bb")
	cfg["mcpServers"] = mcpServers

	output, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, output, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Successfully removed bb MCP server from Claude Desktop\n")
	fmt.Fprintf(os.Stderr, "Config file: %s\n", configPath)
	return nil
}

func runStatus(client string) error {
	if client != "claude-code" {
		return fmt.Errorf("status is currently only supported for claude-code")
	}

	claudePath, err := findClaude()
	if err != nil {
		return err
	}

	//nolint:gosec // Arguments are constructed from trusted sources
	cmd := exec.Command(claudePath, "mcp", "get", "bb")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bb is not registered as an MCP server in Claude Code (or failed to check): %w", err)
	}

	return nil
}
