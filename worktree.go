package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// getWorktreePath calculates the worktree path: ../../<folder>/<dirname>
func getWorktreePath(dir, folderName string) string {
	dirName := filepath.Base(dir)
	parentDir := filepath.Dir(dir)
	grandparentDir := filepath.Dir(parentDir)
	return filepath.Join(grandparentDir, folderName, dirName)
}

// createWorktree creates a worktree for the given directory, folder name, and branch
func createWorktree(dir, folderName, branchName string) error {
	dirName := filepath.Base(dir)
	worktreePath := getWorktreePath(dir, folderName)

	fmt.Printf("\n[%s] Creating worktree at %s\n", dirName, worktreePath)

	// Check if worktree already exists
	if _, err := os.Stat(worktreePath); err == nil {
		fmt.Printf("[%s] Worktree already exists at %s\n", dirName, worktreePath)
		return nil
	}

	// Create parent directory for worktree if needed
	worktreeParent := filepath.Dir(worktreePath)
	if err := os.MkdirAll(worktreeParent, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Determine if branch exists locally, remotely, or needs to be created
	var cmd *exec.Cmd
	if branchExists(dir, branchName) {
		// Branch exists locally, use it
		fmt.Printf("[%s] Using existing local branch '%s'\n", dirName, branchName)
		cmd = exec.Command("git", "worktree", "add", worktreePath, branchName)
	} else if remoteBranchExists(dir, branchName) {
		// Branch exists on remote, track it
		fmt.Printf("[%s] Tracking remote branch '%s'\n", dirName, branchName)
		cmd = exec.Command("git", "worktree", "add", "--track", "-b", branchName, worktreePath, "origin/"+branchName)
	} else {
		// Branch doesn't exist, create it
		fmt.Printf("[%s] Creating new branch '%s'\n", dirName, branchName)
		cmd = exec.Command("git", "worktree", "add", "-b", branchName, worktreePath)
	}

	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git worktree add failed: %w", err)
	}

	fmt.Printf("[%s] Worktree created successfully\n", dirName)

	// Create symlinks for gitignored files/directories
	if err := createIgnoredSymlinks(dir, worktreePath); err != nil {
		fmt.Fprintf(os.Stderr, "[%s] Warning: failed to create some symlinks: %v\n", dirName, err)
	}

	return nil
}

// removeWorktree removes a worktree for the given directory and folder name
func removeWorktree(dir, folderName, branchName string) error {
	dirName := filepath.Base(dir)
	worktreePath := getWorktreePath(dir, folderName)

	fmt.Printf("\n[%s] Removing worktree at %s\n", dirName, worktreePath)

	// Check if worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		fmt.Printf("[%s] Worktree does not exist at %s\n", dirName, worktreePath)
		return nil
	}

	// Remove the worktree
	cmd := exec.Command("git", "worktree", "remove", worktreePath)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Try force remove if normal remove fails
		fmt.Printf("[%s] Normal remove failed, trying force remove...\n", dirName)
		cmd = exec.Command("git", "worktree", "remove", "--force", worktreePath)
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git worktree remove failed: %w", err)
		}
	}

	fmt.Printf("[%s] Worktree removed successfully\n", dirName)
	return nil
}
