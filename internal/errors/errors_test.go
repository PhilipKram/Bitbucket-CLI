package errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestBBError_Error(t *testing.T) {
	tests := []struct {
		name     string
		bbErr    *BBError
		wantMsg  string
		wantSugg bool
		wantErr  bool
	}{
		{
			name: "message only",
			bbErr: &BBError{
				Message: "something went wrong",
			},
			wantMsg:  "something went wrong",
			wantSugg: false,
			wantErr:  false,
		},
		{
			name: "message with suggestion",
			bbErr: &BBError{
				Message:    "authentication failed",
				Suggestion: "run 'bb auth login' to authenticate",
			},
			wantMsg:  "authentication failed",
			wantSugg: true,
			wantErr:  false,
		},
		{
			name: "message with underlying error",
			bbErr: &BBError{
				Message: "network request failed",
				Err:     fmt.Errorf("connection timeout"),
			},
			wantMsg:  "network request failed",
			wantSugg: false,
			wantErr:  true,
		},
		{
			name: "complete error with all fields",
			bbErr: &BBError{
				Message:    "permission denied",
				Suggestion: "contact administrator",
				StatusCode: 403,
				Err:        fmt.Errorf("insufficient permissions"),
			},
			wantMsg:  "permission denied",
			wantSugg: true,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.bbErr.Error()
			if !strings.Contains(got, tt.wantMsg) {
				t.Errorf("Error() message = %q, want to contain %q", got, tt.wantMsg)
			}
			if tt.wantSugg && !strings.Contains(got, "Suggestion:") {
				t.Errorf("Error() = %q, want to contain suggestion", got)
			}
			if tt.wantErr && !strings.Contains(got, "Caused by:") {
				t.Errorf("Error() = %q, want to contain underlying error", got)
			}
		})
	}
}

func TestBBError_Unwrap(t *testing.T) {
	underlyingErr := fmt.Errorf("underlying error")
	bbErr := &BBError{
		Message: "wrapper error",
		Err:     underlyingErr,
	}

	unwrapped := bbErr.Unwrap()
	if unwrapped != underlyingErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}

	// Test with no underlying error
	bbErrNoUnderlying := &BBError{
		Message: "standalone error",
	}
	if bbErrNoUnderlying.Unwrap() != nil {
		t.Errorf("Unwrap() = %v, want nil", bbErrNoUnderlying.Unwrap())
	}
}

func TestNew(t *testing.T) {
	msg := "test error message"
	err := New(msg)

	if err == nil {
		t.Fatal("New() returned nil")
	}
	if err.Message != msg {
		t.Errorf("New() message = %q, want %q", err.Message, msg)
	}
	if err.Suggestion != "" {
		t.Errorf("New() suggestion = %q, want empty", err.Suggestion)
	}
	if err.StatusCode != 0 {
		t.Errorf("New() status code = %d, want 0", err.StatusCode)
	}
	if err.Err != nil {
		t.Errorf("New() underlying error = %v, want nil", err.Err)
	}
}

func TestWrap(t *testing.T) {
	underlyingErr := fmt.Errorf("original error")
	wrapMsg := "wrapped message"

	err := Wrap(underlyingErr, wrapMsg)
	if err == nil {
		t.Fatal("Wrap() returned nil")
	}
	if err.Message != wrapMsg {
		t.Errorf("Wrap() message = %q, want %q", err.Message, wrapMsg)
	}
	if err.Err != underlyingErr {
		t.Errorf("Wrap() underlying error = %v, want %v", err.Err, underlyingErr)
	}

	// Test wrapping nil error
	nilErr := Wrap(nil, "should return nil")
	if nilErr != nil {
		t.Errorf("Wrap(nil) = %v, want nil", nilErr)
	}
}

