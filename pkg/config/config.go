package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Remote struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Endpoint  string `json:"endpoint,omitempty"`
}

func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get user home directory: %w", err)
	}
	return filepath.Join(home, ".borg", "config.json"), nil
}

func LoadRemotes() ([]Remote, error) {
	path, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []Remote{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %w", err)
	}

	var remotes []Remote
	err = json.Unmarshal(data, &remotes)
	if err != nil {
		return nil, fmt.Errorf("could not parse config file: %w", err)
	}

	return remotes, nil
}

func SaveRemotes(remotes []Remote) error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(remotes, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal config data: %w", err)
	}

	configDir := filepath.Dir(path)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		err = os.MkdirAll(configDir, 0755)
		if err != nil {
			return fmt.Errorf("could not create config directory: %w", err)
		}
	}

	return os.WriteFile(path, data, 0644)
}
