package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// ResourceHandler is a function that handles reading a resource by URI.
type ResourceHandler func(uri string) (*ResourceReadResult, error)

// RegisteredResource combines a resource template with its handler function.
type RegisteredResource struct {
	Template ResourceTemplate
	Handler  ResourceHandler
}

// PromptHandler is a function that handles a prompt get request.
type PromptHandler func(args map[string]string) (*PromptGetResult, error)

// RegisteredPrompt combines a prompt definition with its handler function.
type RegisteredPrompt struct {
	Prompt  Prompt
	Handler PromptHandler
}

// Server implements an MCP (Model Context Protocol) server that communicates
// via JSON-RPC 2.0 over stdio (stdin/stdout).
type Server struct {
	reader *bufio.Reader
	writer *bufio.Writer
	mu     sync.Mutex // protects writes to stdout

	handlersMu sync.RWMutex
	handlers   map[string]RequestHandler

	resources map[string]*RegisteredResource
	prompts   map[string]*RegisteredPrompt

	// Server info
	name        string
	version     string
	description string

	// State
	initialized bool
	ctx         context.Context
	cancel      context.CancelFunc
}

// RequestHandler is a function that handles a JSON-RPC request.
type RequestHandler func(req *Request) (map[string]interface{}, error)

// NewServer creates a new MCP server that reads from stdin and writes to stdout.
func NewServer(name, version, description string) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{
		reader:      bufio.NewReader(os.Stdin),
		writer:      bufio.NewWriter(os.Stdout),
		handlers:    make(map[string]RequestHandler),
		resources:   make(map[string]*RegisteredResource),
		prompts:     make(map[string]*RegisteredPrompt),
		name:        name,
		version:     version,
		description: description,
		ctx:         ctx,
		cancel:      cancel,
	}

	// Register built-in handlers
	s.registerBuiltinHandlers()

	return s
}

// NewServerWith creates a server with custom reader/writer for testing.
func NewServerWith(reader io.Reader, writer io.Writer, name, version, description string) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{
		reader:      bufio.NewReader(reader),
		writer:      bufio.NewWriter(writer),
		handlers:    make(map[string]RequestHandler),
		resources:   make(map[string]*RegisteredResource),
		prompts:     make(map[string]*RegisteredPrompt),
		name:        name,
		version:     version,
		description: description,
		ctx:         ctx,
		cancel:      cancel,
	}

	s.registerBuiltinHandlers()

	return s
}

// RegisterHandler registers a custom handler for a specific JSON-RPC method.
func (s *Server) RegisterHandler(method string, handler RequestHandler) {
	s.handlersMu.Lock()
	defer s.handlersMu.Unlock()
	s.handlers[method] = handler
}

// AddResourceTemplate registers a resource template with its handler.
func (s *Server) AddResourceTemplate(template ResourceTemplate, handler ResourceHandler) {
	s.resources[template.Name] = &RegisteredResource{Template: template, Handler: handler}
}

// AddPrompt registers a prompt with its handler.
func (s *Server) AddPrompt(prompt Prompt, handler PromptHandler) {
	s.prompts[prompt.Name] = &RegisteredPrompt{Prompt: prompt, Handler: handler}
}

// registerBuiltinHandlers registers the core MCP protocol handlers.
func (s *Server) registerBuiltinHandlers() {
	s.RegisterHandler("initialize", s.handleInitialize)
	s.RegisterHandler("initialized", s.handleInitialized)
	s.RegisterHandler("tools/list", s.handleToolsList)
	s.RegisterHandler("tools/call", s.handleToolsCall)
	s.RegisterHandler("resources/list", s.handleResourcesList)
	s.RegisterHandler("resources/read", s.handleResourcesRead)
	s.RegisterHandler("prompts/list", s.handlePromptsList)
	s.RegisterHandler("prompts/get", s.handlePromptsGet)
}

// Start begins listening for JSON-RPC requests on stdin and processing them.
// This blocks until the context is cancelled or an unrecoverable error occurs.
func (s *Server) Start() error {
	for {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
			if err := s.processOneRequest(); err != nil {
				if err == io.EOF {
					return nil // Clean shutdown
				}
				// Log error but continue processing
				fmt.Fprintf(os.Stderr, "Error processing request: %v\n", err)
			}
		}
	}
}

// Stop gracefully stops the server.
func (s *Server) Stop() {
	s.cancel()
}

