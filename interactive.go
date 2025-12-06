package main

import (
	"fmt"
	"sort"

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

	// Build items with "folder -> branch" format
	items := make([]string, len(folders)+1)
	for i, folder := range folders {
		items[i] = fmt.Sprintf("%s -> %s", folder, config.Mappings[folder])
	}
	items[len(folders)] = "Cancel"

	idx := runSelect("Select a worktree to remove:", items)

	if idx == -1 || idx == len(folders) {
		fmt.Println("Cancelled.")
		return "", "", false
	}

	selectedFolder := folders[idx]
	return selectedFolder, config.Mappings[selectedFolder], true
}
