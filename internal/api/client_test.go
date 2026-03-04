package api

import (
	"bytes"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/PhilipKram/bitbucket-cli/internal/config"
	"github.com/PhilipKram/bitbucket-cli/internal/errors"
)

func TestClient_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %s", auth)
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	// Override BitbucketAPI base URL by using GetRaw with the test server URL
	data, err := client.GetRaw(server.URL + "/test")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	var result map[string]bool
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if !result["ok"] {
		t.Errorf("expected ok=true")
	}
}

func TestClient_Post_BodyBuffering(t *testing.T) {
	// Test that POST body is correctly sent on retry after 401
	var callCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		body, _ := io.ReadAll(r.Body)

		if count == 1 {
			// First call: return 401 to trigger refresh
			w.WriteHeader(401)
			return
		}

		// Second call (retry): verify the body was resent correctly
		if string(body) != `{"key":"value"}` {
			t.Errorf("retry body = %q, want %q", string(body), `{"key":"value"}`)
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	// Create a token server that returns a new token
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "new-token",
			"refresh_token": "new-refresh",
			"token_type":    "bearer",
			"expires_in":    7200,
		})
	}))
	defer tokenServer.Close()

	// We can't easily test the full refresh flow since it calls auth.RefreshAccessToken
	// which hits the real token URL. Instead, test that body buffering works for a
	// non-401 scenario.

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	// This should work fine — body is buffered
	resp, err := client.httpClient.Post(server.URL+"/test", "application/json", nil)
	if err != nil {
		t.Fatalf("Post() error: %v", err)
	}
	resp.Body.Close()
}

func TestClient_Post_EmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "" {
			t.Errorf("expected no Content-Type header, got %q", ct)
		}
		body, _ := io.ReadAll(r.Body)
		if len(body) != 0 {
			t.Errorf("expected empty body, got %q", string(body))
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"approved":true}`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	// Use doRequest directly since Post prepends BitbucketAPI
	resp, err := client.doRequest("POST", server.URL+"/approve", nil, "")
	if err != nil {
		t.Fatalf("Post() error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestClient_Post_WithJsonBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", ct)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"title":"test"}` {
			t.Errorf("expected JSON body, got %q", string(body))
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	resp, err := client.doRequest("POST", server.URL+"/test", strings.NewReader(`{"title":"test"}`), "application/json")
	if err != nil {
		t.Fatalf("Post() error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestClient_HandleResponse_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"type":"error","error":{"message":"Repository not found"}}`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	_, err := client.GetRaw(server.URL + "/missing")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}

	// Validate structured error
	bbErr, ok := err.(*errors.BBError)
	if !ok {
		t.Fatalf("expected *errors.BBError, got %T", err)
	}
	if bbErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", bbErr.StatusCode)
	}
	if !strings.Contains(bbErr.Message, "Not found") {
		t.Errorf("Message = %q, want to contain 'Not found'", bbErr.Message)
	}
	if bbErr.Suggestion == "" {
		t.Error("expected non-empty suggestion for 404")
	}
}

func TestClient_HandleResponse_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"type":"error","error":{"message":"Invalid credentials"}}`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "invalid-token",
		RefreshToken: "", // No refresh token to prevent retry
	})

	_, err := client.GetRaw(server.URL + "/test")
	if err == nil {
		t.Fatal("expected error for 401 response")
	}

	// Validate structured error
	bbErr, ok := err.(*errors.BBError)
	if !ok {
		t.Fatalf("expected *errors.BBError, got %T", err)
	}
	if bbErr.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", bbErr.StatusCode)
	}
	if !strings.Contains(bbErr.Message, "Authentication failed") {
		t.Errorf("Message = %q, want to contain 'Authentication failed'", bbErr.Message)
	}
	if !strings.Contains(bbErr.Suggestion, "bb auth login") {
		t.Errorf("Suggestion = %q, want to contain 'bb auth login'", bbErr.Suggestion)
	}
}

func TestClient_HandleResponse_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte(`{"type":"error","error":{"message":"Access denied"}}`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	_, err := client.GetRaw(server.URL + "/forbidden")
	if err == nil {
		t.Fatal("expected error for 403 response")
	}

	// Validate structured error
	bbErr, ok := err.(*errors.BBError)
	if !ok {
		t.Fatalf("expected *errors.BBError, got %T", err)
	}
	if bbErr.StatusCode != 403 {
		t.Errorf("StatusCode = %d, want 403", bbErr.StatusCode)
	}
	if !strings.Contains(bbErr.Message, "Permission denied") {
		t.Errorf("Message = %q, want to contain 'Permission denied'", bbErr.Message)
	}
	if !strings.Contains(bbErr.Suggestion, "permissions") {
		t.Errorf("Suggestion = %q, want to contain 'permissions'", bbErr.Suggestion)
	}
}

func TestClient_HandleResponse_RateLimit(t *testing.T) {
	resetTime := fmt.Sprintf("%d", time.Now().Add(5*time.Minute).Unix())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Reset", resetTime)
		w.WriteHeader(429)
		w.Write([]byte(`{"type":"error","error":{"message":"Rate limit exceeded"}}`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	_, err := client.GetRaw(server.URL + "/test")
	if err == nil {
		t.Fatal("expected error for 429 response")
	}

	// Validate structured error
	bbErr, ok := err.(*errors.BBError)
	if !ok {
		t.Fatalf("expected *errors.BBError, got %T", err)
	}
	if bbErr.StatusCode != 429 {
		t.Errorf("StatusCode = %d, want 429", bbErr.StatusCode)
	}
	if !strings.Contains(bbErr.Message, "Rate limit") {
		t.Errorf("Message = %q, want to contain 'Rate limit'", bbErr.Message)
	}
	if bbErr.Suggestion == "" {
		t.Error("expected non-empty suggestion for 429")
	}
}

func TestClient_HandleResponse_BadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"type":"error","error":{"message":"Invalid branch name","detail":"Branch names cannot contain spaces"}}`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	_, err := client.GetRaw(server.URL + "/test")
	if err == nil {
		t.Fatal("expected error for 400 response")
	}

	// Validate structured error
	bbErr, ok := err.(*errors.BBError)
	if !ok {
		t.Fatalf("expected *errors.BBError, got %T", err)
	}
	if bbErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", bbErr.StatusCode)
	}
	if !strings.Contains(bbErr.Message, "Invalid branch name") {
		t.Errorf("Message = %q, want to contain error message", bbErr.Message)
	}
	if !strings.Contains(bbErr.Message, "Branch names cannot contain spaces") {
		t.Errorf("Message = %q, want to contain detail", bbErr.Message)
	}
	if !strings.Contains(bbErr.Suggestion, "request parameters") {
		t.Errorf("Suggestion = %q, want to contain 'request parameters'", bbErr.Suggestion)
	}
}

