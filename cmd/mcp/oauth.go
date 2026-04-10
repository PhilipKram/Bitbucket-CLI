package mcp

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	authPkg "github.com/PhilipKram/bitbucket-cli/internal/auth"
	"github.com/PhilipKram/bitbucket-cli/internal/buildinfo"
	"github.com/PhilipKram/bitbucket-cli/internal/config"
	mcpPkg "github.com/PhilipKram/bitbucket-cli/internal/mcp"
)

func signalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
}

// --- Session types ---

type mcpSession struct {
	BearerToken     string `json:"bearer_token"`
	AccessToken     string `json:"access_token"`
	RefreshToken    string `json:"refresh_token,omitempty"`      // Bitbucket refresh token
	MCPRefreshToken string `json:"mcp_refresh_token,omitempty"` // MCP session refresh token
	TokenExpiresAt  int64  `json:"token_expires_at,omitempty"`
	ClientID        string `json:"client_id"`
	Username        string `json:"username,omitempty"`
}

type oauthRegisteredClient struct {
	ClientID     string   `json:"client_id"`
	ClientName   string   `json:"client_name"`
	RedirectURIs []string `json:"redirect_uris"`
}

type oauthAuthRequest struct {
	ClientID     string `json:"client_id"`
	RedirectURI  string `json:"redirect_uri"`
	State        string `json:"state"`        // client's state
	CodeVerifier string `json:"code_verifier"` // server-generated PKCE for Bitbucket
	BBState      string `json:"bb_state"`      // state sent to Bitbucket
	CodeChallenge string `json:"code_challenge,omitempty"` // client's PKCE challenge
	CodeChallengeMethod string `json:"code_challenge_method,omitempty"`
}

type oauthAuthCode struct {
	Code        string `json:"code"`
	ClientID    string `json:"client_id"`
	RedirectURI string `json:"redirect_uri"`
	Session     *mcpSession `json:"session"`
	CodeChallenge string `json:"code_challenge,omitempty"`
	CodeChallengeMethod string `json:"code_challenge_method,omitempty"`
}

// --- Session store ---

type mcpSessionStore struct {
	mu           sync.RWMutex
	path         string // file path for persistence
	sessions     map[string]*mcpSession
	clients      map[string]*oauthRegisteredClient
	pending      map[string]*oauthAuthRequest // keyed by BB state
	codes        map[string]*oauthAuthCode
	clientID     string // Bitbucket OAuth consumer key
	clientSecret string // Bitbucket OAuth consumer secret
	callbackURL  string // OAuth callback URL (Bitbucket redirects here after user authorizes)
}

func newSessionStore(bbClientID, bbClientSecret, persistPath, callbackURL string) *mcpSessionStore {
	s := &mcpSessionStore{
		sessions:     make(map[string]*mcpSession),
		clients:      make(map[string]*oauthRegisteredClient),
		pending:      make(map[string]*oauthAuthRequest),
		codes:        make(map[string]*oauthAuthCode),
		clientID:     bbClientID,
		clientSecret: bbClientSecret,
		path:         persistPath,
		callbackURL:  callbackURL,
	}
	s.loadFromDisk()
	return s
}

func (s *mcpSessionStore) getSession(bearerToken string) *mcpSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[bearerToken]
}

func (s *mcpSessionStore) putSession(sess *mcpSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sess.BearerToken] = sess
	s.saveToDisk()
}

func (s *mcpSessionStore) putClient(c *oauthRegisteredClient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[c.ClientID] = c
	s.saveToDisk()
}

func (s *mcpSessionStore) getClient(clientID string) *oauthRegisteredClient {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.clients[clientID]
}

func (s *mcpSessionStore) putPending(bbState string, req *oauthAuthRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending[bbState] = req
}

func (s *mcpSessionStore) popPending(bbState string) *oauthAuthRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	req := s.pending[bbState]
	delete(s.pending, bbState)
	return req
}

func (s *mcpSessionStore) putCode(code string, ac *oauthAuthCode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[code] = ac
}

