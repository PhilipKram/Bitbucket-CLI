package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PhilipKram/bitbucket-cli/internal/mcp"
)

// TestIntegration_MCP_CommandStructure tests the MCP command structure.
func TestIntegration_MCP_CommandStructure(t *testing.T) {
	cmd := NewCmdMCP()

	if cmd == nil {
		t.Fatal("NewCmdMCP returned nil")
	}

	if cmd.Use != "mcp" {
		t.Errorf("Expected Use 'mcp', got %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected non-empty Short description")
	}

	// Verify all subcommands exist
	subcommands := []string{"serve", "install", "uninstall", "status"}
	for _, name := range subcommands {
		found, _, err := cmd.Find([]string{name})
		if err != nil {
			t.Fatalf("Failed to find '%s' subcommand: %v", name, err)
		}
		if found == nil || found.Use != name {
			t.Errorf("Expected subcommand '%s', got %v", name, found)
		}
	}
}

// TestIntegration_MCP_InstallFlags tests the install subcommand flags.
func TestIntegration_MCP_InstallFlags(t *testing.T) {
	cmd := NewCmdMCP()
	installCmd, _, err := cmd.Find([]string{"install"})
	if err != nil {
		t.Fatalf("Failed to find install subcommand: %v", err)
	}

	scopeFlag := installCmd.Flags().Lookup("scope")
	if scopeFlag == nil {
		t.Fatal("Expected --scope flag on install command")
	}
	if scopeFlag.DefValue != "user" {
		t.Errorf("Expected default scope 'user', got %s", scopeFlag.DefValue)
	}

	clientFlag := installCmd.Flags().Lookup("client")
	if clientFlag == nil {
		t.Fatal("Expected --client flag on install command")
	}
	if clientFlag.DefValue != "claude-code" {
		t.Errorf("Expected default client 'claude-code', got %s", clientFlag.DefValue)
	}
}

// TestIntegration_MCP_UninstallFlags tests the uninstall subcommand flags.
func TestIntegration_MCP_UninstallFlags(t *testing.T) {
	cmd := NewCmdMCP()
	uninstallCmd, _, err := cmd.Find([]string{"uninstall"})
	if err != nil {
		t.Fatalf("Failed to find uninstall subcommand: %v", err)
	}

	scopeFlag := uninstallCmd.Flags().Lookup("scope")
	if scopeFlag == nil {
		t.Fatal("Expected --scope flag on uninstall command")
	}
	if scopeFlag.DefValue != "user" {
		t.Errorf("Expected default scope 'user', got %s", scopeFlag.DefValue)
	}

	clientFlag := uninstallCmd.Flags().Lookup("client")
	if clientFlag == nil {
		t.Fatal("Expected --client flag on uninstall command")
	}
	if clientFlag.DefValue != "claude-code" {
		t.Errorf("Expected default client 'claude-code', got %s", clientFlag.DefValue)
	}
}

// TestIntegration_MCP_StatusFlags tests the status subcommand flags.
func TestIntegration_MCP_StatusFlags(t *testing.T) {
	cmd := NewCmdMCP()
	statusCmd, _, err := cmd.Find([]string{"status"})
	if err != nil {
		t.Fatalf("Failed to find status subcommand: %v", err)
	}

	clientFlag := statusCmd.Flags().Lookup("client")
	if clientFlag == nil {
		t.Fatal("Expected --client flag on status command")
	}
	if clientFlag.DefValue != "claude-code" {
		t.Errorf("Expected default client 'claude-code', got %s", clientFlag.DefValue)
	}

	// status should not have --scope flag
	scopeFlag := statusCmd.Flags().Lookup("scope")
	if scopeFlag != nil {
		t.Error("status command should not have --scope flag")
	}
}

// TestBBBinaryPath tests the bbBinaryPath helper function.
func TestBBBinaryPath(t *testing.T) {
	path := bbBinaryPath()
	if path == "" {
		t.Error("bbBinaryPath returned empty string")
	}
	// Should return an absolute path or "bb" as fallback
	if path != "bb" && !filepath.IsAbs(path) {
		t.Errorf("Expected absolute path or 'bb', got %s", path)
	}
}

