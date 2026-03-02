package errors

import (
	"fmt"
)

// BBError represents a structured error with user-friendly messages and suggestions.
// It provides context about what went wrong and actionable steps to resolve the issue.
type BBError struct {
	// Message is the human-readable description of what went wrong
	Message string

	// Suggestion provides actionable guidance for resolving the error
	Suggestion string

	// StatusCode is the HTTP status code if this is an API error (0 if not applicable)
	StatusCode int

	// Err is the underlying error that caused this error (can be nil)
	Err error
}

// Error implements the error interface.
// It returns a formatted error message that includes the message, suggestion, and underlying error.
func (e *BBError) Error() string {
	msg := e.Message

	if e.Suggestion != "" {
		msg += fmt.Sprintf("\n\nSuggestion: %s", e.Suggestion)
	}

	if e.Err != nil {
		msg += fmt.Sprintf("\n\nCaused by: %v", e.Err)
	}

	return msg
}

// Unwrap returns the underlying error, allowing errors.Is and errors.As to work correctly.
func (e *BBError) Unwrap() error {
	return e.Err
}

// New creates a new BBError with the given message.
func New(message string) *BBError {
	return &BBError{
		Message: message,
	}
}

// Wrap wraps an existing error with additional context.
func Wrap(err error, message string) *BBError {
	if err == nil {
		return nil
	}

	return &BBError{
		Message: message,
		Err:     err,
	}
}

// WithSuggestion adds a suggestion to an existing BBError or creates a new one.
func WithSuggestion(err error, suggestion string) *BBError {
	if bbErr, ok := err.(*BBError); ok {
		bbErr.Suggestion = suggestion
		return bbErr
	}

	return &BBError{
		Message:    err.Error(),
		Suggestion: suggestion,
		Err:        err,
	}
}

// NotFound creates an error for when a resource is not found (404).
func NotFound(resourceType, identifier string) *BBError {
	return &BBError{
		Message:    fmt.Sprintf("%s not found: %s", resourceType, identifier),
		Suggestion: fmt.Sprintf("Check that the %s exists and you have permission to access it. Verify the workspace and repository names are correct.", resourceType),
		StatusCode: 404,
	}
}

// Unauthorized creates an error for authentication failures (401).
func Unauthorized(message string) *BBError {
	if message == "" {
		message = "Authentication required"
	}

	return &BBError{
		Message:    message,
		Suggestion: "Try running 'bb auth login' to authenticate with Bitbucket.",
		StatusCode: 401,
	}
}

// Forbidden creates an error for authorization failures (403).
func Forbidden(action string) *BBError {
	message := "Permission denied"
	if action != "" {
		message = fmt.Sprintf("Permission denied: %s", action)
	}

	return &BBError{
		Message:    message,
		Suggestion: "Check that your Bitbucket account has the necessary permissions for this resource. You may need to request access from the workspace administrator.",
		StatusCode: 403,
	}
}

// RateLimit creates an error for rate limiting (429).
func RateLimit(resetTime string) *BBError {
	message := "Rate limit exceeded"
	suggestion := "Wait a few moments before trying again."

	if resetTime != "" {
		suggestion = fmt.Sprintf("Rate limit resets at %s. Wait until then before retrying.", resetTime)
	}

	return &BBError{
		Message:    message,
		Suggestion: suggestion,
		StatusCode: 429,
	}
}

// NetworkError creates an error for network-related failures.
func NetworkError(err error) *BBError {
	return &BBError{
		Message:    "Network request failed",
		Suggestion: "Check your internet connection. If the problem persists, try increasing the timeout by setting the BB_HTTP_TIMEOUT environment variable (e.g., BB_HTTP_TIMEOUT=60s).",
		Err:        err,
	}
}

// InvalidInput creates an error for invalid user input.
func InvalidInput(field, reason string) *BBError {
	return &BBError{
		Message:    fmt.Sprintf("Invalid %s: %s", field, reason),
		Suggestion: fmt.Sprintf("Check the format of the %s and try again.", field),
	}
}

// GitError creates an error for git-related failures.
func GitError(operation string, err error) *BBError {
	return &BBError{
		Message:    fmt.Sprintf("Git operation failed: %s", operation),
		Suggestion: "Ensure you are in a git repository and the remote is configured correctly.",
		Err:        err,
	}
}

// ConfigError creates an error for configuration-related failures.
func ConfigError(message string) *BBError {
	return &BBError{
		Message:    fmt.Sprintf("Configuration error: %s", message),
		Suggestion: "Check your configuration file (~/.config/bb/config.yaml) or environment variables.",
	}
}
