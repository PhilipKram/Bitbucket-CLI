package mcp

import (
	"fmt"
	"strings"
)

// validateRepoArg validates that a repository argument is in the expected
// workspace/repo-slug format to prevent path injection via MCP tool inputs.
func validateRepoArg(repo string) error {
	parts := strings.SplitN(repo, "/", 3)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid repository format %q: expected workspace/repo-slug", repo)
	}
	return nil
}