func TestWithSuggestion(t *testing.T) {
	suggestion := "try running auth login"

	t.Run("add suggestion to BBError", func(t *testing.T) {
		bbErr := &BBError{
			Message: "authentication required",
		}
		result := WithSuggestion(bbErr, suggestion)

		if result.Suggestion != suggestion {
			t.Errorf("WithSuggestion() suggestion = %q, want %q", result.Suggestion, suggestion)
		}
		if result != bbErr {
			t.Error("WithSuggestion() should modify and return the same BBError")
		}
	})

	t.Run("convert standard error to BBError with suggestion", func(t *testing.T) {
		stdErr := fmt.Errorf("standard error")
		result := WithSuggestion(stdErr, suggestion)

		if result.Suggestion != suggestion {
			t.Errorf("WithSuggestion() suggestion = %q, want %q", result.Suggestion, suggestion)
		}
		if result.Message != stdErr.Error() {
			t.Errorf("WithSuggestion() message = %q, want %q", result.Message, stdErr.Error())
		}
		if result.Err != stdErr {
			t.Errorf("WithSuggestion() underlying error = %v, want %v", result.Err, stdErr)
		}
	})
}

func TestNotFound(t *testing.T) {
	resourceType := "Repository"
	identifier := "my-repo"

	err := NotFound(resourceType, identifier)
	if err == nil {
		t.Fatal("NotFound() returned nil")
	}
	if err.StatusCode != 404 {
		t.Errorf("NotFound() status code = %d, want 404", err.StatusCode)
	}
	if !strings.Contains(err.Message, resourceType) {
		t.Errorf("NotFound() message = %q, want to contain %q", err.Message, resourceType)
	}
	if !strings.Contains(err.Message, identifier) {
		t.Errorf("NotFound() message = %q, want to contain %q", err.Message, identifier)
	}
	if err.Suggestion == "" {
		t.Error("NotFound() should have a suggestion")
	}
}

func TestUnauthorized(t *testing.T) {
	t.Run("with custom message", func(t *testing.T) {
		msg := "Invalid token"
		err := Unauthorized(msg)

		if err == nil {
			t.Fatal("Unauthorized() returned nil")
		}
		if err.StatusCode != 401 {
			t.Errorf("Unauthorized() status code = %d, want 401", err.StatusCode)
		}
		if err.Message != msg {
			t.Errorf("Unauthorized() message = %q, want %q", err.Message, msg)
		}
		if err.Suggestion == "" {
			t.Error("Unauthorized() should have a suggestion")
		}
	})

	t.Run("with empty message", func(t *testing.T) {
		err := Unauthorized("")

		if err.Message != "Authentication required" {
			t.Errorf("Unauthorized(\"\") message = %q, want %q", err.Message, "Authentication required")
		}
	})
}

func TestForbidden(t *testing.T) {
	t.Run("with action", func(t *testing.T) {
		action := "delete repository"
		err := Forbidden(action)

		if err == nil {
			t.Fatal("Forbidden() returned nil")
		}
		if err.StatusCode != 403 {
			t.Errorf("Forbidden() status code = %d, want 403", err.StatusCode)
		}
		if !strings.Contains(err.Message, action) {
			t.Errorf("Forbidden() message = %q, want to contain %q", err.Message, action)
		}
		if err.Suggestion == "" {
			t.Error("Forbidden() should have a suggestion")
		}
	})

	t.Run("without action", func(t *testing.T) {
		err := Forbidden("")

		if err.Message != "Permission denied" {
			t.Errorf("Forbidden(\"\") message = %q, want %q", err.Message, "Permission denied")
		}
	})
}

