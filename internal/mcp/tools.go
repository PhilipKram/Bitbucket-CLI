package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// ToolHandler is a function that executes a tool with the given arguments.
// It returns content to be included in the tool call result, or an error.
type ToolHandler func(ctx context.Context, args map[string]interface{}) ([]Content, error)

// RegisteredTool combines a tool definition with its handler function.
type RegisteredTool struct {
	Tool    Tool
	Handler ToolHandler
}

// ToolRegistry manages the collection of available MCP tools.
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]*RegisteredTool
}

// NewToolRegistry creates a new empty tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]*RegisteredTool),
	}
}

// Register adds a tool to the registry with its handler.
// If a tool with the same name already exists, it will be replaced.
func (r *ToolRegistry) Register(tool Tool, handler ToolHandler) error {
	if tool.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	if handler == nil {
		return fmt.Errorf("tool handler cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.tools[tool.Name] = &RegisteredTool{
		Tool:    tool,
		Handler: handler,
	}

	return nil
}

// Unregister removes a tool from the registry.
// Returns true if the tool was found and removed, false otherwise.
func (r *ToolRegistry) Unregister(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; exists {
		delete(r.tools, name)
		return true
	}
	return false
}

// Get retrieves a registered tool by name.
// Returns nil if the tool is not found.
func (r *ToolRegistry) Get(name string) *RegisteredTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.tools[name]
}

// List returns all registered tools.
// The returned slice is a copy and safe to modify.
func (r *ToolRegistry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, rt := range r.tools {
		tools = append(tools, rt.Tool)
	}

	return tools
}

// Execute runs a tool by name with the given arguments.
// Returns a ToolCallResult with the tool's output or error.
func (r *ToolRegistry) Execute(ctx context.Context, name string, args map[string]interface{}) ToolCallResult {
	rt := r.Get(name)
	if rt == nil {
		return ToolCallResult{
			Content: []Content{
				NewTextContent(fmt.Sprintf("Tool not found: %s", name)),
			},
			IsError: true,
		}
	}

	// Execute the tool handler
	content, err := rt.Handler(ctx, args)
	if err != nil {
		return ToolCallResult{
			Content: []Content{
				NewTextContent(fmt.Sprintf("Tool execution failed: %v", err)),
			},
			IsError: true,
		}
	}

	// Return successful result
	return ToolCallResult{
		Content: content,
		IsError: false,
	}
}

// Count returns the number of registered tools.
func (r *ToolRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tools)
}

// SetRegistry sets the tool registry for the server.
// This allows the server to use custom tool definitions.
func (s *Server) SetRegistry(registry *ToolRegistry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Override the tools/list handler to use the registry
	s.RegisterHandler("tools/list", func(req *Request) (map[string]interface{}, error) {
		tools := registry.List()
		result := ToolsListResult{
			Tools: tools,
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
	})

	// Override the tools/call handler to use the registry
	s.RegisterHandler("tools/call", func(req *Request) (map[string]interface{}, error) {
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

		// Execute the tool
		result := registry.Execute(s.ctx, callReq.Name, callReq.Arguments)

		// Convert to map
		resultBytes, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tool call result: %w", err)
		}

		var resultMap map[string]interface{}
		if err := json.Unmarshal(resultBytes, &resultMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal result map: %w", err)
		}

		return resultMap, nil
	})
}

// NewJSONSchema creates a JSON schema map for tool input parameters.
// This is a helper function for creating tool input schemas.
func NewJSONSchema(schemaType string, properties map[string]interface{}, required []string) map[string]interface{} {
	schema := map[string]interface{}{
		"type":       schemaType,
		"properties": properties,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

// NewStringProperty creates a JSON schema property for a string parameter.
func NewStringProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "string",
		"description": description,
	}
}

// NewNumberProperty creates a JSON schema property for a number parameter.
func NewNumberProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "number",
		"description": description,
	}
}

// NewBooleanProperty creates a JSON schema property for a boolean parameter.
func NewBooleanProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "boolean",
		"description": description,
	}
}

