package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/PhilipKram/bitbucket-cli/internal/buildinfo"
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

func newCmdServe() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start MCP server",
		Long: `Start an MCP (Model Context Protocol) server on stdio.

The server exposes Bitbucket CLI capabilities as MCP tools that can be
invoked by AI agents and LLM-powered development tools.

The server communicates via JSON-RPC 2.0 over stdin/stdout.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create MCP server
			server := mcp.NewServer(
				"bb-mcp",
				buildinfo.Version,
				"Bitbucket CLI MCP server - exposes bb commands as MCP tools",
			)

			// Create tool registry and register default tools
			registry := mcp.NewToolRegistry()
			if err := mcp.RegisterDefaultTools(registry); err != nil {
				return err
			}

			// Set registry on server
			server.SetRegistry(registry)

			// Register default resources
			mcp.RegisterDefaultResources(server)

			// Register default prompts
			mcp.RegisterDefaultPrompts(server)

			return server.Start()
		},
	}

	return cmd
}

func newCmdInstall() *cobra.Command {
	var scope string
	var client string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Register bb as an MCP server in an AI client",
		Long: `Register bb as an MCP server in an AI client configuration.

Supported clients:
  claude-code     - Claude Code CLI (default)
  claude-desktop  - Claude Desktop application

Supported scopes (claude-code only):
  user    - User-level configuration (default)
  local   - Local project configuration
  project - Project-level configuration`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(client, scope)
		},
	}

	cmd.Flags().StringVar(&scope, "scope", "user", "Configuration scope: user, local, or project (claude-code only)")
	cmd.Flags().StringVar(&client, "client", "claude-code", "AI client: claude-code or claude-desktop")

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

// bbBinaryPath returns the absolute path to the running binary.
// Falls back to "bb" if the path cannot be determined.
func bbBinaryPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "bb"
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return exe
	}
	return resolved
}

// mcpConfigJSON builds the MCP server configuration JSON for the given binary path.
func mcpConfigJSON(bbPath string) (string, error) {
	config := map[string]interface{}{
		"command": bbPath,
		"args":    []string{"mcp", "serve"},
	}
	data, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal MCP config: %w", err)
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

func runInstall(client, scope string) error {
	switch client {
	case "claude-code":
		return installClaudeCode(scope)
	case "claude-desktop":
		return installClaudeDesktop()
	default:
		return fmt.Errorf("unsupported client: %s (supported: claude-code, claude-desktop)", client)
	}
}

func installClaudeCode(scope string) error {
	claudePath, err := findClaude()
	if err != nil {
		return err
	}

	bbPath := bbBinaryPath()
	configJSON, err := mcpConfigJSON(bbPath)
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

func installClaudeDesktop() error {
	configPath, err := claudeDesktopConfigPath()
	if err != nil {
		return err
	}

	bbPath := bbBinaryPath()

	// Read existing config or start with empty object
	var config map[string]interface{}
	data, err := os.ReadFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		config = make(map[string]interface{})
	} else {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Ensure mcpServers key exists
	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
	}

	// Add bb entry
	mcpServers["bb"] = map[string]interface{}{
		"command": bbPath,
		"args":    []string{"mcp", "serve"},
	}
	config["mcpServers"] = mcpServers

	// Write config back
	output, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure parent directory exists
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

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("bb is not registered as an MCP server in Claude Desktop")
	}

	if _, exists := mcpServers["bb"]; !exists {
		return fmt.Errorf("bb is not registered as an MCP server in Claude Desktop")
	}

	delete(mcpServers, "bb")
	config["mcpServers"] = mcpServers

	output, err := json.MarshalIndent(config, "", "  ")
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
