package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestParseBitbucketRemote_SSH(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantWS    string
		wantRepo  string
		wantError bool
	}{
		{
			name:     "SSH with .git suffix",
			url:      "git@bitbucket.org:myworkspace/myrepo.git",
			wantWS:   "myworkspace",
			wantRepo: "myrepo",
		},
		{
			name:     "SSH without .git suffix",
			url:      "git@bitbucket.org:myworkspace/myrepo",
			wantWS:   "myworkspace",
			wantRepo: "myrepo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParseBitbucketRemote(tt.url)
			if tt.wantError {
				if err == nil {
					t.Error("ParseBitbucketRemote() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseBitbucketRemote() error: %v", err)
			}
			if info.Workspace != tt.wantWS {
				t.Errorf("Workspace = %q, want %q", info.Workspace, tt.wantWS)
			}
			if info.Repo != tt.wantRepo {
				t.Errorf("Repo = %q, want %q", info.Repo, tt.wantRepo)
			}
		})
	}
}

func TestParseBitbucketRemote_HTTPS(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantWS    string
		wantRepo  string
		wantError bool
	}{
		{
			name:     "HTTPS with .git suffix",
			url:      "https://bitbucket.org/myworkspace/myrepo.git",
			wantWS:   "myworkspace",
			wantRepo: "myrepo",
		},
		{
			name:     "HTTPS without .git suffix",
			url:      "https://bitbucket.org/myworkspace/myrepo",
			wantWS:   "myworkspace",
			wantRepo: "myrepo",
		},
		{
			name:     "HTTP with .git suffix",
			url:      "http://bitbucket.org/myworkspace/myrepo.git",
			wantWS:   "myworkspace",
			wantRepo: "myrepo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParseBitbucketRemote(tt.url)
			if tt.wantError {
				if err == nil {
					t.Error("ParseBitbucketRemote() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseBitbucketRemote() error: %v", err)
			}
			if info.Workspace != tt.wantWS {
				t.Errorf("Workspace = %q, want %q", info.Workspace, tt.wantWS)
			}
			if info.Repo != tt.wantRepo {
				t.Errorf("Repo = %q, want %q", info.Repo, tt.wantRepo)
			}
		})
	}
}

func TestParseBitbucketRemote_InvalidURLs(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{
			name: "empty URL",
			url:  "",
		},
		{
			name: "GitHub SSH URL",
			url:  "git@github.com:user/repo.git",
		},
		{
			name: "GitHub HTTPS URL",
			url:  "https://github.com/user/repo.git",
		},
		{
			name: "non-git URL",
			url:  "https://example.com",
		},
		{
			name: "malformed URL",
			url:  "not-a-url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBitbucketRemote(tt.url)
			if err == nil {
				t.Errorf("ParseBitbucketRemote(%q) expected error, got nil", tt.url)
			}
		})
	}
}

func TestGetCurrentBranch_InGitRepo(t *testing.T) {
	// Create a temporary git repository
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user (required for commits)
	if err := exec.Command("git", "config", "user.email", "test@example.com").Run(); err != nil {
		t.Fatalf("failed to set git user.email: %v", err)
	}
	if err := exec.Command("git", "config", "user.name", "Test User").Run(); err != nil {
		t.Fatalf("failed to set git user.name: %v", err)
	}

	// Create an initial commit (git needs at least one commit to have a branch)
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	if err := exec.Command("git", "add", "test.txt").Run(); err != nil {
		t.Fatalf("failed to git add: %v", err)
	}
	if err := exec.Command("git", "commit", "-m", "initial commit").Run(); err != nil {
		t.Fatalf("failed to git commit: %v", err)
	}

	branch, err := GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch() error: %v", err)
	}

	// Default branch is typically "master" or "main"
	if branch != "master" && branch != "main" {
		t.Logf("GetCurrentBranch() = %q (expected master or main, but accepting any valid branch)", branch)
	}
	if branch == "" {
		t.Error("GetCurrentBranch() returned empty string")
	}
}

func TestGetCurrentBranch_NotInGitRepo(t *testing.T) {
	// Create a temporary directory that is NOT a git repo
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	_, err := GetCurrentBranch()
	if err == nil {
		t.Error("GetCurrentBranch() expected error when not in git repo, got nil")
	}
}

