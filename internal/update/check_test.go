package update

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/PhilipKram/bitbucket-cli/internal/config"
)

// setTempHome overrides environment variables so config.ConfigDir() resolves
// under tmpDir on all platforms, preventing writes to the real config dir.
func setTempHome(t *testing.T, tmpDir string) {
	t.Helper()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", "")
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmpDir)
	}
}

func TestCheckForUpdate_DevVersion(t *testing.T) {
	// Dev versions should not check for updates
	info := CheckForUpdate("dev")
	if info != nil {
		t.Error("expected nil for dev version")
	}
}

func TestCheckForUpdate_EmptyVersion(t *testing.T) {
	// Empty versions should not check for updates
	info := CheckForUpdate("")
	if info != nil {
		t.Error("expected nil for empty version")
	}
}

func TestCheckForUpdate_SameVersion(t *testing.T) {
	setTempHome(t, t.TempDir())

	// Test with cached same version
	setupTestCache(t, "1.0.0", time.Now())
	defer cleanupTestCache(t)

	info := CheckForUpdate("v1.0.0")
	if info != nil {
		t.Error("expected nil when current version matches latest")
	}
}

func TestCheckForUpdate_NewerVersionAvailable(t *testing.T) {
	setTempHome(t, t.TempDir())

	setupTestCache(t, "v2.0.0", time.Now())
	defer cleanupTestCache(t)

	info := CheckForUpdate("v1.0.0")
	if info == nil {
		t.Fatal("expected UpdateInfo when newer version available")
	}
	if info.Current != "1.0.0" {
		t.Errorf("expected current=1.0.0, got %s", info.Current)
	}
	if info.Latest != "2.0.0" {
		t.Errorf("expected latest=2.0.0, got %s", info.Latest)
	}
}

func TestCheckForUpdate_WithoutVPrefix(t *testing.T) {
	setTempHome(t, t.TempDir())

	setupTestCache(t, "2.0.0", time.Now())
	defer cleanupTestCache(t)

	info := CheckForUpdate("1.0.0")
	if info == nil {
		t.Fatal("expected UpdateInfo when newer version available")
	}
	if info.Current != "1.0.0" {
		t.Errorf("expected current=1.0.0, got %s", info.Current)
	}
	if info.Latest != "2.0.0" {
		t.Errorf("expected latest=2.0.0, got %s", info.Latest)
	}
}

func TestCheckForUpdate_ExpiredCache(t *testing.T) {
	// This test triggers a real network call to GitHub without assertions.
	// Skip until update fetching is refactored to be testable without real network calls.
	t.Skip("skipping until update fetching is refactored to allow mocking the network")
}

func TestReadCache_Valid(t *testing.T) {
	setTempHome(t, t.TempDir())

	setupTestCache(t, "v1.5.0", time.Now())
	defer cleanupTestCache(t)

	c, err := readCache()
	if err != nil {
		t.Fatalf("readCache() error: %v", err)
	}
	if c.LatestVersion != "v1.5.0" {
		t.Errorf("expected latest=v1.5.0, got %s", c.LatestVersion)
	}
}

func TestReadCache_NotExists(t *testing.T) {
	setTempHome(t, t.TempDir())

	_, err := readCache()
	if err == nil {
		t.Error("expected error when cache doesn't exist")
	}
}

func TestReadCache_Expired(t *testing.T) {
	setTempHome(t, t.TempDir())

	setupTestCache(t, "v1.0.0", time.Now().Add(-25*time.Hour))
	defer cleanupTestCache(t)

	_, err := readCache()
	if err == nil {
		t.Error("expected error for expired cache")
	}
	if !os.IsNotExist(err) {
		t.Errorf("expected os.ErrNotExist for expired cache, got %v", err)
	}
}

func TestReadCache_InvalidJSON(t *testing.T) {
	setTempHome(t, t.TempDir())

	dir, err := config.ConfigDir()
	if err != nil {
		t.Fatalf("failed to get config dir: %v", err)
	}
	cachePath := filepath.Join(dir, cacheName)

	// Write invalid JSON
	if err := os.WriteFile(cachePath, []byte("not valid json"), 0600); err != nil {
		t.Fatalf("failed to write invalid cache: %v", err)
	}
	defer os.Remove(cachePath)

	_, err = readCache()
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestWriteCache(t *testing.T) {
	setTempHome(t, t.TempDir())
	defer cleanupTestCache(t)

	c := &cache{
		LatestVersion: "v1.2.3",
		CheckedAt:     time.Now(),
	}

	err := writeCache(c)
	if err != nil {
		t.Fatalf("writeCache() error: %v", err)
	}

	// Verify it was written correctly
	read, err := readCache()
	if err != nil {
		t.Fatalf("readCache() error: %v", err)
	}
	if read.LatestVersion != "v1.2.3" {
		t.Errorf("expected v1.2.3, got %s", read.LatestVersion)
	}
}

func TestFetchLatestVersion_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(GHRelease{TagName: "v1.2.3"})
	}))
	defer server.Close()

	// We can't easily test fetchLatestVersion directly since it uses a hardcoded URL,
	// but we can test the response parsing logic by simulating the HTTP call
	client := &http.Client{Timeout: fetchTimeout}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	defer resp.Body.Close()

	var rel GHRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if rel.TagName != "v1.2.3" {
		t.Errorf("expected v1.2.3, got %s", rel.TagName)
	}
}

func TestFetchLatestVersion_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	// Test error handling
	client := &http.Client{Timeout: fetchTimeout}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Error("expected non-200 status code")
	}
}

func TestFetchLatestVersion_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := &http.Client{Timeout: fetchTimeout}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	defer resp.Body.Close()

	var rel GHRelease
	err = json.NewDecoder(resp.Body).Decode(&rel)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestUpdateInfo_Structure(t *testing.T) {
	info := &UpdateInfo{
		Current: "1.0.0",
		Latest:  "2.0.0",
	}

	if info.Current != "1.0.0" {
		t.Errorf("expected current=1.0.0, got %s", info.Current)
	}
	if info.Latest != "2.0.0" {
		t.Errorf("expected latest=2.0.0, got %s", info.Latest)
	}
}

// Helper functions

func setupTestCache(t *testing.T, version string, checkedAt time.Time) {
	t.Helper()
	c := &cache{
		LatestVersion: version,
		CheckedAt:     checkedAt,
	}
	if err := writeCache(c); err != nil {
		t.Fatalf("failed to setup test cache: %v", err)
	}
}

func cleanupTestCache(t *testing.T) {
	t.Helper()
	dir, err := config.ConfigDir()
	if err != nil {
		return
	}
	cachePath := filepath.Join(dir, cacheName)
	os.Remove(cachePath) // Ignore errors
}
