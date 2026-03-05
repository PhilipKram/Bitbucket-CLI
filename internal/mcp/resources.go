package mcp

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
)

// RegisterDefaultResources registers all default resource templates with the server.
func RegisterDefaultResources(s *Server) {
	// README resource
	s.AddResourceTemplate(ResourceTemplate{
		URITemplate: "bitbucket:///{repo}/README.md",
		Name:        "readme",
		Description: "Read the README.md file from a Bitbucket repository",
		MimeType:    "text/markdown",
	}, handleReadmeResource)

	// Pipeline step log resource
	s.AddResourceTemplate(ResourceTemplate{
		URITemplate: "bitbucket:///{repo}/pipeline/{uuid}/step/{step_uuid}/log",
		Name:        "pipeline-step-log",
		Description: "Read the log output of a pipeline step",
		MimeType:    "text/plain",
	}, handlePipelineStepLogResource)

	// PR diff resource
	s.AddResourceTemplate(ResourceTemplate{
		URITemplate: "bitbucket:///{repo}/pr/{id}/diff",
		Name:        "pr-diff",
		Description: "Read the diff of a pull request",
		MimeType:    "text/plain",
	}, handlePRDiffResource)
}

// handleReadmeResource reads the README.md from a repository.
func handleReadmeResource(uri string) (*ResourceReadResult, error) {
	repo, err := extractRepoFromResourceURI(uri)
	if err != nil {
		return nil, err
	}

	client, err := api.NewClient()
	if err != nil {
		return nil, err
	}

	data, err := client.Get(fmt.Sprintf("/repositories/%s/src/HEAD/README.md", repo))
	if err != nil {
		return nil, fmt.Errorf("failed to read README.md: %w", err)
	}

	return &ResourceReadResult{
		Contents: []ResourceContents{
			{
				URI:      uri,
				MimeType: "text/markdown",
				Text:     string(data),
			},
		},
	}, nil
}

// handlePipelineStepLogResource reads the log output of a pipeline step.
func handlePipelineStepLogResource(uri string) (*ResourceReadResult, error) {
	repo, err := extractRepoFromResourceURI(uri)
	if err != nil {
		return nil, err
	}

	// Parse pipeline UUID and step UUID from URI
	// Format: bitbucket:///{owner}/{repo}/pipeline/{uuid}/step/{step_uuid}/log
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URI: %w", err)
	}

	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	// parts: [owner, repo, "pipeline", uuid, "step", step_uuid, "log"]
	if len(parts) < 7 || parts[2] != "pipeline" || parts[4] != "step" {
		return nil, fmt.Errorf("invalid pipeline step log URI: expected /{owner}/{repo}/pipeline/{uuid}/step/{step_uuid}/log format")
	}

	pipelineUUID := parts[3]
	stepUUID := parts[5]

	client, err := api.NewClient()
	if err != nil {
		return nil, err
	}

	data, err := client.Get(fmt.Sprintf("/repositories/%s/pipelines/%s/steps/%s/log", repo, pipelineUUID, stepUUID))
	if err != nil {
		return nil, fmt.Errorf("failed to read pipeline step log: %w", err)
	}

	// Limit to 1 MiB
	const maxSize = 1 << 20
	text := string(data)
	if len(text) > maxSize {
		text = text[:maxSize]
	}

	return &ResourceReadResult{
		Contents: []ResourceContents{
			{
				URI:      uri,
				MimeType: "text/plain",
				Text:     text,
			},
		},
	}, nil
}

// handlePRDiffResource reads the diff of a pull request.
func handlePRDiffResource(uri string) (*ResourceReadResult, error) {
	repo, err := extractRepoFromResourceURI(uri)
	if err != nil {
		return nil, err
	}

	// Parse PR ID from URI
	// Format: bitbucket:///{owner}/{repo}/pr/{id}/diff
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URI: %w", err)
	}

	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	// parts: [owner, repo, "pr", id, "diff"]
	if len(parts) < 5 || parts[2] != "pr" {
		return nil, fmt.Errorf("invalid PR diff URI: expected /{owner}/{repo}/pr/{id}/diff format")
	}

	prID := parts[3]

	client, err := api.NewClient()
	if err != nil {
		return nil, err
	}

	data, err := client.Get(fmt.Sprintf("/repositories/%s/pullrequests/%s/diff", repo, prID))
	if err != nil {
		return nil, fmt.Errorf("failed to read PR diff: %w", err)
	}

	return &ResourceReadResult{
		Contents: []ResourceContents{
			{
				URI:      uri,
				MimeType: "text/plain",
				Text:     string(data),
			},
		},
	}, nil
}

// extractRepoFromResourceURI extracts the "owner/repo" from a bitbucket:/// URI.
func extractRepoFromResourceURI(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid URI: expected /{owner}/{repo}/... format")
	}
	return parts[0] + "/" + parts[1], nil
}
