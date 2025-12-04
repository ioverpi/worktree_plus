package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	// Define flags
	dirsFlag := flag.String("dirs", "", "Comma-separated list of directories to create worktrees for. If not set, uses all directories with .git subfolder")
	removeFlag := flag.Bool("remove", false, "Remove worktrees instead of creating them")

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: worktree_plus [-dirs=dir1,dir2,...] [-remove] <branch-name>")
		fmt.Fprintln(os.Stderr, "\nFlags must come before the branch name.")
		fmt.Fprintln(os.Stderr, "")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Get positional arguments (branch name is required)
	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}
	branchName := args[0]

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	// Determine which directories to process
	var targetDirs []string
	if *dirsFlag != "" {
		// Use specified directories
		for _, d := range strings.Split(*dirsFlag, ",") {
			d = strings.TrimSpace(d)
			if d != "" {
				// Make absolute if relative
				if !filepath.IsAbs(d) {
					d = filepath.Join(cwd, d)
				}
				targetDirs = append(targetDirs, d)
			}
		}
	} else {
		// Find all directories with .git subfolder
		targetDirs, err = findGitDirs(cwd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding git directories: %v\n", err)
			os.Exit(1)
		}
	}

	if len(targetDirs) == 0 {
		fmt.Fprintln(os.Stderr, "No directories found to process")
		os.Exit(1)
	}

	fmt.Printf("Processing %d directories for branch '%s'\n", len(targetDirs), branchName)

	// Process each directory
	for _, dir := range targetDirs {
		if *removeFlag {
			err = removeWorktree(dir, branchName)
		} else {
			err = createWorktree(dir, branchName)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", dir, err)
		}
	}

	// Clean up empty branch directory after all removals (e.g., ../cool-feature)
	if *removeFlag {
		branchDir := filepath.Join(filepath.Dir(cwd), branchName)
		entries, err := os.ReadDir(branchDir)
		if err == nil && len(entries) == 0 {
			if err := os.Remove(branchDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not remove empty branch directory %s: %v\n", branchDir, err)
			} else {
				fmt.Printf("\nRemoved empty branch directory %s\n", branchDir)
			}
		}
	}

	// Symlink root directory files to branch directory after creating worktrees
	if !*removeFlag {
		branchDir := filepath.Join(filepath.Dir(cwd), branchName)
		if err := symlinkRootFiles(cwd, branchDir, targetDirs); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to symlink some root files: %v\n", err)
		}
	}
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

// getWorktreePath calculates the worktree path: ../../<branch>/<dirname>
func getWorktreePath(dir, branchName string) string {
	dirName := filepath.Base(dir)
	parentDir := filepath.Dir(dir)
	grandparentDir := filepath.Dir(parentDir)
	return filepath.Join(grandparentDir, branchName, dirName)
}

// createWorktree creates a worktree for the given directory and branch
func createWorktree(dir, branchName string) error {
	dirName := filepath.Base(dir)
	worktreePath := getWorktreePath(dir, branchName)

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

// removeWorktree removes a worktree for the given directory and branch
func removeWorktree(dir, branchName string) error {
	dirName := filepath.Base(dir)
	worktreePath := getWorktreePath(dir, branchName)

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

// addToGitIgnore adds items to the worktree's .gitignore and marks it as assume-unchanged
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
