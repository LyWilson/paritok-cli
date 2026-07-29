package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	ConfigFileName     = ".paritok.json"
	EnvAPIKey          = "PARITOK_API_KEY"
	EnvNvidiaAPIKey    = "NVIDIA_API_KEY"
	EnvBaseURL         = "PARITOK_BASE_URL"
	DefaultBaseURL     = "http://127.0.0.1:8080"
)

type Config struct {
	APIKey       string `json:"api_key"`
	NvidiaAPIKey string `json:"nvidia_api_key,omitempty"`
	BaseURL      string `json:"base_url,omitempty"`
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ConfigFileName), nil
}

func Load() (*Config, error) {
	cfg := &Config{}

	if key := os.Getenv(EnvAPIKey); key != "" {
		cfg.APIKey = key
	}
	if key := os.Getenv(EnvNvidiaAPIKey); key != "" {
		cfg.NvidiaAPIKey = key
	}

	cfg.BaseURL = os.Getenv(EnvBaseURL)
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}

	path, err := configPath()
	if err != nil {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var fileCfg Config
	if err := json.Unmarshal(data, &fileCfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if fileCfg.APIKey != "" {
		cfg.APIKey = fileCfg.APIKey
	}
	if fileCfg.NvidiaAPIKey != "" {
		cfg.NvidiaAPIKey = fileCfg.NvidiaAPIKey
	}
	if cfg.BaseURL == DefaultBaseURL && fileCfg.BaseURL != "" {
		cfg.BaseURL = fileCfg.BaseURL
	}

	return cfg, nil
}

func Save(apiKey string) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	cfg := Config{APIKey: apiKey}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

func SaveNvidia(apiKey string) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	cfg := Config{}
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &cfg)
	}
	cfg.NvidiaAPIKey = apiKey
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