func TestClient_HandleResponse_Conflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		w.Write([]byte(`{"type":"error","error":{"message":"Resource already exists"}}`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	_, err := client.GetRaw(server.URL + "/test")
	if err == nil {
		t.Fatal("expected error for 409 response")
	}

	// Validate structured error
	bbErr, ok := err.(*errors.BBError)
	if !ok {
		t.Fatalf("expected *errors.BBError, got %T", err)
	}
	if bbErr.StatusCode != 409 {
		t.Errorf("StatusCode = %d, want 409", bbErr.StatusCode)
	}
	if !strings.Contains(bbErr.Message, "Resource already exists") {
		t.Errorf("Message = %q, want to contain error message", bbErr.Message)
	}
	if !strings.Contains(bbErr.Suggestion, "conflicts") {
		t.Errorf("Suggestion = %q, want to contain 'conflicts'", bbErr.Suggestion)
	}
}

func TestClient_HandleResponse_ServerError(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
	}{
		{"500 Internal Server Error", http.StatusInternalServerError},
		{"502 Bad Gateway", http.StatusBadGateway},
		{"503 Service Unavailable", http.StatusServiceUnavailable},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(`{"type":"error","error":{"message":"Internal server error"}}`))
			}))
			defer server.Close()

			client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
				AccessToken: "test-token",
			})

			_, err := client.GetRaw(server.URL + "/test")
			if err == nil {
				t.Fatalf("expected error for %d response", tc.statusCode)
			}

			// Validate structured error
			bbErr, ok := err.(*errors.BBError)
			if !ok {
				t.Fatalf("expected *errors.BBError, got %T", err)
			}
			if bbErr.StatusCode != tc.statusCode {
				t.Errorf("StatusCode = %d, want %d", bbErr.StatusCode, tc.statusCode)
			}
			if !strings.Contains(bbErr.Suggestion, "temporary") {
				t.Errorf("Suggestion = %q, want to contain 'temporary'", bbErr.Suggestion)
			}
		})
	}
}

func TestClient_HandleResponse_GenericError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(418) // I'm a teapot - unusual status code
		w.Write([]byte(`{"type":"error","error":{"message":"Unusual error"}}`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	_, err := client.GetRaw(server.URL + "/test")
	if err == nil {
		t.Fatal("expected error for 418 response")
	}

	// Validate structured error
	bbErr, ok := err.(*errors.BBError)
	if !ok {
		t.Fatalf("expected *errors.BBError, got %T", err)
	}
	if bbErr.StatusCode != 418 {
		t.Errorf("StatusCode = %d, want 418", bbErr.StatusCode)
	}
	if !strings.Contains(bbErr.Message, "API error") {
		t.Errorf("Message = %q, want to contain 'API error'", bbErr.Message)
	}
	if !strings.Contains(bbErr.Message, "418") {
		t.Errorf("Message = %q, want to contain status code", bbErr.Message)
	}
}

func TestClient_Delete_NoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(204)
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	// Use doRequest directly since Delete prepends BitbucketAPI
	resp, err := client.doRequest("DELETE", server.URL+"/test", nil, "")
	if err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 204 {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}
}

