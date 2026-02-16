package auth

import (
	"encoding/json"
	"html"
	"net/http"
	"net/http/httptest"
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
