package mcp

import "encoding/json"

// Protocol version for MCP 2025-11-25 specification
const ProtocolVersion = "2025-11-25"

// JSON-RPC 2.0 base types

// Request represents a JSON-RPC 2.0 request message.
type Request struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      json.RawMessage        `json:"id,omitempty"` // string or number; absent for notifications
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 success response message.
type Response struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      json.RawMessage        `json:"id"`
	Result  map[string]interface{} `json:"result"`
}

// ErrorResponse represents a JSON-RPC 2.0 error response message.
type ErrorResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Error   RPCError        `json:"error"`
}

// RPCError represents a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Notification represents a JSON-RPC 2.0 notification message (no response expected).
type Notification struct {
	JSONRPC string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// Standard JSON-RPC error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// MCP Protocol Types

// InitializeRequest represents the parameters for an initialize request.
type InitializeRequest struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      Implementation     `json:"clientInfo"`
}

// InitializeResult represents the result of an initialize request.
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
	Instructions    string             `json:"instructions,omitempty"`
}

// Implementation describes a client or server implementation.
type Implementation struct {
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
	WebsiteURL  string `json:"websiteUrl,omitempty"`
	Icons       []Icon `json:"icons,omitempty"`
}

// Icon represents a visual identifier for implementations, tools, prompts, or resources.
type Icon struct {
	Src      string   `json:"src"`               // URI to icon (HTTP/HTTPS or data URI)
	MimeType string   `json:"mimeType,omitempty"` // Optional MIME type
	Sizes    []string `json:"sizes,omitempty"`    // e.g., ["48x48"], ["any"] for SVG
	Theme    string   `json:"theme,omitempty"`    // "light" or "dark"
}

// ClientCapabilities describes optional features supported by the client.
type ClientCapabilities struct {
	Roots       *RootsCapability       `json:"roots,omitempty"`
	Sampling    *SamplingCapability    `json:"sampling,omitempty"`
	Elicitation *ElicitationCapability `json:"elicitation,omitempty"`
	Tasks       *TasksCapability       `json:"tasks,omitempty"`
	Experimental map[string]interface{} `json:"experimental,omitempty"`
}

// ServerCapabilities describes optional features supported by the server.
type ServerCapabilities struct {
	Logging     *LoggingCapability     `json:"logging,omitempty"`
	Prompts     *PromptsCapability     `json:"prompts,omitempty"`
	Resources   *ResourcesCapability   `json:"resources,omitempty"`
	Tools       *ToolsCapability       `json:"tools,omitempty"`
	Tasks       *TasksCapability       `json:"tasks,omitempty"`
	Completions *CompletionsCapability `json:"completions,omitempty"`
	Experimental map[string]interface{} `json:"experimental,omitempty"`
}

// RootsCapability indicates support for filesystem roots.
type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// SamplingCapability indicates support for LLM sampling requests.
type SamplingCapability struct{}

// ElicitationCapability indicates support for server elicitation requests.
type ElicitationCapability struct {
	Form map[string]interface{} `json:"form,omitempty"`
	URL  map[string]interface{} `json:"url,omitempty"`
}

// LoggingCapability indicates support for structured log messages.
type LoggingCapability struct{}

// PromptsCapability indicates support for prompt templates.
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability indicates support for readable resources.
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// ToolsCapability indicates support for callable tools.
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// CompletionsCapability indicates support for argument autocompletion.
type CompletionsCapability struct{}

// TasksCapability indicates support for task-augmented requests.
type TasksCapability struct {
	List     map[string]interface{} `json:"list,omitempty"`
	Cancel   map[string]interface{} `json:"cancel,omitempty"`
	Requests map[string]interface{} `json:"requests,omitempty"`
}

// Tool Definition Types

// Tool represents an MCP tool definition.
type Tool struct {
	Name         string                 `json:"name"`
	Title        string                 `json:"title,omitempty"`
	Description  string                 `json:"description"`
	InputSchema  map[string]interface{} `json:"inputSchema"`
	OutputSchema map[string]interface{} `json:"outputSchema,omitempty"`
	Icons        []Icon                 `json:"icons,omitempty"`
	Annotations  *Annotations           `json:"annotations,omitempty"`
}

