package main

import (
	"github.com/charmbracelet/lipgloss"
)

// render home screen
func (m model) homeView() string {
	const (
		base    = "#1e1e2e" // Catppuccin Mocha Base
		surface = "#313244" // Catppuccin Mocha Surface0
		text    = "#cdd6f4" // Catppuccin Mocha Text
		subtext = "#a6adc8" // Catppuccin Mocha Subtext0
		mauve   = "#cba6f7" // Catppuccin Mocha Mauve
		pink    = "#f5c2e7" // Catppuccin Mocha Pink
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
╚═════╝ ╚══════╝╚═╝  ╚═╝╚══════╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝   ╚══════╝╚═╝  ╚═╝
`

	// Title color
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color(pink)).
		Bold(true).
		Align(lipgloss.Center).
		Render(titleASCII)

	// subtitle
	sub := lipgloss.NewStyle(). 
		Foreground(lipgloss.Color(text)).
		Align(lipgloss.Center).
		Render("ADB-powered Android package manager")

	// button
	btn := lipgloss.NewStyle().
		Foreground(lipgloss.Color(base)).
		Background(lipgloss.Color(mauve)).
		Padding(0, 5).
		MarginTop(1).
		Bold(true).
		Render(" START ")

	// Footer
	footer := lipgloss.NewStyle(). // Navigation hints at the bottom
		Foreground(lipgloss.Color(subtext)).
		Render("enter • q quit")

	// Combine all elements vertically
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		sub,
		"",
		btn,
		"",
		footer,
	)

	// Styled card container for the content
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