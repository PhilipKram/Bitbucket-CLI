package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// BitbucketAPIError represents the JSON error response from the Bitbucket API.
// See: https://developer.atlassian.com/cloud/bitbucket/rest/intro/#error-responses
type BitbucketAPIError struct {
	Type    string `json:"type"`
	Error   Error  `json:"error"`
	Message string `json:"message,omitempty"`
}

// Error is the detailed error object in Bitbucket API responses.
type Error struct {
	Message string                 `json:"message"`
	Detail  string                 `json:"detail,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// ParseAPIError parses an HTTP response and creates an appropriate BBError.
// It extracts error details from the Bitbucket API JSON response and rate limit headers.
func ParseAPIError(resp *http.Response, body []byte) *BBError {
	statusCode := resp.StatusCode

	// Try to parse Bitbucket API error JSON
	var apiErr BitbucketAPIError
	var errorMessage string
	if len(body) > 0 {
		if err := json.Unmarshal(body, &apiErr); err == nil {
			// Successfully parsed API error
			if apiErr.Error.Message != "" {
				errorMessage = apiErr.Error.Message
				// Include detail if available
				if apiErr.Error.Detail != "" {
					errorMessage += ": " + apiErr.Error.Detail
				}
			} else if apiErr.Message != "" {
				errorMessage = apiErr.Message
			}
		}
	}

	// Fallback to raw body if we couldn't parse JSON
	if errorMessage == "" && len(body) > 0 {
		errorMessage = string(body)
		// Truncate very long error messages
		if len(errorMessage) > 500 {
			errorMessage = errorMessage[:500] + "..."
		}
	}

	// Create appropriate error based on status code
	switch statusCode {
	case http.StatusUnauthorized: // 401
		err := Unauthorized("Authentication failed")
		if errorMessage != "" {
			err.Message = "Authentication failed: " + errorMessage
		}
		return err

	case http.StatusForbidden: // 403
		action := "access this resource"
		if errorMessage != "" {
			action = errorMessage
		}
		return Forbidden(action)

	case http.StatusNotFound: // 404
		// Try to extract resource type from error message
		resourceType := "Resource"
		identifier := ""
		if errorMessage != "" {
			resourceType = errorMessage
		}
		err := NotFound(resourceType, identifier)
		err.Message = fmt.Sprintf("Not found: %s", errorMessage)
		if errorMessage == "" {
			err.Message = "Resource not found (404)"
		}
		return err

	case http.StatusTooManyRequests: // 429
		resetTime := extractRateLimitReset(resp)
		return RateLimit(resetTime)

	case http.StatusBadRequest: // 400
		message := "Bad request"
		if errorMessage != "" {
			message = errorMessage
		}
		return &BBError{
			Message:    message,
			Suggestion: "Check the request parameters and try again. Ensure all required fields are provided with valid values.",
			StatusCode: statusCode,
		}

	case http.StatusConflict: // 409
		message := "Conflict"
		if errorMessage != "" {
			message = errorMessage
		}
		return &BBError{
			Message:    message,
			Suggestion: "The resource may be in a state that conflicts with this operation. Try refreshing and attempting the operation again.",
			StatusCode: statusCode,
		}

	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable: // 500, 502, 503
		message := "Bitbucket API error"
		if errorMessage != "" {
			message = errorMessage
		}
		return &BBError{
			Message:    message,
			Suggestion: "This is a temporary issue with the Bitbucket API. Wait a few moments and try again.",
			StatusCode: statusCode,
		}

	default:
		// Generic error for other status codes
		message := fmt.Sprintf("API error (HTTP %d)", statusCode)
		if errorMessage != "" {
			message = fmt.Sprintf("API error (HTTP %d): %s", statusCode, errorMessage)
		}
		return &BBError{
			Message:    message,
			Suggestion: "Check the Bitbucket API documentation or try the operation again.",
			StatusCode: statusCode,
		}
	}
}

// extractRateLimitReset extracts the rate limit reset time from response headers.
// It looks for X-RateLimit-Reset header and formats it as a human-readable time.
func extractRateLimitReset(resp *http.Response) string {
	resetHeader := resp.Header.Get("X-RateLimit-Reset")
	if resetHeader == "" {
		return ""
	}

	// Try to parse as Unix timestamp
	var resetTime time.Time
	if timestamp, err := time.Parse(time.RFC3339, resetHeader); err == nil {
		resetTime = timestamp
	} else {
		// Some APIs use Unix epoch seconds
		var epochSecs int64
		if _, err := fmt.Sscanf(resetHeader, "%d", &epochSecs); err == nil {
			resetTime = time.Unix(epochSecs, 0)
		} else {
			// Couldn't parse, return raw value
			return resetHeader
		}
	}

	// Format as human-readable time
	now := time.Now()
	if resetTime.After(now) {
		duration := resetTime.Sub(now)
		if duration < time.Minute {
			return fmt.Sprintf("%d seconds", int(duration.Seconds()))
		} else if duration < time.Hour {
			return fmt.Sprintf("%d minutes", int(duration.Minutes()))
		} else {
			return resetTime.Format("3:04 PM MST")
		}
	}

	return resetTime.Format("3:04 PM MST")
}

// IsNotFound checks if an error is a 404 Not Found error.
func IsNotFound(err error) bool {
	if bbErr, ok := err.(*BBError); ok {
		return bbErr.StatusCode == http.StatusNotFound
	}
	return false
}

// IsUnauthorized checks if an error is a 401 Unauthorized error.
func IsUnauthorized(err error) bool {
	if bbErr, ok := err.(*BBError); ok {
		return bbErr.StatusCode == http.StatusUnauthorized
	}
	return false
}

// IsForbidden checks if an error is a 403 Forbidden error.
func IsForbidden(err error) bool {
	if bbErr, ok := err.(*BBError); ok {
		return bbErr.StatusCode == http.StatusForbidden
	}
	return false
}

// IsRateLimit checks if an error is a 429 Rate Limit error.
func IsRateLimit(err error) bool {
	if bbErr, ok := err.(*BBError); ok {
		return bbErr.StatusCode == http.StatusTooManyRequests
	}
	return false
}
