package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/PhilipKram/bitbucket-cli/internal/auth"
	"github.com/PhilipKram/bitbucket-cli/internal/config"
)

// Client wraps HTTP calls to the Bitbucket 2.0 API with automatic token refresh.
type Client struct {
	httpClient *http.Client
	token      *config.TokenData
	cfg        *config.Config
}

// PaginatedResponse is the standard paginated response envelope from Bitbucket.
type PaginatedResponse struct {
	Size     int              `json:"size"`
	Page     int              `json:"page"`
	PageLen  int              `json:"pagelen"`
	Next     string           `json:"next"`
	Previous string           `json:"previous"`
	Values   json.RawMessage  `json:"values"`
}

// NewClient creates an authenticated API client.
func NewClient() (*Client, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	token, err := config.LoadToken()
	if err != nil {
		return nil, fmt.Errorf("not authenticated. Run 'bb auth login' first")
	}

	return &Client{
		httpClient: &http.Client{},
		token:      token,
		cfg:        cfg,
	}, nil
}

// GetConfig returns the loaded configuration.
func (c *Client) GetConfig() *config.Config {
	return c.cfg
}

func (c *Client) doRequest(method, urlStr string, body io.Reader, contentType string) (*http.Response, error) {
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token.AccessToken)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Attempt token refresh on 401
	if resp.StatusCode == http.StatusUnauthorized && c.token.RefreshToken != "" {
		resp.Body.Close()
		if err := c.refreshToken(); err != nil {
			return nil, fmt.Errorf("session expired, please run 'bb auth login' again: %w", err)
		}
		// Retry the request with the new token
		req2, err := http.NewRequest(method, urlStr, body)
		if err != nil {
			return nil, err
		}
		req2.Header.Set("Authorization", "Bearer "+c.token.AccessToken)
		if contentType != "" {
			req2.Header.Set("Content-Type", contentType)
		}
		return c.httpClient.Do(req2)
	}

	return resp, nil
}

func (c *Client) refreshToken() error {
	cfg := c.cfg
	if cfg.OAuthKey == "" || cfg.OAuthSecret == "" {
		return fmt.Errorf("OAuth credentials not configured")
	}
	newToken, err := auth.RefreshAccessToken(cfg.OAuthKey, cfg.OAuthSecret, c.token.RefreshToken)
	if err != nil {
		return err
	}
	c.token = newToken
	return config.SaveToken(newToken)
}

// Get performs a GET request to the Bitbucket API.
func (c *Client) Get(path string) ([]byte, error) {
	u := config.BitbucketAPI + path
	resp, err := c.doRequest("GET", u, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return handleResponse(resp)
}

// GetRaw performs a GET to an absolute URL (for pagination "next" links).
func (c *Client) GetRaw(rawURL string) ([]byte, error) {
	resp, err := c.doRequest("GET", rawURL, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return handleResponse(resp)
}

// Post performs a POST with JSON body.
func (c *Client) Post(path string, jsonBody string) ([]byte, error) {
	u := config.BitbucketAPI + path
	resp, err := c.doRequest("POST", u, strings.NewReader(jsonBody), "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return handleResponse(resp)
}

// PostForm performs a POST with form-encoded body.
func (c *Client) PostForm(path string, data url.Values) ([]byte, error) {
	u := config.BitbucketAPI + path
	resp, err := c.doRequest("POST", u, strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return handleResponse(resp)
}

// Put performs a PUT with JSON body.
func (c *Client) Put(path string, jsonBody string) ([]byte, error) {
	u := config.BitbucketAPI + path
	resp, err := c.doRequest("PUT", u, strings.NewReader(jsonBody), "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return handleResponse(resp)
}

// Delete performs a DELETE request.
func (c *Client) Delete(path string) ([]byte, error) {
	u := config.BitbucketAPI + path
	resp, err := c.doRequest("DELETE", u, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// DELETE often returns 204 No Content
	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	return handleResponse(resp)
}

func handleResponse(resp *http.Response) ([]byte, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return body, nil
}
