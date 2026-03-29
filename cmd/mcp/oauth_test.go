package mcp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mcpPkg "github.com/PhilipKram/bitbucket-cli/internal/mcp"
)

// --- Session store tests ---

func TestNewSessionStore(t *testing.T) {
	store := newSessionStore("key", "secret", "")
	if store.clientID != "key" {
		t.Errorf("Expected clientID 'key', got %s", store.clientID)
	}
	if store.clientSecret != "secret" {
		t.Errorf("Expected clientSecret 'secret', got %s", store.clientSecret)
	}
	if len(store.sessions) != 0 {
		t.Errorf("Expected 0 sessions, got %d", len(store.sessions))
	}
}

func TestSessionStore_PutGetSession(t *testing.T) {
	store := newSessionStore("key", "secret", "")

	sess := &mcpSession{
		BearerToken: "bearer-123",
		AccessToken: "access-456",
		ClientID:    "client-789",
	}
	store.putSession(sess)

	got := store.getSession("bearer-123")
	if got == nil {
		t.Fatal("Expected session, got nil")
	}
	if got.AccessToken != "access-456" {
		t.Errorf("Expected AccessToken 'access-456', got %s", got.AccessToken)
	}
	if got.ClientID != "client-789" {
		t.Errorf("Expected ClientID 'client-789', got %s", got.ClientID)
	}
}

func TestSessionStore_GetSession_NotFound(t *testing.T) {
	store := newSessionStore("key", "secret", "")
	got := store.getSession("nonexistent")
	if got != nil {
		t.Errorf("Expected nil, got %v", got)
	}
}

func TestSessionStore_PutGetClient(t *testing.T) {
	store := newSessionStore("key", "secret", "")

	client := &oauthRegisteredClient{
		ClientID:     "client-abc",
		ClientName:   "test",
		RedirectURIs: []string{"http://localhost/callback"},
	}
	store.putClient(client)

	got := store.getClient("client-abc")
	if got == nil {
		t.Fatal("Expected client, got nil")
	}
	if got.ClientName != "test" {
		t.Errorf("Expected ClientName 'test', got %s", got.ClientName)
	}
}

func TestSessionStore_PutPopPending(t *testing.T) {
	store := newSessionStore("key", "secret", "")

	req := &oauthAuthRequest{
		ClientID:    "client-1",
		RedirectURI: "http://localhost/callback",
		State:       "state-1",
		BBState:     "bb-state-1",
	}
	store.putPending("bb-state-1", req)

	got := store.popPending("bb-state-1")
	if got == nil {
		t.Fatal("Expected pending request, got nil")
	}
	if got.ClientID != "client-1" {
		t.Errorf("Expected ClientID 'client-1', got %s", got.ClientID)
	}

	// Should be removed after pop
	got2 := store.popPending("bb-state-1")
	if got2 != nil {
		t.Error("Expected nil after pop, got pending request")
	}
}

func TestSessionStore_PutPopCode(t *testing.T) {
	store := newSessionStore("key", "secret", "")

	ac := &oauthAuthCode{
		Code:        "code-1",
		ClientID:    "client-1",
		RedirectURI: "http://localhost/callback",
		Session:     &mcpSession{BearerToken: "bearer-1", AccessToken: "access-1"},
	}
	store.putCode("code-1", ac)

	got := store.popCode("code-1")
	if got == nil {
		t.Fatal("Expected auth code, got nil")
	}
	if got.Session.BearerToken != "bearer-1" {
		t.Errorf("Expected BearerToken 'bearer-1', got %s", got.Session.BearerToken)
	}

	// Should be removed after pop
	got2 := store.popCode("code-1")
	if got2 != nil {
		t.Error("Expected nil after pop, got auth code")
	}
}

// --- Session persistence tests ---

func TestSessionStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")

	// Create store and save a session
	store1 := newSessionStore("key", "secret", path)
	store1.putSession(&mcpSession{
		BearerToken: "bearer-persist",
		AccessToken: "access-persist",
		ClientID:    "client-persist",
	})

	// Verify file was written
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read sessions file: %v", err)
	}
	if !strings.Contains(string(data), "bearer-persist") {
		t.Error("Sessions file does not contain expected bearer token")
	}

	// Create new store from same file — should load the session
	store2 := newSessionStore("key", "secret", path)
	got := store2.getSession("bearer-persist")
	if got == nil {
		t.Fatal("Expected session loaded from disk, got nil")
	}
	if got.AccessToken != "access-persist" {
		t.Errorf("Expected AccessToken 'access-persist', got %s", got.AccessToken)
	}
}

func TestSessionStore_PersistenceNoPath(t *testing.T) {
	// With empty path, persistence should be a no-op (no panic)
	store := newSessionStore("key", "secret", "")
	store.putSession(&mcpSession{BearerToken: "test", AccessToken: "test"})
}

// --- OAuth metadata handler tests ---

func TestOAuthMetadataHandler(t *testing.T) {
	handler := oauthMetadataHandler("http://localhost:8181")

	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", ct)
	}

	var metadata map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &metadata); err != nil {
		t.Fatalf("Failed to parse metadata: %v", err)
	}

	if metadata["issuer"] != "http://localhost:8181" {
		t.Errorf("Expected issuer http://localhost:8181, got %v", metadata["issuer"])
	}
	if metadata["authorization_endpoint"] != "http://localhost:8181/oauth/authorize" {
		t.Errorf("Expected auth endpoint, got %v", metadata["authorization_endpoint"])
	}
	if metadata["token_endpoint"] != "http://localhost:8181/oauth/token" {
		t.Errorf("Expected token endpoint, got %v", metadata["token_endpoint"])
	}
	if metadata["registration_endpoint"] != "http://localhost:8181/oauth/register" {
		t.Errorf("Expected registration endpoint, got %v", metadata["registration_endpoint"])
	}
}

func TestOAuthMetadataHandler_MethodNotAllowed(t *testing.T) {
	handler := oauthMetadataHandler("http://localhost:8181")

	req := httptest.NewRequest(http.MethodPost, "/.well-known/oauth-authorization-server", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", w.Code)
	}
}

func TestOAuthProtectedResourceHandler(t *testing.T) {
	handler := oauthProtectedResourceHandler("http://localhost:8181", "/mcp")

	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	var resource map[string]interface{}
	body, _ := io.ReadAll(w.Result().Body)
	json.Unmarshal(body, &resource)

	if resource["resource"] != "http://localhost:8181/mcp" {
		t.Errorf("Expected resource http://localhost:8181/mcp, got %v", resource["resource"])
	}
}

// --- OAuth register handler tests ---

func TestOAuthRegisterHandler(t *testing.T) {
	store := newSessionStore("key", "secret", "")
	handler := oauthRegisterHandler(store)

	body := `{"client_name":"test-client","redirect_uris":["http://localhost:9999/callback"]}`
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &result)

	if result["client_name"] != "test-client" {
		t.Errorf("Expected client_name 'test-client', got %v", result["client_name"])
	}

	clientID, ok := result["client_id"].(string)
	if !ok || clientID == "" {
		t.Error("Expected non-empty client_id")
	}

	// Verify client was stored
	stored := store.getClient(clientID)
	if stored == nil {
		t.Fatal("Expected client to be stored")
	}
	if stored.ClientName != "test-client" {
		t.Errorf("Expected stored ClientName 'test-client', got %s", stored.ClientName)
	}
}

func TestOAuthRegisterHandler_MissingRedirectURIs(t *testing.T) {
	store := newSessionStore("key", "secret", "")
	handler := oauthRegisterHandler(store)

	body := `{"client_name":"test-client","redirect_uris":[]}`
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestOAuthRegisterHandler_MethodNotAllowed(t *testing.T) {
	store := newSessionStore("key", "secret", "")
	handler := oauthRegisterHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/oauth/register", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", w.Code)
	}
}

// --- OAuth authorize handler tests ---

