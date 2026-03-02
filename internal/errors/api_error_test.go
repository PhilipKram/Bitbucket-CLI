package errors

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseAPIError_Unauthorized(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusUnauthorized,
		Header:     http.Header{},
	}
	body := []byte(`{"type":"error","error":{"message":"Invalid authentication credentials"}}`)

	err := ParseAPIError(resp, body)
	if err == nil {
		t.Fatal("ParseAPIError() returned nil")
	}
	if err.StatusCode != 401 {
		t.Errorf("ParseAPIError() status code = %d, want 401", err.StatusCode)
	}
	if !strings.Contains(err.Message, "Authentication failed") {
		t.Errorf("ParseAPIError() message = %q, want to contain 'Authentication failed'", err.Message)
	}
	if !strings.Contains(err.Suggestion, "bb auth login") {
		t.Errorf("ParseAPIError() suggestion = %q, want to contain 'bb auth login'", err.Suggestion)
	}
}

func TestParseAPIError_Forbidden(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Header:     http.Header{},
	}
	body := []byte(`{"type":"error","error":{"message":"You do not have permission to access this resource"}}`)

	err := ParseAPIError(resp, body)
	if err == nil {
		t.Fatal("ParseAPIError() returned nil")
	}
	if err.StatusCode != 403 {
		t.Errorf("ParseAPIError() status code = %d, want 403", err.StatusCode)
	}
	if !strings.Contains(err.Message, "Permission denied") {
		t.Errorf("ParseAPIError() message = %q, want to contain 'Permission denied'", err.Message)
	}
	if !strings.Contains(err.Suggestion, "permissions") {
		t.Errorf("ParseAPIError() suggestion = %q, want to contain 'permissions'", err.Suggestion)
	}
}

func TestParseAPIError_NotFound(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusNotFound,
		Header:     http.Header{},
	}
	body := []byte(`{"type":"error","error":{"message":"Repository not found"}}`)

	err := ParseAPIError(resp, body)
	if err == nil {
		t.Fatal("ParseAPIError() returned nil")
	}
	if err.StatusCode != 404 {
		t.Errorf("ParseAPIError() status code = %d, want 404", err.StatusCode)
	}
	if !strings.Contains(err.Message, "Not found") {
		t.Errorf("ParseAPIError() message = %q, want to contain 'Not found'", err.Message)
	}
	if err.Suggestion == "" {
		t.Error("ParseAPIError() should have a suggestion for 404")
	}
}

func TestParseAPIError_RateLimit(t *testing.T) {
	resetTime := fmt.Sprintf("%d", time.Now().Add(5*time.Minute).Unix())
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header: http.Header{
			"X-RateLimit-Reset": []string{resetTime},
		},
	}
	body := []byte(`{"type":"error","error":{"message":"Rate limit exceeded"}}`)

	err := ParseAPIError(resp, body)
	if err == nil {
		t.Fatal("ParseAPIError() returned nil")
	}
	if err.StatusCode != 429 {
		t.Errorf("ParseAPIError() status code = %d, want 429", err.StatusCode)
	}
	if !strings.Contains(err.Message, "Rate limit") {
		t.Errorf("ParseAPIError() message = %q, want to contain 'Rate limit'", err.Message)
	}
	if err.Suggestion == "" {
		t.Error("ParseAPIError() should have a suggestion for 429")
	}
}

func TestParseAPIError_BadRequest(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{},
	}
	body := []byte(`{"type":"error","error":{"message":"Invalid branch name","detail":"Branch names cannot contain spaces"}}`)

	err := ParseAPIError(resp, body)
	if err == nil {
		t.Fatal("ParseAPIError() returned nil")
	}
	if err.StatusCode != 400 {
		t.Errorf("ParseAPIError() status code = %d, want 400", err.StatusCode)
	}
	if !strings.Contains(err.Message, "Invalid branch name") {
		t.Errorf("ParseAPIError() message = %q, want to contain error message", err.Message)
	}
	if !strings.Contains(err.Message, "Branch names cannot contain spaces") {
		t.Errorf("ParseAPIError() message = %q, want to contain detail", err.Message)
	}
	if !strings.Contains(err.Suggestion, "request parameters") {
		t.Errorf("ParseAPIError() suggestion = %q, want to contain 'request parameters'", err.Suggestion)
	}
}

