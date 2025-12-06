package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Define flags
	dirsFlag := flag.String("dirs", "", "Comma-separated list of directories to create worktrees for. If not set, uses all directories with .git subfolder")
	removeFlag := flag.Bool("remove", false, "Remove worktrees instead of creating them")
	folderFlag := flag.String("folder", "", "Custom folder name for the worktree (defaults to branch name). Mapping is saved for later use.")
	listFlag := flag.Bool("list", false, "List all saved folder-to-branch mappings")

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: worktree_plus [-dirs=dir1,dir2,...] [-folder=name] [-remove] <branch-name>")
		fmt.Fprintln(os.Stderr, "       worktree_plus -remove    (interactive selection)")
		fmt.Fprintln(os.Stderr, "       worktree_plus -list")
		fmt.Fprintln(os.Stderr, "\nFlags must come before the branch name.")
		fmt.Fprintln(os.Stderr, "")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	// Load config
	config, err := loadConfig(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Handle -list flag
	if *listFlag {
		folders := getRecentFolders(config)
		if len(folders) == 0 {
			fmt.Println("No folder history.")
		} else {
			fmt.Println("Folder history (most recent first):")
			for _, f := range folders {
				status := "inactive"
				if f.IsActive {
					status = "active"
				}
				fmt.Printf("  %s -> %s (%s, %s)\n", f.Name, f.Branch, status, formatTimeAgo(f.LastUsed))
			}
		}
		return
	}

	// Get positional arguments
	args := flag.Args()

	var branchName, folderName string

	// Handle interactive remove when no branch name provided
	if *removeFlag && len(args) < 1 {
		var ok bool
		folderName, branchName, ok = interactiveSelectMapping(config)
		if !ok {
			os.Exit(0)
		}
	} else {
		// Branch name is required for non-interactive operations
		if len(args) < 1 {
			flag.Usage()
			os.Exit(1)
		}
		branchName = args[0]

		// Determine folder name
		if *folderFlag != "" {
			// Use specified folder name
			folderName = *folderFlag
		} else if existingFolder, ok := findFolderByBranch(config, branchName); ok {
			// Look up existing active mapping by branch name
			folderName = existingFolder
			fmt.Printf("Using existing mapping: folder '%s' -> branch '%s'\n", folderName, branchName)
		} else if !*removeFlag {
			// When creating without -folder, offer folder selection
			var ok bool
			folderName, ok = selectFolderForBranch(config, branchName)
			if !ok {
				fmt.Println("Cancelled.")
				os.Exit(0)
			}
		} else {
			// Default to branch name as folder name
			folderName = branchName
		}

		// When creating, check for conflicts
		if !*removeFlag {
			// Check if this exact folder+branch is already active
			if isExactMatch(config, folderName, branchName) {
				fmt.Printf("Worktrees for folder '%s' with branch '%s' already exist. Nothing to do.\n", folderName, branchName)
				os.Exit(0)
			}

			// Check if branch is already in use with a different folder
			if conflictFolder := checkBranchConflict(config, folderName, branchName); conflictFolder != "" {
				fmt.Fprintf(os.Stderr, "Error: branch '%s' is already active in folder '%s'\n", branchName, conflictFolder)
				fmt.Fprintf(os.Stderr, "Remove the existing worktrees first with: worktree_plus -remove %s\n", branchName)
				os.Exit(1)
			}

			// Save/update the mapping
			touchFolder(config, folderName, branchName)
			if err := saveConfig(cwd, config); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save config: %v\n", err)
			} else if *folderFlag != "" {
				fmt.Printf("Saved mapping: folder '%s' -> branch '%s'\n", folderName, branchName)
			}
		}
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

	fmt.Printf("Processing %d directories for branch '%s' (folder: '%s')\n", len(targetDirs), branchName, folderName)

	// Process each directory
	for _, dir := range targetDirs {
		if *removeFlag {
			err = removeWorktree(dir, folderName, branchName)
		} else {
			err = createWorktree(dir, folderName, branchName)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", dir, err)
		}
	}

	// Clean up folder directory after all removals
	if *removeFlag {
		folderDir := filepath.Join(filepath.Dir(cwd), folderName)

		// Clean up symlinks and handle remaining files
		if err := cleanupFolderDir(folderDir, cwd); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: error during cleanup: %v\n", err)
		}

		// Try to remove the folder directory if it's now empty
		entries, err := os.ReadDir(folderDir)
		if err == nil && len(entries) == 0 {
			if err := os.Remove(folderDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not remove empty folder directory %s: %v\n", folderDir, err)
			} else {
				fmt.Printf("Removed empty folder directory %s\n", folderDir)
			}
		} else if err == nil && len(entries) > 0 {
			fmt.Printf("Folder directory %s not removed (still contains files)\n", folderDir)
		}

		// Deactivate the folder in config (keeps history)
		deactivateFolder(config, folderName)
		if err := saveConfig(cwd, config); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update config: %v\n", err)
		} else {
			fmt.Printf("Deactivated folder '%s' (kept in history)\n", folderName)
		}
	}

	// Symlink root directory files to folder directory after creating worktrees
	if !*removeFlag {
		folderDir := filepath.Join(filepath.Dir(cwd), folderName)
		if err := symlinkRootFiles(cwd, folderDir, targetDirs); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to symlink some root files: %v\n", err)
		}
	}
}
