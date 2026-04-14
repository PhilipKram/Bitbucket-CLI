package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
)

// Variable represents a Bitbucket pipeline variable.
type Variable struct {
	UUID    string `json:"uuid"`
	Key     string `json:"key"`
	Value   string `json:"value"`
	Secured bool   `json:"secured"`
}

func listVariables(client *api.Client, repository string) ([]Variable, error) {
	path := fmt.Sprintf("/repositories/%s/pipelines_config/variables?pagelen=100", repository)
	return api.GetPaginated[Variable](client, path)
}

func findVariableByKey(variables []Variable, key string) (*Variable, error) {
	for i := range variables {
		if variables[i].Key == key {
			return &variables[i], nil
		}
	}
	return nil, fmt.Errorf("variable %q not found", key)
}

// VariableListHandler handles the variable_list tool invocation.
func VariableListHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
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

	variables, err := listVariables(client, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to list variables: %w", err)
	}

	data, err := json.Marshal(variables)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal variables: %w", err)
	}

	return []Content{NewTextContent(string(data))}, nil
}

// VariableGetHandler handles the variable_get tool invocation.
func VariableGetHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	key, ok := args["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("key parameter is required")
	}

	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	variables, err := listVariables(client, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to list variables: %w", err)
	}

	v, err := findVariableByKey(variables, key)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal variable: %w", err)
	}

	return []Content{NewTextContent(string(data))}, nil
}

// VariableSetHandler handles the variable_set tool invocation.
func VariableSetHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	key, ok := args["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("key parameter is required")
	}

	value, ok := args["value"].(string)
	if !ok {
		return nil, fmt.Errorf("value parameter is required")
	}

	secured := false
	if s, ok := args["secured"].(bool); ok {
		secured = s
	}

	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	body := map[string]interface{}{
		"key":     key,
		"value":   value,
		"secured": secured,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	path := fmt.Sprintf("/repositories/%s/pipelines_config/variables", repository)
	data, err := client.Post(path, string(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create variable: %w", err)
	}

	var v Variable
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("failed to unmarshal variable: %w", err)
	}

	result, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal variable: %w", err)
	}

	return []Content{NewTextContent(string(result))}, nil
}

// VariableUpdateHandler handles the variable_update tool invocation.
func VariableUpdateHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	key, ok := args["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("key parameter is required")
	}

	value, ok := args["value"].(string)
	if !ok {
		return nil, fmt.Errorf("value parameter is required")
	}

	secured := false
	if s, ok := args["secured"].(bool); ok {
		secured = s
	}

	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Find existing variable UUID by key
	variables, err := listVariables(client, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to list variables: %w", err)
	}

	existing, err := findVariableByKey(variables, key)
	if err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"key":     key,
		"value":   value,
		"secured": secured,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	path := fmt.Sprintf("/repositories/%s/pipelines_config/variables/%s",
		repository, url.PathEscape(existing.UUID))
	data, err := client.Put(path, string(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to update variable: %w", err)
	}

	var v Variable
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("failed to unmarshal variable: %w", err)
	}

	result, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal variable: %w", err)
	}

	return []Content{NewTextContent(string(result))}, nil
}

// VariableDeleteHandler handles the variable_delete tool invocation.
func VariableDeleteHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required")
	}
	if err := validateRepoArg(repository); err != nil {
		return nil, err
	}

	key, ok := args["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("key parameter is required")
	}

	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Find existing variable UUID by key
	variables, err := listVariables(client, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to list variables: %w", err)
	}

	existing, err := findVariableByKey(variables, key)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/repositories/%s/pipelines_config/variables/%s",
		repository, url.PathEscape(existing.UUID))
	_, err = client.Delete(path)
	if err != nil {
		return nil, fmt.Errorf("failed to delete variable: %w", err)
	}

	return []Content{NewTextContent(fmt.Sprintf("Variable %q deleted.", key))}, nil
}