func (s *mcpSessionStore) popCode(code string) *oauthAuthCode {
	s.mu.Lock()
	defer s.mu.Unlock()
	ac := s.codes[code]
	delete(s.codes, code)
	return ac
}

// --- Persistence ---

// defaultTokenExpiry is the lifetime (in seconds) advertised to MCP clients
// via the expires_in field. A long expiry minimises re-authentication prompts
// across Claude Code sessions. The server still refreshes the underlying
// Bitbucket token proactively.
const defaultTokenExpiry = 30 * 24 * 3600 // 30 days

type sessionPersistence struct {
	Sessions []*mcpSession            `json:"sessions"`
	Clients  []*oauthRegisteredClient `json:"clients,omitempty"`
}

func (s *mcpSessionStore) saveToDisk() {
	if s.path == "" {
		return
	}
	p := sessionPersistence{}
	for _, sess := range s.sessions {
		p.Sessions = append(p.Sessions, sess)
	}
	for _, c := range s.clients {
		p.Clients = append(p.Clients, c)
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not marshal sessions: %v\n", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not create session dir: %v\n", err)
		return
	}
	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not persist sessions: %v\n", err)
	}
}

func (s *mcpSessionStore) loadFromDisk() {
	if s.path == "" {
		return
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var p sessionPersistence
	if err := json.Unmarshal(data, &p); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not parse sessions file: %v\n", err)
		return
	}
	for _, sess := range p.Sessions {
		s.sessions[sess.BearerToken] = sess
	}
	for _, c := range p.Clients {
		s.clients[c.ClientID] = c
	}
	if len(s.sessions) > 0 || len(p.Clients) > 0 {
		fmt.Fprintf(os.Stderr, "Loaded %d persisted session(s), %d registered client(s)\n", len(s.sessions), len(p.Clients))
	}
}

// --- Auth middleware ---

// tokenRefreshBuffer is how many seconds before expiry we proactively refresh.
const tokenRefreshBuffer = 300 // 5 minutes

func (s *mcpSessionStore) sessionBearerAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hdr := r.Header.Get("Authorization")
		if !strings.HasPrefix(hdr, "Bearer ") {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(hdr, "Bearer ")
		sess := s.getSession(token)
		if sess == nil {
			w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Auto-refresh Bitbucket token if expired or expiring soon.
		// Also refresh if TokenExpiresAt is 0 (unknown expiry) and we have a refresh token,
		// since the token was likely obtained hours ago.
		needsRefresh := sess.RefreshToken != "" && (sess.TokenExpiresAt == 0 || time.Now().Unix() >= sess.TokenExpiresAt-tokenRefreshBuffer)
		if needsRefresh {
			fmt.Fprintf(os.Stderr, "Refreshing Bitbucket token for session...\n")
			newToken, err := authPkg.RefreshAccessToken(s.clientID, s.clientSecret, sess.RefreshToken)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Token refresh failed: %v\n", err)
				// If we know the token is expired (not just unknown), reject the request
				// so Claude Code triggers re-authorization
				if sess.TokenExpiresAt > 0 && time.Now().Unix() >= sess.TokenExpiresAt {
					w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token", error_description="token expired and refresh failed"`)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				// TokenExpiresAt is 0 (unknown) — let the request through, Bitbucket will tell us
			} else {
				sess.AccessToken = newToken.AccessToken
				if newToken.RefreshToken != "" {
					sess.RefreshToken = newToken.RefreshToken
				}
				if newToken.ExpiresIn > 0 {
					sess.TokenExpiresAt = time.Now().Unix() + int64(newToken.ExpiresIn)
				}
				s.putSession(sess)
				fmt.Fprintf(os.Stderr, "Token refreshed successfully (expires in %ds)\n", newToken.ExpiresIn)
			}
		}

		next.ServeHTTP(w, r)
	})
}

// --- OAuth handlers ---

