package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// HTTPHandler adapts an MCP Server to handle JSON-RPC requests over HTTP.
// Each POST request contains a JSON-RPC message and receives a JSON-RPC response.
// Supports both regular JSON responses and Server-Sent Events (SSE) streaming.
type HTTPHandler struct {
	// serverFactory creates a Server for each request. This allows stateless
	// operation where each request gets a fresh server, or stateful operation
	// where a shared server is returned.
	serverFactory func(r *http.Request) *Server

	// stateless controls whether each request gets an independent server.
	stateless bool

	// sharedServer is used in stateful mode.
	sharedServer *Server
	once         sync.Once
}

// HTTPHandlerOptions configures the HTTP handler behavior.
type HTTPHandlerOptions struct {
	Stateless bool
}

// NewHTTPHandler creates an HTTP handler that processes MCP JSON-RPC requests.
// The serverFactory function is called for each request to obtain the Server instance.
func NewHTTPHandler(serverFactory func(r *http.Request) *Server, opts *HTTPHandlerOptions) *HTTPHandler {
	h := &HTTPHandler{
		serverFactory: serverFactory,
	}
	if opts != nil {
		h.stateless = opts.Stateless
	}
	return h
}

// ServeHTTP implements http.Handler. It accepts POST requests containing
// JSON-RPC messages, processes them through the MCP server, and returns
// the JSON-RPC response.
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only accept POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(body) == 0 {
		http.Error(w, "Empty request body", http.StatusBadRequest)
		return
	}

	// Get the server for this request
	server := h.serverFactory(r)
	if server == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Create a pipe to capture the server's response
	input := bytes.NewReader(append(body, '\n'))
	var output bytes.Buffer

	// Build a per-request context. If the request carries a Bearer token,
	// attach it so tool handlers can create per-user API clients.
	reqCtx := server.ctx
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimPrefix(auth, "Bearer ")
		reqCtx = ContextWithToken(reqCtx, token)
	}

	// Create a temporary server with our custom reader/writer to process
	// this single request through the existing JSON-RPC machinery.
	tmpServer := &Server{
		reader:      bufio.NewReader(input),
		writer:      bufio.NewWriter(&output),
		handlers:    server.handlers,
		resources:   server.resources,
		prompts:     server.prompts,
		name:        server.name,
		version:     server.version,
		description: server.description,
		initialized: server.initialized,
		ctx:         reqCtx,
		cancel:      server.cancel,
	}

	// Process the single request
	if err := tmpServer.processOneRequest(); err != nil {
		if err == io.EOF {
			http.Error(w, "Empty request", http.StatusBadRequest)
			return
		}
		errResp := NewErrorResponse(nil, InternalError, fmt.Sprintf("Internal error: %v", err), nil)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Check if the client accepts SSE
	accept := r.Header.Get("Accept")
	if accept == "text/event-stream" {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		responseData := bytes.TrimSpace(output.Bytes())
		if len(responseData) > 0 {
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", responseData)
		}

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		return
	}

	// Regular JSON response
	responseData := bytes.TrimSpace(output.Bytes())
	if len(responseData) == 0 {
		// Notification — no response body expected
		w.WriteHeader(http.StatusAccepted)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseData)
}
