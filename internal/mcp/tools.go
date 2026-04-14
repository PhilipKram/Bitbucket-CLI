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
	if registry == nil {
		return
	}
	s.handlersMu.Lock()
	defer s.handlersMu.Unlock()

	// Override the tools/list handler to use the registry
	s.handlers["tools/list"] = func(req *Request) (map[string]interface{}, error) {
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
	}

	// Override the tools/call handler to use the registry
	s.handlers["tools/call"] = func(req *Request) (map[string]interface{}, error) {
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
	}
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

// NewPRApproveTool creates a tool definition for approving a pull request.
func NewPRApproveTool() Tool {
	return Tool{
		Name:        "pr_approve",
		Title:       "Approve Pull Request",
		Description: "Approve a pull request in a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
			"pr_id":      NewStringProperty("Pull request ID"),
		}, []string{"repository", "pr_id"}),
	}
}

// NewPRMergeTool creates a tool definition for merging a pull request.
func NewPRMergeTool() Tool {
	return Tool{
		Name:        "pr_merge",
		Title:       "Merge Pull Request",
		Description: "Merge a pull request in a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository":          NewStringProperty("Repository in format workspace/repo-slug"),
			"pr_id":               NewStringProperty("Pull request ID"),
			"merge_strategy":      NewStringProperty("Optional merge strategy: merge_commit, squash, or fast_forward"),
			"close_source_branch": NewBooleanProperty("Optional: close source branch after merge"),
		}, []string{"repository", "pr_id"}),
	}
}

// NewPRDeclineTool creates a tool definition for declining a pull request.
func NewPRDeclineTool() Tool {
	return Tool{
		Name:        "pr_decline",
		Title:       "Decline Pull Request",
		Description: "Decline a pull request in a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
			"pr_id":      NewStringProperty("Pull request ID"),
		}, []string{"repository", "pr_id"}),
	}
}

// NewPRDiffTool creates a tool definition for getting a pull request diff.
func NewPRDiffTool() Tool {
	return Tool{
		Name:        "pr_diff",
		Title:       "Get Pull Request Diff",
		Description: "Get the diff of a pull request in a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
			"pr_id":      NewStringProperty("Pull request ID"),
		}, []string{"repository", "pr_id"}),
	}
}

// NewPRCommentTool creates a tool definition for commenting on a pull request.
func NewPRCommentTool() Tool {
	return Tool{
		Name:        "pr_comment",
		Title:       "Comment on Pull Request",
		Description: "Add a comment to a pull request in a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
			"pr_id":      NewStringProperty("Pull request ID"),
			"content":    NewStringProperty("Comment content in raw/markdown format"),
		}, []string{"repository", "pr_id", "content"}),
	}
}

// NewPRCommentsListTool creates a tool definition for listing PR comments.
func NewPRCommentsListTool() Tool {
	return Tool{
		Name:        "pr_comments",
		Title:       "List Pull Request Comments",
		Description: "List all comments on a pull request, including inline code review comments and general comments",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
			"pr_id":      NewStringProperty("Pull request ID"),
		}, []string{"repository", "pr_id"}),
	}
}

// NewPREditTool creates a tool definition for editing a pull request.
func NewPREditTool() Tool {
	return Tool{
		Name:        "pr_edit",
		Title:       "Edit Pull Request",
		Description: "Edit a pull request in a Bitbucket repository (update title, description, destination branch, or close-branch setting)",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository":          NewStringProperty("Repository in format workspace/repo-slug"),
			"pr_id":               NewStringProperty("Pull request ID"),
			"title":               NewStringProperty("Optional: new PR title"),
			"description":         NewStringProperty("Optional: new PR description"),
			"destination":         NewStringProperty("Optional: new destination branch"),
			"close_source_branch": NewBooleanProperty("Optional: close source branch after merge"),
		}, []string{"repository", "pr_id"}),
	}
}

