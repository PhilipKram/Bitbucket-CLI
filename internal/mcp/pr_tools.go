package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
)

// PRListHandler handles the pr_list tool invocation.
func PRListHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	// Extract required parameter
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
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
	path := fmt.Sprintf("/repositories/%s/pullrequests?pagelen=25&page=%d", repository, page)
	if state != "" {
		path += "&state=" + url.QueryEscape(strings.ToUpper(state))
	}

	// Fetch pull requests
	prs, err := api.GetPaginated[PullRequest](client, path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pull requests: %w", err)
	}

	// Convert to JSON
	data, err := json.Marshal(prs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pull requests: %w", err)
	}

	return []Content{NewTextContent(string(data))}, nil
}

// PRViewHandler handles the pr_view tool invocation.
func PRViewHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	// Extract required parameters
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}

	prID, ok := args["pr_id"].(string)
	if !ok || prID == "" {
		return nil, fmt.Errorf("pr_id parameter is required")
	}

	// Create API client
	client, err := api.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Fetch pull request
	path := fmt.Sprintf("/repositories/%s/pullrequests/%s", repository, prID)
	data, err := client.Get(path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pull request: %w", err)
	}

	var pr PullRequest
	if err := json.Unmarshal(data, &pr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pull request: %w", err)
	}

	// Convert to JSON
	result, err := json.Marshal(pr)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pull request: %w", err)
	}

	return []Content{NewTextContent(string(result))}, nil
}

// PRCreateHandler handles the pr_create tool invocation.
func PRCreateHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	// Extract required parameters
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}

	title, ok := args["title"].(string)
	if !ok || title == "" {
		return nil, fmt.Errorf("title parameter is required")
	}

	source, ok := args["source"].(string)
	if !ok || source == "" {
		return nil, fmt.Errorf("source parameter is required")
	}

	// Extract optional parameters
	description, _ := args["description"].(string)
	destination, _ := args["destination"].(string)
	closeBranch := false
	if cb, ok := args["close_branch"].(bool); ok {
		closeBranch = cb
	}

	// Create API client
	client, err := api.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Build request body
	body := map[string]interface{}{
		"title":               title,
		"description":         description,
		"close_source_branch": closeBranch,
		"source": map[string]interface{}{
			"branch": map[string]string{"name": source},
		},
	}
	if destination != "" {
		body["destination"] = map[string]interface{}{
			"branch": map[string]string{"name": destination},
		}
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create pull request
	path := fmt.Sprintf("/repositories/%s/pullrequests", repository)
	data, err := client.Post(path, string(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	var pr PullRequest
	if err := json.Unmarshal(data, &pr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal created pull request: %w", err)
	}

	// Convert to JSON
	result, err := json.Marshal(pr)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pull request: %w", err)
	}

	return []Content{NewTextContent(string(result))}, nil
}

// PullRequest represents a Bitbucket pull request.
// This is copied from cmd/pr/pr.go to avoid circular dependencies.
type PullRequest struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	CreatedOn   string `json:"created_on"`
	UpdatedOn   string `json:"updated_on"`
	Author      struct {
		DisplayName string `json:"display_name"`
	} `json:"author"`
	Source struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	} `json:"source"`
	Destination struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
	} `json:"destination"`
	CloseSourceBranch bool `json:"close_source_branch"`
	MergeCommit       *struct {
		Hash string `json:"hash"`
	} `json:"merge_commit"`
	CommentCount int `json:"comment_count"`
	TaskCount    int `json:"task_count"`
	Links        struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
	Reviewers []struct {
		DisplayName string `json:"display_name"`
		UUID        string `json:"uuid"`
	} `json:"reviewers"`
	Participants []struct {
		User struct {
			DisplayName string `json:"display_name"`
		} `json:"user"`
		Role     string `json:"role"`
		Approved bool   `json:"approved"`
	} `json:"participants"`
}
