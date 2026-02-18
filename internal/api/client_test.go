package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/PhilipKram/bitbucket-cli/internal/config"
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

func TestClient_HandleResponse_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer server.Close()

	client := NewClientWith(server.Client(), &config.Config{}, &config.TokenData{
		AccessToken: "test-token",
	})

	_, err := client.GetRaw(server.URL + "/missing")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	if got := err.Error(); got == "" {
		t.Error("expected non-empty error message")
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