// NewObjectProperty creates a JSON schema property for an object parameter.
func NewObjectProperty(description string, properties map[string]interface{}, required []string) map[string]interface{} {
	prop := map[string]interface{}{
		"type":        "object",
		"description": description,
		"properties":  properties,
	}

	if len(required) > 0 {
		prop["required"] = required
	}

	return prop
}

// NewArrayProperty creates a JSON schema property for an array parameter.
func NewArrayProperty(description string, items map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"type":        "array",
		"description": description,
		"items":       items,
	}
}

// PR Tool Definitions

// NewPRListTool creates a tool definition for listing pull requests.
func NewPRListTool() Tool {
	return Tool{
		Name:        "pr_list",
		Title:       "List Pull Requests",
		Description: "List pull requests in a Bitbucket repository with optional state filtering",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
			"state":      NewStringProperty("Optional state filter: OPEN, MERGED, DECLINED, or SUPERSEDED"),
			"page":       NewNumberProperty("Optional page number (default: 1)"),
		}, []string{"repository"}),
	}
}

// NewPRViewTool creates a tool definition for viewing a pull request.
func NewPRViewTool() Tool {
	return Tool{
		Name:        "pr_view",
		Title:       "View Pull Request",
		Description: "View detailed information about a specific pull request",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
			"pr_id":      NewStringProperty("Pull request ID"),
		}, []string{"repository", "pr_id"}),
	}
}

// NewPRCreateTool creates a tool definition for creating a pull request.
func NewPRCreateTool() Tool {
	return Tool{
		Name:        "pr_create",
		Title:       "Create Pull Request",
		Description: "Create a new pull request in a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository":   NewStringProperty("Repository in format workspace/repo-slug"),
			"title":        NewStringProperty("Pull request title"),
			"source":       NewStringProperty("Source branch name"),
			"description":  NewStringProperty("Optional pull request description"),
			"destination":  NewStringProperty("Optional destination branch (defaults to main branch)"),
			"close_branch": NewBooleanProperty("Optional: close source branch after merge (default: false)"),
		}, []string{"repository", "title", "source"}),
	}
}

// Issue Tool Definitions

// NewIssueListTool creates a tool definition for listing issues.
func NewIssueListTool() Tool {
	return Tool{
		Name:        "issue_list",
		Title:       "List Issues",
		Description: "List issues in a Bitbucket repository with optional state filtering",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
			"state":      NewStringProperty("Optional state filter: new, open, resolved, on hold, invalid, duplicate, wontfix, closed"),
			"page":       NewNumberProperty("Optional page number (default: 1)"),
		}, []string{"repository"}),
	}
}

// NewIssueCreateTool creates a tool definition for creating an issue.
func NewIssueCreateTool() Tool {
	return Tool{
		Name:        "issue_create",
		Title:       "Create Issue",
		Description: "Create a new issue in a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
			"title":      NewStringProperty("Issue title"),
			"content":    NewStringProperty("Optional issue description"),
			"kind":       NewStringProperty("Optional issue kind: bug, enhancement, proposal, task (default: bug)"),
			"priority":   NewStringProperty("Optional priority: trivial, minor, major, critical, blocker (default: major)"),
		}, []string{"repository", "title"}),
	}
}

// Pipeline Tool Definitions

// NewPipelineListTool creates a tool definition for listing pipelines.
func NewPipelineListTool() Tool {
	return Tool{
		Name:        "pipeline_list",
		Title:       "List Pipelines",
		Description: "List CI/CD pipelines in a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
			"page":       NewNumberProperty("Optional page number (default: 1)"),
		}, []string{"repository"}),
	}
}

// NewPipelineTriggerTool creates a tool definition for triggering a pipeline.
func NewPipelineTriggerTool() Tool {
	return Tool{
		Name:        "pipeline_trigger",
		Title:       "Trigger Pipeline",
		Description: "Trigger a new CI/CD pipeline in a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
			"branch":     NewStringProperty("Optional branch to run pipeline on (default: main)"),
			"pattern":    NewStringProperty("Optional custom pipeline pattern name"),
			"custom":     NewBooleanProperty("Optional: trigger a custom pipeline (default: false)"),
		}, []string{"repository"}),
	}
}
