package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
)

// PipelineListHandler handles the pipeline_list tool invocation.
func PipelineListHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
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
	client, err := api.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Build API path
	path := fmt.Sprintf("/repositories/%s/pipelines/?pagelen=20&page=%d&sort=-created_on", repository, page)

	// Fetch pipelines
	pipelines, err := api.GetPaginated[Pipeline](client, path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pipelines: %w", err)
	}

	// Convert to JSON
	data, err := json.Marshal(pipelines)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pipelines: %w", err)
	}

	return []Content{NewTextContent(string(data))}, nil
}

// PipelineTriggerHandler handles the pipeline_trigger tool invocation.
func PipelineTriggerHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	// Extract required parameter
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	// Extract optional parameters
	branch, _ := args["branch"].(string)
	if branch == "" {
		branch = "main" // Default to main
	}
	pattern, _ := args["pattern"].(string)
	custom := false
	if customFlag, ok := args["custom"].(bool); ok {
		custom = customFlag
	}

	// Create API client
	client, err := api.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Build request body
	target := map[string]interface{}{
		"ref_type": "branch",
		"type":     "pipeline_ref_target",
		"ref_name": branch,
	}
	if custom && pattern != "" {
		target["selector"] = map[string]string{
			"type":    "custom",
			"pattern": pattern,
		}
	}

	body := map[string]interface{}{
		"target": target,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Trigger pipeline
	path := fmt.Sprintf("/repositories/%s/pipelines/", repository)
	data, err := client.Post(path, string(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to trigger pipeline: %w", err)
	}

	var pipeline Pipeline
	if err := json.Unmarshal(data, &pipeline); err != nil {
		return nil, fmt.Errorf("failed to unmarshal triggered pipeline: %w", err)
	}

	// Convert to JSON
	result, err := json.Marshal(pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pipeline: %w", err)
	}

	return []Content{NewTextContent(string(result))}, nil
}

// Pipeline represents a Bitbucket pipeline.
// This is copied from cmd/pipeline/pipeline.go to avoid circular dependencies.
type Pipeline struct {
	UUID        string `json:"uuid"`
	BuildNumber int    `json:"build_number"`
	State       struct {
		Name   string `json:"name"`
		Result *struct {
			Name string `json:"name"`
		} `json:"result"`
		Stage *struct {
			Name string `json:"name"`
		} `json:"stage"`
	} `json:"state"`
	Target struct {
		Type     string `json:"type"`
		RefType  string `json:"ref_type"`
		RefName  string `json:"ref_name"`
		Selector struct {
			Type    string `json:"type"`
			Pattern string `json:"pattern"`
		} `json:"selector"`
	} `json:"target"`
	Creator struct {
		DisplayName string `json:"display_name"`
	} `json:"creator"`
	CreatedOn         string `json:"created_on"`
	CompletedOn       string `json:"completed_on"`
	DurationInSeconds int    `json:"duration_in_seconds"`
}