func oauthMetadataHandler(baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		metadata := map[string]interface{}{
			"issuer":                                baseURL,
			"authorization_endpoint":                baseURL + "/oauth/authorize",
			"token_endpoint":                        baseURL + "/oauth/token",
			"registration_endpoint":                 baseURL + "/oauth/register",
			"response_types_supported":              []string{"code"},
			"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
			"code_challenge_methods_supported":       []string{"S256"},
			"token_endpoint_auth_methods_supported": []string{"none"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metadata)
	}
}

func oauthProtectedResourceHandler(baseURL, basePath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		resource := map[string]interface{}{
			"resource":              baseURL + basePath,
			"authorization_servers": []string{baseURL},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resource)
	}
}

func oauthRegisterHandler(store *mcpSessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			ClientName   string   `json:"client_name"`
			RedirectURIs []string `json:"redirect_uris"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if len(req.RedirectURIs) == 0 {
			http.Error(w, "redirect_uris is required", http.StatusBadRequest)
			return
		}

		id, err := generateToken()
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		client := &oauthRegisteredClient{
			ClientID:     id,
			ClientName:   req.ClientName,
			RedirectURIs: req.RedirectURIs,
		}
		store.putClient(client)

		resp := map[string]interface{}{
			"client_id":                id,
			"client_name":             req.ClientName,
			"redirect_uris":           req.RedirectURIs,
			"grant_types":             []string{"authorization_code"},
			"response_types":          []string{"code"},
			"token_endpoint_auth_method": "none",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}

func oauthAuthorizeHandler(store *mcpSessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		q := r.URL.Query()
		clientID := q.Get("client_id")
		redirectURI := q.Get("redirect_uri")
		state := q.Get("state")
		codeChallenge := q.Get("code_challenge")
		codeChallengeMethod := q.Get("code_challenge_method")

		if clientID == "" || redirectURI == "" {
			http.Error(w, "client_id and redirect_uri are required", http.StatusBadRequest)
			return
		}

		// Validate client registration
		client := store.getClient(clientID)
		if client == nil {
			http.Error(w, "Unknown client_id", http.StatusBadRequest)
			return
		}

		// Validate redirect URI
		validRedirect := false
		for _, uri := range client.RedirectURIs {
			if uri == redirectURI {
				validRedirect = true
				break
			}
		}
		if !validRedirect {
			http.Error(w, "Invalid redirect_uri", http.StatusBadRequest)
			return
		}

		// Generate PKCE verifier for Bitbucket code exchange (server-side)
		verifier, err := authPkg.GenerateCodeVerifier()
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		bbState, err := authPkg.GenerateState()
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		// Store pending auth request
		store.putPending(bbState, &oauthAuthRequest{
			ClientID:     clientID,
			RedirectURI:  redirectURI,
			State:        state,
			CodeVerifier: verifier,
			BBState:      bbState,
			CodeChallenge: codeChallenge,
			CodeChallengeMethod: codeChallengeMethod,
		})

		// Redirect to Bitbucket's authorization page
		bbAuthURL := fmt.Sprintf("%s?client_id=%s&response_type=code&redirect_uri=%s&state=%s",
			config.AuthURL,
			url.QueryEscape(store.clientID),
			url.QueryEscape(store.callbackURL),
			url.QueryEscape(bbState),
		)

		http.Redirect(w, r, bbAuthURL, http.StatusFound)
	}
}

func oauthBitbucketCallbackHandler(store *mcpSessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		q := r.URL.Query()
		code := q.Get("code")
		bbState := q.Get("state")

		if code == "" || bbState == "" {
			errMsg := q.Get("error_description")
			if errMsg == "" {
				errMsg = q.Get("error")
			}
			if errMsg == "" {
				errMsg = "no authorization code received"
			}
			http.Error(w, fmt.Sprintf("Authorization failed: %s", errMsg), http.StatusBadRequest)
			return
		}

		// Look up pending auth request
		pending := store.popPending(bbState)
		if pending == nil {
			http.Error(w, "Unknown or expired authorization request", http.StatusBadRequest)
			return
		}

		// Exchange code with Bitbucket using the same callback URI
		bbToken, err := authPkg.ExchangeCodeServerSide(store.clientID, store.clientSecret, code, store.callbackURL)
		if err != nil {
			http.Error(w, fmt.Sprintf("Token exchange failed: %v", err), http.StatusBadGateway)
			return
		}

		// Generate session bearer token
		sessionToken, err := generateToken()
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		sess := &mcpSession{
			BearerToken: sessionToken,
			AccessToken: bbToken.AccessToken,
			RefreshToken: bbToken.RefreshToken,
			ClientID:    pending.ClientID,
		}
		if bbToken.ExpiresIn > 0 {
			sess.TokenExpiresAt = time.Now().Unix() + int64(bbToken.ExpiresIn)
		}

		// Generate our own auth code to return to the MCP client
		ourCode, err := generateToken()
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		store.putCode(ourCode, &oauthAuthCode{
			Code:        ourCode,
			ClientID:    pending.ClientID,
			RedirectURI: pending.RedirectURI,
			Session:     sess,
			CodeChallenge: pending.CodeChallenge,
			CodeChallengeMethod: pending.CodeChallengeMethod,
		})

		// Redirect back to the MCP client with our auth code
		redirectURL := pending.RedirectURI + "?code=" + url.QueryEscape(ourCode)
		if pending.State != "" {
			redirectURL += "&state=" + url.QueryEscape(pending.State)
		}

		http.Redirect(w, r, redirectURL, http.StatusFound)
	}
}

func oauthTokenHandler(store *mcpSessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		grantType := r.FormValue("grant_type")

		switch grantType {
		case "authorization_code":
			handleAuthorizationCodeGrant(w, r, store)
		case "refresh_token":
			handleRefreshTokenGrant(w, r, store)
		default:
			jsonError(w, "unsupported_grant_type", "Supported: authorization_code, refresh_token", http.StatusBadRequest)
		}
	}
}

// handleAuthorizationCodeGrant handles the authorization_code grant type.
func handleAuthorizationCodeGrant(w http.ResponseWriter, r *http.Request, store *mcpSessionStore) {
	code := r.FormValue("code")
	clientID := r.FormValue("client_id")
	redirectURI := r.FormValue("redirect_uri")
	codeVerifier := r.FormValue("code_verifier")

	ac := store.popCode(code)
	if ac == nil {
		jsonError(w, "invalid_grant", "Invalid or expired authorization code", http.StatusBadRequest)
		return
	}

	if ac.ClientID != clientID {
		jsonError(w, "invalid_grant", "client_id mismatch", http.StatusBadRequest)
		return
	}

	if ac.RedirectURI != redirectURI {
		jsonError(w, "invalid_grant", "redirect_uri mismatch", http.StatusBadRequest)
		return
	}

	if ac.CodeChallenge != "" && ac.CodeChallengeMethod == "S256" {
		expected := authPkg.GenerateCodeChallenge(codeVerifier)
		if subtle.ConstantTimeCompare([]byte(expected), []byte(ac.CodeChallenge)) != 1 {
			jsonError(w, "invalid_grant", "PKCE verification failed", http.StatusBadRequest)
			return
		}
	}

	// Generate a refresh token for the MCP session
	mcpRefreshToken, err := generateToken()
	if err != nil {
		jsonError(w, "server_error", "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}
	ac.Session.MCPRefreshToken = mcpRefreshToken

	store.putSession(ac.Session)

	fmt.Fprintf(os.Stderr, "New OAuth session created for client %s\n", clientID)

	resp := map[string]interface{}{
		"access_token":  ac.Session.BearerToken,
		"token_type":    "bearer",
		"expires_in":    defaultTokenExpiry,
		"refresh_token": mcpRefreshToken,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleRefreshTokenGrant handles the refresh_token grant type.
// This refreshes the MCP session by getting a new Bitbucket token.
func handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request, store *mcpSessionStore) {
	mcpRefreshToken := r.FormValue("refresh_token")
	if mcpRefreshToken == "" {
		jsonError(w, "invalid_request", "refresh_token is required", http.StatusBadRequest)
		return
	}

	// Find session by MCP refresh token
	var sess *mcpSession
	store.mu.RLock()
	for _, s := range store.sessions {
		if s.MCPRefreshToken == mcpRefreshToken {
			sess = s
			break
		}
	}
	store.mu.RUnlock()

	if sess == nil {
		jsonError(w, "invalid_grant", "Invalid refresh token", http.StatusBadRequest)
		return
	}

	// Refresh the Bitbucket token
	if sess.RefreshToken == "" {
		jsonError(w, "invalid_grant", "No Bitbucket refresh token available", http.StatusBadRequest)
		return
	}

	newBBToken, err := authPkg.RefreshAccessToken(store.clientID, store.clientSecret, sess.RefreshToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Bitbucket token refresh failed during MCP refresh: %v\n", err)
		jsonError(w, "invalid_grant", "Bitbucket token refresh failed — re-authorization required", http.StatusBadRequest)
		return
	}

	// Generate new session bearer token
	newBearer, err := generateToken()
	if err != nil {
		jsonError(w, "server_error", "Failed to generate token", http.StatusInternalServerError)
		return
	}

	newMCPRefresh, err := generateToken()
	if err != nil {
		jsonError(w, "server_error", "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	// Remove old session, create new one
	store.mu.Lock()
	delete(store.sessions, sess.BearerToken)
	store.mu.Unlock()

	newSess := &mcpSession{
		BearerToken:     newBearer,
		AccessToken:     newBBToken.AccessToken,
		RefreshToken:    newBBToken.RefreshToken,
		MCPRefreshToken: newMCPRefresh,
		ClientID:        sess.ClientID,
		Username:        sess.Username,
	}
	if newBBToken.ExpiresIn > 0 {
		newSess.TokenExpiresAt = time.Now().Unix() + int64(newBBToken.ExpiresIn)
	}
	// Preserve refresh token if Bitbucket didn't return a new one
	if newSess.RefreshToken == "" {
		newSess.RefreshToken = sess.RefreshToken
	}

	store.putSession(newSess)

	fmt.Fprintf(os.Stderr, "MCP session refreshed for client %s\n", sess.ClientID)

	resp := map[string]interface{}{
		"access_token":  newBearer,
		"token_type":    "bearer",
		"expires_in":    defaultTokenExpiry,
		"refresh_token": newMCPRefresh,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func jsonError(w http.ResponseWriter, errorCode, description string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":             errorCode,
		"error_description": description,
	})
}

// serveHTTPOAuth starts the MCP server over HTTP with per-user OAuth authentication.
func serveHTTPOAuth(host string, port int, basePath, bbClientID, bbClientSecret, externalURL string) error {
	// Determine session persistence path
	var sessionsPath string
	dir, err := config.ConfigDir()
	if err == nil {
		sessionsPath = filepath.Join(dir, "sessions.json")
	}

	store := newSessionStore(bbClientID, bbClientSecret, sessionsPath, "")

	// Per-user server factory: each request gets a server with the user's Bitbucket token
	handler := mcpPkg.NewHTTPHandler(func(r *http.Request) *mcpPkg.Server {
		hdr := r.Header.Get("Authorization")
		if !strings.HasPrefix(hdr, "Bearer ") {
			return nil
		}
		token := strings.TrimPrefix(hdr, "Bearer ")
		sess := store.getSession(token)
		if sess == nil {
			return nil
		}
		return createMCPServerWithToken(sess.AccessToken)
	}, &mcpPkg.HTTPHandlerOptions{})

	addr := fmt.Sprintf("%s:%d", host, port)
	baseURL := strings.TrimRight(externalURL, "/")
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://%s", addr)
	}

	// Determine callback URL:
	// - If external-url is set → use {external-url}/callback (works for remote servers)
	// - Otherwise → use localhost:8817/callback (matches bb auth login consumer config)
	callbackURL := fmt.Sprintf("http://localhost:%d/callback", authPkg.OAuthCallbackPort)
	isRemote := externalURL != "" && !strings.Contains(externalURL, "localhost") && !strings.Contains(externalURL, "127.0.0.1")
	if isRemote {
		callbackURL = baseURL + "/callback"
	}

	store.callbackURL = callbackURL

	mux := http.NewServeMux()

	// OAuth discovery endpoints
	mux.HandleFunc("GET /.well-known/oauth-authorization-server", oauthMetadataHandler(baseURL))
	mux.HandleFunc("GET /.well-known/oauth-protected-resource", oauthProtectedResourceHandler(baseURL, basePath))

	// OAuth flow endpoints on main server
	mux.HandleFunc("POST /oauth/register", oauthRegisterHandler(store))
	mux.HandleFunc("GET /oauth/authorize", oauthAuthorizeHandler(store))
	mux.HandleFunc("POST /oauth/token", oauthTokenHandler(store))

	// Callback handler on main server (for remote deployments)
	mux.HandleFunc("GET /callback", oauthBitbucketCallbackHandler(store))

	// MCP endpoint with session auth
	mux.Handle(basePath, store.sessionBearerAuth(handler))

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Also start callback on port 8817 for localhost compatibility with existing
	// Bitbucket consumer configs (matches bb auth login).
	var callbackServer *http.Server
	if !isRemote {
		callbackMux := http.NewServeMux()
		callbackMux.HandleFunc("GET /callback", oauthBitbucketCallbackHandler(store))
		callbackAddr := fmt.Sprintf("0.0.0.0:%d", authPkg.OAuthCallbackPort)
		callbackServer = &http.Server{
			Addr:              callbackAddr,
			Handler:           callbackMux,
			ReadHeaderTimeout: 10 * time.Second,
		}
	}

	fmt.Fprintf(os.Stderr, "bb MCP server (OAuth mode) running on %s%s\n", baseURL, basePath)
	fmt.Fprintf(os.Stderr, "OAuth metadata: %s/.well-known/oauth-authorization-server\n", baseURL)
	fmt.Fprintf(os.Stderr, "OAuth callback: %s\n", callbackURL)
	fmt.Fprintf(os.Stderr, "Bitbucket consumer: %s\n", bbClientID)
	fmt.Fprintf(os.Stderr, "bb-mcp version: %s\n", buildinfo.Version)
	if isRemote {
		fmt.Fprintf(os.Stderr, "NOTE: Update your Bitbucket OAuth consumer callback URL to: %s\n", callbackURL)
	}

	// Graceful shutdown
	ctx, stop := signalContext()
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- httpServer.ListenAndServe()
	}()
	if callbackServer != nil {
		go func() {
			if err := callbackServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				fmt.Fprintf(os.Stderr, "OAuth callback server error: %v\n", err)
			}
		}()
	}

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("HTTP server error: %w", err)
		}
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, "\nShutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if callbackServer != nil {
			callbackServer.Shutdown(shutdownCtx)
		}
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown error: %w", err)
		}
	}

	return nil
}

// createMCPServerWithToken creates an MCP server whose context carries the given
// Bitbucket access token, so tool handlers use it for API calls.
func createMCPServerWithToken(accessToken string) *mcpPkg.Server {
	server := mcpPkg.NewServer(
		"bb-mcp",
		buildinfo.Version,
		"Bitbucket CLI MCP server - exposes bb commands as MCP tools",
	)

	registry := mcpPkg.NewToolRegistry()
	if err := mcpPkg.RegisterDefaultTools(registry); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to register tools: %v\n", err)
		return server
	}

	server.SetRegistry(registry)
	mcpPkg.RegisterDefaultResources(server)
	mcpPkg.RegisterDefaultPrompts(server)

	// Inject the Bitbucket token into the server's context
	server.SetContext(mcpPkg.ContextWithToken(server.Context(), accessToken))

	return server
}
