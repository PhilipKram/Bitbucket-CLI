package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestNewToolRegistry(t *testing.T) {
	registry := NewToolRegistry()
	if registry == nil {
		t.Fatal("NewToolRegistry returned nil")
	}

	if registry.Count() != 0 {
		t.Errorf("expected empty registry, got %d tools", registry.Count())
	}
}

func TestToolRegistry_Register(t *testing.T) {
	registry := NewToolRegistry()

	tool := Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"param1": NewStringProperty("First parameter"),
		}, []string{"param1"}),
	}

	handler := func(ctx context.Context, args map[string]interface{}) ([]Content, error) {
		return []Content{NewTextContent("test result")}, nil
	}

	err := registry.Register(tool, handler)
	if err != nil {
		t.Errorf("Register failed: %v", err)
	}

	if registry.Count() != 1 {
		t.Errorf("expected 1 tool, got %d", registry.Count())
	}

	rt := registry.Get("test_tool")
	if rt == nil {
		t.Fatal("Get returned nil for registered tool")
	}

	if rt.Tool.Name != "test_tool" {
		t.Errorf("expected tool name 'test_tool', got '%s'", rt.Tool.Name)
	}
}

func TestToolRegistry_RegisterEmptyName(t *testing.T) {
	registry := NewToolRegistry()

	tool := Tool{
		Name:        "",
		Description: "Invalid tool",
	}

	handler := func(ctx context.Context, args map[string]interface{}) ([]Content, error) {
		return nil, nil
	}

	err := registry.Register(tool, handler)
	if err == nil {
		t.Error("expected error for empty tool name, got nil")
	}
}

func TestToolRegistry_RegisterNilHandler(t *testing.T) {
	registry := NewToolRegistry()

	tool := Tool{
		Name:        "test_tool",
		Description: "A test tool",
	}

	err := registry.Register(tool, nil)
	if err == nil {
		t.Error("expected error for nil handler, got nil")
	}
}

func TestToolRegistry_RegisterReplacement(t *testing.T) {
	registry := NewToolRegistry()

	tool1 := Tool{
		Name:        "test_tool",
		Description: "First version",
	}

	handler1 := func(ctx context.Context, args map[string]interface{}) ([]Content, error) {
		return []Content{NewTextContent("v1")}, nil
	}

	tool2 := Tool{
		Name:        "test_tool",
		Description: "Second version",
	}

	handler2 := func(ctx context.Context, args map[string]interface{}) ([]Content, error) {
		return []Content{NewTextContent("v2")}, nil
	}

	registry.Register(tool1, handler1)
	registry.Register(tool2, handler2)

	if registry.Count() != 1 {
		t.Errorf("expected 1 tool after replacement, got %d", registry.Count())
	}

	rt := registry.Get("test_tool")
	if rt.Tool.Description != "Second version" {
		t.Errorf("expected tool to be replaced, got description '%s'", rt.Tool.Description)
	}
}

func TestToolRegistry_Unregister(t *testing.T) {
	registry := NewToolRegistry()

	tool := Tool{
		Name:        "test_tool",
		Description: "A test tool",
	}

	handler := func(ctx context.Context, args map[string]interface{}) ([]Content, error) {
		return nil, nil
	}

	registry.Register(tool, handler)

	// Unregister existing tool
	if !registry.Unregister("test_tool") {
		t.Error("Unregister returned false for existing tool")
	}

	if registry.Count() != 0 {
		t.Errorf("expected 0 tools after unregister, got %d", registry.Count())
	}

	// Unregister non-existent tool
	if registry.Unregister("nonexistent") {
		t.Error("Unregister returned true for non-existent tool")
	}
}

func TestToolRegistry_Get(t *testing.T) {
	registry := NewToolRegistry()

	tool := Tool{
		Name:        "test_tool",
		Description: "A test tool",
	}

	handler := func(ctx context.Context, args map[string]interface{}) ([]Content, error) {
		return nil, nil
	}

	registry.Register(tool, handler)

	// Get existing tool
	rt := registry.Get("test_tool")
	if rt == nil {
		t.Fatal("Get returned nil for existing tool")
	}

	// Get non-existent tool
	rt = registry.Get("nonexistent")
	if rt != nil {
		t.Error("Get returned non-nil for non-existent tool")
	}
}

