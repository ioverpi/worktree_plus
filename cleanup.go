package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// cleanupFolderDir removes symlinks and handles remaining files in the folder directory
func cleanupFolderDir(folderDir, cwd string) error {
	entries, err := os.ReadDir(folderDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	fmt.Printf("\nCleaning up folder directory %s\n", folderDir)

	var symlinks []string
	var regularFiles []string

	for _, entry := range entries {
		path := filepath.Join(folderDir, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			continue
		}

		if info.Mode()&os.ModeSymlink != 0 {
			symlinks = append(symlinks, entry.Name())
		} else {
			regularFiles = append(regularFiles, entry.Name())
		}
	}

	// Remove symlinks
	for _, name := range symlinks {
		path := filepath.Join(folderDir, name)
		if err := os.Remove(path); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not remove symlink %s: %v\n", name, err)
		} else {
			fmt.Printf("  Removed symlink: %s\n", name)
		}
	}

	// Handle remaining regular files
	if len(regularFiles) > 0 {
		fmt.Printf("\nThe following non-symlink files remain in %s:\n", folderDir)
		for _, name := range regularFiles {
			fmt.Printf("  - %s\n", name)
		}

		idx := runSelect("What would you like to do?", []string{
			"Remove them permanently",
			"Move them to current working directory",
			"Do nothing (leave them)",
		})

		switch idx {
		case 0:
			// Remove files permanently
			for _, name := range regularFiles {
				path := filepath.Join(folderDir, name)
				if err := os.RemoveAll(path); err != nil {
					fmt.Fprintf(os.Stderr, "  Warning: could not remove %s: %v\n", name, err)
				} else {
					fmt.Printf("  Removed: %s\n", name)
				}
			}
		case 1:
			// Move files to current working directory
			for _, name := range regularFiles {
				srcPath := filepath.Join(folderDir, name)
				dstPath := filepath.Join(cwd, name)

				// Check if destination already exists
				if _, err := os.Stat(dstPath); err == nil {
					fmt.Fprintf(os.Stderr, "  Warning: %s already exists in %s, skipping\n", name, cwd)
					continue
				}

				if err := os.Rename(srcPath, dstPath); err != nil {
					fmt.Fprintf(os.Stderr, "  Warning: could not move %s: %v\n", name, err)
				} else {
					fmt.Printf("  Moved: %s -> %s\n", name, dstPath)
				}
			}
		case 2, -1:
			fmt.Println("Leaving files as-is.")
		}
	}

	return nil
}
