// Copied from https://github.com/iximiuz/labctl/blob/main/internal/config/config.go
// License: https://github.com/iximiuz/labctl?tab=Apache-2.0-1-ov-file#readme
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/goccy/go-yaml"
)

const (
	defaultBaseURL = "https://labs.iximiuz.com"

	defaultAPIBaseURL = defaultBaseURL + "/api"

	defaultSSHIdentityFile = "iximiuz_labs_user"
)

type Config struct {
	mu sync.RWMutex

	FilePath string `yaml:"-"`

	BaseURL string `yaml:"base_url"`

	APIBaseURL string `yaml:"api_base_url"`

	SessionID string `yaml:"session_id"`

	AccessToken string `yaml:"access_token"`
}

func ConfigFilePath(homeDir string) string {
	return filepath.Join(homeDir, ".iximiuz", "labctl", "config.yaml")
}

func Default(homeDir string) *Config {
	configFilePath := ConfigFilePath(homeDir)

	cfg := &Config{
		FilePath:   configFilePath,
		BaseURL:    defaultBaseURL,
		APIBaseURL: defaultAPIBaseURL,
	}

	applyEnvOverrides(cfg)

	return cfg
}

func Load(homeDir string) (*Config, error) {
	path := ConfigFilePath(homeDir)

	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return Default(homeDir), nil
	}
	if err != nil {
		return nil, fmt.Errorf("unable to open config file: %s", err)
	}
	defer file.Close()

	var cfg Config
	if err := yaml.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode config from YAML: %s", err)
	}

	// Migrations
	if cfg.BaseURL == "" {
		cfg.BaseURL = strings.TrimSuffix(cfg.APIBaseURL, "/api")
	}

	applyEnvOverrides(&cfg)

	cfg.FilePath = path

	return &cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if sessionID := os.Getenv("IXIMIUZ_SESSION_ID"); sessionID != "" {
		cfg.SessionID = sessionID
	}
	if accessToken := os.Getenv("IXIMIUZ_ACCESS_TOKEN"); accessToken != "" {
		cfg.AccessToken = accessToken
	}
}
