package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
)

// IssueListHandler handles the issue_list tool invocation.
func IssueListHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	// Extract required parameter
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	// Extract optional parameters
	state, _ := args["state"].(string)
	page := 1
	if pageNum, ok := args["page"].(float64); ok {
		page = int(pageNum)
	}

	// Create API client
	client, err := api.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Build API path
	path := fmt.Sprintf("/repositories/%s/issues?pagelen=25&page=%d", repository, page)
	if state != "" {
		path += fmt.Sprintf("&q=state%%3D%%22%s%%22", url.QueryEscape(state))
	}

	// Fetch issues
	issues, err := api.GetPaginated[Issue](client, path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issues: %w", err)
	}

	// Convert to JSON
	data, err := json.Marshal(issues)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal issues: %w", err)
	}

	return []Content{NewTextContent(string(data))}, nil
}

// IssueCreateHandler handles the issue_create tool invocation.
func IssueCreateHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	// Extract required parameters
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	title, ok := args["title"].(string)
	if !ok || title == "" {
		return nil, fmt.Errorf("title parameter is required")
	}

	// Extract optional parameters
	content, _ := args["content"].(string)
	kind, _ := args["kind"].(string)
	if kind == "" {
		kind = "bug" // Default to bug
	}
	priority, _ := args["priority"].(string)
	if priority == "" {
		priority = "major" // Default to major
	}

	// Create API client
	client, err := api.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Build request body
	body := map[string]interface{}{
		"title":    title,
		"kind":     kind,
		"priority": priority,
	}
	if content != "" {
		body["content"] = map[string]interface{}{
			"raw": content,
		}
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create issue
	path := fmt.Sprintf("/repositories/%s/issues", repository)
	data, err := client.Post(path, string(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}

	var issue Issue
	if err := json.Unmarshal(data, &issue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal created issue: %w", err)
	}

	// Convert to JSON
	result, err := json.Marshal(issue)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal issue: %w", err)
	}

	return []Content{NewTextContent(string(result))}, nil
}

// Issue represents a Bitbucket issue.
// This is copied from cmd/issue/issue.go to avoid circular dependencies.
type Issue struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	State    string `json:"state"`
	Priority string `json:"priority"`
	Kind     string `json:"kind"`
	Content  struct {
		Raw string `json:"raw"`
	} `json:"content"`
	Reporter struct {
		DisplayName string `json:"display_name"`
	} `json:"reporter"`
	Assignee *struct {
		DisplayName string `json:"display_name"`
	} `json:"assignee"`
	CreatedOn string `json:"created_on"`
	UpdatedOn string `json:"updated_on"`
	Votes     int    `json:"votes"`
	Component *struct {
		Name string `json:"name"`
	} `json:"component"`
	Milestone *struct {
		Name string `json:"name"`
	} `json:"milestone"`
	Version *struct {
		Name string `json:"name"`
	} `json:"version"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}