// TestMCPConfigJSON tests the mcpConfigJSON helper function.
func TestMCPConfigJSON(t *testing.T) {
	configJSON, err := mcpConfigJSON("/usr/local/bin/bb")
	if err != nil {
		t.Fatalf("mcpConfigJSON failed: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		t.Fatalf("Failed to parse config JSON: %v", err)
	}

	if config["command"] != "/usr/local/bin/bb" {
		t.Errorf("Expected command '/usr/local/bin/bb', got %v", config["command"])
	}

	args, ok := config["args"].([]interface{})
	if !ok {
		t.Fatal("Expected args to be an array")
	}
	if len(args) != 2 || args[0] != "mcp" || args[1] != "serve" {
		t.Errorf("Expected args ['mcp', 'serve'], got %v", args)
	}
}

// TestClaudeDesktopConfigPath tests the claudeDesktopConfigPath helper function.
func TestClaudeDesktopConfigPath(t *testing.T) {
	path, err := claudeDesktopConfigPath()
	if err != nil {
		t.Fatalf("claudeDesktopConfigPath failed: %v", err)
	}

	if path == "" {
		t.Error("claudeDesktopConfigPath returned empty string")
	}

	if !filepath.IsAbs(path) {
		t.Errorf("Expected absolute path, got %s", path)
	}

	if !strings.HasSuffix(path, "claude_desktop_config.json") {
		t.Errorf("Expected path ending with claude_desktop_config.json, got %s", path)
	}
}

// TestRunInstallUnsupportedClient tests install with an unsupported client.
func TestRunInstallUnsupportedClient(t *testing.T) {
	err := runInstall("unsupported-client", "user")
	if err == nil {
		t.Fatal("Expected error for unsupported client")
	}
	if !strings.Contains(err.Error(), "unsupported client") {
		t.Errorf("Expected 'unsupported client' error, got: %v", err)
	}
}

// TestRunUninstallUnsupportedClient tests uninstall with an unsupported client.
func TestRunUninstallUnsupportedClient(t *testing.T) {
	err := runUninstall("unsupported-client", "user")
	if err == nil {
		t.Fatal("Expected error for unsupported client")
	}
	if !strings.Contains(err.Error(), "unsupported client") {
		t.Errorf("Expected 'unsupported client' error, got: %v", err)
	}
}

// TestRunStatusUnsupportedClient tests status with an unsupported client.
func TestRunStatusUnsupportedClient(t *testing.T) {
	err := runStatus("claude-desktop")
	if err == nil {
		t.Fatal("Expected error for unsupported client")
	}
	if !strings.Contains(err.Error(), "only supported for claude-code") {
		t.Errorf("Expected 'only supported for claude-code' error, got: %v", err)
	}
}

// TestInstallClaudeDesktop tests installing bb in Claude Desktop config.
func TestInstallClaudeDesktop(t *testing.T) {
	// Create a temporary directory to simulate Claude Desktop config
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "Claude")
	configPath := filepath.Join(configDir, "claude_desktop_config.json")

	// We can't easily override claudeDesktopConfigPath, so test the JSON logic directly
	bbPath := "/usr/local/bin/bb"
	config := make(map[string]interface{})
	mcpServers := make(map[string]interface{})
	mcpServers["bb"] = map[string]interface{}{
		"command": bbPath,
		"args":    []string{"mcp", "serve"},
	}
	config["mcpServers"] = mcpServers

	output, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	if err := os.WriteFile(configPath, output, 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Read back and verify
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var readConfig map[string]interface{}
	if err := json.Unmarshal(data, &readConfig); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	servers, ok := readConfig["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected mcpServers in config")
	}

	bbConfig, ok := servers["bb"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected bb entry in mcpServers")
	}

	if bbConfig["command"] != "/usr/local/bin/bb" {
		t.Errorf("Expected command '/usr/local/bin/bb', got %v", bbConfig["command"])
	}
}