func TestToolRegistry_List(t *testing.T) {
	registry := NewToolRegistry()

	// Empty registry
	tools := registry.List()
	if len(tools) != 0 {
		t.Errorf("expected 0 tools in empty registry, got %d", len(tools))
	}

	// Add tools
	handler := func(ctx context.Context, args map[string]interface{}) ([]Content, error) {
		return nil, nil
	}

	registry.Register(Tool{Name: "tool1", Description: "Tool 1"}, handler)
	registry.Register(Tool{Name: "tool2", Description: "Tool 2"}, handler)
	registry.Register(Tool{Name: "tool3", Description: "Tool 3"}, handler)

	tools = registry.List()
	if len(tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(tools))
	}

	// Verify all tool names are present
	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name] = true
	}

	expectedNames := []string{"tool1", "tool2", "tool3"}
	for _, name := range expectedNames {
		if !names[name] {
			t.Errorf("expected tool '%s' in list", name)
		}
	}
}

func TestToolRegistry_Execute(t *testing.T) {
	registry := NewToolRegistry()

	// Register a successful tool
	tool := Tool{
		Name:        "echo_tool",
		Description: "Echoes the input",
	}

	handler := func(ctx context.Context, args map[string]interface{}) ([]Content, error) {
		msg, ok := args["message"].(string)
		if !ok {
			return nil, fmt.Errorf("message parameter is required")
		}
		return []Content{NewTextContent(msg)}, nil
	}

	registry.Register(tool, handler)

	// Execute successfully
	ctx := context.Background()
	result := registry.Execute(ctx, "echo_tool", map[string]interface{}{
		"message": "Hello, World!",
	})

	if result.IsError {
		t.Errorf("expected successful execution, got error: %v", result.Content[0].Text)
	}

	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}

	if result.Content[0].Text != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got '%s'", result.Content[0].Text)
	}
}

func TestToolRegistry_ExecuteError(t *testing.T) {
	registry := NewToolRegistry()

	// Register a tool that returns an error
	tool := Tool{
		Name:        "error_tool",
		Description: "Always returns an error",
	}

	handler := func(ctx context.Context, args map[string]interface{}) ([]Content, error) {
		return nil, fmt.Errorf("intentional error")
	}

	registry.Register(tool, handler)

	// Execute and expect error
	ctx := context.Background()
	result := registry.Execute(ctx, "error_tool", nil)

	if !result.IsError {
		t.Error("expected error execution, got success")
	}

	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}

	if result.Content[0].Type != "text" {
		t.Errorf("expected text content, got '%s'", result.Content[0].Type)
	}
}

func TestToolRegistry_ExecuteNotFound(t *testing.T) {
	registry := NewToolRegistry()

	// Execute non-existent tool
	ctx := context.Background()
	result := registry.Execute(ctx, "nonexistent", nil)

	if !result.IsError {
		t.Error("expected error for non-existent tool")
	}

	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}

	expectedMsg := "Tool not found: nonexistent"
	if result.Content[0].Text != expectedMsg {
		t.Errorf("expected '%s', got '%s'", expectedMsg, result.Content[0].Text)
	}
}

func TestToolRegistry_Count(t *testing.T) {
	registry := NewToolRegistry()

	if registry.Count() != 0 {
		t.Errorf("expected 0 tools, got %d", registry.Count())
	}

	handler := func(ctx context.Context, args map[string]interface{}) ([]Content, error) {
		return nil, nil
	}

	registry.Register(Tool{Name: "tool1"}, handler)
	if registry.Count() != 1 {
		t.Errorf("expected 1 tool, got %d", registry.Count())
	}

	registry.Register(Tool{Name: "tool2"}, handler)
	if registry.Count() != 2 {
		t.Errorf("expected 2 tools, got %d", registry.Count())
	}

	registry.Unregister("tool1")
	if registry.Count() != 1 {
		t.Errorf("expected 1 tool after unregister, got %d", registry.Count())
	}
}

func TestToolRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewToolRegistry()

	handler := func(ctx context.Context, args map[string]interface{}) ([]Content, error) {
		return []Content{NewTextContent("ok")}, nil
	}

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	// Concurrent registrations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				toolName := fmt.Sprintf("tool_%d_%d", id, j)
				tool := Tool{Name: toolName, Description: "Concurrent tool"}
				registry.Register(tool, handler)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				registry.List()
				registry.Count()
			}
		}()
	}

	wg.Wait()

	// Verify final state
	expectedCount := numGoroutines * numOperations
	if registry.Count() != expectedCount {
		t.Errorf("expected %d tools, got %d", expectedCount, registry.Count())
	}
}

