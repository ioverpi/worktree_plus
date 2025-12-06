package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// FolderInfo holds information about a folder
type FolderInfo struct {
	Branch    string    `json:"branch"`
	LastUsed  time.Time `json:"last_used"`
	IsActive  bool      `json:"is_active"` // true if worktrees currently exist
}

// Config holds folder history with timestamps
type Config struct {
	Folders map[string]*FolderInfo `json:"folders,omitempty"`
}

const configFileName = ".worktree_plus.json"

// loadConfig loads the config file from the given directory
func loadConfig(dir string) (*Config, error) {
	configPath := filepath.Join(dir, configFileName)
	config := &Config{
		Folders: make(map[string]*FolderInfo),
	}

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

	if config.Folders == nil {
		config.Folders = make(map[string]*FolderInfo)
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

// findFolderByBranch looks up a folder name by branch name in active folders
func findFolderByBranch(config *Config, branchName string) (string, bool) {
	for folder, info := range config.Folders {
		if info.IsActive && info.Branch == branchName {
			return folder, true
		}
	}
	return "", false
}

// touchFolder updates the last used time for a folder and marks it active
func touchFolder(config *Config, folderName, branchName string) {
	if info, exists := config.Folders[folderName]; exists {
		info.LastUsed = time.Now()
		info.IsActive = true
		info.Branch = branchName
	} else {
		config.Folders[folderName] = &FolderInfo{
			Branch:   branchName,
			LastUsed: time.Now(),
			IsActive: true,
		}
	}
}

// deactivateFolder marks a folder as inactive but keeps it in history
func deactivateFolder(config *Config, folderName string) {
	if info, exists := config.Folders[folderName]; exists {
		info.IsActive = false
		info.LastUsed = time.Now()
	}
}

// FolderHistory represents a folder with its history info for display
type FolderHistory struct {
	Name     string
	Branch   string
	LastUsed time.Time
	IsActive bool
}

// checkBranchConflict checks if a branch is already active with a different folder
// Returns the conflicting folder name if there's a conflict, empty string otherwise
func checkBranchConflict(config *Config, folderName, branchName string) string {
	for folder, info := range config.Folders {
		if info.IsActive && info.Branch == branchName && folder != folderName {
			return folder
		}
	}
	return ""
}

// isExactMatch checks if folder+branch exactly matches an active session
func isExactMatch(config *Config, folderName, branchName string) bool {
	if info, exists := config.Folders[folderName]; exists {
		return info.IsActive && info.Branch == branchName
	}
	return false
}

// getRecentFolders returns folders sorted by last used date (most recent first)
func getRecentFolders(config *Config) []FolderHistory {
	var folders []FolderHistory
	for name, info := range config.Folders {
		folders = append(folders, FolderHistory{
			Name:     name,
			Branch:   info.Branch,
			LastUsed: info.LastUsed,
			IsActive: info.IsActive,
		})
	}

	sort.Slice(folders, func(i, j int) bool {
		return folders[i].LastUsed.After(folders[j].LastUsed)
	})

	return folders
}
