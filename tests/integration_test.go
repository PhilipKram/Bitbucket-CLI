package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
	"github.com/PhilipKram/bitbucket-cli/internal/auth"
	"github.com/PhilipKram/bitbucket-cli/internal/config"
)

// TestIntegration_AuthFlow_TokenRefresh tests the full auth workflow including token refresh.
func TestIntegration_AuthFlow_TokenRefresh(t *testing.T) {
	var tokenCallCount int32

	// Mock OAuth token server
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&tokenCallCount, 1)

		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Error("expected basic auth")
		}
		if user != "test-client-id" || pass != "test-client-secret" {
			t.Errorf("unexpected credentials: user=%q pass=%q", user, pass)
		}

		r.ParseForm()
		grantType := r.FormValue("grant_type")

		if grantType == "refresh_token" {
			if r.FormValue("refresh_token") != "test-refresh-token" {
				t.Errorf("expected refresh_token=test-refresh-token, got %s", r.FormValue("refresh_token"))
			}
			w.WriteHeader(200)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token":  "refreshed-access-token",
				"refresh_token": "new-refresh-token",
				"token_type":    "bearer",
				"expires_in":    7200,
			})
		} else {
			w.WriteHeader(400)
			w.Write([]byte(`{"error":"unsupported_grant_type"}`))
		}
	}))
	defer tokenServer.Close()

	// Override TokenURL for testing
	orig := config.TokenURL
	config.TokenURL = tokenServer.URL
	defer func() { config.TokenURL = orig }()

	// Test token refresh
	newToken, err := auth.RefreshAccessToken("test-client-id", "test-client-secret", "test-refresh-token")
	if err != nil {
		t.Fatalf("RefreshAccessToken() error: %v", err)
	}

	if newToken.AccessToken != "refreshed-access-token" {
		t.Errorf("AccessToken = %q, want %q", newToken.AccessToken, "refreshed-access-token")
	}
	if newToken.RefreshToken != "new-refresh-token" {
		t.Errorf("RefreshToken = %q, want %q", newToken.RefreshToken, "new-refresh-token")
	}
	if atomic.LoadInt32(&tokenCallCount) != 1 {
		t.Errorf("expected 1 token call, got %d", tokenCallCount)
	}
}

// TestIntegration_APIClient_WithTokenRefresh tests the API client with automatic token refresh.
func TestIntegration_APIClient_WithTokenRefresh(t *testing.T) {
	// Use a temporary config directory so that any token refresh writes do not
	// pollute the real user configuration directory.
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)
	config.ResetConfigDirCache()

	var apiCallCount int32
	var tokenRefreshCount int32

	// Mock OAuth token refresh server
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&tokenRefreshCount, 1)

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "refreshed-token",
			"refresh_token": "new-refresh",
			"token_type":    "bearer",
			"expires_in":    7200,
		})
	}))
	defer tokenServer.Close()

	// Mock Bitbucket API server
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&apiCallCount, 1)

		auth := r.Header.Get("Authorization")
		if count == 1 {
			// First call with expired token - return 401
			if auth != "Bearer expired-token" {
				t.Errorf("expected Bearer expired-token on first call, got %s", auth)
			}
			w.WriteHeader(401)
			w.Write([]byte(`{"error":"token_expired"}`))
		} else {
			// Second call after refresh - should succeed
			if auth != "Bearer refreshed-token" {
				t.Errorf("expected Bearer refreshed-token on second call, got %s", auth)
			}
			w.WriteHeader(200)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]string{{"name": "repo1"}},
			})
		}
	}))
	defer apiServer.Close()

	orig := config.TokenURL
	config.TokenURL = tokenServer.URL
	defer func() { config.TokenURL = orig }()

	// Create client with expired token
	client := api.NewClientWith(apiServer.Client(), &config.Config{
		OAuthKey:    "test-key",
		OAuthSecret: "test-secret",
	}, &config.TokenData{
		AccessToken:  "expired-token",
		RefreshToken: "old-refresh",
	})

	// This should trigger a 401, refresh the token, and retry successfully.
	_, err := client.GetRaw(apiServer.URL + "/repos")
	if err != nil {
		t.Fatalf("GetRaw() error: %v", err)
	}

	if got := atomic.LoadInt32(&apiCallCount); got != 2 {
		t.Errorf("apiCallCount = %d, want 2", got)
	}
	if got := atomic.LoadInt32(&tokenRefreshCount); got != 1 {
		t.Errorf("tokenRefreshCount = %d, want 1", got)
	}
}