// NewPRUnapproveTool creates a tool definition for removing approval from a pull request.
func NewPRUnapproveTool() Tool {
	return Tool{
		Name:        "pr_unapprove",
		Title:       "Unapprove Pull Request",
		Description: "Remove approval from a pull request in a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
			"pr_id":      NewStringProperty("Pull request ID"),
		}, []string{"repository", "pr_id"}),
	}
}

// NewPRActivityTool creates a tool definition for viewing pull request activity.
func NewPRActivityTool() Tool {
	return Tool{
		Name:        "pr_activity",
		Title:       "Pull Request Activity",
		Description: "View the activity log of a pull request (updates, approvals, comments)",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
			"pr_id":      NewStringProperty("Pull request ID"),
		}, []string{"repository", "pr_id"}),
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

// NewIssueViewTool creates a tool definition for viewing an issue.
func NewIssueViewTool() Tool {
	return Tool{
		Name:        "issue_view",
		Title:       "View Issue",
		Description: "View detailed information about a specific issue in a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
			"issue_id":   NewStringProperty("Issue ID"),
		}, []string{"repository", "issue_id"}),
	}
}

// NewIssueEditTool creates a tool definition for editing an issue.
func NewIssueEditTool() Tool {
	return Tool{
		Name:        "issue_edit",
		Title:       "Edit Issue",
		Description: "Edit an existing issue in a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
			"issue_id":   NewStringProperty("Issue ID"),
			"title":      NewStringProperty("Optional new issue title"),
			"content":    NewStringProperty("Optional new issue description"),
			"kind":       NewStringProperty("Optional issue kind: bug, enhancement, proposal, task"),
			"priority":   NewStringProperty("Optional priority: trivial, minor, major, critical, blocker"),
			"state":      NewStringProperty("Optional state: new, open, resolved, on hold, invalid, duplicate, wontfix, closed"),
		}, []string{"repository", "issue_id"}),
	}
}

// NewIssueDeleteTool creates a tool definition for deleting an issue.
func NewIssueDeleteTool() Tool {
	return Tool{
		Name:        "issue_delete",
		Title:       "Delete Issue",
		Description: "Delete an issue from a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
			"issue_id":   NewStringProperty("Issue ID"),
		}, []string{"repository", "issue_id"}),
	}
}

// NewIssueCommentTool creates a tool definition for commenting on an issue.
func NewIssueCommentTool() Tool {
	return Tool{
		Name:        "issue_comment",
		Title:       "Comment on Issue",
		Description: "Add a comment to an issue in a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
			"issue_id":   NewStringProperty("Issue ID"),
			"content":    NewStringProperty("Comment content in raw/markdown format"),
		}, []string{"repository", "issue_id", "content"}),
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

// NewPipelineViewTool creates a tool definition for viewing a pipeline.
func NewPipelineViewTool() Tool {
	return Tool{
		Name:        "pipeline_view",
		Title:       "View Pipeline",
		Description: "View detailed information about a specific pipeline in a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository":    NewStringProperty("Repository in format workspace/repo-slug"),
			"pipeline_uuid": NewStringProperty("Pipeline UUID"),
		}, []string{"repository", "pipeline_uuid"}),
	}
}

// NewPipelineStopTool creates a tool definition for stopping a pipeline.
func NewPipelineStopTool() Tool {
	return Tool{
		Name:        "pipeline_stop",
		Title:       "Stop Pipeline",
		Description: "Stop a running pipeline in a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository":    NewStringProperty("Repository in format workspace/repo-slug"),
			"pipeline_uuid": NewStringProperty("Pipeline UUID"),
		}, []string{"repository", "pipeline_uuid"}),
	}
}

// Repo Tool Definitions

// NewRepoListTool creates a tool definition for listing repositories.
func NewRepoListTool() Tool {
	return Tool{
		Name:        "repo_list",
		Title:       "List Repositories",
		Description: "List repositories in a Bitbucket workspace",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"workspace": NewStringProperty("Bitbucket workspace slug"),
			"page":      NewNumberProperty("Optional page number (default: 1)"),
		}, []string{"workspace"}),
	}
}