func TestRateLimit(t *testing.T) {
	t.Run("with reset time", func(t *testing.T) {
		resetTime := "2:30 PM EST"
		err := RateLimit(resetTime)

		if err == nil {
			t.Fatal("RateLimit() returned nil")
		}
		if err.StatusCode != 429 {
			t.Errorf("RateLimit() status code = %d, want 429", err.StatusCode)
		}
		if err.Message != "Rate limit exceeded" {
			t.Errorf("RateLimit() message = %q, want %q", err.Message, "Rate limit exceeded")
		}
		if !strings.Contains(err.Suggestion, resetTime) {
			t.Errorf("RateLimit() suggestion = %q, want to contain %q", err.Suggestion, resetTime)
		}
	})

	t.Run("without reset time", func(t *testing.T) {
		err := RateLimit("")

		if err.Suggestion == "" {
			t.Error("RateLimit(\"\") should have a suggestion")
		}
		if !strings.Contains(err.Suggestion, "Wait") {
			t.Errorf("RateLimit(\"\") suggestion should mention waiting, got %q", err.Suggestion)
		}
	})
}

func TestNetworkError(t *testing.T) {
	underlyingErr := fmt.Errorf("connection refused")
	err := NetworkError(underlyingErr)

	if err == nil {
		t.Fatal("NetworkError() returned nil")
	}
	if err.Message != "Network request failed" {
		t.Errorf("NetworkError() message = %q, want %q", err.Message, "Network request failed")
	}
	if err.Err != underlyingErr {
		t.Errorf("NetworkError() underlying error = %v, want %v", err.Err, underlyingErr)
	}
	if err.Suggestion == "" {
		t.Error("NetworkError() should have a suggestion")
	}
	if !strings.Contains(err.Suggestion, "internet connection") {
		t.Errorf("NetworkError() suggestion should mention internet connection, got %q", err.Suggestion)
	}
}

func TestInvalidInput(t *testing.T) {
	field := "repository name"
	reason := "contains invalid characters"

	err := InvalidInput(field, reason)
	if err == nil {
		t.Fatal("InvalidInput() returned nil")
	}
	if !strings.Contains(err.Message, field) {
		t.Errorf("InvalidInput() message = %q, want to contain %q", err.Message, field)
	}
	if !strings.Contains(err.Message, reason) {
		t.Errorf("InvalidInput() message = %q, want to contain %q", err.Message, reason)
	}
	if err.Suggestion == "" {
		t.Error("InvalidInput() should have a suggestion")
	}
}

func TestGitError(t *testing.T) {
	operation := "clone"
	underlyingErr := fmt.Errorf("repository not found")

	err := GitError(operation, underlyingErr)
	if err == nil {
		t.Fatal("GitError() returned nil")
	}
	if !strings.Contains(err.Message, operation) {
		t.Errorf("GitError() message = %q, want to contain %q", err.Message, operation)
	}
	if err.Err != underlyingErr {
		t.Errorf("GitError() underlying error = %v, want %v", err.Err, underlyingErr)
	}
	if err.Suggestion == "" {
		t.Error("GitError() should have a suggestion")
	}
}

func TestConfigError(t *testing.T) {
	msg := "missing required field 'workspace'"

	err := ConfigError(msg)
	if err == nil {
		t.Fatal("ConfigError() returned nil")
	}
	if !strings.Contains(err.Message, msg) {
		t.Errorf("ConfigError() message = %q, want to contain %q", err.Message, msg)
	}
	if err.Suggestion == "" {
		t.Error("ConfigError() should have a suggestion")
	}
	if !strings.Contains(err.Suggestion, "config") {
		t.Errorf("ConfigError() suggestion should mention config, got %q", err.Suggestion)
	}
}

func TestBBError_ErrorsIsAs(t *testing.T) {
	// Test that errors.Is and errors.As work with BBError due to Unwrap
	baseErr := fmt.Errorf("base error")
	wrappedErr := Wrap(baseErr, "wrapped")

	if !errors.Is(wrappedErr, baseErr) {
		t.Error("errors.Is() should work with BBError.Unwrap()")
	}

	var bbErr *BBError
	if !errors.As(wrappedErr, &bbErr) {
		t.Error("errors.As() should work with BBError")
	}
	if bbErr.Message != "wrapped" {
		t.Errorf("errors.As() extracted message = %q, want %q", bbErr.Message, "wrapped")
	}
}