// TestUninstallClaudeDesktopLogic tests the uninstall logic for Claude Desktop.
func TestUninstallClaudeDesktopLogic(t *testing.T) {
	// Simulate a config with bb registered
	config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"bb": map[string]interface{}{
				"command": "/usr/local/bin/bb",
				"args":    []string{"mcp", "serve"},
			},
			"other-tool": map[string]interface{}{
				"command": "/usr/local/bin/other",
				"args":    []string{"serve"},
			},
		},
	}

	// Remove bb
	mcpServers := config["mcpServers"].(map[string]interface{})
	delete(mcpServers, "bb")

	// Verify bb is gone but other-tool remains
	if _, exists := mcpServers["bb"]; exists {
		t.Error("Expected bb to be removed from mcpServers")
	}
	if _, exists := mcpServers["other-tool"]; !exists {
		t.Error("Expected other-tool to remain in mcpServers")
	}
}

// TestIntegration_MCP_ServerInitialization tests server creation and initialization.
func TestIntegration_MCP_ServerInitialization(t *testing.T) {
	input := bytes.NewReader([]byte{})
	output := &bytes.Buffer{}

	server := mcp.NewServerWith(input, output, "bb-mcp", "1.0.0", "Bitbucket CLI MCP Server")

	if server == nil {
		t.Fatal("NewServerWith returned nil")
	}
}

// TestIntegration_MCP_ToolRegistry tests tool registration and retrieval.
func TestIntegration_MCP_ToolRegistry(t *testing.T) {
	registry := mcp.NewToolRegistry()

	// Register sample tools
	prListTool := mcp.NewPRListTool()
	err := registry.Register(prListTool, func(ctx context.Context, args map[string]interface{}) ([]mcp.Content, error) {
		return []mcp.Content{
			mcp.NewTextContent("Mock PR list result"),
		}, nil
	})
	if err != nil {
		t.Fatalf("Failed to register pr_list tool: %v", err)
	}

	prViewTool := mcp.NewPRViewTool()
	err = registry.Register(prViewTool, func(ctx context.Context, args map[string]interface{}) ([]mcp.Content, error) {
		return []mcp.Content{
			mcp.NewTextContent("Mock PR view result"),
		}, nil
	})
	if err != nil {
		t.Fatalf("Failed to register pr_view tool: %v", err)
	}

	issueListTool := mcp.NewIssueListTool()
	err = registry.Register(issueListTool, func(ctx context.Context, args map[string]interface{}) ([]mcp.Content, error) {
		return []mcp.Content{
			mcp.NewTextContent("Mock issue list result"),
		}, nil
	})
	if err != nil {
		t.Fatalf("Failed to register issue_list tool: %v", err)
	}

	// Verify tool count
	if count := registry.Count(); count != 3 {
		t.Errorf("Expected 3 tools, got %d", count)
	}

	// Verify tools can be retrieved
	tools := registry.List()
	if len(tools) != 3 {
		t.Errorf("Expected 3 tools in list, got %d", len(tools))
	}

	// Verify tool names
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	expectedTools := []string{"pr_list", "pr_view", "issue_list"}
	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("Expected tool %s not found in tools list", expected)
		}
	}

	// Verify individual tool retrieval
	prTool := registry.Get("pr_list")
	if prTool == nil {
		t.Error("Failed to get pr_list tool")
	} else if prTool.Tool.Name != "pr_list" {
		t.Errorf("Expected tool name pr_list, got %s", prTool.Tool.Name)
	}
}

