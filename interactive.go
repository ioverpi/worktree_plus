package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// interactiveSelectMapping displays mappings and lets user choose one to remove
func interactiveSelectMapping(config *Config) (folderName, branchName string, ok bool) {
	if len(config.Mappings) == 0 {
		fmt.Println("No mappings saved. Nothing to remove.")
		return "", "", false
	}

	// Sort folders for consistent display
	folders := make([]string, 0, len(config.Mappings))
	for folder := range config.Mappings {
		folders = append(folders, folder)
	}
	sort.Strings(folders)

	fmt.Println("Select a worktree to remove:")
	for i, folder := range folders {
		fmt.Printf("  [%d] %s -> %s\n", i+1, folder, config.Mappings[folder])
	}
	fmt.Printf("  [0] Cancel\n")
	fmt.Print("\nEnter choice: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		return "", "", false
	}

	input = strings.TrimSpace(input)
	choice, err := strconv.Atoi(input)
	if err != nil || choice < 0 || choice > len(folders) {
		fmt.Println("Invalid choice.")
		return "", "", false
	}

	if choice == 0 {
		fmt.Println("Cancelled.")
		return "", "", false
	}

	selectedFolder := folders[choice-1]
	return selectedFolder, config.Mappings[selectedFolder], true
}
