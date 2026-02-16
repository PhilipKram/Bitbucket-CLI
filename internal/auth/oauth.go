package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/PhilipKram/bitbucket-cli/internal/config"
)

// Login performs the OAuth 2.0 Authorization Code flow.
// It starts a local HTTP server to receive the callback, opens the browser
// for user authorization, and exchanges the code for tokens.
func Login(clientID, clientSecret string) (*config.TokenData, error) {
	// Find an available port for the callback server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to start local server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)

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
			// HTML-escape user-controlled error messages to prevent injection
			fmt.Fprintf(w, "<html><body><h2>Authentication Failed</h2><p>%s</p><p>You can close this window.</p></body></html>", html.EscapeString(errMsg))
			errCh <- fmt.Errorf("authorization failed: %s", errMsg)
			return
		}
		fmt.Fprint(w, "<html><body><h2>Authentication Successful!</h2><p>You can close this window and return to the terminal.</p></body></html>")
		codeCh <- code
	})

	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	defer server.Shutdown(context.Background())

	authURL := fmt.Sprintf("%s?client_id=%s&response_type=code&redirect_uri=%s",
		config.AuthURL, url.QueryEscape(clientID), url.QueryEscape(redirectURI))

	// Attempt to open the browser automatically; fall back to printing the URL.
	if err := openBrowser(authURL); err != nil {
		fmt.Println("Open this URL in your browser to authenticate:")
		fmt.Println()
		fmt.Println("  " + authURL)
	} else {
		fmt.Println("Opened browser for authentication.")
		fmt.Println("If it didn't open, visit:")
		fmt.Println()
		fmt.Println("  " + authURL)
	}
	fmt.Println()
	fmt.Println("Waiting for authorization...")

	// Wait for the callback
	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return nil, err
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("authorization timed out after 5 minutes")
	}

	// Exchange authorization code for tokens
	return exchangeCode(clientID, clientSecret, code, redirectURI)
}

func exchangeCode(clientID, clientSecret, code, redirectURI string) (*config.TokenData, error) {
	data := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {redirectURI},
	}

	req, err := http.NewRequest("POST", config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var token config.TokenData
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}
	return &token, nil
}

// RefreshAccessToken uses the refresh token to obtain a new access token.
func RefreshAccessToken(clientID, clientSecret, refreshToken string) (*config.TokenData, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}

	req, err := http.NewRequest("POST", config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var token config.TokenData
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}
	return &token, nil
}

// openBrowser attempts to open the given URL in the user's default browser.
func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}
