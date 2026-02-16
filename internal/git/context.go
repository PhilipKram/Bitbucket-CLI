package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// RemoteInfo holds the workspace and repository extracted from a Bitbucket remote URL.
type RemoteInfo struct {
	Workspace string
	Repo      string
}

// GetCurrentBranch returns the name of the current git branch.
// Returns an error if not in a git repository or if the HEAD is detached.
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch (not in a git repository?): %w", err)
	}

	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return "", fmt.Errorf("no current branch detected")
	}
	if branch == "HEAD" {
		return "", fmt.Errorf("HEAD is detached; not on any branch")
	}

	return branch, nil
}

// GetRemoteURL returns the URL of the specified git remote (e.g., "origin").
// Returns an error if the remote does not exist.
func GetRemoteURL(remoteName string) (string, error) {
	if remoteName == "" {
		remoteName = "origin"
	}

	cmd := exec.Command("git", "config", "--get", fmt.Sprintf("remote.%s.url", remoteName))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("remote '%s' not found: %w", remoteName, err)
	}

	url := strings.TrimSpace(string(output))
	if url == "" {
		return "", fmt.Errorf("remote '%s' has no URL configured", remoteName)
	}

	return url, nil
}

// ParseBitbucketRemote extracts workspace and repository name from a Bitbucket remote URL.
// Supports both SSH and HTTPS formats:
//   - SSH: git@bitbucket.org:workspace/repo.git
//   - HTTPS: https://bitbucket.org/workspace/repo.git
//
// Returns an error if the URL is not a valid Bitbucket remote URL.
func ParseBitbucketRemote(url string) (*RemoteInfo, error) {
	if url == "" {
		return nil, fmt.Errorf("remote URL is empty")
	}

	// SSH format: git@bitbucket.org:workspace/repo.git
	sshPattern := regexp.MustCompile(`^git@bitbucket\.org:([^/]+)/(.+?)(?:\.git)?$`)
	if matches := sshPattern.FindStringSubmatch(url); matches != nil {
		return &RemoteInfo{
			Workspace: matches[1],
			Repo:      matches[2],
		}, nil
	}

	// HTTPS format: https://bitbucket.org/workspace/repo.git
	// Also handle http:// for edge cases
	httpsPattern := regexp.MustCompile(`^https?://bitbucket\.org/([^/]+)/(.+?)(?:\.git)?$`)
	if matches := httpsPattern.FindStringSubmatch(url); matches != nil {
		return &RemoteInfo{
			Workspace: matches[1],
			Repo:      matches[2],
		}, nil
	}

	return nil, fmt.Errorf("not a valid Bitbucket remote URL: %s", url)
}

// GetBitbucketContext attempts to auto-detect the Bitbucket workspace, repository,
// and current branch from the git repository in the current directory.
// Returns an error if not in a git repository or if the remote is not a Bitbucket URL.
func GetBitbucketContext(remoteName string) (workspace, repo, branch string, err error) {
	branch, err = GetCurrentBranch()
	if err != nil {
		return "", "", "", err
	}

	url, err := GetRemoteURL(remoteName)
	if err != nil {
		return "", "", "", err
	}

	info, err := ParseBitbucketRemote(url)
	if err != nil {
		return "", "", "", err
	}

	return info.Workspace, info.Repo, branch, nil
}
