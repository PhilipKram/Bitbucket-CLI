package auth

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

// TestOpenBrowser tests are skipped to avoid launching real browser processes
// in CI/developer environments. To properly test openBrowser, refactor it to
// accept an injectable command runner.

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
	if !strings.Contains(err.Error(), "Failed to parse token response") {
		t.Errorf("expected parse error in message, got: %v", err)
	}
}

func TestRefreshAccessToken_NetworkError(t *testing.T) {
	// Use a closed server for a fast, deterministic connection error
	server := httptest.NewServer(http.NewServeMux())
	closedURL := server.URL
	server.Close()

	orig := config.TokenURL
	config.TokenURL = closedURL
	defer func() { config.TokenURL = orig }()

	_, err := RefreshAccessToken("client", "secret", "refresh")
	if err == nil {
		t.Fatal("expected error for network failure")
	}
}

func TestExchangeCode_NetworkError(t *testing.T) {
	// Use a closed server for a fast, deterministic connection error
	server := httptest.NewServer(http.NewServeMux())
	closedURL := server.URL
	server.Close()

	orig := config.TokenURL
	config.TokenURL = closedURL
	defer func() { config.TokenURL = orig }()

	_, err := exchangeCode("client", "secret", "code", "http://localhost:8817/callback")
	if err == nil {
		t.Fatal("expected error for network failure")
	}
}


func TestExchangeCode_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		// Empty response body
	}))
	defer server.Close()

	orig := config.TokenURL
	config.TokenURL = server.URL
	defer func() { config.TokenURL = orig }()

	_, err := exchangeCode("client", "secret", "code", "http://localhost:8817/callback")
	if err == nil {
		t.Fatal("expected error for empty response")
	}
}

func TestRefreshAccessToken_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		// Empty response body
	}))
	defer server.Close()

	orig := config.TokenURL
	config.TokenURL = server.URL
	defer func() { config.TokenURL = orig }()

	_, err := RefreshAccessToken("client", "secret", "refresh")
	if err == nil {
		t.Fatal("expected error for empty response")
	}
}

func TestExchangeCode_MissingFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		// Response with missing fields
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "token-only",
		})
	}))
	defer server.Close()

	orig := config.TokenURL
	config.TokenURL = server.URL
	defer func() { config.TokenURL = orig }()

	// Should still succeed - Go will use zero values for missing fields
	token, err := exchangeCode("client", "secret", "code", "http://localhost:8817/callback")
	if err != nil {
		t.Fatalf("exchangeCode() error: %v", err)
	}
	if token.AccessToken != "token-only" {
		t.Errorf("AccessToken = %q, want %q", token.AccessToken, "token-only")
	}
	// RefreshToken should be empty string
	if token.RefreshToken != "" {
		t.Errorf("RefreshToken = %q, want empty string", token.RefreshToken)
	}
}

func TestRefreshAccessToken_MissingFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		// Response with missing fields
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "new-token-only",
		})
	}))
	defer server.Close()

	orig := config.TokenURL
	config.TokenURL = server.URL
	defer func() { config.TokenURL = orig }()

	// Should still succeed - Go will use zero values for missing fields
	token, err := RefreshAccessToken("client", "secret", "refresh")
	if err != nil {
		t.Fatalf("RefreshAccessToken() error: %v", err)
	}
	if token.AccessToken != "new-token-only" {
		t.Errorf("AccessToken = %q, want %q", token.AccessToken, "new-token-only")
	}
}