// Annotations provides metadata about tool behavior.
type Annotations struct {
	Audience     []string `json:"audience,omitempty"`     // e.g., ["user", "assistant"]
	Priority     float64  `json:"priority,omitempty"`     // 0.0 to 1.0
	LastModified string   `json:"lastModified,omitempty"` // ISO 8601 timestamp
}

// ToolsListRequest represents parameters for tools/list request.
type ToolsListRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

// ToolsListResult represents the result of a tools/list request.
type ToolsListResult struct {
	Tools      []Tool `json:"tools"`
	NextCursor string `json:"nextCursor,omitempty"`
}

// ToolCallRequest represents parameters for tools/call request.
type ToolCallRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ToolCallResult represents the result of a tools/call request.
type ToolCallResult struct {
	Content           []Content   `json:"content"`
	IsError           bool        `json:"isError,omitempty"`
	StructuredContent interface{} `json:"structuredContent,omitempty"`
}

// Content Types

// Content represents a content item in a tool result.
// The actual type is determined by the "type" field.
type Content struct {
	Type string `json:"type"` // "text", "image", "audio", "resource", "resource_link"

	// Text content
	Text string `json:"text,omitempty"`

	// Image content
	Data     string `json:"data,omitempty"`     // base64-encoded
	MimeType string `json:"mimeType,omitempty"` // e.g., "image/png"

	// Resource link content
	URI         string `json:"uri,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`

	// Embedded resource content
	Resource *Resource `json:"resource,omitempty"`

	// Annotations for all content types
	Annotations *Annotations `json:"annotations,omitempty"`
}

// Resource represents an embedded resource in a tool result.
type Resource struct {
	URI         string       `json:"uri"`
	MimeType    string       `json:"mimeType,omitempty"`
	Text        string       `json:"text,omitempty"`
	Blob        string       `json:"blob,omitempty"` // base64-encoded binary
	Annotations *Annotations `json:"annotations,omitempty"`
}

// Helper Functions

// NewRequest creates a new JSON-RPC request with the given method and params.
func NewRequest(id json.RawMessage, method string, params map[string]interface{}) *Request {
	return &Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
}

// NewResponse creates a new JSON-RPC success response.
func NewResponse(id json.RawMessage, result map[string]interface{}) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

// NewErrorResponse creates a new JSON-RPC error response.
func NewErrorResponse(id json.RawMessage, code int, message string, data interface{}) *ErrorResponse {
	return &ErrorResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

// NewNotification creates a new JSON-RPC notification.
func NewNotification(method string, params map[string]interface{}) *Notification {
	return &Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
}

// NewTextContent creates a text content item.
func NewTextContent(text string) Content {
	return Content{
		Type: "text",
		Text: text,
	}
}

// NewImageContent creates an image content item.
func NewImageContent(data, mimeType string) Content {
	return Content{
		Type:     "image",
		Data:     data,
		MimeType: mimeType,
	}
}

// NewResourceLinkContent creates a resource link content item.
func NewResourceLinkContent(uri, name, description, mimeType string) Content {
	return Content{
		Type:        "resource_link",
		URI:         uri,
		Name:        name,
		Description: description,
		MimeType:    mimeType,
	}
}

// MarshalJSON ensures proper JSON serialization of requests.
func (r *Request) MarshalJSON() ([]byte, error) {
	type Alias Request
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	})
}

// MarshalJSON ensures proper JSON serialization of responses.
func (r *Response) MarshalJSON() ([]byte, error) {
	type Alias Response
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	})
}

// MarshalJSON ensures proper JSON serialization of error responses.
func (e *ErrorResponse) MarshalJSON() ([]byte, error) {
	type Alias ErrorResponse
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(e),
	})
}

// MarshalJSON ensures proper JSON serialization of notifications.
func (n *Notification) MarshalJSON() ([]byte, error) {
	type Alias Notification
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(n),
	})
}
