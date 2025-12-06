package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the folder name to branch name mappings
type Config struct {
	Mappings map[string]string `json:"mappings"`
}

const configFileName = ".worktree_plus.json"

// loadConfig loads the config file from the given directory
func loadConfig(dir string) (*Config, error) {
	configPath := filepath.Join(dir, configFileName)
	config := &Config{Mappings: make(map[string]string)}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil // Return empty config if file doesn't exist
		}
		return nil, err
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if config.Mappings == nil {
		config.Mappings = make(map[string]string)
	}

	return config, nil
}

// saveConfig saves the config file to the given directory
func saveConfig(dir string, config *Config) error {
	configPath := filepath.Join(dir, configFileName)

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// findFolderByBranch looks up a folder name by branch name in the config
func findFolderByBranch(config *Config, branchName string) (string, bool) {
	for folder, branch := range config.Mappings {
		if branch == branchName {
			return folder, true
		}
	}
	return "", false
}
