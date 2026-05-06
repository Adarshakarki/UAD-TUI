package main

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type pkgItem struct {
	id       string
	name     string
	selected bool
}

type packagesLoadedMsg []pkgItem

// retrieve list of installed packages using ADB and enrich with metadata
func fetchPackages(metadata map[string]string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("adb", "shell", "pm", "list", "packages", "-u")
		out, err := cmd.Output()
		if err != nil {
			return packagesLoadedMsg{}
		}

		lines := strings.Split(string(out), "\n")
		var pkgs []pkgItem
		for _, line := range lines {
			id := strings.TrimPrefix(strings.TrimSpace(line), "package:")
			if id == "" {
				continue
			}
			name, ok := metadata[id]
			if !ok {
				name = "Unknown App"
			}
			pkgs = append(pkgs, pkgItem{id: id, name: name})
		}
		return packagesLoadedMsg(pkgs)
	}
}

// Renders the package list view with selection and navigation.
func (m model) packageListView() string {
	const (
		mauve = "#cba6f7"
		text  = "#cdd6f4"
		sub   = "#a6adc8"
		base  = "#1e1e2e"
	)

	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color(mauve)).
		Bold(true).
		Render("SELECT PACKAGES TO UNINSTALL")

		// Build the list of packages.
	var list strings.Builder
	// Calculate list window for scrolling
	start := 0
	if m.cursor > 10 {
		start = m.cursor - 10
	}
	end := start + 15
	if end > len(m.packages) {
		end = len(m.packages)
	}

	for i := start; i < end; i++ {
		p := m.packages[i]
		cursor := "  "
		if m.cursor == i { // Highlight the current cursor position.
			cursor = "> "
		}

		checked := "[ ]"
		if p.selected { // Show if package is selected.
			checked = "[x]"
		}

		lineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(text))
		if m.cursor == i {
			lineStyle = lineStyle.Foreground(lipgloss.Color(mauve)).Bold(true)
		}

		item := fmt.Sprintf("%s%s %-25s %s", cursor, checked, p.name, p.id)
		list.WriteString(lineStyle.Render(item) + "\n")
	}
	// Instructions for user interaction.
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color(sub)).
		PaddingTop(1).
		Render("space: select  •  enter: uninstall  •  q: quit")

	content := lipgloss.JoinVertical(lipgloss.Left, header, "", list.String(), footer)
	// Center the content in the terminal.
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		lipgloss.NewStyle().Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(mauve)).Render(content),
		lipgloss.WithWhitespaceBackground(lipgloss.Color(base)),
	)
}
