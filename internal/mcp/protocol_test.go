package mcp

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestRequest_EncodeDecode(t *testing.T) {
	tests := []struct {
		name string
		req  *Request
	}{
		{
			name: "request with string ID",
			req: &Request{
				JSONRPC: "2.0",
				ID:      "test-id-123",
				Method:  "initialize",
				Params: map[string]interface{}{
					"protocolVersion": ProtocolVersion,
					"capabilities":    map[string]interface{}{},
				},
			},
		},
		{
			name: "request with numeric ID",
			req: &Request{
				JSONRPC: "2.0",
				ID:      42,
				Method:  "tools/list",
				Params:  map[string]interface{}{},
			},
		},
		{
			name: "request without params",
			req: &Request{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "ping",
			},
		},
		{
			name: "request with complex params",
			req: &Request{
				JSONRPC: "2.0",
				ID:      "complex-1",
				Method:  "tools/call",
				Params: map[string]interface{}{
					"name": "test_tool",
					"arguments": map[string]interface{}{
						"nested": map[string]interface{}{
							"value": 123,
							"text":  "hello",
						},
						"array": []interface{}{"a", "b", "c"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			data, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			// Decode
			var decoded Request
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal request: %v", err)
			}

			// Verify required fields
			if decoded.JSONRPC != tt.req.JSONRPC {
				t.Errorf("JSONRPC = %v, want %v", decoded.JSONRPC, tt.req.JSONRPC)
			}

			// ID comparison needs special handling due to JSON number conversion
			if !compareID(decoded.ID, tt.req.ID) {
				t.Errorf("ID = %v (type %T), want %v (type %T)", decoded.ID, decoded.ID, tt.req.ID, tt.req.ID)
			}

			if decoded.Method != tt.req.Method {
				t.Errorf("Method = %v, want %v", decoded.Method, tt.req.Method)
			}

			// Params can be nil or empty map - omitempty means empty maps become nil
			if tt.req.Params != nil && len(tt.req.Params) > 0 && decoded.Params == nil {
				t.Error("Params with values should not be nil")
			}
		})
	}
}

func TestResponse_EncodeDecode(t *testing.T) {
	tests := []struct {
		name string
		resp *Response
	}{
		{
			name: "response with string ID",
			resp: &Response{
				JSONRPC: "2.0",
				ID:      "resp-123",
				Result: map[string]interface{}{
					"protocolVersion": ProtocolVersion,
					"capabilities":    map[string]interface{}{},
				},
			},
		},
		{
			name: "response with numeric ID",
			resp: &Response{
				JSONRPC: "2.0",
				ID:      99,
				Result: map[string]interface{}{
					"success": true,
				},
			},
		},
		{
			name: "response with empty result",
			resp: &Response{
				JSONRPC: "2.0",
				ID:      1,
				Result:  map[string]interface{}{},
			},
		},
		{
			name: "response with complex result",
			resp: &Response{
				JSONRPC: "2.0",
				ID:      "complex-resp",
				Result: map[string]interface{}{
					"tools": []interface{}{
						map[string]interface{}{
							"name":        "tool1",
							"description": "Test tool",
						},
					},
					"nextCursor": "cursor-123",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			data, err := json.Marshal(tt.resp)
			if err != nil {
				t.Fatalf("Failed to marshal response: %v", err)
			}

			// Decode
			var decoded Response
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			// Verify required fields
			if decoded.JSONRPC != tt.resp.JSONRPC {
				t.Errorf("JSONRPC = %v, want %v", decoded.JSONRPC, tt.resp.JSONRPC)
			}

			if !compareID(decoded.ID, tt.resp.ID) {
				t.Errorf("ID = %v (type %T), want %v (type %T)", decoded.ID, decoded.ID, tt.resp.ID, tt.resp.ID)
			}

			if decoded.Result == nil {
				t.Error("Result should not be nil")
			}
		})
	}
}

func TestErrorResponse_EncodeDecode(t *testing.T) {
	tests := []struct {
		name    string
		errResp *ErrorResponse
	}{
		{
			name: "error with string ID",
			errResp: &ErrorResponse{
				JSONRPC: "2.0",
				ID:      "err-123",
				Error: RPCError{
					Code:    MethodNotFound,
					Message: "Method not found",
				},
			},
		},
		{
			name: "error with numeric ID",
			errResp: &ErrorResponse{
				JSONRPC: "2.0",
				ID:      42,
				Error: RPCError{
					Code:    InvalidParams,
					Message: "Invalid parameters",
				},
			},
		},
		{
			name: "error without ID (parse error)",
			errResp: &ErrorResponse{
				JSONRPC: "2.0",
				Error: RPCError{
					Code:    ParseError,
					Message: "Parse error",
				},
			},
		},
		{
			name: "error with data",
			errResp: &ErrorResponse{
				JSONRPC: "2.0",
				ID:      1,
				Error: RPCError{
					Code:    InternalError,
					Message: "Internal error",
					Data: map[string]interface{}{
						"details": "Something went wrong",
						"code":    500,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			data, err := json.Marshal(tt.errResp)
			if err != nil {
				t.Fatalf("Failed to marshal error response: %v", err)
			}

			// Decode
			var decoded ErrorResponse
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal error response: %v", err)
			}

			// Verify required fields
			if decoded.JSONRPC != tt.errResp.JSONRPC {
				t.Errorf("JSONRPC = %v, want %v", decoded.JSONRPC, tt.errResp.JSONRPC)
			}

			if tt.errResp.ID != nil && !compareID(decoded.ID, tt.errResp.ID) {
				t.Errorf("ID = %v (type %T), want %v (type %T)", decoded.ID, decoded.ID, tt.errResp.ID, tt.errResp.ID)
			}

			if decoded.Error.Code != tt.errResp.Error.Code {
				t.Errorf("Error.Code = %v, want %v", decoded.Error.Code, tt.errResp.Error.Code)
			}

			if decoded.Error.Message != tt.errResp.Error.Message {
				t.Errorf("Error.Message = %v, want %v", decoded.Error.Message, tt.errResp.Error.Message)
			}
		})
	}
}

func TestNotification_EncodeDecode(t *testing.T) {
	tests := []struct {
		name  string
		notif *Notification
	}{
		{
			name: "notification without params",
			notif: &Notification{
				JSONRPC: "2.0",
				Method:  "initialized",
			},
		},
		{
			name: "notification with params",
			notif: &Notification{
				JSONRPC: "2.0",
				Method:  "progress",
				Params: map[string]interface{}{
					"token": "token-123",
					"value": map[string]interface{}{
						"kind":       "report",
						"percentage": 50,
					},
				},
			},
		},
		{
			name: "notification with empty params",
			notif: &Notification{
				JSONRPC: "2.0",
				Method:  "cancelled",
				Params:  map[string]interface{}{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			data, err := json.Marshal(tt.notif)
			if err != nil {
				t.Fatalf("Failed to marshal notification: %v", err)
			}

			// Decode
			var decoded Notification
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal notification: %v", err)
			}

			// Verify required fields
			if decoded.JSONRPC != tt.notif.JSONRPC {
				t.Errorf("JSONRPC = %v, want %v", decoded.JSONRPC, tt.notif.JSONRPC)
			}

			if decoded.Method != tt.notif.Method {
				t.Errorf("Method = %v, want %v", decoded.Method, tt.notif.Method)
			}

			// Verify notification has no ID field in JSON
			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("Failed to unmarshal to map: %v", err)
			}

			if _, hasID := raw["id"]; hasID {
				t.Error("Notification should not have an 'id' field")
			}
		})
	}
}

func TestNewRequest(t *testing.T) {
	req := NewRequest(123, "test/method", map[string]interface{}{
		"param1": "value1",
	})

	if req.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %v, want 2.0", req.JSONRPC)
	}

	if req.ID != 123 {
		t.Errorf("ID = %v, want 123", req.ID)
	}

	if req.Method != "test/method" {
		t.Errorf("Method = %v, want test/method", req.Method)
	}

	if req.Params["param1"] != "value1" {
		t.Errorf("Params[param1] = %v, want value1", req.Params["param1"])
	}
}

func TestNewResponse(t *testing.T) {
	resp := NewResponse("test-id", map[string]interface{}{
		"success": true,
	})

	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %v, want 2.0", resp.JSONRPC)
	}

	if resp.ID != "test-id" {
		t.Errorf("ID = %v, want test-id", resp.ID)
	}

	if resp.Result["success"] != true {
		t.Errorf("Result[success] = %v, want true", resp.Result["success"])
	}
}

func TestNewErrorResponse(t *testing.T) {
	errResp := NewErrorResponse(456, MethodNotFound, "Method not found", map[string]interface{}{
		"method": "unknown",
	})

	if errResp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %v, want 2.0", errResp.JSONRPC)
	}

	if errResp.ID != 456 {
		t.Errorf("ID = %v, want 456", errResp.ID)
	}

	if errResp.Error.Code != MethodNotFound {
		t.Errorf("Error.Code = %v, want %v", errResp.Error.Code, MethodNotFound)
	}

	if errResp.Error.Message != "Method not found" {
		t.Errorf("Error.Message = %v, want Method not found", errResp.Error.Message)
	}

	if errResp.Error.Data == nil {
		t.Error("Error.Data should not be nil")
	}
}

