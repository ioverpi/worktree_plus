# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Project Does

worktree_plus is a CLI tool that creates git worktrees for multiple repositories at once. It's designed for monorepo-like setups where you have multiple git repositories in subdirectories and want to create worktrees for all of them on the same branch simultaneously. It also symlinks gitignored files (like node_modules, build artifacts) from the source to the worktree.

## Build Commands

```bash
go build -o ~/.local/bin/worktree_plus.exe .   # Build to PATH
```

No external dependencies - uses only Go standard library.

## Usage

```bash
worktree_plus <branch-name>                    # Create worktrees for all git repos in cwd
worktree_plus -folder=feature1 <branch-name>  # Use custom folder name
worktree_plus -remove                          # Interactive removal
worktree_plus -remove <branch-name>           # Remove specific worktree
worktree_plus -list                            # List saved folder-branch mappings
worktree_plus -dirs=repo1,repo2 <branch-name> # Only process specific directories
```

## Architecture

The codebase is split by concern:

- **main.go** - Entry point, flag parsing, orchestration flow
- **config.go** - `Config` struct and JSON persistence (`.worktree_plus.json`)
- **worktree.go** - Core `createWorktree`/`removeWorktree` operations
- **git.go** - Git command wrappers (`branchExists`, `remoteBranchExists`, `getIgnoredItems`, `addToGitExclude`)
- **symlink.go** - Symlink creation for root files and gitignored items
- **cleanup.go** - `cleanupFolderDir` with interactive prompts for leftover files
- **interactive.go** - `interactiveSelectMapping` for `-remove` without branch name
- **files.go** - `findGitDirs` to discover git repositories

## Key Behaviors

- Worktrees are created at `../../<folder>/<dirname>` relative to each git repo
- Folder-to-branch mappings are persisted in `.worktree_plus.json`
- Gitignored items are symlinked from source to worktree and added to worktree's `.gitignore` with `assume-unchanged`
- On removal, symlinks are cleaned up and user is prompted about remaining non-symlink files