func TestJSONSchemaHelpers(t *testing.T) {
	// Test NewJSONSchema
	schema := NewJSONSchema("object", map[string]interface{}{
		"name": NewStringProperty("User name"),
		"age":  NewNumberProperty("User age"),
	}, []string{"name"})

	if schema["type"] != "object" {
		t.Errorf("expected type 'object', got '%v'", schema["type"])
	}

	if schema["properties"] == nil {
		t.Error("expected properties to be set")
	}

	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("expected required to be []string")
	}

	if len(required) != 1 || required[0] != "name" {
		t.Errorf("expected required=['name'], got %v", required)
	}

	// Test NewStringProperty
	strProp := NewStringProperty("A string parameter")
	if strProp["type"] != "string" {
		t.Errorf("expected type 'string', got '%v'", strProp["type"])
	}

	// Test NewNumberProperty
	numProp := NewNumberProperty("A number parameter")
	if numProp["type"] != "number" {
		t.Errorf("expected type 'number', got '%v'", numProp["type"])
	}

	// Test NewBooleanProperty
	boolProp := NewBooleanProperty("A boolean parameter")
	if boolProp["type"] != "boolean" {
		t.Errorf("expected type 'boolean', got '%v'", boolProp["type"])
	}

	// Test NewObjectProperty
	objProp := NewObjectProperty("An object parameter", map[string]interface{}{
		"field1": NewStringProperty("Field 1"),
	}, []string{"field1"})
	if objProp["type"] != "object" {
		t.Errorf("expected type 'object', got '%v'", objProp["type"])
	}

	// Test NewArrayProperty
	arrProp := NewArrayProperty("An array parameter", map[string]interface{}{
		"type": "string",
	})
	if arrProp["type"] != "array" {
		t.Errorf("expected type 'array', got '%v'", arrProp["type"])
	}
}

func TestServer_SetRegistry(t *testing.T) {
	// Create a server
	server := NewServer("test-server", "1.0.0", "Test server")

	// Create and populate a registry
	registry := NewToolRegistry()
	tool := Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"param1": NewStringProperty("First parameter"),
		}, []string{"param1"}),
	}

	handler := func(ctx context.Context, args map[string]interface{}) ([]Content, error) {
		return []Content{NewTextContent("test result")}, nil
	}

	registry.Register(tool, handler)

	// Set the registry on the server
	server.SetRegistry(registry)

	// Verify the server's handlers are updated
	// We can test this by calling the handlers directly

	// Test tools/list handler
	listReq := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "tools/list",
		Params:  map[string]interface{}{},
	}

	listHandler := server.handlers["tools/list"]
	if listHandler == nil {
		t.Fatal("tools/list handler not registered")
	}

	listResult, err := listHandler(listReq)
	if err != nil {
		t.Errorf("tools/list handler failed: %v", err)
	}

	// Verify the result contains our tool
	tools, ok := listResult["tools"].([]interface{})
	if !ok {
		t.Fatal("expected tools to be a slice")
	}

	if len(tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(tools))
	}

	// Test tools/call handler
	callReq := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      "test_tool",
			"arguments": map[string]interface{}{"param1": "value1"},
		},
	}

	callHandler := server.handlers["tools/call"]
	if callHandler == nil {
		t.Fatal("tools/call handler not registered")
	}

	callResult, err := callHandler(callReq)
	if err != nil {
		t.Errorf("tools/call handler failed: %v", err)
	}

	// Verify the result
	isError, ok := callResult["isError"].(bool)
	if ok && isError {
		t.Error("expected successful tool call, got error")
	}

	content, ok := callResult["content"].([]interface{})
	if !ok {
		t.Fatal("expected content to be a slice")
	}

	if len(content) != 1 {
		t.Errorf("expected 1 content item, got %d", len(content))
	}
}

// PR Tool Tests