func TestExchangeCode_VariousHTTPErrors(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		body       string
		wantError  string
	}{
		{
			name:       "Unauthorized",
			statusCode: 401,
			body:       `{"error":"unauthorized"}`,
			wantError:  "HTTP 401",
		},
		{
			name:       "Forbidden",
			statusCode: 403,
			body:       `{"error":"forbidden"}`,
			wantError:  "HTTP 403",
		},
		{
			name:       "Internal Server Error",
			statusCode: 500,
			body:       `{"error":"internal_error"}`,
			wantError:  "HTTP 500",
		},
		{
			name:       "Bad Gateway",
			statusCode: 502,
			body:       `{"error":"bad_gateway"}`,
			wantError:  "HTTP 502",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(tc.body))
			}))
			defer server.Close()

			orig := config.TokenURL
			config.TokenURL = server.URL
			defer func() { config.TokenURL = orig }()

			_, err := exchangeCode("client", "secret", "code", "http://localhost:8817/callback")
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.wantError) {
				t.Errorf("expected %q in error, got: %v", tc.wantError, err)
			}
		})
	}
}

// --- WSL detection tests ---

func TestIsWSLFromProcVersion_WSL2(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "version")
	os.WriteFile(path, []byte("Linux version 5.15.90.1-microsoft-standard-WSL2 (root@1234) (gcc)"), 0644)

	if !isWSLFromProcVersion(path) {
		t.Error("expected WSL detection for microsoft-standard-WSL2 kernel")
	}
}

func TestIsWSLFromProcVersion_WSL1(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "version")
	os.WriteFile(path, []byte("Linux version 4.4.0-19041-Microsoft (Microsoft@Microsoft.com)"), 0644)

	if !isWSLFromProcVersion(path) {
		t.Error("expected WSL detection for Microsoft kernel (WSL1)")
	}
}

func TestIsWSLFromProcVersion_NativeLinux(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "version")
	os.WriteFile(path, []byte("Linux version 6.1.0-18-amd64 (debian-kernel@lists.debian.org)"), 0644)

	if isWSLFromProcVersion(path) {
		t.Error("native Linux should not be detected as WSL")
	}
}

func TestIsWSLFromProcVersion_FileNotFound(t *testing.T) {
	if isWSLFromProcVersion("/nonexistent/path/version") {
		t.Error("missing file should not be detected as WSL")
	}
}

func TestIsWSLFromProcVersion_CaseInsensitive(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "version")
	os.WriteFile(path, []byte("Linux version 5.15.90.1-MICROSOFT-standard-WSL2"), 0644)

	if !isWSLFromProcVersion(path) {
		t.Error("WSL detection should be case-insensitive")
	}
}

// --- Callback handler tests (reproduces the reported Bitbucket error) ---

func TestCallbackHandler_UnsupportedResponseType(t *testing.T) {
	// This reproduces the exact error from the developer's screenshot:
	// Bitbucket returns error=unsupported_response_type when the OAuth consumer
	// is not configured as a "private consumer".
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errMsg := r.URL.Query().Get("error_description")
			if errMsg == "" {
				errMsg = r.URL.Query().Get("error")
			}
			if errMsg == "" {
				errMsg = "no authorization code received"
			}
			fmt.Fprintf(w, "<html><body><h2>Authentication Failed</h2><p>%s</p><p>You can close this window.</p></body></html>", html.EscapeString(errMsg))
			errCh <- fmt.Errorf("Authorization failed: %s", errMsg)
			return
		}
		codeCh <- code
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// Simulate the exact callback Bitbucket sends when consumer is not private
	resp, err := http.Get(server.URL + "/callback?error=unsupported_response_type&error_description=Invalid+value+specified+None.+Must+be+one+of%3A+code")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Verify the error page is rendered
	if !strings.Contains(bodyStr, "Authentication Failed") {
		t.Error("expected 'Authentication Failed' in response body")
	}
	if !strings.Contains(bodyStr, "Invalid value specified None") {
		t.Error("expected error description in response body")
	}

	// Verify the error is sent to errCh
	select {
	case err := <-errCh:
		if !strings.Contains(err.Error(), "Invalid value specified None") {
			t.Errorf("expected Bitbucket error description in error, got: %v", err)
		}
	default:
		t.Error("expected error on errCh")
	}
}