// NewRepoViewTool creates a tool definition for viewing a repository.
func NewRepoViewTool() Tool {
	return Tool{
		Name:        "repo_view",
		Title:       "View Repository",
		Description: "View detailed information about a specific Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
		}, []string{"repository"}),
	}
}

// Snippet Tool Definitions

// NewSnippetListTool creates a tool definition for listing snippets.
func NewSnippetListTool() Tool {
	return Tool{
		Name:        "snippet_list",
		Title:       "List Snippets",
		Description: "List snippets in a Bitbucket workspace",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"workspace": NewStringProperty("Bitbucket workspace slug"),
		}, []string{"workspace"}),
	}
}

// NewSnippetViewTool creates a tool definition for viewing a snippet.
func NewSnippetViewTool() Tool {
	return Tool{
		Name:        "snippet_view",
		Title:       "View Snippet",
		Description: "View detailed information about a specific snippet",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"workspace":  NewStringProperty("Bitbucket workspace slug"),
			"snippet_id": NewStringProperty("Snippet ID"),
		}, []string{"workspace", "snippet_id"}),
	}
}

// Branch Tool Definitions

// NewBranchListTool creates a tool definition for listing branches.
func NewBranchListTool() Tool {
	return Tool{
		Name:        "branch_list",
		Title:       "List Branches",
		Description: "List branches in a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
			"page":       NewNumberProperty("Optional page number (default: 1)"),
		}, []string{"repository"}),
	}
}

// Workspace Tool Definitions

// NewWorkspaceListTool creates a tool definition for listing workspaces.
func NewWorkspaceListTool() Tool {
	return Tool{
		Name:        "workspace_list",
		Title:       "List Workspaces",
		Description: "List Bitbucket workspaces you belong to",
		InputSchema: NewJSONSchema("object", map[string]interface{}{}, nil),
	}
}

// NewWorkspaceViewTool creates a tool definition for viewing a workspace.
func NewWorkspaceViewTool() Tool {
	return Tool{
		Name:        "workspace_view",
		Title:       "View Workspace",
		Description: "View details of a Bitbucket workspace",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"workspace": NewStringProperty("Workspace slug"),
		}, []string{"workspace"}),
	}
}

// NewWorkspaceMembersTool creates a tool definition for listing workspace members.
func NewWorkspaceMembersTool() Tool {
	return Tool{
		Name:        "workspace_members",
		Title:       "List Workspace Members",
		Description: "List members of a Bitbucket workspace",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"workspace": NewStringProperty("Workspace slug"),
		}, []string{"workspace"}),
	}
}

// NewWorkspaceProjectsTool creates a tool definition for listing workspace projects.
func NewWorkspaceProjectsTool() Tool {
	return Tool{
		Name:        "workspace_projects",
		Title:       "List Workspace Projects",
		Description: "List projects in a Bitbucket workspace",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"workspace": NewStringProperty("Workspace slug"),
		}, []string{"workspace"}),
	}
}

// NewWorkspaceProjectCreateTool creates a tool definition for creating a project.
func NewWorkspaceProjectCreateTool() Tool {
	return Tool{
		Name:        "workspace_project_create",
		Title:       "Create Workspace Project",
		Description: "Create a new project in a Bitbucket workspace",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"workspace":   NewStringProperty("Workspace slug"),
			"key":         NewStringProperty("Project key (short uppercase identifier)"),
			"name":        NewStringProperty("Project name"),
			"description": NewStringProperty("Optional project description"),
			"is_private":  NewBooleanProperty("Optional: make project private (default: true)"),
		}, []string{"workspace", "key", "name"}),
	}
}