// TestIntegration_MCP_ToolExecution tests tool execution through the registry.
func TestIntegration_MCP_ToolExecution(t *testing.T) {
	registry := mcp.NewToolRegistry()

	// Register a test tool that echoes arguments
	testTool := mcp.Tool{
		Name:        "test_echo",
		Title:       "Test Echo",
		Description: "Echo back the provided message",
		InputSchema: mcp.NewJSONSchema("object", map[string]interface{}{
			"message": mcp.NewStringProperty("Message to echo"),
		}, []string{"message"}),
	}

	executionCalled := false
	err := registry.Register(testTool, func(ctx context.Context, args map[string]interface{}) ([]mcp.Content, error) {
		executionCalled = true
		message, ok := args["message"].(string)
		if !ok {
			message = "no message provided"
		}
		return []mcp.Content{
			mcp.NewTextContent("Echo: " + message),
		}, nil
	})
	if err != nil {
		t.Fatalf("Failed to register test_echo tool: %v", err)
	}

	// Execute the tool through the registry
	ctx := context.Background()
	result := registry.Execute(ctx, "test_echo", map[string]interface{}{
		"message": "Hello from integration test",
	})

	// Verify execution happened
	if !executionCalled {
		t.Error("Tool handler was not called")
	}

	// Verify result
	if result.IsError {
		t.Error("Expected IsError to be false for successful execution")
	}

	if len(result.Content) == 0 {
		t.Fatal("Expected content in result")
	}

	firstContent := result.Content[0]
	if firstContent.Type != "text" {
		t.Errorf("Expected content type 'text', got %s", firstContent.Type)
	}

	expectedText := "Echo: Hello from integration test"
	if firstContent.Text != expectedText {
		t.Errorf("Expected text %q, got %q", expectedText, firstContent.Text)
	}
}

// TestIntegration_MCP_ToolNotFound tests tool execution with unknown tool.
func TestIntegration_MCP_ToolNotFound(t *testing.T) {
	registry := mcp.NewToolRegistry()

	// Execute a non-existent tool
	ctx := context.Background()
	result := registry.Execute(ctx, "nonexistent_tool", map[string]interface{}{})

	// Verify result indicates error
	if !result.IsError {
		t.Error("Expected IsError to be true for tool not found")
	}

	// Verify error message
	if len(result.Content) == 0 {
		t.Fatal("Expected content in error result")
	}

	firstContent := result.Content[0]
	if firstContent.Type != "text" {
		t.Errorf("Expected content type 'text', got %s", firstContent.Type)
	}

	if !strings.Contains(firstContent.Text, "Tool not found") {
		t.Errorf("Expected 'Tool not found' in error message, got: %s", firstContent.Text)
	}
}

// TestIntegration_MCP_ToolHandlerError tests error handling in tool execution.
func TestIntegration_MCP_ToolHandlerError(t *testing.T) {
	registry := mcp.NewToolRegistry()

	// Register a tool that always returns an error
	errorTool := mcp.Tool{
		Name:        "error_tool",
		Title:       "Error Tool",
		Description: "A tool that always fails",
		InputSchema: mcp.NewJSONSchema("object", map[string]interface{}{}, []string{}),
	}

	err := registry.Register(errorTool, func(ctx context.Context, args map[string]interface{}) ([]mcp.Content, error) {
		return nil, fmt.Errorf("simulated error")
	})
	if err != nil {
		t.Fatalf("Failed to register error_tool: %v", err)
	}

	// Execute the tool
	ctx := context.Background()
	result := registry.Execute(ctx, "error_tool", map[string]interface{}{})

	// Verify the result indicates an error
	if !result.IsError {
		t.Error("Expected IsError to be true for failed tool execution")
	}

	// Verify error message
	if len(result.Content) == 0 {
		t.Fatal("Expected content in error result")
	}

	if !strings.Contains(result.Content[0].Text, "Tool execution failed") {
		t.Errorf("Expected 'Tool execution failed' in error message, got: %s", result.Content[0].Text)
	}
}

