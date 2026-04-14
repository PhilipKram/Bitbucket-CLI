package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
)

// Download represents a Bitbucket repository download.
type Download struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	CreatedOn string `json:"created_on"`
	Downloads int    `json:"downloads"`
	Links     struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"links"`
}

func parseRepoArg(arg string) (string, string, error) {
	parts := strings.SplitN(arg, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("repository must be in format workspace/repo-slug")
	}
	return parts[0], parts[1], nil
}

// DownloadListHandler handles the download_list tool invocation.
func DownloadListHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	ws, repo, err := parseRepoArg(repository)
	if err != nil {
		return nil, err
	}

	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	path := fmt.Sprintf("/repositories/%s/%s/downloads?pagelen=25",
		url.PathEscape(ws), url.PathEscape(repo))
	downloads, err := api.GetPaginated[Download](client, path)
	if err != nil {
		return nil, fmt.Errorf("failed to list downloads: %w", err)
	}

	data, err := json.Marshal(downloads)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal downloads: %w", err)
	}

	return []Content{NewTextContent(string(data))}, nil
}

// DownloadDeleteHandler handles the download_delete tool invocation.
func DownloadDeleteHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	filename, ok := args["filename"].(string)
	if !ok || filename == "" {
		return nil, fmt.Errorf("filename parameter is required")
	}

	ws, repo, err := parseRepoArg(repository)
	if err != nil {
		return nil, err
	}

	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	path := fmt.Sprintf("/repositories/%s/%s/downloads/%s",
		url.PathEscape(ws), url.PathEscape(repo), url.PathEscape(filename))
	_, err = client.Delete(path)
	if err != nil {
		return nil, fmt.Errorf("failed to delete download: %w", err)
	}

	return []Content{NewTextContent(fmt.Sprintf("Deleted '%s' from %s/%s downloads.", filename, ws, repo))}, nil
}