func TestNewNotification(t *testing.T) {
	notif := NewNotification("test/notification", map[string]interface{}{
		"event": "started",
	})

	if notif.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %v, want 2.0", notif.JSONRPC)
	}

	if notif.Method != "test/notification" {
		t.Errorf("Method = %v, want test/notification", notif.Method)
	}

	if notif.Params["event"] != "started" {
		t.Errorf("Params[event] = %v, want started", notif.Params["event"])
	}
}

func TestNewTextContent(t *testing.T) {
	content := NewTextContent("Hello, world!")

	if content.Type != "text" {
		t.Errorf("Type = %v, want text", content.Type)
	}

	if content.Text != "Hello, world!" {
		t.Errorf("Text = %v, want Hello, world!", content.Text)
	}

	// Verify it encodes/decodes correctly
	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("Failed to marshal content: %v", err)
	}

	var decoded Content
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal content: %v", err)
	}

	if decoded.Type != "text" {
		t.Errorf("Decoded Type = %v, want text", decoded.Type)
	}

	if decoded.Text != "Hello, world!" {
		t.Errorf("Decoded Text = %v, want Hello, world!", decoded.Text)
	}
}

func TestNewImageContent(t *testing.T) {
	content := NewImageContent("base64data", "image/png")

	if content.Type != "image" {
		t.Errorf("Type = %v, want image", content.Type)
	}

	if content.Data != "base64data" {
		t.Errorf("Data = %v, want base64data", content.Data)
	}

	if content.MimeType != "image/png" {
		t.Errorf("MimeType = %v, want image/png", content.MimeType)
	}

	// Verify it encodes/decodes correctly
	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("Failed to marshal content: %v", err)
	}

	var decoded Content
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal content: %v", err)
	}

	if decoded.Type != "image" {
		t.Errorf("Decoded Type = %v, want image", decoded.Type)
	}

	if decoded.Data != "base64data" {
		t.Errorf("Decoded Data = %v, want base64data", decoded.Data)
	}

	if decoded.MimeType != "image/png" {
		t.Errorf("Decoded MimeType = %v, want image/png", decoded.MimeType)
	}
}

