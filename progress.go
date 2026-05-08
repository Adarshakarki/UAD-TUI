package main

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type uninstallFinishedMsg []string

// runUninstall executes ADB uninstall commands for all selected packages.
// Results are tagged with "OK" or "FAIL" for downstream rendering.
func runUninstall(pkgs []pkgItem) tea.Cmd {
	return func() tea.Msg {
		var results []string
		for _, p := range pkgs {
			if !p.selected {
				continue
			}
			cmd := exec.Command("adb", "shell", "pm", "uninstall", "--user", "0", p.id)
			err := cmd.Run()
			status := "OK"
			if err != nil {
				status = "FAIL"
			}
			// Encode status in the string for parsing in resultsView
			results = append(results, fmt.Sprintf("%s|%s", status, p.id))
		}
		return uninstallFinishedMsg(results)
	}
}

// processingView shows an animated spinner while the uninstall runs.
func (m model) processingView() string {
	const (
		base  = "#1e1e2e"
		peach = "#fab387"
		sub   = "#a6adc8"
	)

	spinner := lipgloss.NewStyle().
		Foreground(lipgloss.Color(peach)).
		Bold(true).
		Render(m.spinner.View())

	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color(peach)).
		Bold(true).
		Render("UNINSTALLING PACKAGES")

	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color(sub)).
		Render("Please keep your device connected...")

	content := lipgloss.JoinVertical(lipgloss.Center,
		spinner+"  "+title,
		"",
		hint,
	)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		lipgloss.NewStyle().
			Padding(2, 6).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(peach)).
			Render(content),
		lipgloss.WithWhitespaceBackground(lipgloss.Color(base)),
	)
}

// resultsView shows the color-coded uninstall summary.
func (m model) resultsView() string {
	const (
		base    = "#1e1e2e"
		mantle  = "#181825"
		surface = "#313244"
		overlay = "#45475a"
		mauve   = "#cba6f7"
		teal    = "#94e2d5"
		red     = "#f38ba8"
		green   = "#a6e3a1"
		sub     = "#a6adc8"
		text    = "#cdd6f4"
	)

	// Header
	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color(base)).
		Background(lipgloss.Color(mauve)).
		Bold(true).
		Padding(0, 2).
		Render(" UNINSTALLATION SUMMARY ")

	// Parse and render results
	successCount := 0
	failCount := 0

	var rows []string
	for _, r := range m.results {
		parts := strings.SplitN(r, "|", 2)
		if len(parts) != 2 {
			continue
		}
		status, pid := parts[0], parts[1]

		if status == "OK" {
			successCount++
			icon := lipgloss.NewStyle().
				Foreground(lipgloss.Color(teal)).Bold(true).
				Render("✓ OK  ")
			pidStr := lipgloss.NewStyle().
				Foreground(lipgloss.Color(text)).
				Render(pid)
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, icon, pidStr))
		} else {
			failCount++
			icon := lipgloss.NewStyle().
				Foreground(lipgloss.Color(red)).Bold(true).
				Render("✗ FAIL")
			pidStr := lipgloss.NewStyle().
				Foreground(lipgloss.Color(sub)).
				Render(pid)
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, icon, " ", pidStr))
		}
	}

	resultsBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(overlay)).
		Background(lipgloss.Color(mantle)).
		Padding(1, 2).
		MaxHeight(20).
		Render(strings.Join(rows, "\n"))

	// Summary counts
	okBadge := lipgloss.NewStyle().
		Foreground(lipgloss.Color(base)).
		Background(lipgloss.Color(teal)).
		Bold(true).
		Padding(0, 2).
		Render(fmt.Sprintf("✓ %d succeeded", successCount))

	var failBadge string
	if failCount > 0 {
		failBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color(base)).
			Background(lipgloss.Color(red)).
			Bold(true).
			Padding(0, 2).
			MarginLeft(1).
			Render(fmt.Sprintf("✗ %d failed", failCount))
	}

	summaryRow := lipgloss.JoinHorizontal(lipgloss.Top, okBadge, failBadge)

	// Footer
	kStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(mauve)).Bold(true)
	dStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(sub))
	sep := dStyle.Render("  •  ")

	footer := lipgloss.NewStyle().MarginTop(1).Render(
		lipgloss.JoinHorizontal(lipgloss.Top,
			kStyle.Render("b/esc"), dStyle.Render(" back to home"),
			sep,
			kStyle.Render("q"), dStyle.Render(" quit"),
		),
	)

	content := lipgloss.JoinVertical(lipgloss.Center,
		header,
		"",
		resultsBox,
		"",
		summaryRow,
		footer,
	)

	card := lipgloss.NewStyle().
		Padding(1, 3).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(mauve)).
		Background(lipgloss.Color(surface)).
		Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		card,
		lipgloss.WithWhitespaceBackground(lipgloss.Color(base)),
	)
}
