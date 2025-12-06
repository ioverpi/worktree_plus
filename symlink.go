package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// createSymlink creates a symlink, handling Windows requirements
func createSymlink(source, target string, isDir bool) error {
	// On Windows, creating symlinks may require admin privileges or Developer Mode
	// Try to create symlink first
	err := os.Symlink(source, target)
	if err != nil {
		// On Windows, if symlink fails, we could try junction for directories
		// or hard link for files, but those have limitations
		return fmt.Errorf("symlink failed: %w", err)
	}
	return nil
}

// symlinkRootFiles creates symlinks for files in the root directory to the branch directory
func symlinkRootFiles(rootDir, branchDir string, excludeDirs []string) error {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return err
	}

	// Build set of directories to exclude (the ones that became worktrees)
	excludeSet := make(map[string]bool)
	for _, dir := range excludeDirs {
		excludeSet[filepath.Base(dir)] = true
	}

	fmt.Printf("\nSymlinking root files to %s\n", branchDir)

	var errors []string
	created := 0

	for _, entry := range entries {
		name := entry.Name()

		// Skip excluded directories (git repos that became worktrees)
		if excludeSet[name] {
			continue
		}

		// Skip hidden git-related files/dirs that shouldn't be shared
		if name == ".git" {
			continue
		}

		sourcePath := filepath.Join(rootDir, name)
		targetPath := filepath.Join(branchDir, name)

		// Skip if target already exists
		if _, err := os.Lstat(targetPath); err == nil {
			continue
		}

		// Get absolute path for symlink
		absSourcePath, err := filepath.Abs(sourcePath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: failed to get absolute path: %v", name, err))
			continue
		}

		// Create symlink
		if err := os.Symlink(absSourcePath, targetPath); err != nil {
			errors = append(errors, fmt.Sprintf("%s: symlink failed: %v", name, err))
			continue
		}

		created++
		fmt.Printf("  Linked: %s\n", name)
	}

	fmt.Printf("Created %d symlinks in branch directory\n", created)

	if len(errors) > 0 {
		return fmt.Errorf("some symlinks failed:\n  %s", strings.Join(errors, "\n  "))
	}

	return nil
}

// createIgnoredSymlinks creates symlinks in the worktree for gitignored items from the source
func createIgnoredSymlinks(sourceDir, worktreeDir string) error {
	dirName := filepath.Base(sourceDir)

	ignoredItems, err := getIgnoredItems(sourceDir)
	if err != nil {
		return err
	}

	if len(ignoredItems) == 0 {
		fmt.Printf("[%s] No gitignored items to symlink\n", dirName)
		return nil
	}

	fmt.Printf("[%s] Creating symlinks for %d gitignored items...\n", dirName, len(ignoredItems))

	var errors []string
	var linkedItems []string
	created := 0

	for _, item := range ignoredItems {
		sourcePath := filepath.Join(sourceDir, item)
		targetPath := filepath.Join(worktreeDir, item)

		// Check if source exists
		sourceInfo, err := os.Stat(sourcePath)
		if err != nil {
			continue // Skip if source doesn't exist
		}

		// Check if target already exists
		if _, err := os.Lstat(targetPath); err == nil {
			continue // Skip if target already exists
		}

		// Create parent directories in worktree if needed
		targetParent := filepath.Dir(targetPath)
		if err := os.MkdirAll(targetParent, 0755); err != nil {
			errors = append(errors, fmt.Sprintf("%s: failed to create parent dir: %v", item, err))
			continue
		}

		// Create symlink
		// On Windows, we need to use absolute paths and handle directory symlinks differently
		absSourcePath, err := filepath.Abs(sourcePath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: failed to get absolute path: %v", item, err))
			continue
		}

		if err := createSymlink(absSourcePath, targetPath, sourceInfo.IsDir()); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", item, err))
			continue
		}

		created++
		linkedItems = append(linkedItems, item)
		fmt.Printf("[%s]   Linked: %s\n", dirName, item)
	}

	fmt.Printf("[%s] Created %d symlinks\n", dirName, created)

	// Add linked items to .gitignore (marked assume-unchanged) so git ignores them
	if len(linkedItems) > 0 {
		if err := addToGitExclude(worktreeDir, linkedItems); err != nil {
			fmt.Fprintf(os.Stderr, "[%s] Warning: failed to update .gitignore: %v\n", dirName, err)
		} else {
			fmt.Printf("[%s] Added %d items to .gitignore (marked assume-unchanged)\n", dirName, len(linkedItems))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("some symlinks failed:\n  %s", strings.Join(errors, "\n  "))
	}

	return nil
}
