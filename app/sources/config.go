package sources

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// ProviderType identifies the type of media provider.
type ProviderType string

const ProviderPlex ProviderType = "plex"

// Account represents a user account for a media provider.
type Account struct {
	ID       string       `json:"id"`
	Type     ProviderType `json:"type"`
	Username string       `json:"username"`
	ClientID string       `json:"client_id"`
	Servers  []*Server    `json:"servers"`
}

// Server represents a media server associated with an account.
type Server struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Enabled     bool   `json:"enabled"`
	URL         string `json:"url"`
	AccessToken string `json:"-"`
}

type sourcesConfig struct {
	Accounts []*Account `json:"accounts"`
}

func configPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.Getenv("HOME")
	}
	return filepath.Join(configDir, "scanline", "sources.json")
}

func loadConfig() ([]*Account, error) {
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read sources config %s: %w", path, err)
	}

	var cfg sourcesConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse sources config %s: %w", path, err)
	}
	return cfg.Accounts, nil
}

func saveConfig(accounts []*Account) {
	path := configPath()

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		slog.Error("failed to create config directory", "error", err)
		return
	}

	cfg := sourcesConfig{Accounts: accounts}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		slog.Error("failed to marshal sources config", "error", err)
		return
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		slog.Error("failed to write sources config", "path", path, "error", err)
	}
}