// NewWorkspacePermissionsTool creates a tool definition for listing workspace permissions.
func NewWorkspacePermissionsTool() Tool {
	return Tool{
		Name:        "workspace_permissions",
		Title:       "List Workspace Permissions",
		Description: "List permissions in a Bitbucket workspace",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"workspace": NewStringProperty("Workspace slug"),
		}, []string{"workspace"}),
	}
}

// User Tool Definitions

// NewUserMeTool creates a tool definition for viewing the current user.
func NewUserMeTool() Tool {
	return Tool{
		Name:        "user_me",
		Title:       "Current User",
		Description: "Show the current authenticated Bitbucket user",
		InputSchema: NewJSONSchema("object", map[string]interface{}{}, nil),
	}
}

// NewUserViewTool creates a tool definition for viewing a user profile.
func NewUserViewTool() Tool {
	return Tool{
		Name:        "user_view",
		Title:       "View User",
		Description: "View a Bitbucket user's profile by UUID or username",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"user_id": NewStringProperty("User UUID or username"),
		}, []string{"user_id"}),
	}
}

// NewUserEmailsTool creates a tool definition for listing user emails.
func NewUserEmailsTool() Tool {
	return Tool{
		Name:        "user_emails",
		Title:       "List User Emails",
		Description: "List email addresses of the current authenticated user",
		InputSchema: NewJSONSchema("object", map[string]interface{}{}, nil),
	}
}

// NewUserSSHKeysTool creates a tool definition for listing SSH keys.
func NewUserSSHKeysTool() Tool {
	return Tool{
		Name:        "user_ssh_keys",
		Title:       "List SSH Keys",
		Description: "List SSH keys of the current authenticated user",
		InputSchema: NewJSONSchema("object", map[string]interface{}{}, nil),
	}
}

// NewUserSSHKeyAddTool creates a tool definition for adding an SSH key.
func NewUserSSHKeyAddTool() Tool {
	return Tool{
		Name:        "user_ssh_key_add",
		Title:       "Add SSH Key",
		Description: "Add an SSH key to the current authenticated user's account",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"key":   NewStringProperty("SSH public key content"),
			"label": NewStringProperty("Optional label for the key"),
		}, []string{"key"}),
	}
}

// Environment Tool Definitions

// NewEnvironmentListTool creates a tool definition for listing environments.
func NewEnvironmentListTool() Tool {
	return Tool{
		Name:        "environment_list",
		Title:       "List Environments",
		Description: "List deployment environments for a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
		}, []string{"repository"}),
	}
}

// NewEnvironmentViewTool creates a tool definition for viewing an environment.
func NewEnvironmentViewTool() Tool {
	return Tool{
		Name:        "environment_view",
		Title:       "View Environment",
		Description: "View details of a deployment environment",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository":       NewStringProperty("Repository in format workspace/repo-slug"),
			"environment_uuid": NewStringProperty("Environment UUID"),
		}, []string{"repository", "environment_uuid"}),
	}
}

// NewEnvironmentCreateTool creates a tool definition for creating an environment.
func NewEnvironmentCreateTool() Tool {
	return Tool{
		Name:        "environment_create",
		Title:       "Create Environment",
		Description: "Create a deployment environment in a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository":       NewStringProperty("Repository in format workspace/repo-slug"),
			"name":             NewStringProperty("Environment name"),
			"environment_type": NewStringProperty("Environment type: Test, Staging, or Production"),
		}, []string{"repository", "name", "environment_type"}),
	}
}

// NewEnvironmentDeleteTool creates a tool definition for deleting an environment.
func NewEnvironmentDeleteTool() Tool {
	return Tool{
		Name:        "environment_delete",
		Title:       "Delete Environment",
		Description: "Delete a deployment environment from a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository":       NewStringProperty("Repository in format workspace/repo-slug"),
			"environment_uuid": NewStringProperty("Environment UUID"),
		}, []string{"repository", "environment_uuid"}),
	}
}

// Variable Tool Definitions