func TestNewResourceLinkContent(t *testing.T) {
	content := NewResourceLinkContent(
		"file:///path/to/file.txt",
		"File Resource",
		"A test file resource",
		"text/plain",
	)

	if content.Type != "resource_link" {
		t.Errorf("Type = %v, want resource_link", content.Type)
	}

	if content.URI != "file:///path/to/file.txt" {
		t.Errorf("URI = %v, want file:///path/to/file.txt", content.URI)
	}

	if content.Name != "File Resource" {
		t.Errorf("Name = %v, want File Resource", content.Name)
	}

	if content.Description != "A test file resource" {
		t.Errorf("Description = %v, want A test file resource", content.Description)
	}

	if content.MimeType != "text/plain" {
		t.Errorf("MimeType = %v, want text/plain", content.MimeType)
	}

	// Verify it encodes/decodes correctly
	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("Failed to marshal content: %v", err)
	}

	var decoded Content
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal content: %v", err)
	}

	if decoded.Type != "resource_link" {
		t.Errorf("Decoded Type = %v, want resource_link", decoded.Type)
	}

	if decoded.URI != "file:///path/to/file.txt" {
		t.Errorf("Decoded URI = %v, want file:///path/to/file.txt", decoded.URI)
	}
}

func TestContent_MultipleTypes(t *testing.T) {
	tests := []struct {
		name    string
		content Content
	}{
		{
			name: "text content",
			content: Content{
				Type: "text",
				Text: "Sample text",
			},
		},
		{
			name: "image content",
			content: Content{
				Type:     "image",
				Data:     "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
				MimeType: "image/png",
			},
		},
		{
			name: "resource_link content",
			content: Content{
				Type:        "resource_link",
				URI:         "https://example.com/resource",
				Name:        "Example Resource",
				Description: "An example resource",
				MimeType:    "application/json",
			},
		},
		{
			name: "resource content with text",
			content: Content{
				Type: "resource",
				Resource: &Resource{
					URI:      "file:///data.json",
					MimeType: "application/json",
					Text:     `{"key":"value"}`,
				},
			},
		},
		{
			name: "resource content with blob",
			content: Content{
				Type: "resource",
				Resource: &Resource{
					URI:      "file:///binary.dat",
					MimeType: "application/octet-stream",
					Blob:     "YmluYXJ5ZGF0YQ==",
				},
			},
		},
		{
			name: "content with annotations",
			content: Content{
				Type: "text",
				Text: "Annotated text",
				Annotations: &Annotations{
					Audience:     []string{"user", "assistant"},
					Priority:     0.8,
					LastModified: "2026-03-04T12:00:00Z",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			data, err := json.Marshal(tt.content)
			if err != nil {
				t.Fatalf("Failed to marshal content: %v", err)
			}

			// Decode
			var decoded Content
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal content: %v", err)
			}

			// Verify type is preserved
			if decoded.Type != tt.content.Type {
				t.Errorf("Type = %v, want %v", decoded.Type, tt.content.Type)
			}
		})
	}
}