func TestPRTools_ToolDefinitions(t *testing.T) {
	// Test PR List tool definition
	listTool := NewPRListTool()
	if listTool.Name != "pr_list" {
		t.Errorf("expected name 'pr_list', got '%s'", listTool.Name)
	}
	if listTool.Description == "" {
		t.Error("expected non-empty description")
	}
	if listTool.InputSchema == nil {
		t.Error("expected input schema to be set")
	}

	// Test PR View tool definition
	viewTool := NewPRViewTool()
	if viewTool.Name != "pr_view" {
		t.Errorf("expected name 'pr_view', got '%s'", viewTool.Name)
	}
	if viewTool.Description == "" {
		t.Error("expected non-empty description")
	}
	if viewTool.InputSchema == nil {
		t.Error("expected input schema to be set")
	}

	// Test PR Create tool definition
	createTool := NewPRCreateTool()
	if createTool.Name != "pr_create" {
		t.Errorf("expected name 'pr_create', got '%s'", createTool.Name)
	}
	if createTool.Description == "" {
		t.Error("expected non-empty description")
	}
	if createTool.InputSchema == nil {
		t.Error("expected input schema to be set")
	}
}

func TestPRTools_HandlerValidation(t *testing.T) {
	ctx := context.Background()

	// Test PR List handler parameter validation
	t.Run("PRList_MissingRepository", func(t *testing.T) {
		_, err := PRListHandler(ctx, map[string]interface{}{})
		if err == nil {
			t.Error("expected error for missing repository parameter")
		}
		if !strings.Contains(err.Error(), "repository") {
			t.Errorf("expected error message to mention repository, got: %v", err)
		}
	})

	// Test PR View handler parameter validation
	t.Run("PRView_MissingRepository", func(t *testing.T) {
		_, err := PRViewHandler(ctx, map[string]interface{}{
			"pr_id": "1",
		})
		if err == nil {
			t.Error("expected error for missing repository parameter")
		}
		if !strings.Contains(err.Error(), "repository") {
			t.Errorf("expected error message to mention repository, got: %v", err)
		}
	})

	t.Run("PRView_MissingPRID", func(t *testing.T) {
		_, err := PRViewHandler(ctx, map[string]interface{}{
			"repository": "workspace/repo",
		})
		if err == nil {
			t.Error("expected error for missing pr_id parameter")
		}
		if !strings.Contains(err.Error(), "pr_id") {
			t.Errorf("expected error message to mention pr_id, got: %v", err)
		}
	})

	// Test PR Create handler parameter validation
	t.Run("PRCreate_MissingRepository", func(t *testing.T) {
		_, err := PRCreateHandler(ctx, map[string]interface{}{
			"title":  "Test PR",
			"source": "feature-branch",
		})
		if err == nil {
			t.Error("expected error for missing repository parameter")
		}
		if !strings.Contains(err.Error(), "repository") {
			t.Errorf("expected error message to mention repository, got: %v", err)
		}
	})

	t.Run("PRCreate_MissingTitle", func(t *testing.T) {
		_, err := PRCreateHandler(ctx, map[string]interface{}{
			"repository": "workspace/repo",
			"source":     "feature-branch",
		})
		if err == nil {
			t.Error("expected error for missing title parameter")
		}
		if !strings.Contains(err.Error(), "title") {
			t.Errorf("expected error message to mention title, got: %v", err)
		}
	})

	t.Run("PRCreate_MissingSource", func(t *testing.T) {
		_, err := PRCreateHandler(ctx, map[string]interface{}{
			"repository": "workspace/repo",
			"title":      "Test PR",
		})
		if err == nil {
			t.Error("expected error for missing source parameter")
		}
		if !strings.Contains(err.Error(), "source") {
			t.Errorf("expected error message to mention source, got: %v", err)
		}
	})
}

