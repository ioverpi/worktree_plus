package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// selectModel is a bubbletea model for selection UI
type selectModel struct {
	label    string
	items    []string
	cursor   int
	selected int
	quit     bool
}

func (m selectModel) Init() tea.Cmd {
	return nil
}

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.selected = -1
			m.quit = true
			return m, tea.Quit
		case "up", "k":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(m.items) - 1 // Wrap to bottom
			}
		case "down", "j":
			m.cursor++
			if m.cursor >= len(m.items) {
				m.cursor = 0 // Wrap to top
			}
		case "enter", " ":
			m.selected = m.cursor
			m.quit = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m selectModel) View() string {
	s := m.label + "\n\n"

	for i, item := range m.items {
		cursor := "  "
		if m.cursor == i {
			cursor = "> "
		}
		s += cursor + item + "\n"
	}

	s += "\n(j/k or arrows to move, enter to select, q to cancel)\n"
	return s
}

// runSelect runs the selection UI and returns the selected index (-1 if cancelled)
func runSelect(label string, items []string) int {
	m := selectModel{
		label:    label,
		items:    items,
		selected: -1,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error running selection: %v\n", err)
		return -1
	}

	return finalModel.(selectModel).selected
}

// interactiveSelectMapping displays active mappings and lets user choose one to remove
func interactiveSelectMapping(config *Config) (folderName, branchName string, ok bool) {
	// Get only active folders
	var activeFolders []FolderHistory
	for _, f := range getRecentFolders(config) {
		if f.IsActive {
			activeFolders = append(activeFolders, f)
		}
	}

	if len(activeFolders) == 0 {
		fmt.Println("No active worktrees. Nothing to remove.")
		return "", "", false
	}

	// Build items with "folder -> branch" format
	items := make([]string, len(activeFolders)+1)
	for i, f := range activeFolders {
		items[i] = fmt.Sprintf("%s -> %s", f.Name, f.Branch)
	}
	items[len(activeFolders)] = "Cancel"

	idx := runSelect("Select a worktree to remove:", items)

	if idx == -1 || idx == len(activeFolders) {
		fmt.Println("Cancelled.")
		return "", "", false
	}

	selected := activeFolders[idx]
	return selected.Name, selected.Branch, true
}

// formatTimeAgo formats a time as a human-readable "time ago" string
func formatTimeAgo(t time.Time) string {
	diff := time.Since(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2, 2006")
	}
}

// textInputModel is a bubbletea model for text input
type textInputModel struct {
	label    string
	value    string
	done     bool
	canceled bool
}

func (m textInputModel) Init() tea.Cmd {
	return nil
}

func (m textInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.canceled = true
			m.done = true
			return m, tea.Quit
		case "enter":
			m.done = true
			return m, tea.Quit
		case "backspace":
			if len(m.value) > 0 {
				m.value = m.value[:len(m.value)-1]
			}
		default:
			// Only add printable characters
			if len(msg.String()) == 1 {
				m.value += msg.String()
			}
		}
	}
	return m, nil
}

func (m textInputModel) View() string {
	return fmt.Sprintf("%s\n\n> %s_\n\n(enter to confirm, esc to cancel)\n", m.label, m.value)
}

// promptTextInput prompts for text input and returns the value
func promptTextInput(label, defaultValue string) (string, bool) {
	m := textInputModel{
		label: label,
		value: defaultValue,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error running input: %v\n", err)
		return "", false
	}

	result := finalModel.(textInputModel)
	if result.canceled || result.value == "" {
		return "", false
	}
	return result.value, true
}

// selectFolderForBranch lets user choose a folder for the branch
// Returns the folder name and whether user confirmed (vs cancelled)
func selectFolderForBranch(config *Config, branchName string) (string, bool) {
	recentFolders := getRecentFolders(config)

	// Filter to only inactive folders (available for reuse)
	var inactiveFolders []FolderHistory
	for _, f := range recentFolders {
		if !f.IsActive {
			inactiveFolders = append(inactiveFolders, f)
		}
	}

	// If no history, just use branch name
	if len(inactiveFolders) == 0 {
		return branchName, true
	}

	// Build menu items - default option first
	items := make([]string, 0, len(inactiveFolders)+3)
	items = append(items, fmt.Sprintf("Create new: %s", branchName))
	items = append(items, "Enter custom folder name...")

	for _, f := range inactiveFolders {
		status := formatTimeAgo(f.LastUsed)
		items = append(items, fmt.Sprintf("%s (was: %s, %s)", f.Name, f.Branch, status))
	}
	items = append(items, "Cancel")

	idx := runSelect("Select folder for worktrees:", items)

	if idx == -1 || idx == len(items)-1 {
		return "", false // Cancelled
	}

	if idx == 0 {
		return branchName, true // Create new with branch name
	}

	if idx == 1 {
		// Custom folder name input
		return promptTextInput("Enter folder name:", "")
	}

	// Selected a previous folder (offset by 2 for the two options at the top)
	return inactiveFolders[idx-2].Name, true
}
