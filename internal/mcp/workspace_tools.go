package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
)

// Workspace represents a Bitbucket workspace.
type Workspace struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	IsPrivate bool   `json:"is_private"`
	CreatedOn string `json:"created_on"`
	Links     struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

// Project represents a Bitbucket project within a workspace.
type Project struct {
	UUID        string `json:"uuid"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"is_private"`
	CreatedOn   string `json:"created_on"`
	Links       struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

// WorkspaceMember represents a member of a Bitbucket workspace.
type WorkspaceMember struct {
	User struct {
		DisplayName string `json:"display_name"`
		UUID        string `json:"uuid"`
		Nickname    string `json:"nickname"`
	} `json:"user"`
}

// WorkspaceListHandler handles the workspace_list tool invocation.
func WorkspaceListHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	workspaces, err := api.GetPaginated[Workspace](client, "/workspaces?pagelen=50")
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}

	data, err := json.Marshal(workspaces)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal workspaces: %w", err)
	}

	return []Content{NewTextContent(string(data))}, nil
}

// WorkspaceViewHandler handles the workspace_view tool invocation.
func WorkspaceViewHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	workspace, ok := args["workspace"].(string)
	if !ok || workspace == "" {
		return nil, fmt.Errorf("workspace parameter is required")
	}

	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	path := fmt.Sprintf("/workspaces/%s", url.PathEscape(workspace))
	data, err := client.Get(path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch workspace: %w", err)
	}

	var ws Workspace
	if err := json.Unmarshal(data, &ws); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workspace: %w", err)
	}

	result, err := json.Marshal(ws)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal workspace: %w", err)
	}

	return []Content{NewTextContent(string(result))}, nil
}

// WorkspaceMembersHandler handles the workspace_members tool invocation.
func WorkspaceMembersHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	workspace, ok := args["workspace"].(string)
	if !ok || workspace == "" {
		return nil, fmt.Errorf("workspace parameter is required")
	}

	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	path := fmt.Sprintf("/workspaces/%s/members?pagelen=50", url.PathEscape(workspace))
	members, err := api.GetPaginated[WorkspaceMember](client, path)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspace members: %w", err)
	}

	data, err := json.Marshal(members)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal members: %w", err)
	}

	return []Content{NewTextContent(string(data))}, nil
}

// WorkspaceProjectsHandler handles the workspace_projects tool invocation.
func WorkspaceProjectsHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	workspace, ok := args["workspace"].(string)
	if !ok || workspace == "" {
		return nil, fmt.Errorf("workspace parameter is required")
	}

	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	path := fmt.Sprintf("/workspaces/%s/projects?pagelen=50", url.PathEscape(workspace))
	projects, err := api.GetPaginated[Project](client, path)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	data, err := json.Marshal(projects)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal projects: %w", err)
	}

	return []Content{NewTextContent(string(data))}, nil
}

// WorkspaceProjectCreateHandler handles the workspace_project_create tool invocation.
func WorkspaceProjectCreateHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	workspace, ok := args["workspace"].(string)
	if !ok || workspace == "" {
		return nil, fmt.Errorf("workspace parameter is required")
	}

	key, ok := args["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("key parameter is required")
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name parameter is required")
	}

	description, _ := args["description"].(string)
	isPrivate := true
	if p, ok := args["is_private"].(bool); ok {
		isPrivate = p
	}

	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	body := map[string]interface{}{
		"name":        name,
		"key":         key,
		"description": description,
		"is_private":  isPrivate,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	path := fmt.Sprintf("/workspaces/%s/projects", url.PathEscape(workspace))
	data, err := client.Post(path, string(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	var project Project
	if err := json.Unmarshal(data, &project); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project: %w", err)
	}

	result, err := json.Marshal(project)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal project: %w", err)
	}

	return []Content{NewTextContent(string(result))}, nil
}

// WorkspacePermissionsHandler handles the workspace_permissions tool invocation.
func WorkspacePermissionsHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	workspace, ok := args["workspace"].(string)
	if !ok || workspace == "" {
		return nil, fmt.Errorf("workspace parameter is required")
	}

	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	path := fmt.Sprintf("/workspaces/%s/permissions?pagelen=50", url.PathEscape(workspace))
	data, err := client.Get(path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch workspace permissions: %w", err)
	}

	return []Content{NewTextContent(string(data))}, nil
}
