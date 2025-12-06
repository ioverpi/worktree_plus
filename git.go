package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// branchExists checks if a branch exists in the repository
func branchExists(repoDir, branchName string) bool {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branchName)
	cmd.Dir = repoDir
	err := cmd.Run()
	return err == nil
}

// remoteBranchExists checks if a remote branch exists
func remoteBranchExists(repoDir, branchName string) bool {
	cmd := exec.Command("git", "ls-remote", "--heads", "origin", branchName)
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(output) > 0
}

// getIgnoredItems returns a list of gitignored files and directories in the repo
func getIgnoredItems(repoDir string) ([]string, error) {
	// Get ignored files that exist on disk
	cmd := exec.Command("git", "ls-files", "--others", "--ignored", "--exclude-standard", "--directory")
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files failed: %w", err)
	}

	var items []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			// Remove trailing slash from directories
			line = strings.TrimSuffix(line, "/")
			items = append(items, line)
		}
	}

	return items, nil
}

// addToGitExclude adds items to the worktree's .gitignore and marks it as assume-unchanged
func addToGitExclude(worktreeDir string, items []string) error {
	gitignorePath := filepath.Join(worktreeDir, ".gitignore")

	// Read existing .gitignore if it exists
	var existingContent string
	if data, err := os.ReadFile(gitignorePath); err == nil {
		existingContent = string(data)
	}

	// Build set of existing patterns to avoid duplicates
	existingPatterns := make(map[string]bool)
	for _, line := range strings.Split(existingContent, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			existingPatterns[line] = true
		}
	}

	// Determine new items to add
	var newItems []string
	for _, item := range items {
		// Use forward slashes and anchor to root
		pattern := "/" + strings.ReplaceAll(item, "\\", "/")
		if !existingPatterns[pattern] {
			newItems = append(newItems, pattern)
		}
	}

	if len(newItems) == 0 {
		return nil // Nothing new to add
	}

	// Append to .gitignore
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("cannot open .gitignore: %w", err)
	}

	// Add a header comment if this is a new section
	if !strings.Contains(existingContent, "# worktree_plus symlinks") {
		if _, err := f.WriteString("\n# worktree_plus symlinks\n"); err != nil {
			f.Close()
			return err
		}
	}

	for _, item := range newItems {
		if _, err := f.WriteString(item + "\n"); err != nil {
			f.Close()
			return err
		}
	}
	f.Close()

	// Mark .gitignore as assume-unchanged so git ignores our modifications
	cmd := exec.Command("git", "update-index", "--assume-unchanged", "--", ".gitignore")
	cmd.Dir = worktreeDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set assume-unchanged on .gitignore: %w", err)
	}

	return nil
}
