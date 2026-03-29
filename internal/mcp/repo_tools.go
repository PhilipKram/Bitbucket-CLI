package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
)

// Repository represents a Bitbucket repository.
type Repository struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	FullName  string `json:"full_name"`
	Slug      string `json:"slug"`
	IsPrivate bool   `json:"is_private"`
	Language  string `json:"language"`
	CreatedOn string `json:"created_on"`
	UpdatedOn string `json:"updated_on"`
	Links     struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
		Clone []struct {
			Href string `json:"href"`
			Name string `json:"name"`
		} `json:"clone"`
	} `json:"links"`
}

// Snippet represents a Bitbucket snippet.
type Snippet struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	IsPrivate bool   `json:"is_private"`
	CreatedOn string `json:"created_on"`
	Creator   struct {
		DisplayName string `json:"display_name"`
	} `json:"creator"`
}

// Branch represents a Bitbucket branch.
type Branch struct {
	Name   string `json:"name"`
	Target struct {
		Hash    string `json:"hash"`
		Date    string `json:"date"`
		Message string `json:"message"`
		Author  struct {
			Raw string `json:"raw"`
		} `json:"author"`
	} `json:"target"`
}

// RepoListHandler handles the repo_list tool invocation.
func RepoListHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	// Extract required parameter
	workspace, ok := args["workspace"].(string)
	if !ok || workspace == "" {
		return nil, fmt.Errorf("workspace parameter is required")
	}

	// Extract optional parameters
	page := 1
	if pageNum, ok := args["page"].(float64); ok {
		page = int(pageNum)
	}

	// Create API client
	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Build API path
	path := fmt.Sprintf("/repositories/%s?pagelen=25&page=%d", url.PathEscape(workspace), page)

	// Fetch repositories
	repos, err := api.GetPaginated[Repository](client, path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repositories: %w", err)
	}

	// Convert to JSON
	data, err := json.Marshal(repos)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal repositories: %w", err)
	}

	return []Content{NewTextContent(string(data))}, nil
}

// RepoViewHandler handles the repo_view tool invocation.
func RepoViewHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	// Extract required parameter
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	// Create API client
	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Fetch repository
	path := fmt.Sprintf("/repositories/%s", repository)
	data, err := client.Get(path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repository: %w", err)
	}

	var repo Repository
	if err := json.Unmarshal(data, &repo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal repository: %w", err)
	}

	// Convert to JSON
	result, err := json.Marshal(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal repository: %w", err)
	}

	return []Content{NewTextContent(string(result))}, nil
}

// SnippetListHandler handles the snippet_list tool invocation.
func SnippetListHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	// Extract required parameter
	workspace, ok := args["workspace"].(string)
	if !ok || workspace == "" {
		return nil, fmt.Errorf("workspace parameter is required")
	}

	// Create API client
	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Build API path
	path := fmt.Sprintf("/snippets/%s?pagelen=25", url.PathEscape(workspace))

	// Fetch snippets
	snippets, err := api.GetPaginated[Snippet](client, path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch snippets: %w", err)
	}

	// Convert to JSON
	data, err := json.Marshal(snippets)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snippets: %w", err)
	}

	return []Content{NewTextContent(string(data))}, nil
}

// SnippetViewHandler handles the snippet_view tool invocation.
func SnippetViewHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	// Extract required parameters
	workspace, ok := args["workspace"].(string)
	if !ok || workspace == "" {
		return nil, fmt.Errorf("workspace parameter is required")
	}

	snippetID, ok := args["snippet_id"].(string)
	if !ok || snippetID == "" {
		return nil, fmt.Errorf("snippet_id parameter is required")
	}

	// Create API client
	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Fetch snippet
	path := fmt.Sprintf("/snippets/%s/%s", url.PathEscape(workspace), url.PathEscape(snippetID))
	data, err := client.Get(path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch snippet: %w", err)
	}

	var snippet Snippet
	if err := json.Unmarshal(data, &snippet); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snippet: %w", err)
	}

	// Convert to JSON
	result, err := json.Marshal(snippet)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snippet: %w", err)
	}

	return []Content{NewTextContent(string(result))}, nil
}

// BranchListHandler handles the branch_list tool invocation.
func BranchListHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	// Extract required parameter
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	// Extract optional parameters
	page := 1
	if pageNum, ok := args["page"].(float64); ok {
		page = int(pageNum)
	}

	// Create API client
	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Build API path
	path := fmt.Sprintf("/repositories/%s/refs/branches?pagelen=25&page=%d", repository, page)

	// Fetch branches
	branches, err := api.GetPaginated[Branch](client, path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch branches: %w", err)
	}

	// Convert to JSON
	data, err := json.Marshal(branches)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal branches: %w", err)
	}

	return []Content{NewTextContent(string(data))}, nil
}