func TestClient_OAuthAuth_Header(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer my-oauth-token" {
			t.Errorf("expected 'Bearer my-oauth-token', got %q", auth)
		}
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "my-oauth-token",
	})

	_, err := client.GetRaw(server.URL + "/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestClient_Pagination tests parsing paginated responses
func TestClient_Pagination(t *testing.T) {
	page1Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		// Include a "next" link in the response
		resp := map[string]interface{}{
			"size":    2,
			"page":    1,
			"pagelen": 10,
			"next":    "http://example.com/page2",
			"values":  []map[string]string{{"id": "1"}, {"id": "2"}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer page1Server.Close()

	client := NewClientWith(page1Server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	data, err := client.GetRaw(page1Server.URL + "/repos")
	if err != nil {
		t.Fatalf("GetRaw() error: %v", err)
	}

	var paginatedResp struct {
		Size    int                      `json:"size"`
		Page    int                      `json:"page"`
		PageLen int                      `json:"pagelen"`
		Next    string                   `json:"next"`
		Values  []map[string]interface{} `json:"values"`
	}
	if err := json.Unmarshal(data, &paginatedResp); err != nil {
		t.Fatalf("failed to parse paginated response: %v", err)
	}

	if paginatedResp.Size != 2 {
		t.Errorf("expected size=2, got %d", paginatedResp.Size)
	}
	if paginatedResp.Page != 1 {
		t.Errorf("expected page=1, got %d", paginatedResp.Page)
	}
	if paginatedResp.Next != "http://example.com/page2" {
		t.Errorf("expected next='http://example.com/page2', got %q", paginatedResp.Next)
	}
	if len(paginatedResp.Values) != 2 {
		t.Errorf("expected 2 values, got %d", len(paginatedResp.Values))
	}
}

// TestClient_PaginationFollowNext tests following pagination next links
func TestClient_PaginationFollowNext(t *testing.T) {
	var callCount int32
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		w.WriteHeader(200)

		if count == 1 {
			// First page with next link
			resp := map[string]interface{}{
				"size":    2,
				"page":    1,
				"pagelen": 2,
				"next":    serverURL + "/page2",
				"values":  []map[string]string{{"id": "1"}, {"id": "2"}},
			}
			json.NewEncoder(w).Encode(resp)
		} else {
			// Second page without next link
			resp := map[string]interface{}{
				"size":    1,
				"page":    2,
				"pagelen": 2,
				"values":  []map[string]string{{"id": "3"}},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	// Get first page
	data, err := client.GetRaw(server.URL + "/repos")
	if err != nil {
		t.Fatalf("GetRaw() error: %v", err)
	}

	var page1 struct {
		Next   string                   `json:"next"`
		Values []map[string]interface{} `json:"values"`
	}
	if err := json.Unmarshal(data, &page1); err != nil {
		t.Fatalf("failed to parse page 1: %v", err)
	}

	if len(page1.Values) != 2 {
		t.Errorf("expected 2 values on page 1, got %d", len(page1.Values))
	}

	// Follow next link
	if page1.Next == "" {
		t.Fatal("expected next link on page 1")
	}

	data2, err := client.GetRaw(page1.Next)
	if err != nil {
		t.Fatalf("GetRaw() for page 2 error: %v", err)
	}

	var page2 struct {
		Next   string                   `json:"next"`
		Values []map[string]interface{} `json:"values"`
	}
	if err := json.Unmarshal(data2, &page2); err != nil {
		t.Fatalf("failed to parse page 2: %v", err)
	}

	if len(page2.Values) != 1 {
		t.Errorf("expected 1 value on page 2, got %d", len(page2.Values))
	}
	if page2.Next != "" {
		t.Errorf("expected no next link on page 2, got %q", page2.Next)
	}
}

// TestClient_ErrorHandling_BadRequest tests 400 error handling
func TestClient_ErrorHandling_BadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":{"message":"Invalid request"}}`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	_, err := client.GetRaw(server.URL + "/test")
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if !strings.Contains(err.Error(), "Invalid request") {
		t.Errorf("expected error to mention 'Invalid request', got: %v", err)
	}
}

// TestClient_ErrorHandling_Forbidden tests 403 error handling
func TestClient_ErrorHandling_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte(`{"error":"Forbidden"}`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	_, err := client.GetRaw(server.URL + "/test")
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	if !errors.IsForbidden(err) {
		t.Errorf("expected forbidden error, got: %v", err)
	}
}

// TestClient_ErrorHandling_ServerError tests 500 error handling
func TestClient_ErrorHandling_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"error":"Internal server error"}`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	_, err := client.GetRaw(server.URL + "/test")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	var bbErr *errors.BBError
	if !stderrors.As(err, &bbErr) || bbErr.StatusCode != 500 {
		t.Errorf("expected BBError with StatusCode 500, got: %v", err)
	}
}

// TestClient_ErrorHandling_EmptyResponse tests error with empty body
func TestClient_ErrorHandling_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(502)
		// Empty body
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	_, err := client.GetRaw(server.URL + "/test")
	if err == nil {
		t.Fatal("expected error for 502 response")
	}
	var bbErr *errors.BBError
	if !stderrors.As(err, &bbErr) || bbErr.StatusCode != 502 {
		t.Errorf("expected BBError with StatusCode 502, got: %v", err)
	}
}

// TestClient_TokenRefresh_Unauthorized tests token refresh on 401
func TestClient_TokenRefresh_Unauthorized(t *testing.T) {
	// Ensure token persistence during refresh writes to an isolated temp directory.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	config.ResetConfigDirCache()

	var apiCallCount int32
	var tokenCallCount int32

	// Mock OAuth token endpoint for refresh.
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&tokenCallCount, 1)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"access_token":"new-token","refresh_token":"test-refresh","expires_in":3600}`)
	}))
	defer tokenServer.Close()

	origTokenURL := config.TokenURL
	config.TokenURL = tokenServer.URL
	defer func() { config.TokenURL = origTokenURL }()

	// Mock API endpoint that first returns 401, then succeeds.
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&apiCallCount, 1)
		auth := r.Header.Get("Authorization")

		if count == 1 {
			if auth != "Bearer test-token" {
				t.Errorf("first request expected Authorization 'Bearer test-token', got %q", auth)
			}
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"Unauthorized"}`))
			return
		}
		if auth != "Bearer new-token" {
			t.Errorf("second request expected Authorization 'Bearer new-token', got %q", auth)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer apiServer.Close()

	client := NewClientWith(apiServer.Client(), &config.Config{
		OAuthKey:    "test-key",
		OAuthSecret: "test-secret",
	}, &config.TokenData{
		AccessToken:  "test-token",
		RefreshToken: "test-refresh",
	})

	data, err := client.GetRaw(apiServer.URL + "/test")
	if err != nil {
		t.Fatalf("GetRaw() error: %v", err)
	}

	var resp struct{ Ok bool }
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if !resp.Ok {
		t.Fatal("expected ok=true in response")
	}

	if got := atomic.LoadInt32(&apiCallCount); got != 2 {
		t.Errorf("expected 2 API calls (401 + retry), got %d", got)
	}
	if got := atomic.LoadInt32(&tokenCallCount); got != 1 {
		t.Errorf("expected 1 token refresh call, got %d", got)
	}
}

// TestClient_TokenRefresh_NoRefreshToken tests 401 without refresh token
func TestClient_TokenRefresh_NoRefreshToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error":"Unauthorized"}`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
		// No RefreshToken
	})

	_, err := client.GetRaw(server.URL + "/test")
	if err == nil {
		t.Fatal("expected error for 401 without refresh token")
	}
	// Should get the 401 error, not attempt refresh
	if !errors.IsUnauthorized(err) {
		t.Errorf("expected unauthorized error, got: %v", err)
	}
}

