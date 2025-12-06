package main

import (
	"os"
	"path/filepath"
)

// findGitDirs finds all immediate subdirectories that contain a .git folder
func findGitDirs(root string) ([]string, error) {
	var gitDirs []string

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirPath := filepath.Join(root, entry.Name())
		gitPath := filepath.Join(dirPath, ".git")

		// Check if .git exists (can be file or directory)
		if _, err := os.Stat(gitPath); err == nil {
			gitDirs = append(gitDirs, dirPath)
		}
	}

	return gitDirs, nil
}