// TestIntegration_Config_SaveAndLoad tests config persistence.
func TestIntegration_Config_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)
	config.ResetConfigDirCache()

	// Save a config
	cfg := &config.Config{
		DefaultWorkspace: "test-workspace",
		DefaultFormat:    "json",
		OAuthKey:         "test-oauth-key",
		OAuthSecret:      "test-oauth-secret",
	}

	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}

	// Load the config back
	loaded, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}

	if loaded.DefaultWorkspace != cfg.DefaultWorkspace {
		t.Errorf("DefaultWorkspace = %q, want %q", loaded.DefaultWorkspace, cfg.DefaultWorkspace)
	}
	if loaded.DefaultFormat != cfg.DefaultFormat {
		t.Errorf("DefaultFormat = %q, want %q", loaded.DefaultFormat, cfg.DefaultFormat)
	}
	if loaded.OAuthKey != cfg.OAuthKey {
		t.Errorf("OAuthKey = %q, want %q", loaded.OAuthKey, cfg.OAuthKey)
	}
	if loaded.OAuthSecret != cfg.OAuthSecret {
		t.Errorf("OAuthSecret = %q, want %q", loaded.OAuthSecret, cfg.OAuthSecret)
	}
}

// TestIntegration_Token_SaveAndLoad tests token persistence.
func TestIntegration_Token_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)
	config.ResetConfigDirCache()

	// Save a token
	token := &config.TokenData{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "bearer",
		ExpiresIn:    7200,
		Scopes:       "repository pullrequest",
	}

	if err := config.SaveToken(token); err != nil {
		t.Fatalf("SaveToken() error: %v", err)
	}

	// Load the token back
	loaded, err := config.LoadToken()
	if err != nil {
		t.Fatalf("LoadToken() error: %v", err)
	}

	if loaded.AccessToken != token.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, token.AccessToken)
	}
	if loaded.RefreshToken != token.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", loaded.RefreshToken, token.RefreshToken)
	}
	if loaded.TokenType != token.TokenType {
		t.Errorf("TokenType = %q, want %q", loaded.TokenType, token.TokenType)
	}

	// Test token clearing
	if err := config.ClearToken(); err != nil {
		t.Fatalf("ClearToken() error: %v", err)
	}

	// Verify token is cleared
	_, err = config.LoadToken()
	if err == nil {
		t.Error("expected error when loading cleared token")
	}
}

// TestIntegration_APIClient_Pagination tests paginated API responses.
func TestIntegration_APIClient_Pagination(t *testing.T) {
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}

		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %s", auth)
		}

		// Return different pages based on query param
		queryPage := r.URL.Query().Get("page")
		var response map[string]interface{}

		if queryPage == "" || queryPage == "1" {
			response = map[string]interface{}{
				"size":    2,
				"page":    1,
				"pagelen": 2,
				"next":    serverURL + "/repos?page=2",
				"values": []map[string]interface{}{
					{"name": "repo1"},
					{"name": "repo2"},
				},
			}
		} else {
			response = map[string]interface{}{
				"size":    1,
				"page":    2,
				"pagelen": 2,
				"values": []map[string]interface{}{
					{"name": "repo3"},
				},
			}
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	serverURL = server.URL

	client := api.NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	// Fetch first page
	data, err := client.GetRaw(server.URL + "/repos")
	if err != nil {
		t.Fatalf("GetRaw() error: %v", err)
	}

	var pageResp api.PaginatedResponse
	if err := json.Unmarshal(data, &pageResp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if pageResp.Size != 2 {
		t.Errorf("Size = %d, want 2", pageResp.Size)
	}
	if pageResp.Page != 1 {
		t.Errorf("Page = %d, want 1", pageResp.Page)
	}
	if pageResp.Next == "" {
		t.Error("expected Next link to be present")
	}

	// Fetch second page
	data, err = client.GetRaw(server.URL + "/repos?page=2")
	if err != nil {
		t.Fatalf("GetRaw() page 2 error: %v", err)
	}

	if err := json.Unmarshal(data, &pageResp); err != nil {
		t.Fatalf("failed to parse page 2 response: %v", err)
	}

	if pageResp.Size != 1 {
		t.Errorf("Page 2 Size = %d, want 1", pageResp.Size)
	}
	if pageResp.Page != 2 {
		t.Errorf("Page 2 Page = %d, want 2", pageResp.Page)
	}
}

// TestIntegration_APIClient_ErrorHandling tests error handling across the stack.
func TestIntegration_APIClient_ErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedErrMsg string
	}{
		{
			name:           "404 Not Found",
			statusCode:     404,
			responseBody:   `{"error":"not_found"}`,
			expectedErrMsg: "Not found",
		},
		{
			name:           "500 Internal Server Error",
			statusCode:     500,
			responseBody:   `{"error":"internal_error"}`,
			expectedErrMsg: "internal_error",
		},
		{
			name:           "403 Forbidden",
			statusCode:     403,
			responseBody:   `{"error":"forbidden"}`,
			expectedErrMsg: "Permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := api.NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
				AccessToken: "test-token",
			})

			_, err := client.GetRaw(server.URL + "/test")
			if err == nil {
				t.Fatalf("expected error for %d response", tt.statusCode)
			}
			if !strings.Contains(err.Error(), tt.expectedErrMsg) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.expectedErrMsg)
			}
		})
	}
}