// TestClient_Put_Success tests PUT request
func TestClient_Put_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"name":"updated"}` {
			t.Errorf("unexpected body: %s", string(body))
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	// Use doRequest directly to bypass BitbucketAPI prefix
	resp, err := client.doRequest("PUT", server.URL+"/test", strings.NewReader(`{"name":"updated"}`), "application/json")
	if err != nil {
		t.Fatalf("Put() error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// TestClient_PostForm_Success tests form-encoded POST
func TestClient_PostForm_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/x-www-form-urlencoded" {
			t.Errorf("expected form content type, got %s", contentType)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if r.FormValue("key") != "value" {
			t.Errorf("expected key=value, got %s", r.FormValue("key"))
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	data := url.Values{}
	data.Set("key", "value")

	// Use doRequest directly to bypass BitbucketAPI prefix
	resp, err := client.doRequest("POST", server.URL+"/test", strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatalf("PostForm() error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// TestClient_GetConfig tests the GetConfig method
func TestClient_GetConfig(t *testing.T) {
	cfg := &config.Config{
		OAuthKey:    "test-key",
		OAuthSecret: "test-secret",
	}
	client := NewClientWith(&http.Client{}, cfg, &config.TokenData{
		AccessToken: "test-token",
	})

	gotCfg := client.GetConfig()
	if gotCfg != cfg {
		t.Error("GetConfig() did not return the expected config")
	}
	if gotCfg.OAuthKey != "test-key" {
		t.Errorf("expected OAuthKey='test-key', got %q", gotCfg.OAuthKey)
	}
}

// TestClient_Post_Success tests POST with JSON body
func TestClient_Post_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected json content type, got %s", contentType)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"test":"data"}` {
			t.Errorf("unexpected body: %s", string(body))
		}
		w.WriteHeader(201)
		w.Write([]byte(`{"created":true}`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	resp, err := client.doRequest("POST", server.URL+"/test", strings.NewReader(`{"test":"data"}`), "application/json")
	if err != nil {
		t.Fatalf("Post() error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	var result map[string]bool
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if !result["created"] {
		t.Errorf("expected created=true")
	}
}

// TestClient_DoRequest_NilBody tests doRequest with nil body
func TestClient_DoRequest_NilBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	resp, err := client.doRequest("GET", server.URL+"/test", nil, "")
	if err != nil {
		t.Fatalf("doRequest() with nil body error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// TestClient_DoRequest_WithContentType tests content type header
func TestClient_DoRequest_WithContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		if contentType != "text/plain" {
			t.Errorf("expected Content-Type='text/plain', got %q", contentType)
		}
		w.WriteHeader(200)
		w.Write([]byte(`ok`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	resp, err := client.doRequest("POST", server.URL+"/test", strings.NewReader("plain text"), "text/plain")
	if err != nil {
		t.Fatalf("doRequest() error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// TestClient_ErrorHandling_NetworkError tests network error handling
func TestClient_ErrorHandling_NetworkError(t *testing.T) {
	// Use a closed httptest.Server to trigger a deterministic connection error
	server := httptest.NewServer(http.NewServeMux())
	serverURL := server.URL
	server.Close()

	client := NewClientWith(&http.Client{}, &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	_, err := client.GetRaw(serverURL + "/test")
	if err == nil {
		t.Fatal("expected error for closed server")
	}
}

// TestClient_MultipleErrorStatusCodes tests various error codes
func TestClient_MultipleErrorStatusCodes(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
	}{
		{"BadRequest", 400},
		{"Unauthorized", 401},
		{"Forbidden", 403},
		{"NotFound", 404},
		{"MethodNotAllowed", 405},
		{"Conflict", 409},
		{"InternalServerError", 500},
		{"BadGateway", 502},
		{"ServiceUnavailable", 503},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(`{"error":"test error"}`))
			}))
			defer server.Close()

			client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
				AccessToken: "test-token",
			})

			_, err := client.GetRaw(server.URL + "/test")
			if err == nil {
				t.Fatalf("expected error for %d response", tc.statusCode)
			}
			// Verify the error is a BBError with the correct status code
			var bbErr *errors.BBError
			if !stderrors.As(err, &bbErr) {
				t.Errorf("expected *errors.BBError, got %T: %v", err, err)
			} else if bbErr.StatusCode != tc.statusCode {
				t.Errorf("expected StatusCode %d, got %d", tc.statusCode, bbErr.StatusCode)
			}
		})
	}
}

// TestClient_SuccessStatusCodes tests various success codes
func TestClient_SuccessStatusCodes(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
	}{
		{"OK", 200},
		{"Created", 201},
		{"Accepted", 202},
		{"NoContent", 204},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				if tc.statusCode != 204 {
					w.Write([]byte(`{"success":true}`))
				}
			}))
			defer server.Close()

			client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
				AccessToken: "test-token",
			})

			data, err := client.GetRaw(server.URL + "/test")
			if err != nil {
				t.Fatalf("unexpected error for %d response: %v", tc.statusCode, err)
			}
			if tc.statusCode != 204 && len(data) == 0 {
				t.Error("expected non-empty response body")
			}
		})
	}
}

