package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type AccountConfig struct {
	AuthMethod  string `json:"auth_method"`
	ServiceURL  string `json:"service_url,omitempty"`
	AccountName string `json:"account_name,omitempty"`
	AccountKey  string `json:"account_key,omitempty"`
	Connection  string `json:"connection,omitempty"`
}

type Config struct {
	DefaultAccount string                    `json:"default_account"`
	Accounts       map[string]*AccountConfig `json:"accounts"`
}

func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".azbutil")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config dir: %w", err)
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}