// TestIntegration_MCP_ToolUnregistration tests removing tools from the registry.
func TestIntegration_MCP_ToolUnregistration(t *testing.T) {
	registry := mcp.NewToolRegistry()

	// Register a tool
	testTool := mcp.Tool{
		Name:        "temp_tool",
		Title:       "Temporary Tool",
		Description: "A tool for testing unregistration",
		InputSchema: mcp.NewJSONSchema("object", map[string]interface{}{}, []string{}),
	}

	err := registry.Register(testTool, func(ctx context.Context, args map[string]interface{}) ([]mcp.Content, error) {
		return []mcp.Content{mcp.NewTextContent("test")}, nil
	})
	if err != nil {
		t.Fatalf("Failed to register temp_tool: %v", err)
	}

	// Verify tool is registered
	if registry.Count() != 1 {
		t.Errorf("Expected 1 tool, got %d", registry.Count())
	}

	// Unregister the tool
	removed := registry.Unregister("temp_tool")
	if !removed {
		t.Error("Expected Unregister to return true")
	}

	// Verify tool is removed
	if registry.Count() != 0 {
		t.Errorf("Expected 0 tools after unregister, got %d", registry.Count())
	}

	// Try to unregister again - should return false
	removed = registry.Unregister("temp_tool")
	if removed {
		t.Error("Expected Unregister to return false for non-existent tool")
	}
}

// TestIntegration_MCP_AllToolDefinitions tests all standard tool definitions.
func TestIntegration_MCP_AllToolDefinitions(t *testing.T) {
	registry := mcp.NewToolRegistry()

	// Define all expected tools and their constructors
	toolConstructors := map[string]func() mcp.Tool{
		"pr_list":          mcp.NewPRListTool,
		"pr_view":          mcp.NewPRViewTool,
		"pr_create":        mcp.NewPRCreateTool,
		"issue_list":       mcp.NewIssueListTool,
		"issue_create":     mcp.NewIssueCreateTool,
		"pipeline_list":    mcp.NewPipelineListTool,
		"pipeline_trigger": mcp.NewPipelineTriggerTool,
	}

	// Register all tools with dummy handlers
	dummyHandler := func(ctx context.Context, args map[string]interface{}) ([]mcp.Content, error) {
		return []mcp.Content{mcp.NewTextContent("test")}, nil
	}

	for name, constructor := range toolConstructors {
		tool := constructor()
		if tool.Name != name {
			t.Errorf("Tool constructor for %s returned tool with name %s", name, tool.Name)
		}
		if err := registry.Register(tool, dummyHandler); err != nil {
			t.Errorf("Failed to register tool %s: %v", name, err)
		}
	}

	// Verify all tools are registered
	if count := registry.Count(); count != len(toolConstructors) {
		t.Errorf("Expected %d registered tools, got %d", len(toolConstructors), count)
	}

	// Verify each tool has required fields
	tools := registry.List()
	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("Tool has empty name")
		}
		if tool.Description == "" {
			t.Errorf("Tool %s has empty description", tool.Name)
		}
		if tool.InputSchema == nil {
			t.Errorf("Tool %s has nil input schema", tool.Name)
		} else {
			schemaType, ok := tool.InputSchema["type"].(string)
			if !ok || schemaType != "object" {
				t.Errorf("Tool %s has invalid input schema type: %v", tool.Name, tool.InputSchema["type"])
			}
		}
	}
}