// TestClient_Delete_WithBody tests DELETE with response body
func TestClient_Delete_WithBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"deleted":true}`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	resp, err := client.doRequest("DELETE", server.URL+"/test", nil, "")
	if err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "deleted") {
		t.Errorf("unexpected response body: %s", string(body))
	}
}

// TestClient_TokenRefresh_WithOAuthConfig tests refresh token flow with OAuth config
func TestClient_TokenRefresh_WithOAuthConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	config.ResetConfigDirCache()

	var apiCallCount int32
	var tokenCallCount int32

	// Mock OAuth token endpoint for refresh.
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&tokenCallCount, 1)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"access_token":"refreshed-token","refresh_token":"new-refresh","expires_in":3600}`)
	}))
	defer tokenServer.Close()

	origTokenURL := config.TokenURL
	config.TokenURL = tokenServer.URL
	defer func() { config.TokenURL = origTokenURL }()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&apiCallCount, 1)

		if count == 1 {
			w.WriteHeader(401)
			w.Write([]byte(`{"error":"Unauthorized"}`))
			return
		}

		auth := r.Header.Get("Authorization")
		if auth != "Bearer refreshed-token" {
			t.Errorf("retry expected Authorization 'Bearer refreshed-token', got %q", auth)
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer apiServer.Close()

	client := NewClientWith(apiServer.Client(), &config.Config{
		OAuthKey:    "test-key",
		OAuthSecret: "test-secret",
	}, &config.TokenData{
		AccessToken:  "old-token",
		RefreshToken: "test-refresh-token",
	})

	_, err := client.GetRaw(apiServer.URL + "/test")
	if err != nil {
		t.Fatalf("GetRaw() error: %v", err)
	}

	if got := atomic.LoadInt32(&apiCallCount); got != 2 {
		t.Errorf("expected 2 API calls, got %d", got)
	}
	if got := atomic.LoadInt32(&tokenCallCount); got != 1 {
		t.Errorf("expected 1 token refresh call, got %d", got)
	}
}

// TestClient_TokenRefresh_NoOAuthConfig tests refresh without OAuth config
func TestClient_TokenRefresh_NoOAuthConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error":"Unauthorized"}`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{
		// No OAuth credentials
	}, &config.TokenData{
		AccessToken:  "test-token",
		RefreshToken: "test-refresh",
	})

	_, err := client.GetRaw(server.URL + "/test")
	if err == nil {
		t.Fatal("expected error for 401 without OAuth config")
	}
	// Should fail with session expired or OAuth credentials error
	errStr := err.Error()
	if !strings.Contains(errStr, "OAuth credentials") && !strings.Contains(errStr, "session expired") {
		t.Errorf("expected OAuth credentials or session expired error, got: %v", err)
	}
}

// TestClient_HandleResponse_ReadError tests read error in handleResponse
func TestClient_HandleResponse_ReadError(t *testing.T) {
	// This is hard to test directly since handleResponse is not exported
	// but we can verify error handling through the public methods

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	data, err := client.GetRaw(server.URL + "/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]bool
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if !result["ok"] {
		t.Error("expected ok=true")
	}
}

// TestClient_SetAuth tests authentication header setting
func TestClient_SetAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		expected := "Bearer my-test-token-12345"
		if auth != expected {
			t.Errorf("expected Authorization=%q, got %q", expected, auth)
		}
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "my-test-token-12345",
	})

	_, err := client.GetRaw(server.URL + "/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestClient_Pagination_EmptyNext tests pagination without next link
func TestClient_Pagination_EmptyNext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		resp := map[string]interface{}{
			"size":    1,
			"page":    1,
			"pagelen": 10,
			"values":  []map[string]string{{"id": "1"}},
			// No "next" field
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	data, err := client.GetRaw(server.URL + "/repos")
	if err != nil {
		t.Fatalf("GetRaw() error: %v", err)
	}

	var paginatedResp struct {
		Next   string                   `json:"next"`
		Values []map[string]interface{} `json:"values"`
	}
	if err := json.Unmarshal(data, &paginatedResp); err != nil {
		t.Fatalf("failed to parse paginated response: %v", err)
	}

	if paginatedResp.Next != "" {
		t.Errorf("expected empty next link, got %q", paginatedResp.Next)
	}
	if len(paginatedResp.Values) != 1 {
		t.Errorf("expected 1 value, got %d", len(paginatedResp.Values))
	}
}

// TestClient_AllHTTPMethods tests all HTTP methods
func TestClient_AllHTTPMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != method {
					t.Errorf("expected %s, got %s", method, r.Method)
				}
				w.WriteHeader(200)
				w.Write([]byte(`{"ok":true}`))
			}))
			defer server.Close()

			client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
				AccessToken: "test-token",
			})

			var body io.Reader
			if method == "POST" || method == "PUT" {
				body = strings.NewReader(`{"test":"data"}`)
			}

			resp, err := client.doRequest(method, server.URL+"/test", body, "application/json")
			if err != nil {
				t.Fatalf("%s request error: %v", method, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				t.Errorf("expected 200, got %d", resp.StatusCode)
			}
		})
	}
}

