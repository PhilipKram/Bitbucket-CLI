package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDir_UsesUserConfigDir(t *testing.T) {
	// Set up a temp dir as XDG_CONFIG_HOME (Linux) or use the default
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error: %v", err)
	}

	want := filepath.Join(tmpDir, AppName)
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
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

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
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

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
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	token := &TokenData{
		AccessToken:  "access123",
		RefreshToken: "refresh456",
		TokenType:    "bearer",
		ExpiresIn:    7200,
		AuthMethod:   AuthMethodOAuth,
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
	if loaded.AuthMethod != token.AuthMethod {
		t.Errorf("AuthMethod = %q, want %q", loaded.AuthMethod, token.AuthMethod)
	}
}

func TestSaveAndLoadToken_AppPassword(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	token := &TokenData{
		AccessToken: "app-pass-123",
		TokenType:   "basic",
		AuthMethod:  AuthMethodToken,
		Username:    "testuser",
	}

	if err := SaveToken(token); err != nil {
		t.Fatalf("SaveToken() error: %v", err)
	}

	loaded, err := LoadToken()
	if err != nil {
		t.Fatalf("LoadToken() error: %v", err)
	}

	if loaded.Username != "testuser" {
		t.Errorf("Username = %q, want %q", loaded.Username, "testuser")
	}
	if loaded.AuthMethod != AuthMethodToken {
		t.Errorf("AuthMethod = %q, want %q", loaded.AuthMethod, AuthMethodToken)
	}
}

func TestClearToken(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	token := &TokenData{
		AccessToken: "access123",
		AuthMethod:  AuthMethodOAuth,
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
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Should not error when no token file exists
	if err := ClearToken(); err != nil {
		t.Fatalf("ClearToken() should not error when no file: %v", err)
	}
}
