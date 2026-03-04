package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// setTempHome overrides environment variables so os.UserConfigDir() resolves
// under tmpDir on all platforms (including Linux CI where XDG_CONFIG_HOME
// would otherwise take precedence over HOME).
func setTempHome(t *testing.T, tmpDir string) {
	t.Helper()
	t.Setenv("HOME", tmpDir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmpDir)
	}
	if runtime.GOOS == "linux" {
		t.Setenv("XDG_CONFIG_HOME", "")
	}
}

func TestConfigDir_UsesUserConfigDir(t *testing.T) {
	// Set up a temp dir and configure HOME to ensure os.UserConfigDir() uses it
	tmpDir := t.TempDir()
	setTempHome(t, tmpDir)
	ResetConfigDirCache()

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error: %v", err)
	}

	// Construct expected path based on OS
	var expectedBase string
	switch runtime.GOOS {
	case "darwin":
		expectedBase = filepath.Join(tmpDir, "Library", "Application Support")
	case "windows":
		expectedBase = filepath.Join(tmpDir, "AppData", "Roaming")
	default:
		// Linux and others
		expectedBase = filepath.Join(tmpDir, ".config")
	}

	want := filepath.Join(expectedBase, AppName)
	if dir != want {
		t.Errorf("ConfigDir() = %q, want %q", dir, want)
	}

	// Verify directory was created
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("ConfigDir created directory does not exist: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("ConfigDir path is not a directory")
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	setTempHome(t, tmpDir)
	ResetConfigDirCache()

	cfg := &Config{
		DefaultWorkspace: "myworkspace",
		DefaultFormat:    "json",
		OAuthKey:         "key123",
		OAuthSecret:      "secret456",
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}

	if loaded.DefaultWorkspace != cfg.DefaultWorkspace {
		t.Errorf("DefaultWorkspace = %q, want %q", loaded.DefaultWorkspace, cfg.DefaultWorkspace)
	}
	if loaded.OAuthKey != cfg.OAuthKey {
		t.Errorf("OAuthKey = %q, want %q", loaded.OAuthKey, cfg.OAuthKey)
	}
	if loaded.OAuthSecret != cfg.OAuthSecret {
		t.Errorf("OAuthSecret = %q, want %q", loaded.OAuthSecret, cfg.OAuthSecret)
	}
}

func TestLoadConfig_DefaultsWhenMissing(t *testing.T) {
	tmpDir := t.TempDir()
	setTempHome(t, tmpDir)
	ResetConfigDirCache()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if cfg.DefaultFormat != DefaultFormat {
		t.Errorf("DefaultFormat = %q, want %q", cfg.DefaultFormat, DefaultFormat)
	}
}

func TestSaveAndLoadToken(t *testing.T) {
	tmpDir := t.TempDir()
	setTempHome(t, tmpDir)
	ResetConfigDirCache()

	token := &TokenData{
		AccessToken:  "access123",
		RefreshToken: "refresh456",
		TokenType:    "bearer",
		ExpiresIn:    7200,
	}

	if err := SaveToken(token); err != nil {
		t.Fatalf("SaveToken() error: %v", err)
	}

	loaded, err := LoadToken()
	if err != nil {
		t.Fatalf("LoadToken() error: %v", err)
	}

	if loaded.AccessToken != token.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, token.AccessToken)
	}
	if loaded.RefreshToken != token.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", loaded.RefreshToken, token.RefreshToken)
	}
}

func TestClearToken(t *testing.T) {
	tmpDir := t.TempDir()
	setTempHome(t, tmpDir)
	ResetConfigDirCache()

	token := &TokenData{
		AccessToken: "access123",
	}
	if err := SaveToken(token); err != nil {
		t.Fatalf("SaveToken() error: %v", err)
	}

	if err := ClearToken(); err != nil {
		t.Fatalf("ClearToken() error: %v", err)
	}

	_, err := LoadToken()
	if err == nil {
		t.Error("LoadToken() should return error after ClearToken()")
	}
}

func TestClearToken_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	setTempHome(t, tmpDir)
	ResetConfigDirCache()

	// Should not error when no token file exists
	if err := ClearToken(); err != nil {
		t.Fatalf("ClearToken() should not error when no file: %v", err)
	}
}

func TestConfigDir_Caching(t *testing.T) {
	tmpDir := t.TempDir()
	setTempHome(t, tmpDir)
	ResetConfigDirCache()

	// First call should compute and cache the directory
	dir1, err1 := ConfigDir()
	if err1 != nil {
		t.Fatalf("First ConfigDir() call error: %v", err1)
	}

	// Subsequent calls should return the same cached result
	dir2, err2 := ConfigDir()
	if err2 != nil {
		t.Fatalf("Second ConfigDir() call error: %v", err2)
	}

	dir3, err3 := ConfigDir()
	if err3 != nil {
		t.Fatalf("Third ConfigDir() call error: %v", err3)
	}

	// All calls should return the exact same values
	if dir1 != dir2 || dir1 != dir3 {
		t.Errorf("ConfigDir() returned different values: %q, %q, %q", dir1, dir2, dir3)
	}

	// Verify the directory exists and was created
	info, err := os.Stat(dir1)
	if err != nil {
		t.Fatalf("Cached directory does not exist: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("Cached path is not a directory")
	}
}
