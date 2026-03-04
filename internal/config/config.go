package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

const (
	AppName       = "bitbucket-cli"
	BitbucketAPI  = "https://api.bitbucket.org/2.0"
	AuthURL       = "https://bitbucket.org/site/oauth2/authorize"
	DefaultFormat = "table"
)

// TokenURL is a variable so it can be overridden in tests.
var TokenURL = "https://bitbucket.org/site/oauth2/access_token"

var (
	once      sync.Once
	cachedDir string
	cachedErr error
)

// ResetConfigDirCache resets the cached config directory. This is used in tests
// to ensure each test gets its own config directory based on its temp HOME.
// Exported so tests in other packages can reset the cache when needed.
func ResetConfigDirCache() {
	once = sync.Once{}
	cachedDir = ""
	cachedErr = nil
}

type Config struct {
	DefaultWorkspace string `json:"default_workspace"`
	DefaultFormat    string `json:"default_format"`
	// OAuth credentials configured by the user
	OAuthKey    string `json:"oauth_key"`
	OAuthSecret string `json:"oauth_secret"`
}

type TokenData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scopes       string `json:"scopes"`
}

func ConfigDir() (string, error) {
	once.Do(func() {
		var base string
		// Check XDG_CONFIG_HOME first (for Linux and tests)
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			base = xdg
		} else {
			// Fall back to os.UserConfigDir() for platform-specific defaults
			var err error
			base, err = os.UserConfigDir()
			if err != nil {
				cachedErr = err
				return
			}
		}
		dir := filepath.Join(base, AppName)
		if err := os.MkdirAll(dir, 0700); err != nil {
			cachedErr = err
			return
		}
		cachedDir = dir
	})
	return cachedDir, cachedErr
}

func LoadConfig() (*Config, error) {
	dir, err := ConfigDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{DefaultFormat: DefaultFormat}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.DefaultFormat == "" {
		cfg.DefaultFormat = DefaultFormat
	}
	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "config.json"), data, 0600)
}

func LoadToken() (*TokenData, error) {
	dir, err := ConfigDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "token.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var token TokenData
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

func SaveToken(token *TokenData) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "token.json"), data, 0600)
}

func ClearToken() error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "token.json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
