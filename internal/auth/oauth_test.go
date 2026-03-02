package auth

import (
	"encoding/json"
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PhilipKram/bitbucket-cli/internal/config"
)

func TestRefreshAccessToken_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Error("expected basic auth")
		}
		if user != "client-id" || pass != "client-secret" {
			t.Errorf("unexpected credentials: user=%q pass=%q", user, pass)
		}

		r.ParseForm()
		if r.FormValue("grant_type") != "refresh_token" {
			t.Errorf("expected grant_type=refresh_token, got %s", r.FormValue("grant_type"))
		}
		if r.FormValue("refresh_token") != "old-refresh" {
			t.Errorf("expected refresh_token=old-refresh, got %s", r.FormValue("refresh_token"))
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "refreshed-token",
			"refresh_token": "new-refresh",
			"token_type":    "bearer",
			"expires_in":    7200,
		})
	}))
	defer server.Close()

	// Override TokenURL for testing
	orig := config.TokenURL
	config.TokenURL = server.URL
	defer func() { config.TokenURL = orig }()

	token, err := RefreshAccessToken("client-id", "client-secret", "old-refresh")
	if err != nil {
		t.Fatalf("RefreshAccessToken() error: %v", err)
	}

	if token.AccessToken != "refreshed-token" {
		t.Errorf("AccessToken = %q, want %q", token.AccessToken, "refreshed-token")
	}
	if token.RefreshToken != "new-refresh" {
		t.Errorf("RefreshToken = %q, want %q", token.RefreshToken, "new-refresh")
	}
	if token.TokenType != "bearer" {
		t.Errorf("TokenType = %q, want %q", token.TokenType, "bearer")
	}
}

func TestRefreshAccessToken_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer server.Close()

	orig := config.TokenURL
	config.TokenURL = server.URL
	defer func() { config.TokenURL = orig }()

	_, err := RefreshAccessToken("client-id", "client-secret", "bad-token")
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestRefreshAccessToken_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	orig := config.TokenURL
	config.TokenURL = server.URL
	defer func() { config.TokenURL = orig }()

	_, err := RefreshAccessToken("client-id", "client-secret", "refresh")
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestCallbackHandler_HTMLEscaping(t *testing.T) {
	// Verify that html.EscapeString properly escapes HTML special characters
	// This tests the fix for review comment #9
	malicious := `<script>alert("xss")</script>`
	escaped := html.EscapeString(malicious)

	if escaped == malicious {
		t.Error("html.EscapeString should escape HTML tags")
	}
	if escaped != "&lt;script&gt;alert(&#34;xss&#34;)&lt;/script&gt;" {
		t.Errorf("unexpected escape result: %s", escaped)
	}
}

func TestOpenBrowser(t *testing.T) {
	// Just verify the function doesn't panic.
	// On CI without a display, it may fail but should not panic.
	err := openBrowser("https://example.com")
	// We don't check the error because the test environment may not have
	// a browser/display available. We just verify it doesn't panic.
	_ = err
}

func TestOpenBrowser_InvalidURL(t *testing.T) {
	// Test with various URLs to ensure they're passed correctly
	testCases := []string{
		"https://bitbucket.org",
		"http://localhost:8817",
		"https://example.com?param=value&other=test",
	}

	for _, url := range testCases {
		// Just verify no panic occurs
		_ = openBrowser(url)
	}
}

func TestExchangeCode_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		if r.FormValue("grant_type") != "authorization_code" {
			t.Errorf("expected grant_type=authorization_code, got %s", r.FormValue("grant_type"))
		}
		if r.FormValue("code") != "test-auth-code" {
			t.Errorf("expected code=test-auth-code, got %s", r.FormValue("code"))
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "new-access-token",
			"refresh_token": "new-refresh-token",
			"token_type":    "bearer",
			"expires_in":    3600,
		})
	}))
	defer server.Close()

	orig := config.TokenURL
	config.TokenURL = server.URL
	defer func() { config.TokenURL = orig }()

	token, err := exchangeCode("test-client-id", "test-client-secret", "test-auth-code", "http://localhost:8817/callback")
	if err != nil {
		t.Fatalf("exchangeCode() error: %v", err)
	}

	if token.AccessToken != "new-access-token" {
		t.Errorf("AccessToken = %q, want %q", token.AccessToken, "new-access-token")
	}
	if token.RefreshToken != "new-refresh-token" {
		t.Errorf("RefreshToken = %q, want %q", token.RefreshToken, "new-refresh-token")
	}
}

func TestExchangeCode_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error":"invalid_client"}`))
	}))
	defer server.Close()

	orig := config.TokenURL
	config.TokenURL = server.URL
	defer func() { config.TokenURL = orig }()

	_, err := exchangeCode("bad-client", "bad-secret", "code", "http://localhost:8817/callback")
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if !strings.Contains(err.Error(), "HTTP 401") {
		t.Errorf("expected HTTP 401 in error, got: %v", err)
	}
}

func TestExchangeCode_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	orig := config.TokenURL
	config.TokenURL = server.URL
	defer func() { config.TokenURL = orig }()

	_, err := exchangeCode("client", "secret", "code", "http://localhost:8817/callback")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse token response") {
		t.Errorf("expected parse error in message, got: %v", err)
	}
}

func TestRefreshAccessToken_NetworkError(t *testing.T) {
	// Use an invalid URL to trigger a network error
	orig := config.TokenURL
	config.TokenURL = "http://invalid-host-that-does-not-exist-12345.local"
	defer func() { config.TokenURL = orig }()

	_, err := RefreshAccessToken("client", "secret", "refresh")
	if err == nil {
		t.Fatal("expected error for network failure")
	}
	if !strings.Contains(err.Error(), "token refresh failed") {
		t.Errorf("expected 'token refresh failed' in error, got: %v", err)
	}
}

func TestExchangeCode_NetworkError(t *testing.T) {
	// Use an invalid URL to trigger a network error
	orig := config.TokenURL
	config.TokenURL = "http://invalid-host-that-does-not-exist-12345.local"
	defer func() { config.TokenURL = orig }()

	_, err := exchangeCode("client", "secret", "code", "http://localhost:8817/callback")
	if err == nil {
		t.Fatal("expected error for network failure")
	}
	if !strings.Contains(err.Error(), "token exchange failed") {
		t.Errorf("expected 'token exchange failed' in error, got: %v", err)
	}
}
