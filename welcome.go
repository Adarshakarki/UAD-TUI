package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const appVersion = "v0.1.0"
const githubURL = "https://github.com/Adarshakarki/UAD-TUI"

// logo
const logoArt = `
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

// gradient colors
var gradientColors = []string{
	"#F38BA8", // pink
	"#F5A0B5", // pink-peach blend
	"#FAB387", // peach
	"#F9C96A", // peach-yellow blend
	"#F9E2AF", // yellow
	"#C6E5A0", // yellow-green blend
	"#A6E3A1", // green
	"#7ED8A8", // green-teal blend
	"#94E2D5", // teal
	"#8DD9E8", // teal-sky blend
	"#89DCF3", // sky
	"#89C5F9", // sky-blue blend
	"#89B4FA", // blue
	"#A4A9FB", // blue-lavender blend
	"#CBA6F7", // lavender
	"#D8A0F0", // lavender-pink blend
	"#F38BA8", // back to pink (wrap)
}

// logo render
func renderGradientLogo(art string) string {
	lines := strings.Split(art, "\n")
	var b strings.Builder
	colorIdx := 0
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			c := gradientColors[colorIdx%len(gradientColors)]
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render(line))
			colorIdx++
		} else {
			b.WriteString(line)
		}
		if i < len(lines)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// Model
type welcomeModel struct {
	width  int
	height int
	ready  bool
}

func newWelcomeModel() welcomeModel { return welcomeModel{} }

func (m welcomeModel) init() tea.Cmd {
	return tea.RequestBackgroundColor
}

func (m welcomeModel) update(msg tea.Msg) (welcomeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		m.ready = true
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			return m, navigate(pageChecks)
		case "g":
			return m, openURL(githubURL)
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m welcomeModel) view() tea.View {
	if !m.ready {
		return tea.NewView("Loading...")
	}

	// Warning banner
	warningText := "⚠︎ WARNING: Removing system packages may affect device stability.\nProceed only if you know what you are doing."
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#1e1e2e")).
		Background(lipgloss.Color("#F38BA8")).
		Padding(1, 2).
		MarginBottom(1).
		Align(lipgloss.Center).
		Render(warningText)

	// Gradient logo
	logo := renderGradientLogo(logoArt)

	// Version badge
	version := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#45475A")).
		MarginTop(1).
		Render(appVersion + "  •  Universal Android Debloater on TUI")

	// Buttons
	btnBase := lipgloss.NewStyle().
		Padding(0, 4).
		Bold(true).
		MarginTop(2)

	startBtn := btnBase.
		Background(lipgloss.Color("#A6E3A1")).
		Foreground(lipgloss.Color("#1e1e2e")).
		Render("[↵] START")

	githubBtn := btnBase.
		Background(lipgloss.Color("#89B4FA")).
		Foreground(lipgloss.Color("#1e1e2e")).
		MarginLeft(2).
		Render("[g] ★ GITHUB")

	quitBtn := btnBase.
		Background(lipgloss.Color("#F38BA8")).
		Foreground(lipgloss.Color("#1e1e2e")).
		MarginLeft(2).
		Render("[q] QUIT")

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, startBtn, githubBtn, quitBtn)

	uiContent := lipgloss.JoinVertical(lipgloss.Center,
		header,
		logo,
		version,
		buttons,
	)

	// Gradient border
	container := lipgloss.NewStyle().
		Width(m.width-2).
		Height(m.height-2).
		Border(lipgloss.RoundedBorder()).
		BorderForegroundBlend(lipgloss.Color("#F38BA8"), lipgloss.Color("#89B4FA")).
		Align(lipgloss.Center, lipgloss.Center).
		Render(uiContent)

	v := tea.NewView(container)
	v.AltScreen = true
	return v
}
