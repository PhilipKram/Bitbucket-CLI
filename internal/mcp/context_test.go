package mcp

import (
	"context"
	"testing"
)

func TestContextWithToken(t *testing.T) {
	ctx := context.Background()
	ctx = ContextWithToken(ctx, "my-token")

	token, ok := TokenFromContext(ctx)
	if !ok {
		t.Fatal("Expected token to be present in context")
	}
	if token != "my-token" {
		t.Errorf("Expected 'my-token', got %s", token)
	}
}

func TestTokenFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	_, ok := TokenFromContext(ctx)
	if ok {
		t.Error("Expected no token in empty context")
	}
}

func TestTokenFromContext_EmptyString(t *testing.T) {
	ctx := context.Background()
	ctx = ContextWithToken(ctx, "")

	_, ok := TokenFromContext(ctx)
	if ok {
		t.Error("Expected empty string token to return false")
	}
}

func TestGetClient_WithToken(t *testing.T) {
	ctx := context.Background()
	ctx = ContextWithToken(ctx, "test-access-token")

	client, err := GetClient(ctx)
	if err != nil {
		t.Fatalf("GetClient failed: %v", err)
	}
	if client == nil {
		t.Fatal("Expected client, got nil")
	}
}
