package cmdutil

import (
	"fmt"
	"net/url"
	"strings"
)

// NormalizeUUID ensures Bitbucket UUIDs are properly formatted for API calls.
// Bitbucket uses UUIDs wrapped in braces (e.g., {abc-def-123}) which must be
// URL-encoded in API paths. This function handles UUIDs in various formats:
//   - {uuid}         -> %7Buuid%7D (add encoding)
//   - %7Buuid%7D     -> %7Buuid%7D (already encoded, pass through)
//   - uuid           -> uuid (no braces, pass through)
//
// This allows users to copy UUIDs directly from the Bitbucket web UI without
// worrying about encoding.
func NormalizeUUID(uuid string) string {
	uuid = strings.TrimSpace(uuid)
	if uuid == "" {
		return uuid
	}

	// Check if already URL-encoded (contains %7B or %7D)
	if strings.Contains(uuid, "%7B") || strings.Contains(uuid, "%7b") ||
		strings.Contains(uuid, "%7D") || strings.Contains(uuid, "%7d") {
		// Already encoded, return as-is
		return uuid
	}

	// Check if it has braces that need encoding
	if strings.HasPrefix(uuid, "{") && strings.HasSuffix(uuid, "}") {
		// Has braces, URL-encode them
		return url.PathEscape(uuid)
	}

	// No braces, return as-is (might be a plain UUID without braces)
	return uuid
}

// ValidateUUID checks if a string looks like a valid Bitbucket UUID.
// Returns an error if the UUID format appears invalid.
func ValidateUUID(uuid string) error {
	uuid = strings.TrimSpace(uuid)
	if uuid == "" {
		return fmt.Errorf("UUID cannot be empty")
	}

	// Decode if URL-encoded
	decoded := uuid
	if strings.Contains(uuid, "%") {
		var err error
		decoded, err = url.PathUnescape(uuid)
		if err != nil {
			return fmt.Errorf("invalid URL-encoded UUID: %w", err)
		}
	}

	// Remove braces if present
	withoutBraces := strings.TrimPrefix(decoded, "{")
	withoutBraces = strings.TrimSuffix(withoutBraces, "}")

	// Basic validation: should have some content and look UUID-ish
	if len(withoutBraces) == 0 {
		return fmt.Errorf("UUID cannot be just braces")
	}

	// Bitbucket UUIDs typically contain hyphens and alphanumeric characters
	// We do a basic sanity check without being too strict
	if !strings.Contains(withoutBraces, "-") {
		// Might be a numeric ID or other identifier, which is also valid
		// Don't fail, just allow it
	}

	return nil
}