// TestIntegration_JSONMarshaling tests JSON encoding/decoding across API boundaries.
func TestIntegration_JSONMarshaling(t *testing.T) {
	// Test complex nested structures like PR payloads
	payload := map[string]interface{}{
		"title":       "Test PR",
		"description": "Test description",
		"source": map[string]interface{}{
			"branch": map[string]string{"name": "feature"},
		},
		"destination": map[string]interface{}{
			"branch": map[string]string{"name": "main"},
		},
		"reviewers": []map[string]string{
			{"uuid": "{reviewer-uuid-1}"},
		},
		"close_source_branch": true,
	}

	// Marshal to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	// Unmarshal back to verify round-trip
	var decoded map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &decoded); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	if decoded["title"] != "Test PR" {
		t.Errorf("title = %q, want %q", decoded["title"], "Test PR")
	}

	source, ok := decoded["source"].(map[string]interface{})
	if !ok {
		t.Fatal("source is not a map")
	}
	branch, ok := source["branch"].(map[string]interface{})
	if !ok {
		t.Fatal("source.branch is not a map")
	}
	if branch["name"] != "feature" {
		t.Errorf("source.branch.name = %q, want %q", branch["name"], "feature")
	}

	if decoded["close_source_branch"] != true {
		t.Errorf("close_source_branch = %v, want true", decoded["close_source_branch"])
	}
}

// TestIntegration_Config_FilePermissions tests that config files have secure permissions.
func TestIntegration_Config_FilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping permission test on Windows (no POSIX permission bits)")
	}

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)
	config.ResetConfigDirCache()

	// Save token (should have secure permissions)
	token := &config.TokenData{
		AccessToken:  "secret-token",
		RefreshToken: "secret-refresh",
	}

	if err := config.SaveToken(token); err != nil {
		t.Fatalf("SaveToken() error: %v", err)
	}

	// Check token file permissions
	dir, _ := config.ConfigDir()
	tokenPath := filepath.Join(dir, "token.json")
	info, err := os.Stat(tokenPath)
	if err != nil {
		t.Fatalf("failed to stat token file: %v", err)
	}

	mode := info.Mode().Perm()
	// Token file should be readable/writable only by owner (0600)
	if mode != 0600 {
		t.Errorf("token file mode = %o, want 0600", mode)
	}

	// Save config (should also have secure permissions)
	cfg := &config.Config{OAuthSecret: "secret"}
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}

	configPath := filepath.Join(dir, "config.json")
	info, err = os.Stat(configPath)
	if err != nil {
		t.Fatalf("failed to stat config file: %v", err)
	}

	mode = info.Mode().Perm()
	if mode != 0600 {
		t.Errorf("config file mode = %o, want 0600", mode)
	}
}

// TestIntegration_ConfigAndToken_Workflow tests the full config/token workflow.
func TestIntegration_ConfigAndToken_Workflow(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)
	config.ResetConfigDirCache()

	// Step 1: Save OAuth config
	cfg := &config.Config{
		DefaultWorkspace: "my-workspace",
		DefaultFormat:    "json",
		OAuthKey:         "oauth-key-123",
		OAuthSecret:      "oauth-secret-456",
	}
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}

	// Step 2: Save token
	token := &config.TokenData{
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
		TokenType:    "bearer",
		ExpiresIn:    7200,
	}
	if err := config.SaveToken(token); err != nil {
		t.Fatalf("SaveToken() error: %v", err)
	}

	// Step 3: Load config back
	loadedCfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if loadedCfg.OAuthKey != cfg.OAuthKey {
		t.Errorf("OAuthKey mismatch: got %q, want %q", loadedCfg.OAuthKey, cfg.OAuthKey)
	}

	// Step 4: Load token back
	loadedToken, err := config.LoadToken()
	if err != nil {
		t.Fatalf("LoadToken() error: %v", err)
	}
	if loadedToken.AccessToken != token.AccessToken {
		t.Errorf("AccessToken mismatch: got %q, want %q", loadedToken.AccessToken, token.AccessToken)
	}

	// Step 5: Create an API client with loaded config and token
	client := api.NewClientWith(&http.Client{}, loadedCfg, loadedToken)
	if client == nil {
		t.Fatal("NewClientWith() returned nil")
	}

	clientCfg := client.GetConfig()
	if clientCfg.DefaultWorkspace != "my-workspace" {
		t.Errorf("client workspace = %q, want %q", clientCfg.DefaultWorkspace, "my-workspace")
	}

	// Step 6: Logout (clear token)
	if err := config.ClearToken(); err != nil {
		t.Fatalf("ClearToken() error: %v", err)
	}

	// Step 7: Verify token is cleared
	_, err = config.LoadToken()
	if err == nil {
		t.Error("expected error after clearing token")
	}
}