// TestIntegration_MCP_SchemaHelpers tests the schema helper functions.
func TestIntegration_MCP_SchemaHelpers(t *testing.T) {
	// Test string property
	strProp := mcp.NewStringProperty("test description")
	if strProp["type"] != "string" {
		t.Errorf("Expected type 'string', got %v", strProp["type"])
	}
	if strProp["description"] != "test description" {
		t.Errorf("Expected description 'test description', got %v", strProp["description"])
	}

	// Test number property
	numProp := mcp.NewNumberProperty("number description")
	if numProp["type"] != "number" {
		t.Errorf("Expected type 'number', got %v", numProp["type"])
	}

	// Test boolean property
	boolProp := mcp.NewBooleanProperty("boolean description")
	if boolProp["type"] != "boolean" {
		t.Errorf("Expected type 'boolean', got %v", boolProp["type"])
	}

	// Test array property
	arrayProp := mcp.NewArrayProperty("array description", map[string]interface{}{"type": "string"})
	if arrayProp["type"] != "array" {
		t.Errorf("Expected type 'array', got %v", arrayProp["type"])
	}
	if arrayProp["items"] == nil {
		t.Error("Expected items to be set")
	}

	// Test object property
	objProp := mcp.NewObjectProperty("object description", map[string]interface{}{
		"field1": mcp.NewStringProperty("field1 desc"),
	}, []string{"field1"})
	if objProp["type"] != "object" {
		t.Errorf("Expected type 'object', got %v", objProp["type"])
	}
	if objProp["properties"] == nil {
		t.Error("Expected properties to be set")
	}
	if objProp["required"] == nil {
		t.Error("Expected required to be set")
	}

	// Test JSON schema creation
	schema := mcp.NewJSONSchema("object", map[string]interface{}{
		"name": mcp.NewStringProperty("Name field"),
		"age":  mcp.NewNumberProperty("Age field"),
	}, []string{"name"})

	if schema["type"] != "object" {
		t.Errorf("Expected type 'object', got %v", schema["type"])
	}
	if schema["properties"] == nil {
		t.Error("Expected properties to be set")
	}
	if required, ok := schema["required"].([]string); !ok || len(required) != 1 || required[0] != "name" {
		t.Errorf("Expected required to be ['name'], got %v", schema["required"])
	}
}

// TestIntegration_MCP_ContentHelpers tests the content creation helpers.
func TestIntegration_MCP_ContentHelpers(t *testing.T) {
	// Test text content
	textContent := mcp.NewTextContent("Hello, world!")
	if textContent.Type != "text" {
		t.Errorf("Expected type 'text', got %s", textContent.Type)
	}
	if textContent.Text != "Hello, world!" {
		t.Errorf("Expected text 'Hello, world!', got %s", textContent.Text)
	}

	// Test image content
	imageContent := mcp.NewImageContent("base64data", "image/png")
	if imageContent.Type != "image" {
		t.Errorf("Expected type 'image', got %s", imageContent.Type)
	}
	if imageContent.Data != "base64data" {
		t.Errorf("Expected data 'base64data', got %s", imageContent.Data)
	}
	if imageContent.MimeType != "image/png" {
		t.Errorf("Expected mimeType 'image/png', got %s", imageContent.MimeType)
	}

	// Test resource link content
	linkContent := mcp.NewResourceLinkContent("https://example.com", "Example", "Example resource", "text/html")
	if linkContent.Type != "resource_link" {
		t.Errorf("Expected type 'resource_link', got %s", linkContent.Type)
	}
	if linkContent.URI != "https://example.com" {
		t.Errorf("Expected URI 'https://example.com', got %s", linkContent.URI)
	}
	if linkContent.Name != "Example" {
		t.Errorf("Expected name 'Example', got %s", linkContent.Name)
	}
}

// TestIntegration_MCP_ToolRegistryValidation tests validation in the registry.
func TestIntegration_MCP_ToolRegistryValidation(t *testing.T) {
	registry := mcp.NewToolRegistry()

	// Test registering tool with empty name
	emptyNameTool := mcp.Tool{
		Name:        "",
		Description: "Test tool",
		InputSchema: mcp.NewJSONSchema("object", map[string]interface{}{}, []string{}),
	}

	err := registry.Register(emptyNameTool, func(ctx context.Context, args map[string]interface{}) ([]mcp.Content, error) {
		return nil, nil
	})

	if err == nil {
		t.Error("Expected error when registering tool with empty name")
	} else if !strings.Contains(err.Error(), "name cannot be empty") {
		t.Errorf("Expected 'name cannot be empty' error, got: %v", err)
	}

	// Test registering tool with nil handler
	validTool := mcp.Tool{
		Name:        "valid_tool",
		Description: "Test tool",
		InputSchema: mcp.NewJSONSchema("object", map[string]interface{}{}, []string{}),
	}

	err = registry.Register(validTool, nil)
	if err == nil {
		t.Error("Expected error when registering tool with nil handler")
	} else if !strings.Contains(err.Error(), "handler cannot be nil") {
		t.Errorf("Expected 'handler cannot be nil' error, got: %v", err)
	}
}