func TestToolCallResult_EncodeDecode(t *testing.T) {
	result := ToolCallResult{
		Content: []Content{
			NewTextContent("Tool executed successfully"),
			NewImageContent("base64imagedata", "image/jpeg"),
		},
		IsError: false,
	}

	// Encode
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal ToolCallResult: %v", err)
	}

	// Decode
	var decoded ToolCallResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ToolCallResult: %v", err)
	}

	if len(decoded.Content) != 2 {
		t.Errorf("Content length = %v, want 2", len(decoded.Content))
	}

	if decoded.Content[0].Type != "text" {
		t.Errorf("Content[0].Type = %v, want text", decoded.Content[0].Type)
	}

	if decoded.Content[1].Type != "image" {
		t.Errorf("Content[1].Type = %v, want image", decoded.Content[1].Type)
	}

	if decoded.IsError != false {
		t.Errorf("IsError = %v, want false", decoded.IsError)
	}
}

func TestToolCallResult_Error(t *testing.T) {
	result := ToolCallResult{
		Content: []Content{
			NewTextContent("Error: something went wrong"),
		},
		IsError: true,
	}

	// Encode
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal error ToolCallResult: %v", err)
	}

	// Decode
	var decoded ToolCallResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal error ToolCallResult: %v", err)
	}

	if !decoded.IsError {
		t.Error("IsError should be true")
	}

	if len(decoded.Content) != 1 {
		t.Errorf("Content length = %v, want 1", len(decoded.Content))
	}
}

func TestInitializeRequest_EncodeDecode(t *testing.T) {
	initReq := InitializeRequest{
		ProtocolVersion: ProtocolVersion,
		Capabilities: ClientCapabilities{
			Roots: &RootsCapability{
				ListChanged: true,
			},
			Sampling: &SamplingCapability{},
		},
		ClientInfo: Implementation{
			Name:    "test-client",
			Version: "1.0.0",
			Title:   "Test Client",
		},
	}

	// Encode
	data, err := json.Marshal(initReq)
	if err != nil {
		t.Fatalf("Failed to marshal InitializeRequest: %v", err)
	}

	// Decode
	var decoded InitializeRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal InitializeRequest: %v", err)
	}

	if decoded.ProtocolVersion != ProtocolVersion {
		t.Errorf("ProtocolVersion = %v, want %v", decoded.ProtocolVersion, ProtocolVersion)
	}

	if decoded.Capabilities.Roots == nil {
		t.Error("Capabilities.Roots should not be nil")
	}

	if decoded.ClientInfo.Name != "test-client" {
		t.Errorf("ClientInfo.Name = %v, want test-client", decoded.ClientInfo.Name)
	}
}

