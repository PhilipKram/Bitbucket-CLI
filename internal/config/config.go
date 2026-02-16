package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	AppName       = "bitbucket-cli"
	BitbucketAPI  = "https://api.bitbucket.org/2.0"
	AuthURL       = "https://bitbucket.org/site/oauth2/authorize"
	TokenURL      = "https://bitbucket.org/site/oauth2/access_token"
	DefaultFormat = "table"
)

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
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", AppName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
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