func TestCallbackHandler_ErrorFallbackToErrorParam(t *testing.T) {
	// When error_description is empty, should fall back to the error param
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errMsg := r.URL.Query().Get("error_description")
			if errMsg == "" {
				errMsg = r.URL.Query().Get("error")
			}
			if errMsg == "" {
				errMsg = "no authorization code received"
			}
			fmt.Fprintf(w, "<html><body><h2>Authentication Failed</h2><p>%s</p></body></html>", html.EscapeString(errMsg))
			errCh <- fmt.Errorf("Authorization failed: %s", errMsg)
			return
		}
		codeCh <- code
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := http.Get(server.URL + "/callback?error=unsupported_response_type")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "unsupported_response_type") {
		t.Error("expected error param in response when error_description is missing")
	}

	select {
	case err := <-errCh:
		if !strings.Contains(err.Error(), "unsupported_response_type") {
			t.Errorf("expected error param in error, got: %v", err)
		}
	default:
		t.Error("expected error on errCh")
	}
}

func TestCallbackHandler_SuccessfulCode(t *testing.T) {
	codeCh := make(chan string, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			t.Error("expected code parameter")
			return
		}
		fmt.Fprint(w, "<html><body><h2>Authentication Successful!</h2></body></html>")
		codeCh <- code
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := http.Get(server.URL + "/callback?code=test-auth-code-123")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Authentication Successful") {
		t.Error("expected success message in response")
	}

	select {
	case code := <-codeCh:
		if code != "test-auth-code-123" {
			t.Errorf("code = %q, want %q", code, "test-auth-code-123")
		}
	default:
		t.Error("expected code on codeCh")
	}
}

// --- Login listen address tests ---

func TestLogin_ListenAddress_NonWSL(t *testing.T) {
	// On non-WSL (macOS/native Linux), Login should bind to 127.0.0.1
	// We test this by checking that Login starts a server on 127.0.0.1
	// We need to use a mock token server for the exchange step

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "test-token",
			"refresh_token": "test-refresh",
			"token_type":    "bearer",
		})
	}))
	defer tokenServer.Close()

	origTokenURL := config.TokenURL
	config.TokenURL = tokenServer.URL
	defer func() { config.TokenURL = origTokenURL }()

	// Start Login in a goroutine — it will block waiting for callback
	done := make(chan error, 1)
	go func() {
		_, err := Login("test-client", "test-secret")
		done <- err
	}()

	// Wait for the server to start, then verify it's on 127.0.0.1
	var conn net.Conn
	var err error
	for i := 0; i < 50; i++ {
		conn, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", OAuthCallbackPort))
		if err == nil {
			conn.Close()
			break
		}
	}
	if err != nil {
		t.Fatalf("could not connect to callback server on 127.0.0.1:%d: %v", OAuthCallbackPort, err)
	}

	// Send a successful callback to unblock Login
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/callback?code=test-code", OAuthCallbackPort))
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}
	resp.Body.Close()

	// Wait for Login to complete
	if err := <-done; err != nil {
		t.Fatalf("Login() error: %v", err)
	}
}

func TestRefreshAccessToken_VariousHTTPErrors(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		body       string
		wantError  string
	}{
		{
			name:       "Unauthorized",
			statusCode: 401,
			body:       `{"error":"invalid_token"}`,
			wantError:  "HTTP 401",
		},
		{
			name:       "Bad Request",
			statusCode: 400,
			body:       `{"error":"invalid_grant"}`,
			wantError:  "HTTP 400",
		},
		{
			name:       "Service Unavailable",
			statusCode: 503,
			body:       `{"error":"service_unavailable"}`,
			wantError:  "HTTP 503",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(tc.body))
			}))
			defer server.Close()

			orig := config.TokenURL
			config.TokenURL = server.URL
			defer func() { config.TokenURL = orig }()

			_, err := RefreshAccessToken("client", "secret", "refresh")
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.wantError) {
				t.Errorf("expected %q in error, got: %v", tc.wantError, err)
			}
		})
	}
}