func TestInitializeResult_EncodeDecode(t *testing.T) {
	initResult := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{
				ListChanged: true,
			},
			Resources: &ResourcesCapability{
				Subscribe:   true,
				ListChanged: true,
			},
		},
		ServerInfo: Implementation{
			Name:        "test-server",
			Version:     "1.0.0",
			Title:       "Test Server",
			Description: "A test MCP server",
		},
		Instructions: "Use this server for testing",
	}

	// Encode
	data, err := json.Marshal(initResult)
	if err != nil {
		t.Fatalf("Failed to marshal InitializeResult: %v", err)
	}

	// Decode
	var decoded InitializeResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal InitializeResult: %v", err)
	}

	if decoded.ProtocolVersion != ProtocolVersion {
		t.Errorf("ProtocolVersion = %v, want %v", decoded.ProtocolVersion, ProtocolVersion)
	}

	if decoded.Capabilities.Tools == nil {
		t.Error("Capabilities.Tools should not be nil")
	}

	if decoded.ServerInfo.Name != "test-server" {
		t.Errorf("ServerInfo.Name = %v, want test-server", decoded.ServerInfo.Name)
	}

	if decoded.Instructions != "Use this server for testing" {
		t.Errorf("Instructions = %v, want Use this server for testing", decoded.Instructions)
	}
}

func TestTool_EncodeDecode(t *testing.T) {
	tool := Tool{
		Name:        "test_tool",
		Title:       "Test Tool",
		Description: "A tool for testing",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param1": map[string]interface{}{
					"type":        "string",
					"description": "First parameter",
				},
			},
			"required": []interface{}{"param1"},
		},
		Icons: []Icon{
			{
				Src:      "https://example.com/icon.png",
				MimeType: "image/png",
				Sizes:    []string{"48x48"},
			},
		},
		Annotations: &Annotations{
			Audience: []string{"user"},
			Priority: 0.9,
		},
	}

	// Encode
	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("Failed to marshal Tool: %v", err)
	}

	// Decode
	var decoded Tool
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Tool: %v", err)
	}

	if decoded.Name != "test_tool" {
		t.Errorf("Name = %v, want test_tool", decoded.Name)
	}

	if decoded.Title != "Test Tool" {
		t.Errorf("Title = %v, want Test Tool", decoded.Title)
	}

	if decoded.InputSchema == nil {
		t.Error("InputSchema should not be nil")
	}

	if len(decoded.Icons) != 1 {
		t.Errorf("Icons length = %v, want 1", len(decoded.Icons))
	}

	if decoded.Annotations == nil {
		t.Error("Annotations should not be nil")
	}
}

func TestErrorCodes(t *testing.T) {
	// Verify standard JSON-RPC error codes match specification
	if ParseError != -32700 {
		t.Errorf("ParseError = %v, want -32700", ParseError)
	}

	if InvalidRequest != -32600 {
		t.Errorf("InvalidRequest = %v, want -32600", InvalidRequest)
	}

	if MethodNotFound != -32601 {
		t.Errorf("MethodNotFound = %v, want -32601", MethodNotFound)
	}

	if InvalidParams != -32602 {
		t.Errorf("InvalidParams = %v, want -32602", InvalidParams)
	}

	if InternalError != -32603 {
		t.Errorf("InternalError = %v, want -32603", InternalError)
	}
}

func TestProtocolVersion(t *testing.T) {
	// Verify protocol version matches specification
	if ProtocolVersion != "2025-11-25" {
		t.Errorf("ProtocolVersion = %v, want 2025-11-25", ProtocolVersion)
	}
}

// compareID compares two IDs, handling JSON number conversion
func compareID(a, b interface{}) bool {
	// If both are the same type and equal, return true
	if reflect.DeepEqual(a, b) {
		return true
	}

	// Handle JSON number conversion (int -> float64)
	aFloat, aIsFloat := a.(float64)
	bFloat, bIsFloat := b.(float64)
	aInt, aIsInt := a.(int)
	bInt, bIsInt := b.(int)

	// a is float64, b is int
	if aIsFloat && bIsInt {
		return aFloat == float64(bInt)
	}

	// a is int, b is float64
	if aIsInt && bIsFloat {
		return float64(aInt) == bFloat
	}

	// Handle string comparison
	aStr, aIsStr := a.(string)
	bStr, bIsStr := b.(string)

	if aIsStr && bIsStr {
		return aStr == bStr
	}

	return false
}