func TestPRTools_RegistryIntegration(t *testing.T) {
	// Create a registry and register all PR tools
	registry := NewToolRegistry()

	err := registry.Register(NewPRListTool(), PRListHandler)
	if err != nil {
		t.Errorf("failed to register pr_list tool: %v", err)
	}

	err = registry.Register(NewPRViewTool(), PRViewHandler)
	if err != nil {
		t.Errorf("failed to register pr_view tool: %v", err)
	}

	err = registry.Register(NewPRCreateTool(), PRCreateHandler)
	if err != nil {
		t.Errorf("failed to register pr_create tool: %v", err)
	}

	// Verify all tools are registered
	if registry.Count() != 3 {
		t.Errorf("expected 3 tools registered, got %d", registry.Count())
	}

	// Verify each tool can be retrieved
	tools := []string{"pr_list", "pr_view", "pr_create"}
	for _, toolName := range tools {
		rt := registry.Get(toolName)
		if rt == nil {
			t.Errorf("tool %s not found in registry", toolName)
		}
	}

	// Verify tools appear in list
	toolList := registry.List()
	if len(toolList) != 3 {
		t.Errorf("expected 3 tools in list, got %d", len(toolList))
	}

	toolNames := make(map[string]bool)
	for _, tool := range toolList {
		toolNames[tool.Name] = true
	}

	for _, expectedName := range tools {
		if !toolNames[expectedName] {
			t.Errorf("expected tool %s in list", expectedName)
		}
	}
}

// TestToolSchemas validates that all tool definitions have proper JSON schemas.
func TestToolSchemas(t *testing.T) {
	// Define all tools that should be validated
	tools := []struct {
		name     string
		tool     Tool
		required []string
	}{
		{
			name:     "pr_list",
			tool:     NewPRListTool(),
			required: []string{"repository"},
		},
		{
			name:     "pr_view",
			tool:     NewPRViewTool(),
			required: []string{"repository", "pr_id"},
		},
		{
			name:     "pr_create",
			tool:     NewPRCreateTool(),
			required: []string{"repository", "title", "source"},
		},
		{
			name:     "issue_list",
			tool:     NewIssueListTool(),
			required: []string{"repository"},
		},
		{
			name:     "issue_create",
			tool:     NewIssueCreateTool(),
			required: []string{"repository", "title"},
		},
		{
			name:     "pipeline_list",
			tool:     NewPipelineListTool(),
			required: []string{"repository"},
		},
		{
			name:     "pipeline_trigger",
			tool:     NewPipelineTriggerTool(),
			required: []string{"repository"},
		},
	}

	for _, tc := range tools {
		t.Run(tc.name, func(t *testing.T) {
			// Verify tool has a name
			if tc.tool.Name == "" {
				t.Error("tool name is empty")
			}

			// Verify tool has a description
			if tc.tool.Description == "" {
				t.Error("tool description is empty")
			}

			// Verify tool has an input schema
			if tc.tool.InputSchema == nil {
				t.Fatal("tool input schema is nil")
			}

			schema := tc.tool.InputSchema

			// Verify schema has type "object"
			schemaType, ok := schema["type"].(string)
			if !ok {
				t.Fatal("schema type is not a string")
			}
			if schemaType != "object" {
				t.Errorf("expected schema type 'object', got '%s'", schemaType)
			}

			// Verify schema has properties
			properties, ok := schema["properties"].(map[string]interface{})
			if !ok {
				t.Fatal("schema properties is not a map")
			}
			if len(properties) == 0 {
				t.Error("schema has no properties")
			}

			// Verify required fields exist
			if len(tc.required) > 0 {
				requiredFields, ok := schema["required"].([]string)
				if !ok {
					t.Fatal("schema required is not a []string")
				}

				// Check that all expected required fields are present
				requiredMap := make(map[string]bool)
				for _, field := range requiredFields {
					requiredMap[field] = true
				}

				for _, expectedField := range tc.required {
					if !requiredMap[expectedField] {
						t.Errorf("expected required field '%s' not found in schema", expectedField)
					}
				}

				// Verify all required fields exist in properties
				for _, field := range requiredFields {
					if _, exists := properties[field]; !exists {
						t.Errorf("required field '%s' not found in properties", field)
					}
				}
			}

			// Verify each property has a type
			for propName, propValue := range properties {
				propMap, ok := propValue.(map[string]interface{})
				if !ok {
					t.Errorf("property '%s' is not a map", propName)
					continue
				}

				propType, ok := propMap["type"].(string)
				if !ok {
					t.Errorf("property '%s' has no type or type is not a string", propName)
					continue
				}

				// Verify type is valid
				validTypes := map[string]bool{
					"string":  true,
					"number":  true,
					"boolean": true,
					"object":  true,
					"array":   true,
					"integer": true,
					"null":    true,
				}
				if !validTypes[propType] {
					t.Errorf("property '%s' has invalid type '%s'", propName, propType)
				}
			}
		})
	}
}