// NewVariableListTool creates a tool definition for listing pipeline variables.
func NewVariableListTool() Tool {
	return Tool{
		Name:        "variable_list",
		Title:       "List Pipeline Variables",
		Description: "List pipeline variables for a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
		}, []string{"repository"}),
	}
}

// NewVariableGetTool creates a tool definition for getting a pipeline variable.
func NewVariableGetTool() Tool {
	return Tool{
		Name:        "variable_get",
		Title:       "Get Pipeline Variable",
		Description: "Get a pipeline variable by key",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
			"key":        NewStringProperty("Variable key"),
		}, []string{"repository", "key"}),
	}
}

// NewVariableSetTool creates a tool definition for creating a pipeline variable.
func NewVariableSetTool() Tool {
	return Tool{
		Name:        "variable_set",
		Title:       "Create Pipeline Variable",
		Description: "Create a new pipeline variable in a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
			"key":        NewStringProperty("Variable key"),
			"value":      NewStringProperty("Variable value"),
			"secured":    NewBooleanProperty("Optional: mark variable as secured/encrypted (default: false)"),
		}, []string{"repository", "key", "value"}),
	}
}

// NewVariableUpdateTool creates a tool definition for updating a pipeline variable.
func NewVariableUpdateTool() Tool {
	return Tool{
		Name:        "variable_update",
		Title:       "Update Pipeline Variable",
		Description: "Update an existing pipeline variable in a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
			"key":        NewStringProperty("Variable key"),
			"value":      NewStringProperty("New variable value"),
			"secured":    NewBooleanProperty("Optional: mark variable as secured/encrypted (default: false)"),
		}, []string{"repository", "key", "value"}),
	}
}

// NewVariableDeleteTool creates a tool definition for deleting a pipeline variable.
func NewVariableDeleteTool() Tool {
	return Tool{
		Name:        "variable_delete",
		Title:       "Delete Pipeline Variable",
		Description: "Delete a pipeline variable from a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
			"key":        NewStringProperty("Variable key"),
		}, []string{"repository", "key"}),
	}
}

// Download Tool Definitions

// NewDownloadListTool creates a tool definition for listing repository downloads.
func NewDownloadListTool() Tool {
	return Tool{
		Name:        "download_list",
		Title:       "List Downloads",
		Description: "List repository downloads in a Bitbucket repository",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
		}, []string{"repository"}),
	}
}

// NewDownloadDeleteTool creates a tool definition for deleting a repository download.
func NewDownloadDeleteTool() Tool {
	return Tool{
		Name:        "download_delete",
		Title:       "Delete Download",
		Description: "Delete a file from repository downloads",
		InputSchema: NewJSONSchema("object", map[string]interface{}{
			"repository": NewStringProperty("Repository in format workspace/repo-slug"),
			"filename":   NewStringProperty("Filename to delete"),
		}, []string{"repository", "filename"}),
	}
}

