package mcp

import (
	"context"
	"fmt"
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
		ID:      1,
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
		ID:      2,
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
