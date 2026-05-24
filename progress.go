package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Styles
var (
	checkMark = lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1")).SetString("✓")
	crossMark = lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8")).SetString("✘")
)

// Messages
type uninstallResultMsg struct {
	pkg string
	err error
}

// Commands
func doUninstallPkg(pkg string) tea.Cmd {
	return func() tea.Msg {
		err := exec.Command("adb", "shell", "pm", "uninstall", "--user", "0", pkg).Run()
		return uninstallResultMsg{pkg: pkg, err: err}
	}
}

// Model
type logEntry struct {
	pkg string
	ok  bool
}

type progressModel struct {
	width     int
	height    int
	spinner   spinner.Model
	bar       progress.Model
	queue     []string
	index     int
	total     int
	current   string
	log       []logEntry
	done      bool
	elapsed   time.Duration
	startedAt time.Time
}

func newProgressModel() progressModel {
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#89B4FA"))
	p := progress.New(
		progress.WithDefaultBlend(),
		progress.WithoutPercentage(),
	)
	return progressModel{spinner: s, bar: p}
}

func (m progressModel) init() tea.Cmd { return nil }

func (m progressModel) failedList() []string {
	var out []string
	for _, e := range m.log {
		if !e.ok {
			out = append(out, e.pkg)
		}
	}
	return out
}

func (m progressModel) update(msg tea.Msg) (progressModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		barW := m.width - 30
		if barW < 10 {
			barW = 10
		}
		m.bar.SetWidth(barW)

	case startUninstallMsg:
		m.queue = msg.packages
		m.total = len(msg.packages)
		m.index = 0
		m.done = false
		m.log = nil
		m.elapsed = 0
		m.startedAt = time.Now()
		if len(m.queue) == 0 {
			m.done = true
			return m, nil
		}
		m.current = m.queue[0]
		m.queue = m.queue[1:]
		barCmd := m.bar.SetPercent(0)
		return m, tea.Batch(barCmd, m.spinner.Tick, doUninstallPkg(m.current))

	case uninstallResultMsg:
		m.log = append(m.log, logEntry{pkg: msg.pkg, ok: msg.err == nil})
		m.index++
		if len(m.queue) == 0 {
			m.done = true
			m.current = ""
			m.elapsed = time.Since(m.startedAt).Round(time.Second)
			return m, m.bar.SetPercent(1.0)
		}
		m.current = m.queue[0]
		m.queue = m.queue[1:]
		barCmd := m.bar.SetPercent(float64(m.index) / float64(m.total))
		return m, tea.Batch(barCmd, doUninstallPkg(m.current))

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case progress.FrameMsg:
		var cmd tea.Cmd
		m.bar, cmd = m.bar.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "h":
			return m, navigate(pageWelcome)
		case "b":
			if m.done {
				return m, navigate(pagePackages)
			}
		case "r":
			if m.done && len(m.failedList()) > 0 {
				retry := m.failedList()
				m.log = nil
				return m, func() tea.Msg { return startUninstallMsg{packages: retry} }
			}
		case "g":
			return m, openURL(githubURL)
		}
	}

	return m, nil
}

// View
func (m progressModel) view() tea.View {
	btn := func(bg, fg, label string) string {
		return lipgloss.NewStyle().
			Padding(0, 3).Bold(true).MarginRight(1).
			Background(lipgloss.Color(bg)).
			Foreground(lipgloss.Color(fg)).
			Render(label)
	}

	// Header — text has background, not the full line
	headerText := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#1e1e2e")).
		Background(lipgloss.Color("#CBA6F7")).
		Padding(0, 2).
		Render("UNINSTALLING PACKAGES")

	// If no packages queued, show message with options to go back or home
	if m.total == 0 {
		content := lipgloss.JoinVertical(lipgloss.Center,
			headerText,
			"\n",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#585B70")).Render("No packages queued. Go back and select packages."),
			"\n",
			lipgloss.JoinHorizontal(lipgloss.Center,
				btn("#A6E3A1", "#1e1e2e", "[h] HOME"),
				btn("#45475A", "#CDD6F4", "[q] QUIT"),
			),
		)
		return m.wrap(content)
	}

	// Log of removed packages
	var logLines []string
	for _, e := range m.log {
		var mark, name string
		if e.ok {
			mark = checkMark.String()
			name = lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1")).Render(e.pkg)
		} else {
			mark = crossMark.String()
			name = lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8")).Render(e.pkg)
		}
		logLines = append(logLines, mark+"  "+name)
	}

	// Active line with spinner and progress bar
	if !m.done && m.current != "" {
		w := len(fmt.Sprintf("%d", m.total))
		count := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#585B70")).
			Render(fmt.Sprintf("%*d/%d", w, m.index+1, m.total))
		pkgLabel := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CBA6F7")).Bold(true).
			Render(m.current)
		activeLine := m.spinner.View() + "  " + pkgLabel + "  " + m.bar.View() + "  " + count
		logLines = append(logLines, activeLine)
	}

	logBlock := strings.Join(logLines, "\n")

	// If done, show summary and options to retry failed, go back, or home
	if m.done {
		failed := m.failedList()
		successCount := m.total - len(failed)

		summary := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6E3A1")).
			Bold(true).
			Render(fmt.Sprintf("Done! Removed %d/%d packages in %s.", successCount, m.total, m.elapsed))

		buttons := lipgloss.JoinHorizontal(lipgloss.Center,
			btn("#A6E3A1", "#1e1e2e", "[h] HOME"),
			btn("#45475A", "#CDD6F4", "[b] BACK"),
			btn("#89B4FA", "#1e1e2e", "[g] ★ GITHUB"),
			btn("#F38BA8", "#1e1e2e", "[q] QUIT"),
		)
		if len(failed) > 0 {
			buttons = lipgloss.JoinHorizontal(lipgloss.Center,
				btn("#A6E3A1", "#1e1e2e", "[h] HOME"),
				btn("#FAB387", "#1e1e2e", "[r] RETRY FAILED"),
				btn("#45475A", "#CDD6F4", "[b] BACK"),
				btn("#89B4FA", "#1e1e2e", "[g] ★ GITHUB"),
				btn("#F38BA8", "#1e1e2e", "[q] QUIT"),
			)
		}

		content := lipgloss.JoinVertical(lipgloss.Center,
			headerText,
			"\n",
			logBlock,
			"\n",
			summary,
			"\n",
			buttons,
		)
		return m.wrap(content)
	}

	// Ongoing progress view
	content := lipgloss.JoinVertical(lipgloss.Center,
		headerText,
		"\n",
		logBlock,
		"\n",
		btn("#F38BA8", "#1e1e2e", "[q] QUIT"),
	)
	return m.wrap(content)
}

func (m progressModel) wrap(content string) tea.View {
	container := lipgloss.NewStyle().
		Width(m.width-2).
		Height(m.height-2).
		Border(lipgloss.RoundedBorder()).
		BorderForegroundBlend(lipgloss.Color("#CBA6F7"), lipgloss.Color("#89B4FA")).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)
	v := tea.NewView(container)
	v.AltScreen = true
	return v
}

// Root model
func openURL(url string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("cmd", "/c", "start", url)
		case "darwin":
			cmd = exec.Command("open", url)
		default:
			cmd = exec.Command("xdg-open", url)
		}
		_ = cmd.Run()
		return nil
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
