package main

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// uninstall results
type uninstallFinishedMsg []string

// executes the ADB commands for all selected packages
func runUninstall(pkgs []pkgItem) tea.Cmd {
	return func() tea.Msg {
		var results []string
		for _, p := range pkgs {
			if p.selected {
				cmd := exec.Command("adb", "shell", "pm", "uninstall", "--user", "0", p.id)
				err := cmd.Run() // Execute the uninstall command
				status := "Success"
				if err != nil {
					status = "Failed"
				}
				results = append(results, fmt.Sprintf("%-30s %s", p.id, status))
			}
		}
		return uninstallFinishedMsg(results)
	}
}

// display the interim progress screen during uninstallation.
func (m model) processingView() string {
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, // Center the processing message
		lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#fab387")).
			Render("UNINSTALLING SELECTED PACKAGES...\nPlease wait."),
		lipgloss.WithWhitespaceBackground(lipgloss.Color("#1e1e2e")),
	)
}

// display the final status of the uninstallation batch.
func (m model) resultsView() string {
	mauve := "#cba6f7"

	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color(mauve)).
		Bold(true).
		Render("UNINSTALLATION SUMMARY")

	var res strings.Builder
	for _, r := range m.results {
		res.WriteString(r + "\n")
	}

	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6adc8")).
		PaddingTop(1).
		Render("b: back to home  •  q: quit")

	content := lipgloss.JoinVertical(lipgloss.Left, header, "", res.String(), footer)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, // Center the results card.
		lipgloss.NewStyle().Padding(1, 4).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(mauve)).Render(content),
		lipgloss.WithWhitespaceBackground(lipgloss.Color("#1e1e2e")),
	)
}