// Issue Tool Tests

func TestIssueTools_ToolDefinitions(t *testing.T) {
	// Test Issue List tool definition
	listTool := NewIssueListTool()
	if listTool.Name != "issue_list" {
		t.Errorf("expected name 'issue_list', got '%s'", listTool.Name)
	}
	if listTool.Description == "" {
		t.Error("expected non-empty description")
	}
	if listTool.InputSchema == nil {
		t.Error("expected input schema to be set")
	}

	// Test Issue Create tool definition
	createTool := NewIssueCreateTool()
	if createTool.Name != "issue_create" {
		t.Errorf("expected name 'issue_create', got '%s'", createTool.Name)
	}
	if createTool.Description == "" {
		t.Error("expected non-empty description")
	}
	if createTool.InputSchema == nil {
		t.Error("expected input schema to be set")
	}
}

func TestIssueTools_HandlerValidation(t *testing.T) {
	ctx := context.Background()

	// Test Issue List handler parameter validation
	t.Run("IssueList_MissingRepository", func(t *testing.T) {
		_, err := IssueListHandler(ctx, map[string]interface{}{})
		if err == nil {
			t.Error("expected error for missing repository parameter")
		}
		if !strings.Contains(err.Error(), "repository") {
			t.Errorf("expected error message to mention repository, got: %v", err)
		}
	})

	// Test Issue Create handler parameter validation
	t.Run("IssueCreate_MissingRepository", func(t *testing.T) {
		_, err := IssueCreateHandler(ctx, map[string]interface{}{
			"title": "Test Issue",
		})
		if err == nil {
			t.Error("expected error for missing repository parameter")
		}
		if !strings.Contains(err.Error(), "repository") {
			t.Errorf("expected error message to mention repository, got: %v", err)
		}
	})

	t.Run("IssueCreate_MissingTitle", func(t *testing.T) {
		_, err := IssueCreateHandler(ctx, map[string]interface{}{
			"repository": "workspace/repo",
		})
		if err == nil {
			t.Error("expected error for missing title parameter")
		}
		if !strings.Contains(err.Error(), "title") {
			t.Errorf("expected error message to mention title, got: %v", err)
		}
	})
}

func TestIssueTools_RegistryIntegration(t *testing.T) {
	// Create a registry and register all Issue tools
	registry := NewToolRegistry()

	err := registry.Register(NewIssueListTool(), IssueListHandler)
	if err != nil {
		t.Errorf("failed to register issue_list tool: %v", err)
	}

	err = registry.Register(NewIssueCreateTool(), IssueCreateHandler)
	if err != nil {
		t.Errorf("failed to register issue_create tool: %v", err)
	}

	// Verify all tools are registered
	if registry.Count() != 2 {
		t.Errorf("expected 2 tools registered, got %d", registry.Count())
	}

	// Verify each tool can be retrieved
	tools := []string{"issue_list", "issue_create"}
	for _, toolName := range tools {
		rt := registry.Get(toolName)
		if rt == nil {
			t.Errorf("tool %s not found in registry", toolName)
		}
	}

	// Verify tools appear in list
	toolList := registry.List()
	if len(toolList) != 2 {
		t.Errorf("expected 2 tools in list, got %d", len(toolList))
	}

	toolNames := make(map[string]bool)
	for _, tool := range toolList {
		toolNames[tool.Name] = true
	}

	for _, expectedName := range tools {
		if !toolNames[expectedName] {
			t.Errorf("expected tool %s in list", expectedName)
		}
	}
}

func TestRegisterDefaultTools(t *testing.T) {
	registry := NewToolRegistry()

	err := RegisterDefaultTools(registry)
	if err != nil {
		t.Fatalf("RegisterDefaultTools failed: %v", err)
	}

	// Verify all expected tools are registered
	expectedTools := []string{
		"pr_list", "pr_view", "pr_create",
		"issue_list", "issue_create",
		"pipeline_list", "pipeline_trigger",
	}

	if registry.Count() != len(expectedTools) {
		t.Errorf("expected %d tools, got %d", len(expectedTools), registry.Count())
	}

	for _, toolName := range expectedTools {
		rt := registry.Get(toolName)
		if rt == nil {
			t.Errorf("expected tool %s to be registered", toolName)
		}
	}
}