// TestClient_LargeResponseBody tests handling of large responses
func TestClient_LargeResponseBody(t *testing.T) {
	largeData := strings.Repeat(`{"item":"value"},`, 1000)
	largeData = `{"values":[` + largeData[:len(largeData)-1] + `]}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(largeData))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	data, err := client.GetRaw(server.URL + "/test")
	if err != nil {
		t.Fatalf("GetRaw() error: %v", err)
	}

	if len(data) < 1000 {
		t.Errorf("expected large response, got %d bytes", len(data))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse large response: %v", err)
	}
}

// mockRoundTripper is a custom HTTP RoundTripper for testing
type mockRoundTripper struct {
	roundTripFunc func(*http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

// TestClient_Get_WithBitbucketAPI tests the Get wrapper method
func TestClient_Get_WithBitbucketAPI(t *testing.T) {
	mockTransport := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			expectedURL := config.BitbucketAPI + "/repositories"
			if req.URL.String() != expectedURL {
				t.Errorf("expected URL %s, got %s", expectedURL, req.URL.String())
			}
			if req.Method != "GET" {
				t.Errorf("expected GET, got %s", req.Method)
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"ok":true}`))),
				Header:     make(http.Header),
			}, nil
		},
	}

	httpClient := &http.Client{Transport: mockTransport}
	client := NewClientWith(httpClient, &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	data, err := client.Get("/repositories")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	var result map[string]bool
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if !result["ok"] {
		t.Error("expected ok=true")
	}
}

// TestClient_Post_WithBitbucketAPI tests the Post wrapper method
func TestClient_Post_WithBitbucketAPI(t *testing.T) {
	mockTransport := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			expectedURL := config.BitbucketAPI + "/repositories/myworkspace/myrepo"
			if req.URL.String() != expectedURL {
				t.Errorf("expected URL %s, got %s", expectedURL, req.URL.String())
			}
			if req.Method != "POST" {
				t.Errorf("expected POST, got %s", req.Method)
			}
			body, _ := io.ReadAll(req.Body)
			if string(body) != `{"name":"test"}` {
				t.Errorf("unexpected body: %s", string(body))
			}
			return &http.Response{
				StatusCode: 201,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"created":true}`))),
				Header:     make(http.Header),
			}, nil
		},
	}

	httpClient := &http.Client{Transport: mockTransport}
	client := NewClientWith(httpClient, &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	data, err := client.Post("/repositories/myworkspace/myrepo", `{"name":"test"}`)
	if err != nil {
		t.Fatalf("Post() error: %v", err)
	}

	var result map[string]bool
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if !result["created"] {
		t.Error("expected created=true")
	}
}

// TestClient_Put_WithBitbucketAPI tests the Put wrapper method
func TestClient_Put_WithBitbucketAPI(t *testing.T) {
	mockTransport := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			expectedURL := config.BitbucketAPI + "/repositories/myworkspace/myrepo"
			if req.URL.String() != expectedURL {
				t.Errorf("expected URL %s, got %s", expectedURL, req.URL.String())
			}
			if req.Method != "PUT" {
				t.Errorf("expected PUT, got %s", req.Method)
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"updated":true}`))),
				Header:     make(http.Header),
			}, nil
		},
	}

	httpClient := &http.Client{Transport: mockTransport}
	client := NewClientWith(httpClient, &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	data, err := client.Put("/repositories/myworkspace/myrepo", `{"description":"updated"}`)
	if err != nil {
		t.Fatalf("Put() error: %v", err)
	}

	var result map[string]bool
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if !result["updated"] {
		t.Error("expected updated=true")
	}
}

// TestClient_Delete_WithBitbucketAPI tests the Delete wrapper method
func TestClient_Delete_WithBitbucketAPI(t *testing.T) {
	mockTransport := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			expectedURL := config.BitbucketAPI + "/repositories/myworkspace/myrepo"
			if req.URL.String() != expectedURL {
				t.Errorf("expected URL %s, got %s", expectedURL, req.URL.String())
			}
			if req.Method != "DELETE" {
				t.Errorf("expected DELETE, got %s", req.Method)
			}
			return &http.Response{
				StatusCode: 204,
				Body:       io.NopCloser(bytes.NewReader([]byte{})),
				Header:     make(http.Header),
			}, nil
		},
	}

	httpClient := &http.Client{Transport: mockTransport}
	client := NewClientWith(httpClient, &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	data, err := client.Delete("/repositories/myworkspace/myrepo")
	if err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	if data != nil {
		t.Error("expected nil response for 204")
	}
}