func TestParseAPIError_Conflict(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusConflict,
		Header:     http.Header{},
	}
	body := []byte(`{"type":"error","error":{"message":"Resource already exists"}}`)

	err := ParseAPIError(resp, body)
	if err == nil {
		t.Fatal("ParseAPIError() returned nil")
	}
	if err.StatusCode != 409 {
		t.Errorf("ParseAPIError() status code = %d, want 409", err.StatusCode)
	}
	if !strings.Contains(err.Message, "Resource already exists") {
		t.Errorf("ParseAPIError() message = %q, want to contain error message", err.Message)
	}
	if !strings.Contains(err.Suggestion, "conflicts") {
		t.Errorf("ParseAPIError() suggestion = %q, want to contain 'conflicts'", err.Suggestion)
	}
}

func TestParseAPIError_ServerError(t *testing.T) {
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
			resp := &http.Response{
				StatusCode: tc.statusCode,
				Header:     http.Header{},
			}
			body := []byte(`{"type":"error","error":{"message":"Internal server error"}}`)

			err := ParseAPIError(resp, body)
			if err == nil {
				t.Fatal("ParseAPIError() returned nil")
			}
			if err.StatusCode != tc.statusCode {
				t.Errorf("ParseAPIError() status code = %d, want %d", err.StatusCode, tc.statusCode)
			}
			if !strings.Contains(err.Suggestion, "temporary") {
				t.Errorf("ParseAPIError() suggestion = %q, want to contain 'temporary'", err.Suggestion)
			}
		})
	}
}

func TestParseAPIError_GenericError(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusTeapot, // 418 - unusual status code
		Header:     http.Header{},
	}
	body := []byte(`{"type":"error","error":{"message":"I'm a teapot"}}`)

	err := ParseAPIError(resp, body)
	if err == nil {
		t.Fatal("ParseAPIError() returned nil")
	}
	if err.StatusCode != 418 {
		t.Errorf("ParseAPIError() status code = %d, want 418", err.StatusCode)
	}
	if !strings.Contains(err.Message, "API error") {
		t.Errorf("ParseAPIError() message = %q, want to contain 'API error'", err.Message)
	}
	if !strings.Contains(err.Message, "418") {
		t.Errorf("ParseAPIError() message = %q, want to contain status code", err.Message)
	}
}

func TestParseAPIError_EmptyBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusNotFound,
		Header:     http.Header{},
	}
	body := []byte{}

	err := ParseAPIError(resp, body)
	if err == nil {
		t.Fatal("ParseAPIError() returned nil")
	}
	if err.StatusCode != 404 {
		t.Errorf("ParseAPIError() status code = %d, want 404", err.StatusCode)
	}
	// Should still create a valid error even with empty body
	if err.Message == "" {
		t.Error("ParseAPIError() message should not be empty")
	}
}

func TestParseAPIError_InvalidJSON(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{},
	}
	body := []byte(`not valid json`)

	err := ParseAPIError(resp, body)
	if err == nil {
		t.Fatal("ParseAPIError() returned nil")
	}
	if err.StatusCode != 400 {
		t.Errorf("ParseAPIError() status code = %d, want 400", err.StatusCode)
	}
	// Should fall back to raw body content
	if !strings.Contains(err.Message, "not valid json") {
		t.Errorf("ParseAPIError() message = %q, want to contain raw body", err.Message)
	}
}

func TestParseAPIError_LongErrorMessage(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{},
	}
	// Create a body longer than 500 characters
	longMsg := strings.Repeat("a", 600)
	body := []byte(longMsg)

	err := ParseAPIError(resp, body)
	if err == nil {
		t.Fatal("ParseAPIError() returned nil")
	}
	// Message should be truncated to 500 characters + "..."
	if len(err.Message) > 550 {
		t.Errorf("ParseAPIError() message length = %d, want <= 550 (truncated)", len(err.Message))
	}
	if !strings.Contains(err.Message, "...") {
		t.Error("ParseAPIError() message should contain '...' for truncation")
	}
}

func TestParseAPIError_AlternateMessageField(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{},
	}
	// Some API responses use the top-level "message" field
	body := []byte(`{"type":"error","message":"Top-level error message"}`)

	err := ParseAPIError(resp, body)
	if err == nil {
		t.Fatal("ParseAPIError() returned nil")
	}
	if !strings.Contains(err.Message, "Top-level error message") {
		t.Errorf("ParseAPIError() message = %q, want to contain 'Top-level error message'", err.Message)
	}
}