func TestOAuthAuthorizeHandler_RedirectsToBitbucket(t *testing.T) {
	store := newSessionStore("bb-consumer-key", "bb-secret", "")

	// Register a client first
	store.putClient(&oauthRegisteredClient{
		ClientID:     "test-client-id",
		ClientName:   "test",
		RedirectURIs: []string{"http://localhost:9999/callback"},
	})

	handler := oauthAuthorizeHandler(store)

	req := httptest.NewRequest(http.MethodGet,
		"/oauth/authorize?client_id=test-client-id&redirect_uri=http://localhost:9999/callback&state=mystate&response_type=code",
		nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusFound {
		t.Errorf("Expected 302, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if !strings.HasPrefix(location, "https://bitbucket.org/site/oauth2/authorize") {
		t.Errorf("Expected redirect to Bitbucket, got %s", location)
	}
	if !strings.Contains(location, "client_id=bb-consumer-key") {
		t.Errorf("Expected Bitbucket consumer key in redirect, got %s", location)
	}
	if !strings.Contains(location, "redirect_uri=http") {
		t.Errorf("Expected redirect_uri in location, got %s", location)
	}

	// Verify pending request was stored
	if len(store.pending) != 1 {
		t.Errorf("Expected 1 pending request, got %d", len(store.pending))
	}
}

func TestOAuthAuthorizeHandler_MissingParams(t *testing.T) {
	store := newSessionStore("key", "secret", "")
	handler := oauthAuthorizeHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestOAuthAuthorizeHandler_UnknownClient(t *testing.T) {
	store := newSessionStore("key", "secret", "")
	handler := oauthAuthorizeHandler(store)

	req := httptest.NewRequest(http.MethodGet,
		"/oauth/authorize?client_id=unknown&redirect_uri=http://localhost/cb",
		nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestOAuthAuthorizeHandler_InvalidRedirectURI(t *testing.T) {
	store := newSessionStore("key", "secret", "")
	store.putClient(&oauthRegisteredClient{
		ClientID:     "client-1",
		ClientName:   "test",
		RedirectURIs: []string{"http://localhost:9999/callback"},
	})

	handler := oauthAuthorizeHandler(store)

	req := httptest.NewRequest(http.MethodGet,
		"/oauth/authorize?client_id=client-1&redirect_uri=http://evil.com/callback",
		nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

// --- OAuth token handler tests ---

func TestOAuthTokenHandler_Success(t *testing.T) {
	store := newSessionStore("key", "secret", "")

	sess := &mcpSession{
		BearerToken: "session-bearer-token",
		AccessToken: "bb-access-token",
		ClientID:    "client-1",
	}
	store.putCode("auth-code-123", &oauthAuthCode{
		Code:        "auth-code-123",
		ClientID:    "client-1",
		RedirectURI: "http://localhost:9999/callback",
		Session:     sess,
	})

	handler := oauthTokenHandler(store)

	body := "grant_type=authorization_code&code=auth-code-123&client_id=client-1&redirect_uri=http://localhost:9999/callback"
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &result)

	if result["access_token"] != "session-bearer-token" {
		t.Errorf("Expected access_token 'session-bearer-token', got %v", result["access_token"])
	}
	if result["token_type"] != "bearer" {
		t.Errorf("Expected token_type 'bearer', got %v", result["token_type"])
	}

	// Verify session was stored
	got := store.getSession("session-bearer-token")
	if got == nil {
		t.Fatal("Expected session to be stored")
	}
	if got.AccessToken != "bb-access-token" {
		t.Errorf("Expected AccessToken 'bb-access-token', got %s", got.AccessToken)
	}
}

func TestOAuthTokenHandler_InvalidCode(t *testing.T) {
	store := newSessionStore("key", "secret", "")
	handler := oauthTokenHandler(store)

	body := "grant_type=authorization_code&code=invalid&client_id=x&redirect_uri=http://localhost/cb"
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestOAuthTokenHandler_ClientIDMismatch(t *testing.T) {
	store := newSessionStore("key", "secret", "")
	store.putCode("code-1", &oauthAuthCode{
		Code:        "code-1",
		ClientID:    "client-1",
		RedirectURI: "http://localhost/cb",
		Session:     &mcpSession{BearerToken: "b", AccessToken: "a"},
	})

	handler := oauthTokenHandler(store)

	body := "grant_type=authorization_code&code=code-1&client_id=wrong-client&redirect_uri=http://localhost/cb"
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestOAuthTokenHandler_RedirectURIMismatch(t *testing.T) {
	store := newSessionStore("key", "secret", "")
	store.putCode("code-1", &oauthAuthCode{
		Code:        "code-1",
		ClientID:    "client-1",
		RedirectURI: "http://localhost/cb",
		Session:     &mcpSession{BearerToken: "b", AccessToken: "a"},
	})

	handler := oauthTokenHandler(store)

	body := "grant_type=authorization_code&code=code-1&client_id=client-1&redirect_uri=http://evil.com/cb"
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestOAuthTokenHandler_UnsupportedGrantType(t *testing.T) {
	store := newSessionStore("key", "secret", "")
	handler := oauthTokenHandler(store)

	body := "grant_type=client_credentials"
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestOAuthTokenHandler_PKCEValidation(t *testing.T) {
	store := newSessionStore("key", "secret", "")

	// Store code with PKCE challenge
	store.putCode("code-pkce", &oauthAuthCode{
		Code:                "code-pkce",
		ClientID:            "client-1",
		RedirectURI:         "http://localhost/cb",
		Session:             &mcpSession{BearerToken: "b", AccessToken: "a"},
		CodeChallenge:       "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk", // S256 of "test_verifier"
		CodeChallengeMethod: "S256",
	})

	handler := oauthTokenHandler(store)

	// Wrong verifier should fail
	body := "grant_type=authorization_code&code=code-pkce&client_id=client-1&redirect_uri=http://localhost/cb&code_verifier=wrong_verifier"
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for wrong PKCE verifier, got %d", w.Code)
	}
}

// --- Session bearer auth middleware tests ---

func TestSessionBearerAuth_NoToken(t *testing.T) {
	store := newSessionStore("key", "secret", "")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := store.sessionBearerAuth(next)
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
	if auth := w.Header().Get("WWW-Authenticate"); auth != "Bearer" {
		t.Errorf("Expected WWW-Authenticate: Bearer, got %s", auth)
	}
}

func TestSessionBearerAuth_InvalidToken(t *testing.T) {
	store := newSessionStore("key", "secret", "")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := store.sessionBearerAuth(next)
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
	if auth := w.Header().Get("WWW-Authenticate"); !strings.Contains(auth, "invalid_token") {
		t.Errorf("Expected WWW-Authenticate with invalid_token, got %s", auth)
	}
}

func TestSessionBearerAuth_ValidToken(t *testing.T) {
	store := newSessionStore("key", "secret", "")
	store.putSession(&mcpSession{
		BearerToken: "valid-token",
		AccessToken: "bb-token",
	})

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := store.sessionBearerAuth(next)
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
	if !called {
		t.Error("Expected next handler to be called")
	}
}

// --- Serve flags tests ---

func TestServeCommand_OAuthFlags(t *testing.T) {
	cmd := NewCmdMCP()
	serveCmd, _, err := cmd.Find([]string{"serve"})
	if err != nil {
		t.Fatalf("Failed to find serve subcommand: %v", err)
	}

	flags := []struct {
		name     string
		defValue string
	}{
		{"client-id", ""},
		{"client-secret", ""},
		{"external-url", ""},
		{"transport", "stdio"},
		{"host", "localhost"},
		{"port", "8080"},
		{"base-path", "/mcp"},
	}

	for _, f := range flags {
		flag := serveCmd.Flags().Lookup(f.name)
		if flag == nil {
			t.Errorf("Expected --%s flag on serve command", f.name)
			continue
		}
		if flag.DefValue != f.defValue {
			t.Errorf("Expected --%s default '%s', got '%s'", f.name, f.defValue, flag.DefValue)
		}
	}
}

// --- createMCPServerWithToken test ---

func TestCreateMCPServerWithToken(t *testing.T) {
	server := createMCPServerWithToken("test-access-token")
	if server == nil {
		t.Fatal("Expected server, got nil")
	}

	// Verify the token is in the server's context
	ctx := server.Context()
	token, ok := mcpPkg.TokenFromContext(ctx)
	if !ok {
		t.Error("Expected token in server context")
	}
	if token != "test-access-token" {
		t.Errorf("Expected token 'test-access-token', got %s", token)
	}
}
