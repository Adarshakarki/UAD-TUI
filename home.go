package main

import (
	"github.com/charmbracelet/lipgloss"
)

func (m model) homeView() string {
	const (
		base    = "#1e1e2e"
		surface = "#313244"
		overlay = "#45475a"
		text    = "#cdd6f4"
		subtext = "#a6adc8"
		mauve   = "#cba6f7"
		pink    = "#f5c2e7"
		teal    = "#94e2d5"
		peach   = "#fab387"
		green   = "#a6e3a1"
	)

	titleASCII := `
██╗   ██╗███╗   ██╗██╗██╗   ██╗███████╗██████╗ ███████╗ █████╗ ██╗     
██║   ██║████╗  ██║██║██║   ██║██╔════╝██╔══██╗██╔════╝██╔══██╗██║     
██║   ██║██╔██╗ ██║██║██║   ██║█████╗  ██████╔╝███████╗███████║██║     
██║   ██║██║╚██╗██║██║╚██╗ ██╔╝██╔══╝  ██╔══██╗╚════██║██╔══██║██║     
╚██████╔╝██║ ╚████║██║ ╚████╔╝ ███████╗██║  ██║███████║██║  ██║███████╗
 ╚═════╝ ╚═╝  ╚═══╝╚═╝  ╚═══╝  ╚══════╝╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚══════╝

 █████╗ ███╗   ██╗██████╗ ██████╗  ██████╗ ██╗██████╗ 
██╔══██╗████╗  ██║██╔══██╗██╔══██╗██╔═══██╗██║██╔══██╗
███████║██╔██╗ ██║██║  ██║██████╔╝██║   ██║██║██║  ██║
██╔══██║██║╚██╗██║██║  ██║██╔══██╗██║   ██║██║██║  ██║
██║  ██║██║ ╚████║██████╔╝██║  ██║╚██████╔╝██║██████╔╝
╚═╝  ╚═╝╚═╝  ╚═══╝╚═════╝ ╚═╝  ╚═╝ ╚═════╝ ╚═╝╚═════╝ 

██████╗ ███████╗██████╗ ██╗      ██████╗  █████╗ ████████╗███████╗██████╗ 
██╔══██╗██╔════╝██╔══██╗██║     ██╔═══██╗██╔══██╗╚══██╔══╝██╔════╝██╔══██╗
██║  ██║█████╗  ██████╔╝██║     ██║   ██║███████║   ██║   █████╗  ██████╔╝
██║  ██║██╔══╝  ██╔══██╗██║     ██║   ██║██╔══██║   ██║   ██╔══╝  ██╔══██╗
██████╔╝███████╗██║  ██║███████╗╚██████╔╝██║  ██║   ██║   ███████╗██║  ██║
╚═════╝ ╚══════╝╚═╝  ╚═╝╚══════╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝   ╚══════╝╚═╝  ╚═╝`

	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color(pink)).
		Bold(true).
		Align(lipgloss.Center).
		Render(titleASCII)

	// Divider line
	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color(overlay)).
		Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	sub := lipgloss.NewStyle().
		Foreground(lipgloss.Color(subtext)).
		Align(lipgloss.Center).
		Render("ADB-powered Android package manager for power users")

	// Feature badges
	type feature struct {
		icon, label, color string
	}
	features := []feature{
		{"◈", "ADB Detection", teal},
		{"◈", "Package Browser", mauve},
		{"◈", "Batch Uninstall", peach},
		{"◈", "Search & Filter", green},
		{"◈", "Catppuccin UI", pink},
	}

	badgeStyle := lipgloss.NewStyle().
		Padding(0, 2).
		MarginLeft(1).
		Border(lipgloss.RoundedBorder())

	var badges []string
	for _, f := range features {
		badges = append(badges,
			badgeStyle.
				Foreground(lipgloss.Color(f.color)).
				BorderForeground(lipgloss.Color(f.color)).
				Render(f.icon+" "+f.label),
		)
	}
	featureRow := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, badges...))

	// Warning notice
	notice := lipgloss.NewStyle().
		Foreground(lipgloss.Color(peach)).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color(peach)).
		PaddingLeft(1).
		Render("Warning: Removing system packages may affect device stability.\nProceed only if you know what you are doing.")

	// CTA Button
	btn := lipgloss.NewStyle().
		Foreground(lipgloss.Color(base)).
		Background(lipgloss.Color(mauve)).
		Padding(0, 8).
		Bold(true).
		Render("  START  →")

	// Footer hints
	kStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(mauve)).Bold(true)
	dStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(subtext))
	sep := dStyle.Render("  •  ")

	footer := lipgloss.JoinHorizontal(lipgloss.Top,
		kStyle.Render("enter"), dStyle.Render(" start"),
		sep,
		kStyle.Render("q"), dStyle.Render(" quit"),
	)

	// Stack all content
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		divider,
		"",
		sub,
		"",
		featureRow,
		"",
		lipgloss.NewStyle().Align(lipgloss.Center).Render(notice),
		"",
		btn,
		"",
		footer,
	)

	card := lipgloss.NewStyle().
		Background(lipgloss.Color(surface)).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(mauve)).
		Padding(1, 4).
		Render(content)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		card,
		lipgloss.WithWhitespaceBackground(lipgloss.Color(base)),
	)
}