// processOneRequest reads and processes a single JSON-RPC request from stdin.
func (s *Server) processOneRequest() error {
	// Read one line (JSON-RPC message)
	line, err := s.reader.ReadBytes('\n')
	if err != nil {
		return err
	}

	// Try to parse as a request
	var req Request
	if err := json.Unmarshal(line, &req); err != nil {
		// Send parse error response
		s.sendError(nil, ParseError, "Parse error: invalid JSON", err.Error())
		return nil // Continue processing
	}

	// Validate request
	if req.JSONRPC != "2.0" {
		s.sendError(req.ID, InvalidRequest, "Invalid Request: jsonrpc must be '2.0'", nil)
		return nil
	}

	if req.Method == "" {
		s.sendError(req.ID, InvalidRequest, "Invalid Request: method is required", nil)
		return nil
	}

	// Check if this is a notification (absent id) vs invalid request (null id)
	if req.ID == nil {
		// No id field present - this is a notification
		return s.handleNotification(&req)
	}
	if string(req.ID) == "null" {
		// Explicit "id": null is invalid per JSON-RPC spec
		s.sendError(nil, InvalidRequest, "Invalid Request: id must not be null", nil)
		return nil
	}

	// Dispatch to handler
	s.handlersMu.RLock()
	handler, exists := s.handlers[req.Method]
	s.handlersMu.RUnlock()
	if !exists {
		s.sendError(req.ID, MethodNotFound, fmt.Sprintf("Method not found: %s", req.Method), nil)
		return nil
	}

	// Call handler
	result, err := handler(&req)
	if err != nil {
		s.sendError(req.ID, InternalError, fmt.Sprintf("Internal error: %v", err), nil)
		return nil
	}

	// Send success response
	return s.sendResponse(req.ID, result)
}

// handleNotification processes a JSON-RPC notification (no response).
func (s *Server) handleNotification(req *Request) error {
	// Notifications are fire-and-forget
	s.handlersMu.RLock()
	handler, exists := s.handlers[req.Method]
	s.handlersMu.RUnlock()
	if !exists {
		// Silently ignore unknown notifications per JSON-RPC spec
		return nil
	}

	_, _ = handler(req)
	return nil
}

// handleInitialize handles the initialize request from the client.
func (s *Server) handleInitialize(req *Request) (map[string]interface{}, error) {
	// Parse initialize params
	var initReq InitializeRequest
	if req.Params != nil {
		paramsBytes, err := json.Marshal(req.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		if err := json.Unmarshal(paramsBytes, &initReq); err != nil {
			return nil, fmt.Errorf("failed to unmarshal initialize request: %w", err)
		}
	}

	// Build initialize result
	result := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{
				ListChanged: false,
			},
			Resources: &ResourcesCapability{},
			Prompts:   &PromptsCapability{},
		},
		ServerInfo: Implementation{
			Name:        s.name,
			Version:     s.version,
			Description: s.description,
		},
		Instructions: "Bitbucket CLI MCP server. Use available tools to interact with Bitbucket repositories, PRs, issues, and pipelines.",
	}

	// Convert to map[string]interface{}
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var resultMap map[string]interface{}
	if err := json.Unmarshal(resultBytes, &resultMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result map: %w", err)
	}

	return resultMap, nil
}

// handleInitialized handles the initialized notification from the client.
func (s *Server) handleInitialized(req *Request) (map[string]interface{}, error) {
	s.initialized = true
	return nil, nil
}

// handleToolsList handles the tools/list request.
func (s *Server) handleToolsList(req *Request) (map[string]interface{}, error) {
	// Default handler returns empty list; overridden when a tool registry is configured
	result := ToolsListResult{
		Tools: []Tool{},
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tools list: %w", err)
	}

	var resultMap map[string]interface{}
	if err := json.Unmarshal(resultBytes, &resultMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result map: %w", err)
	}

	return resultMap, nil
}

// handleToolsCall handles the tools/call request.
func (s *Server) handleToolsCall(req *Request) (map[string]interface{}, error) {
	// Parse tool call params
	var callReq ToolCallRequest
	if req.Params != nil {
		paramsBytes, err := json.Marshal(req.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		if err := json.Unmarshal(paramsBytes, &callReq); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tool call request: %w", err)
		}
	}

	// Default handler returns not-found; overridden when a tool registry is configured
	result := ToolCallResult{
		Content: []Content{
			NewTextContent(fmt.Sprintf("Tool not found: %s", callReq.Name)),
		},
		IsError: true,
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tool call result: %w", err)
	}

	var resultMap map[string]interface{}
	if err := json.Unmarshal(resultBytes, &resultMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result map: %w", err)
	}

	return resultMap, nil
}

