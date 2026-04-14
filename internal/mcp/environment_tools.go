package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
)

// Environment represents a Bitbucket deployment environment.
type Environment struct {
	UUID            string `json:"uuid"`
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	EnvironmentType struct {
		Name string `json:"name"`
	} `json:"environment_type"`
	Rank     int `json:"rank"`
	Category struct {
		Name string `json:"name"`
	} `json:"category"`
	Lock *struct {
		Name string `json:"name"`
	} `json:"lock"`
	DeploymentGate *struct {
		Name string `json:"name"`
	} `json:"deployment_gate"`
}

// EnvironmentListHandler handles the environment_list tool invocation.
func EnvironmentListHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	path := fmt.Sprintf("/repositories/%s/environments?pagelen=25", repository)
	envs, err := api.GetPaginated[Environment](client, path)
	if err != nil {
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}

	data, err := json.Marshal(envs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal environments: %w", err)
	}

	return []Content{NewTextContent(string(data))}, nil
}

// EnvironmentViewHandler handles the environment_view tool invocation.
func EnvironmentViewHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	envUUID, ok := args["environment_uuid"].(string)
	if !ok || envUUID == "" {
		return nil, fmt.Errorf("environment_uuid parameter is required")
	}

	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	path := fmt.Sprintf("/repositories/%s/environments/%s", repository, envUUID)
	data, err := client.Get(path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch environment: %w", err)
	}

	var env Environment
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("failed to unmarshal environment: %w", err)
	}

	result, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal environment: %w", err)
	}

	return []Content{NewTextContent(string(result))}, nil
}

// EnvironmentCreateHandler handles the environment_create tool invocation.
func EnvironmentCreateHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name parameter is required")
	}

	envType, ok := args["environment_type"].(string)
	if !ok || envType == "" {
		return nil, fmt.Errorf("environment_type parameter is required")
	}

	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	body := map[string]interface{}{
		"name": name,
		"environment_type": map[string]string{
			"name": envType,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	path := fmt.Sprintf("/repositories/%s/environments", repository)
	data, err := client.Post(path, string(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create environment: %w", err)
	}

	var env Environment
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("failed to unmarshal environment: %w", err)
	}

	result, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal environment: %w", err)
	}

	return []Content{NewTextContent(string(result))}, nil
}

// EnvironmentDeleteHandler handles the environment_delete tool invocation.
func EnvironmentDeleteHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	envUUID, ok := args["environment_uuid"].(string)
	if !ok || envUUID == "" {
		return nil, fmt.Errorf("environment_uuid parameter is required")
	}

	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	path := fmt.Sprintf("/repositories/%s/environments/%s", repository, envUUID)
	_, err = client.Delete(path)
	if err != nil {
		return nil, fmt.Errorf("failed to delete environment: %w", err)
	}

	return []Content{NewTextContent(fmt.Sprintf("Environment %s deleted.", envUUID))}, nil
}