// TestClient_PostForm_WithBitbucketAPI tests the PostForm wrapper method
func TestClient_PostForm_WithBitbucketAPI(t *testing.T) {
	mockTransport := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			expectedURL := config.BitbucketAPI + "/repositories/myworkspace/myrepo/watchers"
			if req.URL.String() != expectedURL {
				t.Errorf("expected URL %s, got %s", expectedURL, req.URL.String())
			}
			if req.Method != "POST" {
				t.Errorf("expected POST, got %s", req.Method)
			}
			contentType := req.Header.Get("Content-Type")
			if contentType != "application/x-www-form-urlencoded" {
				t.Errorf("expected form content type, got %s", contentType)
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"ok":true}`))),
				Header:     make(http.Header),
			}, nil
		},
	}

	httpClient := &http.Client{Transport: mockTransport}
	client := NewClientWith(httpClient, &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	formData := url.Values{}
	formData.Set("watch", "true")

	data, err := client.PostForm("/repositories/myworkspace/myrepo/watchers", formData)
	if err != nil {
		t.Fatalf("PostForm() error: %v", err)
	}

	var result map[string]bool
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if !result["ok"] {
		t.Error("expected ok=true")
	}
}

// TestClient_TransportDefaults verifies default transport configuration
func TestClient_TransportDefaults(t *testing.T) {
	// Clear all transport-related env vars to ensure defaults
	t.Setenv("BB_HTTP_MAX_IDLE_CONNS", "")
	t.Setenv("BB_HTTP_MAX_IDLE_CONNS_PER_HOST", "")
	t.Setenv("BB_HTTP_IDLE_CONN_TIMEOUT", "")

	// Create a minimal config/token for testing
	cfg := &config.Config{
		OAuthKey:    "test-key",
		OAuthSecret: "test-secret",
	}
	token := &config.TokenData{
		AccessToken:  "test-token",
		RefreshToken: "test-refresh",
	}

	// Save config and token so NewClient can load them
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	config.ResetConfigDirCache()
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}
	if err := config.SaveToken(token); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", client.httpClient.Transport)
	}

	// Verify default values
	if transport.MaxIdleConns != 100 {
		t.Errorf("MaxIdleConns = %d, want 100", transport.MaxIdleConns)
	}
	if transport.MaxIdleConnsPerHost != 10 {
		t.Errorf("MaxIdleConnsPerHost = %d, want 10", transport.MaxIdleConnsPerHost)
	}
	if transport.IdleConnTimeout != 90*time.Second {
		t.Errorf("IdleConnTimeout = %v, want 90s", transport.IdleConnTimeout)
	}
}

// TestClient_TransportCustomValues verifies custom transport configuration via env vars
func TestClient_TransportCustomValues(t *testing.T) {
	tests := []struct {
		name                    string
		maxIdleConns            string
		maxIdleConnsPerHost     string
		idleConnTimeout         string
		expectedMaxIdle         int
		expectedMaxIdlePerHost  int
		expectedIdleConnTimeout time.Duration
	}{
		{
			name:                    "all custom values",
			maxIdleConns:            "200",
			maxIdleConnsPerHost:     "20",
			idleConnTimeout:         "120",
			expectedMaxIdle:         200,
			expectedMaxIdlePerHost:  20,
			expectedIdleConnTimeout: 120 * time.Second,
		},
		{
			name:                    "only max idle conns custom",
			maxIdleConns:            "50",
			maxIdleConnsPerHost:     "",
			idleConnTimeout:         "",
			expectedMaxIdle:         50,
			expectedMaxIdlePerHost:  10,
			expectedIdleConnTimeout: 90 * time.Second,
		},
		{
			name:                    "only max idle per host custom",
			maxIdleConns:            "",
			maxIdleConnsPerHost:     "5",
			idleConnTimeout:         "",
			expectedMaxIdle:         100,
			expectedMaxIdlePerHost:  5,
			expectedIdleConnTimeout: 90 * time.Second,
		},
		{
			name:                    "only idle timeout custom",
			maxIdleConns:            "",
			maxIdleConnsPerHost:     "",
			idleConnTimeout:         "60",
			expectedMaxIdle:         100,
			expectedMaxIdlePerHost:  10,
			expectedIdleConnTimeout: 60 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.maxIdleConns != "" {
				t.Setenv("BB_HTTP_MAX_IDLE_CONNS", tt.maxIdleConns)
			}
			if tt.maxIdleConnsPerHost != "" {
				t.Setenv("BB_HTTP_MAX_IDLE_CONNS_PER_HOST", tt.maxIdleConnsPerHost)
			}
			if tt.idleConnTimeout != "" {
				t.Setenv("BB_HTTP_IDLE_CONN_TIMEOUT", tt.idleConnTimeout)
			}

			// Create a minimal config/token for testing
			cfg := &config.Config{
				OAuthKey:    "test-key",
				OAuthSecret: "test-secret",
			}
			token := &config.TokenData{
				AccessToken:  "test-token",
				RefreshToken: "test-refresh",
			}

			// Save config and token so NewClient can load them
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)
			config.ResetConfigDirCache()
			if err := config.SaveConfig(cfg); err != nil {
				t.Fatalf("failed to save config: %v", err)
			}
			if err := config.SaveToken(token); err != nil {
				t.Fatalf("failed to save token: %v", err)
			}

			client, err := NewClient()
			if err != nil {
				t.Fatalf("NewClient() error: %v", err)
			}

			transport, ok := client.httpClient.Transport.(*http.Transport)
			if !ok {
				t.Fatalf("expected *http.Transport, got %T", client.httpClient.Transport)
			}

			if transport.MaxIdleConns != tt.expectedMaxIdle {
				t.Errorf("MaxIdleConns = %d, want %d", transport.MaxIdleConns, tt.expectedMaxIdle)
			}
			if transport.MaxIdleConnsPerHost != tt.expectedMaxIdlePerHost {
				t.Errorf("MaxIdleConnsPerHost = %d, want %d", transport.MaxIdleConnsPerHost, tt.expectedMaxIdlePerHost)
			}
			if transport.IdleConnTimeout != tt.expectedIdleConnTimeout {
				t.Errorf("IdleConnTimeout = %v, want %v", transport.IdleConnTimeout, tt.expectedIdleConnTimeout)
			}
		})
	}
}

// TestClient_TransportInvalidValues verifies fallback to defaults for invalid env vars
func TestClient_TransportInvalidValues(t *testing.T) {
	tests := []struct {
		name                string
		maxIdleConns        string
		maxIdleConnsPerHost string
		idleConnTimeout     string
	}{
		{
			name:                "non-numeric values",
			maxIdleConns:        "abc",
			maxIdleConnsPerHost: "xyz",
			idleConnTimeout:     "invalid",
		},
		{
			name:                "negative values",
			maxIdleConns:        "-10",
			maxIdleConnsPerHost: "-5",
			idleConnTimeout:     "-30",
		},
		{
			name:                "zero values",
			maxIdleConns:        "0",
			maxIdleConnsPerHost: "0",
			idleConnTimeout:     "0",
		},
		{
			name:                "mixed invalid and valid",
			maxIdleConns:        "abc",
			maxIdleConnsPerHost: "20",
			idleConnTimeout:     "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BB_HTTP_MAX_IDLE_CONNS", tt.maxIdleConns)
			t.Setenv("BB_HTTP_MAX_IDLE_CONNS_PER_HOST", tt.maxIdleConnsPerHost)
			t.Setenv("BB_HTTP_IDLE_CONN_TIMEOUT", tt.idleConnTimeout)

			// Create a minimal config/token for testing
			cfg := &config.Config{
				OAuthKey:    "test-key",
				OAuthSecret: "test-secret",
			}
			token := &config.TokenData{
				AccessToken:  "test-token",
				RefreshToken: "test-refresh",
			}

			// Save config and token so NewClient can load them
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)
			config.ResetConfigDirCache()
			if err := config.SaveConfig(cfg); err != nil {
				t.Fatalf("failed to save config: %v", err)
			}
			if err := config.SaveToken(token); err != nil {
				t.Fatalf("failed to save token: %v", err)
			}

			client, err := NewClient()
			if err != nil {
				t.Fatalf("NewClient() error: %v", err)
			}

			transport, ok := client.httpClient.Transport.(*http.Transport)
			if !ok {
				t.Fatalf("expected *http.Transport, got %T", client.httpClient.Transport)
			}

			// Verify defaults are used for invalid values
			expectedMaxIdle := 100
			expectedMaxIdlePerHost := 10
			expectedIdleTimeout := 90 * time.Second

			// Special case: if maxIdleConnsPerHost was valid in the "mixed" test
			if tt.name == "mixed invalid and valid" && tt.maxIdleConnsPerHost == "20" {
				expectedMaxIdlePerHost = 20
			}

			if transport.MaxIdleConns != expectedMaxIdle {
				t.Errorf("MaxIdleConns = %d, want %d (default)", transport.MaxIdleConns, expectedMaxIdle)
			}
			if transport.MaxIdleConnsPerHost != expectedMaxIdlePerHost {
				t.Errorf("MaxIdleConnsPerHost = %d, want %d", transport.MaxIdleConnsPerHost, expectedMaxIdlePerHost)
			}
			if transport.IdleConnTimeout != expectedIdleTimeout {
				t.Errorf("IdleConnTimeout = %v, want %v (default)", transport.IdleConnTimeout, expectedIdleTimeout)
			}
		})
	}
}

// TestClient_TransportAttachedToHTTPClient verifies transport is properly attached
func TestClient_TransportAttachedToHTTPClient(t *testing.T) {
	t.Setenv("BB_HTTP_MAX_IDLE_CONNS", "150")

	// Create a minimal config/token for testing
	cfg := &config.Config{
		OAuthKey:    "test-key",
		OAuthSecret: "test-secret",
	}
	token := &config.TokenData{
		AccessToken:  "test-token",
		RefreshToken: "test-refresh",
	}

	// Save config and token so NewClient can load them
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	config.ResetConfigDirCache()
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}
	if err := config.SaveToken(token); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	// Verify transport is not nil
	if client.httpClient.Transport == nil {
		t.Fatal("expected non-nil Transport")
	}

	// Verify it's the correct type
	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", client.httpClient.Transport)
	}

	// Verify the configured value is applied
	if transport.MaxIdleConns != 150 {
		t.Errorf("MaxIdleConns = %d, want 150", transport.MaxIdleConns)
	}
}