// handleResourcesList handles the resources/list request.
func (s *Server) handleResourcesList(req *Request) (map[string]interface{}, error) {
	templates := make([]ResourceTemplate, 0, len(s.resources))
	for _, r := range s.resources {
		templates = append(templates, r.Template)
	}

	result := ResourcesListResult{
		ResourceTemplates: templates,
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resources list: %w", err)
	}

	var resultMap map[string]interface{}
	if err := json.Unmarshal(resultBytes, &resultMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result map: %w", err)
	}

	return resultMap, nil
}

// handleResourcesRead handles the resources/read request.
func (s *Server) handleResourcesRead(req *Request) (map[string]interface{}, error) {
	uri, _ := req.Params["uri"].(string)
	if uri == "" {
		return nil, fmt.Errorf("missing required parameter: uri")
	}

	// Try each registered resource handler
	for _, r := range s.resources {
		if matchesTemplate(r.Template.URITemplate, uri) {
			result, err := r.Handler(uri)
			if err != nil {
				return nil, err
			}

			resultBytes, err := json.Marshal(result)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal resource read result: %w", err)
			}

			var resultMap map[string]interface{}
			if err := json.Unmarshal(resultBytes, &resultMap); err != nil {
				return nil, fmt.Errorf("failed to unmarshal result map: %w", err)
			}

			return resultMap, nil
		}
	}

	return nil, fmt.Errorf("no resource handler found for URI: %s", uri)
}

// matchesTemplate checks if a URI matches a URI template pattern.
// It performs a simple prefix-based match by comparing the static parts of the template.
func matchesTemplate(template, uri string) bool {
	// Simple matching: split template on '{' tokens and check that the URI
	// contains the static prefix portions in order.
	parts := strings.SplitN(template, "{", 2)
	if len(parts) == 0 {
		return template == uri
	}
	return strings.HasPrefix(uri, parts[0])
}

// handlePromptsList handles the prompts/list request.
func (s *Server) handlePromptsList(req *Request) (map[string]interface{}, error) {
	prompts := make([]Prompt, 0, len(s.prompts))
	for _, p := range s.prompts {
		prompts = append(prompts, p.Prompt)
	}

	result := PromptsListResult{
		Prompts: prompts,
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal prompts list: %w", err)
	}

	var resultMap map[string]interface{}
	if err := json.Unmarshal(resultBytes, &resultMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result map: %w", err)
	}

	return resultMap, nil
}

// handlePromptsGet handles the prompts/get request.
func (s *Server) handlePromptsGet(req *Request) (map[string]interface{}, error) {
	name, _ := req.Params["name"].(string)
	if name == "" {
		return nil, fmt.Errorf("missing required parameter: name")
	}

	rp, exists := s.prompts[name]
	if !exists {
		return nil, fmt.Errorf("prompt not found: %s", name)
	}

	// Extract arguments from params
	args := make(map[string]string)
	if argsRaw, ok := req.Params["arguments"].(map[string]interface{}); ok {
		for k, v := range argsRaw {
			if str, ok := v.(string); ok {
				args[k] = str
			}
		}
	}

	result, err := rp.Handler(args)
	if err != nil {
		return nil, fmt.Errorf("prompt handler error: %w", err)
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal prompt result: %w", err)
	}

	var resultMap map[string]interface{}
	if err := json.Unmarshal(resultBytes, &resultMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result map: %w", err)
	}

	return resultMap, nil
}

// sendResponse sends a JSON-RPC success response to stdout.
func (s *Server) sendResponse(id json.RawMessage, result map[string]interface{}) error {
	resp := NewResponse(id, result)
	return s.writeJSON(resp)
}

// sendError sends a JSON-RPC error response to stdout.
func (s *Server) sendError(id json.RawMessage, code int, message string, data interface{}) error {
	errResp := NewErrorResponse(id, code, message, data)
	return s.writeJSON(errResp)
}

// writeJSON writes a JSON object to stdout followed by a newline.
func (s *Server) writeJSON(v interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if _, err := s.writer.Write(data); err != nil {
		return fmt.Errorf("failed to write to stdout: %w", err)
	}

	if _, err := s.writer.Write([]byte("\n")); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	if err := s.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush writer: %w", err)
	}

	return nil
}
