package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PhilipKram/bitbucket-cli/internal/config"
)

const (
	releaseURL   = "https://api.github.com/repos/PhilipKram/Bitbucket-CLI/releases/latest"
	cacheName    = "update_check.json"
	cacheTTL     = 24 * time.Hour
	fetchTimeout = 2 * time.Second
)

// UpdateInfo holds the result of an update check.
type UpdateInfo struct {
	Current string
	Latest  string
}

type cache struct {
	LatestVersion string    `json:"latest_version"`
	CheckedAt     time.Time `json:"checked_at"`
}

// GHRelease represents a GitHub release.
type GHRelease struct {
	TagName string           `json:"tag_name"`
	Assets  []GHReleaseAsset `json:"assets"`
}

// GHReleaseAsset represents a downloadable asset in a GitHub release.
type GHReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// CheckForUpdate checks whether a newer version is available.
// It returns nil if the current version is up-to-date, if the check
// fails, or if the current version is "dev".
func CheckForUpdate(currentVersion string) *UpdateInfo {
	if currentVersion == "dev" || currentVersion == "" {
		return nil
	}

	current := strings.TrimPrefix(currentVersion, "v")

	// Try cached result first.
	if c, err := readCache(); err == nil {
		latest := strings.TrimPrefix(c.LatestVersion, "v")
		if latest != "" && latest != current {
			return &UpdateInfo{Current: current, Latest: latest}
		}
		return nil
	}

	// Fetch from GitHub.
	latest := fetchLatestVersion()
	if latest == "" {
		return nil
	}

	// Write cache regardless of comparison result.
	_ = writeCache(&cache{LatestVersion: latest, CheckedAt: time.Now()})

	latest = strings.TrimPrefix(latest, "v")
	if latest != current {
		return &UpdateInfo{Current: current, Latest: latest}
	}
	return nil
}

func readCache() (*cache, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, cacheName))
	if err != nil {
		return nil, err
	}
	var c cache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	if time.Since(c.CheckedAt) > cacheTTL {
		return nil, os.ErrNotExist // expired
	}
	return &c, nil
}

func writeCache(c *cache) error {
	dir, err := config.ConfigDir()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, cacheName), data, 0600)
}

func fetchLatestVersion() string {
	client := &http.Client{Timeout: fetchTimeout}
	resp, err := client.Get(releaseURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		return ""
	}
	defer resp.Body.Close()

	var rel GHRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return ""
	}
	return rel.TagName
}

// FetchLatestRelease fetches the full latest release info from GitHub,
// including assets. Uses the provided timeout (longer than background check).
func FetchLatestRelease(timeout time.Duration) (*GHRelease, error) {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(releaseURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var rel GHRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}