// RegisterDefaultTools registers all default bb tools with the given registry.
// This includes PR, Issue, Pipeline, Repo, Snippet, Branch, Workspace, User, Environment, Variable, and Download tools.
func RegisterDefaultTools(registry *ToolRegistry) error {
	// PR Tools
	if err := registry.Register(NewPRListTool(), PRListHandler); err != nil {
		return fmt.Errorf("failed to register pr_list: %w", err)
	}
	if err := registry.Register(NewPRViewTool(), PRViewHandler); err != nil {
		return fmt.Errorf("failed to register pr_view: %w", err)
	}
	if err := registry.Register(NewPRCreateTool(), PRCreateHandler); err != nil {
		return fmt.Errorf("failed to register pr_create: %w", err)
	}
	if err := registry.Register(NewPRApproveTool(), PRApproveHandler); err != nil {
		return fmt.Errorf("failed to register pr_approve: %w", err)
	}
	if err := registry.Register(NewPRMergeTool(), PRMergeHandler); err != nil {
		return fmt.Errorf("failed to register pr_merge: %w", err)
	}
	if err := registry.Register(NewPRDeclineTool(), PRDeclineHandler); err != nil {
		return fmt.Errorf("failed to register pr_decline: %w", err)
	}
	if err := registry.Register(NewPRDiffTool(), PRDiffHandler); err != nil {
		return fmt.Errorf("failed to register pr_diff: %w", err)
	}
	if err := registry.Register(NewPRCommentTool(), PRCommentHandler); err != nil {
		return fmt.Errorf("failed to register pr_comment: %w", err)
	}
	if err := registry.Register(NewPRCommentsListTool(), PRCommentsListHandler); err != nil {
		return fmt.Errorf("failed to register pr_comments: %w", err)
	}
	if err := registry.Register(NewPREditTool(), PREditHandler); err != nil {
		return fmt.Errorf("failed to register pr_edit: %w", err)
	}
	if err := registry.Register(NewPRUnapproveTool(), PRUnapproveHandler); err != nil {
		return fmt.Errorf("failed to register pr_unapprove: %w", err)
	}
	if err := registry.Register(NewPRActivityTool(), PRActivityHandler); err != nil {
		return fmt.Errorf("failed to register pr_activity: %w", err)
	}

	// Issue Tools
	if err := registry.Register(NewIssueListTool(), IssueListHandler); err != nil {
		return fmt.Errorf("failed to register issue_list: %w", err)
	}
	if err := registry.Register(NewIssueCreateTool(), IssueCreateHandler); err != nil {
		return fmt.Errorf("failed to register issue_create: %w", err)
	}
	if err := registry.Register(NewIssueViewTool(), IssueViewHandler); err != nil {
		return fmt.Errorf("failed to register issue_view: %w", err)
	}
	if err := registry.Register(NewIssueEditTool(), IssueEditHandler); err != nil {
		return fmt.Errorf("failed to register issue_edit: %w", err)
	}
	if err := registry.Register(NewIssueDeleteTool(), IssueDeleteHandler); err != nil {
		return fmt.Errorf("failed to register issue_delete: %w", err)
	}
	if err := registry.Register(NewIssueCommentTool(), IssueCommentHandler); err != nil {
		return fmt.Errorf("failed to register issue_comment: %w", err)
	}

	// Pipeline Tools
	if err := registry.Register(NewPipelineListTool(), PipelineListHandler); err != nil {
		return fmt.Errorf("failed to register pipeline_list: %w", err)
	}
	if err := registry.Register(NewPipelineTriggerTool(), PipelineTriggerHandler); err != nil {
		return fmt.Errorf("failed to register pipeline_trigger: %w", err)
	}
	if err := registry.Register(NewPipelineViewTool(), PipelineViewHandler); err != nil {
		return fmt.Errorf("failed to register pipeline_view: %w", err)
	}
	if err := registry.Register(NewPipelineStopTool(), PipelineStopHandler); err != nil {
		return fmt.Errorf("failed to register pipeline_stop: %w", err)
	}

	// Repo Tools
	if err := registry.Register(NewRepoListTool(), RepoListHandler); err != nil {
		return fmt.Errorf("failed to register repo_list: %w", err)
	}
	if err := registry.Register(NewRepoViewTool(), RepoViewHandler); err != nil {
		return fmt.Errorf("failed to register repo_view: %w", err)
	}

	// Snippet Tools
	if err := registry.Register(NewSnippetListTool(), SnippetListHandler); err != nil {
		return fmt.Errorf("failed to register snippet_list: %w", err)
	}
	if err := registry.Register(NewSnippetViewTool(), SnippetViewHandler); err != nil {
		return fmt.Errorf("failed to register snippet_view: %w", err)
	}

	// Branch Tools
	if err := registry.Register(NewBranchListTool(), BranchListHandler); err != nil {
		return fmt.Errorf("failed to register branch_list: %w", err)
	}

	// Workspace Tools
	if err := registry.Register(NewWorkspaceListTool(), WorkspaceListHandler); err != nil {
		return fmt.Errorf("failed to register workspace_list: %w", err)
	}
	if err := registry.Register(NewWorkspaceViewTool(), WorkspaceViewHandler); err != nil {
		return fmt.Errorf("failed to register workspace_view: %w", err)
	}
	if err := registry.Register(NewWorkspaceMembersTool(), WorkspaceMembersHandler); err != nil {
		return fmt.Errorf("failed to register workspace_members: %w", err)
	}
	if err := registry.Register(NewWorkspaceProjectsTool(), WorkspaceProjectsHandler); err != nil {
		return fmt.Errorf("failed to register workspace_projects: %w", err)
	}
	if err := registry.Register(NewWorkspaceProjectCreateTool(), WorkspaceProjectCreateHandler); err != nil {
		return fmt.Errorf("failed to register workspace_project_create: %w", err)
	}
	if err := registry.Register(NewWorkspacePermissionsTool(), WorkspacePermissionsHandler); err != nil {
		return fmt.Errorf("failed to register workspace_permissions: %w", err)
	}

	// User Tools
	if err := registry.Register(NewUserMeTool(), UserMeHandler); err != nil {
		return fmt.Errorf("failed to register user_me: %w", err)
	}
	if err := registry.Register(NewUserViewTool(), UserViewHandler); err != nil {
		return fmt.Errorf("failed to register user_view: %w", err)
	}
	if err := registry.Register(NewUserEmailsTool(), UserEmailsHandler); err != nil {
		return fmt.Errorf("failed to register user_emails: %w", err)
	}
	if err := registry.Register(NewUserSSHKeysTool(), UserSSHKeysHandler); err != nil {
		return fmt.Errorf("failed to register user_ssh_keys: %w", err)
	}
	if err := registry.Register(NewUserSSHKeyAddTool(), UserSSHKeyAddHandler); err != nil {
		return fmt.Errorf("failed to register user_ssh_key_add: %w", err)
	}

	// Environment Tools
	if err := registry.Register(NewEnvironmentListTool(), EnvironmentListHandler); err != nil {
		return fmt.Errorf("failed to register environment_list: %w", err)
	}
	if err := registry.Register(NewEnvironmentViewTool(), EnvironmentViewHandler); err != nil {
		return fmt.Errorf("failed to register environment_view: %w", err)
	}
	if err := registry.Register(NewEnvironmentCreateTool(), EnvironmentCreateHandler); err != nil {
		return fmt.Errorf("failed to register environment_create: %w", err)
	}
	if err := registry.Register(NewEnvironmentDeleteTool(), EnvironmentDeleteHandler); err != nil {
		return fmt.Errorf("failed to register environment_delete: %w", err)
	}

	// Variable Tools
	if err := registry.Register(NewVariableListTool(), VariableListHandler); err != nil {
		return fmt.Errorf("failed to register variable_list: %w", err)
	}
	if err := registry.Register(NewVariableGetTool(), VariableGetHandler); err != nil {
		return fmt.Errorf("failed to register variable_get: %w", err)
	}
	if err := registry.Register(NewVariableSetTool(), VariableSetHandler); err != nil {
		return fmt.Errorf("failed to register variable_set: %w", err)
	}
	if err := registry.Register(NewVariableUpdateTool(), VariableUpdateHandler); err != nil {
		return fmt.Errorf("failed to register variable_update: %w", err)
	}
	if err := registry.Register(NewVariableDeleteTool(), VariableDeleteHandler); err != nil {
		return fmt.Errorf("failed to register variable_delete: %w", err)
	}

	// Download Tools
	if err := registry.Register(NewDownloadListTool(), DownloadListHandler); err != nil {
		return fmt.Errorf("failed to register download_list: %w", err)
	}
	if err := registry.Register(NewDownloadDeleteTool(), DownloadDeleteHandler); err != nil {
		return fmt.Errorf("failed to register download_delete: %w", err)
	}

	return nil
}