func TestGetRemoteURL_Success(t *testing.T) {
	// Create a temporary git repository
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Add a remote
	remoteURL := "git@bitbucket.org:testworkspace/testrepo.git"
	if err := exec.Command("git", "remote", "add", "origin", remoteURL).Run(); err != nil {
		t.Fatalf("failed to add remote: %v", err)
	}

	url, err := GetRemoteURL("origin")
	if err != nil {
		t.Fatalf("GetRemoteURL() error: %v", err)
	}
	if url != remoteURL {
		t.Errorf("GetRemoteURL() = %q, want %q", url, remoteURL)
	}
}

func TestGetRemoteURL_DefaultToOrigin(t *testing.T) {
	// Create a temporary git repository
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	remoteURL := "https://bitbucket.org/testworkspace/testrepo.git"
	if err := exec.Command("git", "remote", "add", "origin", remoteURL).Run(); err != nil {
		t.Fatalf("failed to add remote: %v", err)
	}

	// Pass empty string to test default behavior
	url, err := GetRemoteURL("")
	if err != nil {
		t.Fatalf("GetRemoteURL(\"\") error: %v", err)
	}
	if url != remoteURL {
		t.Errorf("GetRemoteURL(\"\") = %q, want %q", url, remoteURL)
	}
}

func TestGetRemoteURL_RemoteNotFound(t *testing.T) {
	// Create a temporary git repository
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	_, err := GetRemoteURL("nonexistent")
	if err == nil {
		t.Error("GetRemoteURL() expected error for nonexistent remote, got nil")
	}
}

func TestGetBitbucketContext_Success(t *testing.T) {
	// Create a temporary git repository
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user
	if err := exec.Command("git", "config", "user.email", "test@example.com").Run(); err != nil {
		t.Fatalf("failed to set git user.email: %v", err)
	}
	if err := exec.Command("git", "config", "user.name", "Test User").Run(); err != nil {
		t.Fatalf("failed to set git user.name: %v", err)
	}

	// Add a Bitbucket remote
	remoteURL := "git@bitbucket.org:myworkspace/myrepo.git"
	if err := exec.Command("git", "remote", "add", "origin", remoteURL).Run(); err != nil {
		t.Fatalf("failed to add remote: %v", err)
	}

	// Create an initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	if err := exec.Command("git", "add", "test.txt").Run(); err != nil {
		t.Fatalf("failed to git add: %v", err)
	}
	if err := exec.Command("git", "commit", "-m", "initial commit").Run(); err != nil {
		t.Fatalf("failed to git commit: %v", err)
	}

	workspace, repo, branch, err := GetBitbucketContext("origin")
	if err != nil {
		t.Fatalf("GetBitbucketContext() error: %v", err)
	}

	if workspace != "myworkspace" {
		t.Errorf("workspace = %q, want %q", workspace, "myworkspace")
	}
	if repo != "myrepo" {
		t.Errorf("repo = %q, want %q", repo, "myrepo")
	}
	if branch == "" {
		t.Error("branch should not be empty")
	}
}

func TestGetBitbucketContext_NotBitbucketRemote(t *testing.T) {
	// Create a temporary git repository
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user
	if err := exec.Command("git", "config", "user.email", "test@example.com").Run(); err != nil {
		t.Fatalf("failed to set git user.email: %v", err)
	}
	if err := exec.Command("git", "config", "user.name", "Test User").Run(); err != nil {
		t.Fatalf("failed to set git user.name: %v", err)
	}

	// Add a GitHub remote (not Bitbucket)
	remoteURL := "git@github.com:user/repo.git"
	if err := exec.Command("git", "remote", "add", "origin", remoteURL).Run(); err != nil {
		t.Fatalf("failed to add remote: %v", err)
	}

	// Create an initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	if err := exec.Command("git", "add", "test.txt").Run(); err != nil {
		t.Fatalf("failed to git add: %v", err)
	}
	if err := exec.Command("git", "commit", "-m", "initial commit").Run(); err != nil {
		t.Fatalf("failed to git commit: %v", err)
	}

	_, _, _, err := GetBitbucketContext("origin")
	if err == nil {
		t.Error("GetBitbucketContext() expected error for non-Bitbucket remote, got nil")
	}
}
