package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestServerInitialize(t *testing.T) {
	// Create input with initialize request
	initReq := Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": ProtocolVersion,
			"capabilities": map[string]interface{}{
				"roots": map[string]interface{}{
					"listChanged": true,
				},
			},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	reqBytes, err := json.Marshal(initReq)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	input := bytes.NewReader(append(reqBytes, '\n'))
	output := &bytes.Buffer{}

	server := NewServerWith(input, output, "bb-mcp", "1.0.0", "Bitbucket CLI MCP Server")

	// Process one request
	if err := server.processOneRequest(); err != nil {
		t.Fatalf("Failed to process request: %v", err)
	}

	// Parse response
	var resp Response
	if err := json.NewDecoder(output).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response
	if resp.JSONRPC != "2.0" {
		t.Errorf("Expected jsonrpc '2.0', got %s", resp.JSONRPC)
	}

	if string(resp.ID) != "1" {
		t.Errorf("Expected id 1, got %s", resp.ID)
	}

	if resp.Result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Check result contains expected fields
	if _, ok := resp.Result["protocolVersion"]; !ok {
		t.Error("Result missing protocolVersion")
	}

	if _, ok := resp.Result["capabilities"]; !ok {
		t.Error("Result missing capabilities")
	}

	if _, ok := resp.Result["serverInfo"]; !ok {
		t.Error("Result missing serverInfo")
	}
}

func TestServerToolsList(t *testing.T) {
	toolsReq := Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  "tools/list",
		Params:  map[string]interface{}{},
	}

	reqBytes, err := json.Marshal(toolsReq)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	input := bytes.NewReader(append(reqBytes, '\n'))
	output := &bytes.Buffer{}

	server := NewServerWith(input, output, "bb-mcp", "1.0.0", "Bitbucket CLI MCP Server")

	// Process one request
	if err := server.processOneRequest(); err != nil {
		t.Fatalf("Failed to process request: %v", err)
	}

	// Parse response
	var resp Response
	if err := json.NewDecoder(output).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response
	if resp.JSONRPC != "2.0" {
		t.Errorf("Expected jsonrpc '2.0', got %s", resp.JSONRPC)
	}

	if string(resp.ID) != "2" {
		t.Errorf("Expected id 2, got %s", resp.ID)
	}

	// Check result contains tools array
	if _, ok := resp.Result["tools"]; !ok {
		t.Error("Result missing tools array")
	}
}

func TestServerToolsCall(t *testing.T) {
	callReq := Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`3`),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      "test_tool",
			"arguments": map[string]interface{}{},
		},
	}

	reqBytes, err := json.Marshal(callReq)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	input := bytes.NewReader(append(reqBytes, '\n'))
	output := &bytes.Buffer{}

	server := NewServerWith(input, output, "bb-mcp", "1.0.0", "Bitbucket CLI MCP Server")

	// Process one request
	if err := server.processOneRequest(); err != nil {
		t.Fatalf("Failed to process request: %v", err)
	}

	// Parse response
	var resp Response
	if err := json.NewDecoder(output).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response
	if resp.JSONRPC != "2.0" {
		t.Errorf("Expected jsonrpc '2.0', got %s", resp.JSONRPC)
	}

	if string(resp.ID) != "3" {
		t.Errorf("Expected id 3, got %s", resp.ID)
	}

	// For now, tools/call should return an error result since no tools are implemented
	if _, ok := resp.Result["content"]; !ok {
		t.Error("Result missing content array")
	}

	if _, ok := resp.Result["isError"]; !ok {
		t.Error("Result missing isError field")
	}
}

func TestServerMethodNotFound(t *testing.T) {
	req := Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`4`),
		Method:  "unknown/method",
		Params:  map[string]interface{}{},
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	input := bytes.NewReader(append(reqBytes, '\n'))
	output := &bytes.Buffer{}

	server := NewServerWith(input, output, "bb-mcp", "1.0.0", "Bitbucket CLI MCP Server")

	// Process one request
	if err := server.processOneRequest(); err != nil {
		t.Fatalf("Failed to process request: %v", err)
	}

	// Parse error response
	var errResp ErrorResponse
	if err := json.NewDecoder(output).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	// Verify error response
	if errResp.JSONRPC != "2.0" {
		t.Errorf("Expected jsonrpc '2.0', got %s", errResp.JSONRPC)
	}

	if errResp.Error.Code != MethodNotFound {
		t.Errorf("Expected error code %d, got %d", MethodNotFound, errResp.Error.Code)
	}

	if !strings.Contains(errResp.Error.Message, "Method not found") {
		t.Errorf("Expected 'Method not found' in error message, got: %s", errResp.Error.Message)
	}
}