func TestExtractRateLimitReset_ThroughParseAPIError(t *testing.T) {
	t.Run("with Unix timestamp in future", func(t *testing.T) {
		futureTime := time.Now().Add(5 * time.Minute)
		resp := &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header: http.Header{
				"X-RateLimit-Reset": []string{fmt.Sprintf("%d", futureTime.Unix())},
			},
		}
		body := []byte(`{"type":"error","error":{"message":"Rate limit exceeded"}}`)

		err := ParseAPIError(resp, body)
		if err == nil {
			t.Fatal("ParseAPIError() returned nil")
		}
		// The suggestion should contain the formatted reset time
		if err.Suggestion == "" {
			t.Error("ParseAPIError() suggestion should not be empty for rate limit with reset time")
		}
	})

	t.Run("without header", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header:     http.Header{},
		}
		body := []byte(`{"type":"error","error":{"message":"Rate limit exceeded"}}`)

		err := ParseAPIError(resp, body)
		if err == nil {
			t.Fatal("ParseAPIError() returned nil")
		}
		// Should still have a suggestion even without reset time
		if err.Suggestion == "" {
			t.Error("ParseAPIError() suggestion should not be empty for rate limit")
		}
		if !strings.Contains(err.Suggestion, "Wait") {
			t.Errorf("ParseAPIError() suggestion should mention waiting, got %q", err.Suggestion)
		}
	})
}

func TestIsNotFound(t *testing.T) {
	notFoundErr := &BBError{StatusCode: 404}
	if !IsNotFound(notFoundErr) {
		t.Error("IsNotFound() = false, want true for 404 error")
	}

	otherErr := &BBError{StatusCode: 500}
	if IsNotFound(otherErr) {
		t.Error("IsNotFound() = true, want false for non-404 error")
	}

	stdErr := fmt.Errorf("standard error")
	if IsNotFound(stdErr) {
		t.Error("IsNotFound() = true, want false for standard error")
	}
}

func TestIsUnauthorized(t *testing.T) {
	unauthorizedErr := &BBError{StatusCode: 401}
	if !IsUnauthorized(unauthorizedErr) {
		t.Error("IsUnauthorized() = false, want true for 401 error")
	}

	otherErr := &BBError{StatusCode: 403}
	if IsUnauthorized(otherErr) {
		t.Error("IsUnauthorized() = true, want false for non-401 error")
	}

	stdErr := fmt.Errorf("standard error")
	if IsUnauthorized(stdErr) {
		t.Error("IsUnauthorized() = true, want false for standard error")
	}
}

func TestIsForbidden(t *testing.T) {
	forbiddenErr := &BBError{StatusCode: 403}
	if !IsForbidden(forbiddenErr) {
		t.Error("IsForbidden() = false, want true for 403 error")
	}

	otherErr := &BBError{StatusCode: 401}
	if IsForbidden(otherErr) {
		t.Error("IsForbidden() = true, want false for non-403 error")
	}

	stdErr := fmt.Errorf("standard error")
	if IsForbidden(stdErr) {
		t.Error("IsForbidden() = true, want false for standard error")
	}
}

func TestIsRateLimit(t *testing.T) {
	rateLimitErr := &BBError{StatusCode: 429}
	if !IsRateLimit(rateLimitErr) {
		t.Error("IsRateLimit() = false, want true for 429 error")
	}

	otherErr := &BBError{StatusCode: 500}
	if IsRateLimit(otherErr) {
		t.Error("IsRateLimit() = true, want false for non-429 error")
	}

	stdErr := fmt.Errorf("standard error")
	if IsRateLimit(stdErr) {
		t.Error("IsRateLimit() = true, want false for standard error")
	}
}

func TestParseAPIError_Integration(t *testing.T) {
	// Test with a real HTTP response (using httptest)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(5*time.Minute).Unix()))
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"type":"error","error":{"message":"API rate limit exceeded"}}`))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	body = body[:n]

	bbErr := ParseAPIError(resp, body)
	if bbErr == nil {
		t.Fatal("ParseAPIError() returned nil")
	}
	if bbErr.StatusCode != 429 {
		t.Errorf("ParseAPIError() status code = %d, want 429", bbErr.StatusCode)
	}
	if !IsRateLimit(bbErr) {
		t.Error("IsRateLimit() should return true for rate limit error")
	}
}
