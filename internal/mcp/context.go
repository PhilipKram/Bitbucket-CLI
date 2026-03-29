package mcp

import (
	"context"

	"github.com/PhilipKram/bitbucket-cli/internal/api"
)

// contextKey is a private type for context keys in this package.
type contextKey int

const tokenKey contextKey = iota

// ContextWithToken returns a new context carrying the given Bitbucket access token.
func ContextWithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, tokenKey, token)
}

// TokenFromContext extracts the Bitbucket access token from the context, if present.
func TokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(tokenKey).(string)
	return token, ok && token != ""
}

// GetClient returns an API client for the given context. If the context carries
// a per-request Bitbucket token (OAuth mode), a token-based client is returned.
// Otherwise, falls back to the default NewClient (env vars / stored token).
func GetClient(ctx context.Context) (*api.Client, error) {
	if token, ok := TokenFromContext(ctx); ok {
		return api.NewClientFromToken(token), nil
	}
	return api.NewClient()
}
