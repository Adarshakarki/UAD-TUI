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

// fetchPackages retrieves all installed packages via ADB and enriches them with metadata.
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
				name = id // use package ID as display name when metadata is unavailable
			}
			pkgs = append(pkgs, pkgItem{id: id, name: name})
		}
		return packagesLoadedMsg(pkgs)
	}
}

// packageListView renders the full package browser UI.
func (m model) packageListView() string {
	const (
		base    = "#1e1e2e"
		mantle  = "#181825"
		surface = "#313244"
		overlay = "#45475a"
		mauve   = "#cba6f7"
		teal    = "#94e2d5"
		red     = "#f38ba8"
		green   = "#a6e3a1"
		yellow  = "#f9e2af"
		peach   = "#fab387"
		text    = "#cdd6f4"
		sub     = "#a6adc8"
	)

	filtered := m.filteredPackages()

	// Compute stats from full package list
	total := len(m.packages)
	selectedCount := 0
	systemCount := 0
	userCount := 0
	for _, p := range m.packages {
		if p.selected {
			selectedCount++
		}
		if isSystemPackage(p.id) {
			systemCount++
		} else {
			userCount++
		}
	}

	// Header bar
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(base)).
		Background(lipgloss.Color(mauve)).
		Bold(true).
		Padding(0, 2)

	statsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(sub)).
		PaddingLeft(2)

	var selBadge string
	if selectedCount > 0 {
		selBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color(base)).
			Background(lipgloss.Color(teal)).
			Bold(true).
			Padding(0, 1).
			MarginLeft(2).
			Render(fmt.Sprintf("✓ %d selected", selectedCount))
	} else {
		selBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color(overlay)).
			PaddingLeft(2).
			Render("nothing selected yet")
	}

	headerLine := lipgloss.JoinHorizontal(lipgloss.Top,
		titleStyle.Render(" PACKAGE BROWSER "),
		statsStyle.Render(fmt.Sprintf("%d total  •  %d sys  •  %d user", total, systemCount, userCount)),
		selBadge,
	)

	// Filter tabs
	type tab struct {
		label string
		color string
		count int
	}
	tabs := []tab{
		{"ALL", mauve, total},
		{"SYSTEM", yellow, systemCount},
		{"USER", teal, userCount},
		{"SELECTED", green, selectedCount},
	}

	activeTabStyle := lipgloss.NewStyle().Bold(true).Padding(0, 2)
	inactiveTabStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(sub)).
		Background(lipgloss.Color(surface)).Padding(0, 2)

	var tabParts []string
	for i, t := range tabs {
		label := fmt.Sprintf("%s (%d)", t.label, t.count)
		if i == m.filterMode {
			tabParts = append(tabParts,
				activeTabStyle.
					Foreground(lipgloss.Color(base)).
					Background(lipgloss.Color(t.color)).
					Render(label),
			)
		} else {
			tabParts = append(tabParts, inactiveTabStyle.Render(label))
		}
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabParts...)

	// Search bar
	var searchBar string
	prompt := lipgloss.NewStyle().Foreground(lipgloss.Color(teal)).Bold(true).Render("/")
	if m.searchMode {
		searchBar = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(teal)).
			Background(lipgloss.Color(mantle)).
			Padding(0, 1).
			Render(prompt + " " + m.searchQuery + lipgloss.NewStyle().
				Foreground(lipgloss.Color(teal)).Render("█"))
	} else if m.searchQuery != "" {
		searchBar = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(overlay)).
			Background(lipgloss.Color(mantle)).
			Padding(0, 1).
			Render(prompt + " " + m.searchQuery +
				lipgloss.NewStyle().Foreground(lipgloss.Color(overlay)).Render("  (/ to edit, esc to clear)"))
	} else {
		searchBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color(overlay)).
			PaddingLeft(1).
			Render("/ to search packages...")
	}

	// Package list
	visibleCount := 15
	if m.height > 36 {
		visibleCount = m.height - 22
	}
	if visibleCount > 25 {
		visibleCount = 25
	}
	if visibleCount < 5 {
		visibleCount = 5
	}

	start := 0
	if m.cursor >= visibleCount {
		start = m.cursor - visibleCount + 3
	}
	end := start + visibleCount
	if end > len(filtered) {
		end = len(filtered)
	}
	if start < 0 {
		start = 0
	}

	var listLines []string

	// Top scroll hint
	if start > 0 {
		listLines = append(listLines,
			lipgloss.NewStyle().
				Foreground(lipgloss.Color(overlay)).
				Render(fmt.Sprintf("  ↑ %d more above  ↑", start)),
		)
	}

	for i := start; i < end; i++ {
		p := filtered[i]
		isCursor := m.cursor == i

		// Cursor arrow
		arrow := "   "
		if isCursor {
			arrow = lipgloss.NewStyle().
				Foreground(lipgloss.Color(mauve)).Bold(true).
				Render(" ▶ ")
		}

		// Checkbox
		var chk string
		if p.selected {
			chk = lipgloss.NewStyle().
				Foreground(lipgloss.Color(teal)).Bold(true).
				Render("[✓]")
		} else {
			chk = lipgloss.NewStyle().
				Foreground(lipgloss.Color(overlay)).
				Render("[ ]")
		}

		// SYS / USR badge
		var badge string
		if isSystemPackage(p.id) {
			badge = lipgloss.NewStyle().
				Foreground(lipgloss.Color(yellow)).
				Width(4).
				Render("SYS")
		} else {
			badge = lipgloss.NewStyle().
				Foreground(lipgloss.Color(green)).
				Width(4).
				Render("USR")
		}

		// Truncate long names
		name := p.name
		if len(name) > 26 {
			name = name[:23] + "..."
		}

		// Truncate long IDs
		pid := p.id
		if len(pid) > 42 {
			pid = pid[:39] + "..."
		}

		nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(text)).Width(27)
		idStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(sub)).Width(43)

		if isCursor {
			nameStyle = nameStyle.Foreground(lipgloss.Color(mauve)).Bold(true)
			idStyle = idStyle.Foreground(lipgloss.Color(text))
		}

		line := lipgloss.JoinHorizontal(lipgloss.Top,
			arrow,
			chk, " ",
			badge, " ",
			nameStyle.Render(name),
			idStyle.Render(pid),
		)
		listLines = append(listLines, line)
	}

	// Bottom scroll hint
	remaining := len(filtered) - end
	if remaining > 0 {
		listLines = append(listLines,
			lipgloss.NewStyle().
				Foreground(lipgloss.Color(overlay)).
				Render(fmt.Sprintf("  ↓ %d more below  ↓", remaining)),
		)
	}

	// Empty state
	if len(filtered) == 0 {
		listLines = append(listLines,
			lipgloss.NewStyle().
				Foreground(lipgloss.Color(overlay)).
				Italic(true).
				PaddingLeft(3).
				Render("No packages match the current filter or search."),
		)
	}

	listStr := strings.Join(listLines, "\n")

	// Position indicator + column headers
	colHeader := lipgloss.NewStyle().
		Foreground(lipgloss.Color(overlay)).
		Render("   CK  TYP  NAME                        PACKAGE ID")

	posStr := ""
	if len(filtered) > 0 {
		posStr = lipgloss.NewStyle().
			Foreground(lipgloss.Color(overlay)).
			Render(fmt.Sprintf("[%d / %d]", m.cursor+1, len(filtered)))
	}

	// Footer
	type hint struct{ k, v string }
	hints := []hint{
		{"↑↓ jk", "move"},
		{"space", "select"},
		{"tab", "filter"},
		{"/", "search"},
		{"enter", "uninstall"},
		{"esc", "home"},
		{"q", "quit"},
	}
	kStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(mauve)).Bold(true)
	dStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(sub))
	sep := dStyle.Render("  ·  ")

	var hintParts []string
	for _, h := range hints {
		hintParts = append(hintParts, kStyle.Render(h.k)+" "+dStyle.Render(h.v))
	}
	footer := strings.Join(hintParts, sep)

	// Uninstall prompt when packages are selected
	var uninstallPrompt string
	if selectedCount > 0 {
		uninstallPrompt = lipgloss.NewStyle().
			Foreground(lipgloss.Color(base)).
			Background(lipgloss.Color(red)).
			Bold(true).
			Padding(0, 2).
			MarginTop(1).
			Render(fmt.Sprintf("  ⚠  %d package(s) selected  →  press enter to uninstall", selectedCount))
	}

	// Assemble
	content := lipgloss.JoinVertical(lipgloss.Left,
		headerLine,
		"",
		tabBar,
		"",
		searchBar,
		"",
		colHeader,
		lipgloss.NewStyle().Foreground(lipgloss.Color(overlay)).Render(
			strings.Repeat("─", 82)),
		listStr,
		lipgloss.NewStyle().Foreground(lipgloss.Color(overlay)).Render(
			strings.Repeat("─", 82)),
		posStr,
		"",
		footer,
		uninstallPrompt,
	)

	card := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(mauve)).
		Background(lipgloss.Color(mantle)).
		Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		card,
		lipgloss.WithWhitespaceBackground(lipgloss.Color(base)),
	)
}