func TestServerInvalidJSON(t *testing.T) {
	input := bytes.NewReader([]byte("invalid json\n"))
	output := &bytes.Buffer{}

	server := NewServerWith(input, output, "bb-mcp", "1.0.0", "Bitbucket CLI MCP Server")

	// Process one request
	if err := server.processOneRequest(); err != nil {
		t.Fatalf("Failed to process request: %v", err)
	}

	// Parse error response
	var errResp ErrorResponse
	if err := json.NewDecoder(output).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	// Verify error response
	if errResp.Error.Code != ParseError {
		t.Errorf("Expected error code %d, got %d", ParseError, errResp.Error.Code)
	}
}

func TestServerInvalidJSONRPCVersion(t *testing.T) {
	req := Request{
		JSONRPC: "1.0",
		ID:      json.RawMessage(`5`),
		Method:  "initialize",
		Params:  map[string]interface{}{},
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	input := bytes.NewReader(append(reqBytes, '\n'))
	output := &bytes.Buffer{}

	server := NewServerWith(input, output, "bb-mcp", "1.0.0", "Bitbucket CLI MCP Server")

	// Process one request
	if err := server.processOneRequest(); err != nil {
		t.Fatalf("Failed to process request: %v", err)
	}

	// Parse error response
	var errResp ErrorResponse
	if err := json.NewDecoder(output).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	// Verify error response
	if errResp.Error.Code != InvalidRequest {
		t.Errorf("Expected error code %d, got %d", InvalidRequest, errResp.Error.Code)
	}
}

func TestServerNotification(t *testing.T) {
	// Create a notification (request without ID)
	notif := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "initialized",
		"params":  map[string]interface{}{},
	}

	notifBytes, err := json.Marshal(notif)
	if err != nil {
		t.Fatalf("Failed to marshal notification: %v", err)
	}

	input := bytes.NewReader(append(notifBytes, '\n'))
	output := &bytes.Buffer{}

	server := NewServerWith(input, output, "bb-mcp", "1.0.0", "Bitbucket CLI MCP Server")

	// Process one request
	if err := server.processOneRequest(); err != nil {
		t.Fatalf("Failed to process notification: %v", err)
	}

	// Notifications should not produce any output
	if output.Len() > 0 {
		t.Errorf("Expected no output for notification, got: %s", output.String())
	}

	// Verify server is marked as initialized
	if !server.initialized {
		t.Error("Expected server to be initialized after 'initialized' notification")
	}
}

func TestServerCustomHandler(t *testing.T) {
	input := bytes.NewReader([]byte{})
	output := &bytes.Buffer{}

	server := NewServerWith(input, output, "bb-mcp", "1.0.0", "Bitbucket CLI MCP Server")

	// Register custom handler
	called := false
	server.RegisterHandler("custom/method", func(req *Request) (map[string]interface{}, error) {
		called = true
		return map[string]interface{}{
			"success": true,
			"data":    "custom response",
		}, nil
	})

	// Create request
	req := Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`6`),
		Method:  "custom/method",
		Params:  map[string]interface{}{},
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	input = bytes.NewReader(append(reqBytes, '\n'))
	server.reader = bufio.NewReader(bytes.NewReader(append(reqBytes, '\n')))

	// Process one request
	if err := server.processOneRequest(); err != nil {
		t.Fatalf("Failed to process request: %v", err)
	}

	// Verify handler was called
	if !called {
		t.Error("Custom handler was not called")
	}

	// Parse response
	var resp Response
	if err := json.NewDecoder(output).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response contains custom data
	if success, ok := resp.Result["success"].(bool); !ok || !success {
		t.Error("Expected success: true in result")
	}

	if data, ok := resp.Result["data"].(string); !ok || data != "custom response" {
		t.Errorf("Expected data: 'custom response', got: %v", resp.Result["data"])
	}
}
