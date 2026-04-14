package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
)

// User represents a Bitbucket user account.
type User struct {
	UUID        string `json:"uuid"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Nickname    string `json:"nickname"`
	AccountID   string `json:"account_id"`
	CreatedOn   string `json:"created_on"`
	Links       struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
		Avatar struct {
			Href string `json:"href"`
		} `json:"avatar"`
	} `json:"links"`
}

// Email represents a user's email address.
type Email struct {
	Email       string `json:"email"`
	IsPrimary   bool   `json:"is_primary"`
	IsConfirmed bool   `json:"is_confirmed"`
}

// SSHKey represents a user's SSH key.
type SSHKey struct {
	UUID      string `json:"uuid"`
	Key       string `json:"key"`
	Label     string `json:"label"`
	Comment   string `json:"comment"`
	CreatedOn string `json:"created_on"`
}

// UserMeHandler handles the user_me tool invocation.
func UserMeHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	data, err := client.Get("/user")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch current user: %w", err)
	}

	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	result, err := json.Marshal(user)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user: %w", err)
	}

	return []Content{NewTextContent(string(result))}, nil
}

// UserViewHandler handles the user_view tool invocation.
func UserViewHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	userID, ok := args["user_id"].(string)
	if !ok || userID == "" {
		return nil, fmt.Errorf("user_id parameter is required")
	}

	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	path := fmt.Sprintf("/users/%s", url.PathEscape(userID))
	data, err := client.Get(path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	result, err := json.Marshal(user)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user: %w", err)
	}

	return []Content{NewTextContent(string(result))}, nil
}

// UserEmailsHandler handles the user_emails tool invocation.
func UserEmailsHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	emails, err := api.GetPaginated[Email](client, "/user/emails")
	if err != nil {
		return nil, fmt.Errorf("failed to list emails: %w", err)
	}

	data, err := json.Marshal(emails)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal emails: %w", err)
	}

	return []Content{NewTextContent(string(data))}, nil
}

// UserSSHKeysHandler handles the user_ssh_keys tool invocation.
func UserSSHKeysHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	keys, err := api.GetPaginated[SSHKey](client, "/user/ssh-keys?pagelen=50")
	if err != nil {
		return nil, fmt.Errorf("failed to list SSH keys: %w", err)
	}

	data, err := json.Marshal(keys)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SSH keys: %w", err)
	}

	return []Content{NewTextContent(string(data))}, nil
}

// UserSSHKeyAddHandler handles the user_ssh_key_add tool invocation.
func UserSSHKeyAddHandler(ctx context.Context, args map[string]interface{}) ([]Content, error) {
	key, ok := args["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("key parameter is required")
	}

	label, _ := args["label"].(string)

	client, err := GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	body := map[string]interface{}{
		"key": key,
	}
	if label != "" {
		body["label"] = label
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	data, err := client.Post("/user/ssh-keys", string(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to add SSH key: %w", err)
	}

	return []Content{NewTextContent(string(data))}, nil
}
