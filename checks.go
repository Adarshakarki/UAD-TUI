package main

import (
	"fmt"
	"os/exec"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

const adbInstallURL = "https://developer.android.com/studio/releases/platform-tools"

// Step status
type stepStatus int

const (
	statusPending stepStatus = iota
	statusRunning
	statusPass
	statusFail
	statusSkipped
)

type checkStep struct {
	label  string
	status stepStatus
	detail string
}

// Messages
type startChecksMsg struct{}
type checkResultMsg struct {
	index  int
	passed bool
	detail string
}

// ADB checks
func checkADBInstalled() (bool, string) {
	path, err := exec.LookPath("adb")
	if err != nil {
		return false, "adb not found in PATH"
	}
	return true, path
}

func checkADBDaemon() (bool, string) {
	out, err := exec.Command("adb", "start-server").CombinedOutput()
	if err != nil {
		return false, strings.TrimSpace(string(out))
	}
	return true, "daemon running"
}

func checkDevices() (bool, string) {
	out, err := exec.Command("adb", "devices").Output()
	if err != nil {
		return false, "could not list devices"
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	count := 0
	for _, l := range lines[1:] {
		l = strings.TrimSpace(strings.TrimRight(l, "\r"))
		if l != "" {
			count++
		}
	}
	if count == 0 {
		return false, "no devices found - enable USB debugging"
	}
	return true, fmt.Sprintf("%d device(s) detected", count)
}

func checkAuthorized() (bool, string) {
	out, err := exec.Command("adb", "devices").Output()
	if err != nil {
		return false, "could not check authorization"
	}
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n")[1:] {
		if strings.Contains(l, "unauthorized") {
			return false, "unauthorized - accept RSA prompt on device"
		}
	}
	return true, "device authorized"
}

var checkFns = []func() (bool, string){
	checkADBInstalled,
	checkADBDaemon,
	checkDevices,
	checkAuthorized,
}

func runStep(index int, fn func() (bool, string)) tea.Cmd {
	return func() tea.Msg {
		passed, detail := fn()
		return checkResultMsg{index: index, passed: passed, detail: detail}
	}
}

// Model
type checksModel struct {
	width      int
	height     int
	spinner    spinner.Model
	steps      []*checkStep
	current    int
	done       bool
	adbMissing bool
	showHelp   bool
}

func newChecksModel() checksModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8"))

	return checksModel{
		spinner: s,
		steps: []*checkStep{
			{label: "Checking ADB Installation"},
			{label: "Verifying ADB Daemon Status"},
			{label: "Detecting Connected Devices"},
			{label: "Authorizing Device Connection"},
		},
		current: -1,
	}
}

func (m checksModel) init() tea.Cmd {
	if m.done && m.allPassed() {
		return m.spinner.Tick
	}
	for _, s := range m.steps {
		s.status = statusPending
		s.detail = ""
	}
	m.done = false
	m.adbMissing = false
	m.current = -1
	return tea.Batch(m.spinner.Tick, func() tea.Msg { return startChecksMsg{} })
}

func (m checksModel) update(msg tea.Msg) (checksModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyPressMsg:
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "b":
			return m, navigate(pageWelcome)
		case "h":
			m.showHelp = true
		case "r":
			for _, s := range m.steps {
				s.status = statusPending
				s.detail = ""
			}
			m.done = false
			m.adbMissing = false
			m.current = -1
			return m, func() tea.Msg { return startChecksMsg{} }
		case "o":
			if m.adbMissing {
				return m, openURL(adbInstallURL)
			}
		case "enter":
			if m.done && m.allPassed() {
				return m, navigate(pagePackages)
			}
		}

	case startChecksMsg:
		m.steps[0].status = statusRunning
		m.current = 0
		return m, runStep(0, checkFns[0])

	case checkResultMsg:
		i := msg.index
		m.steps[i].detail = msg.detail
		if msg.passed {
			m.steps[i].status = statusPass
			next := i + 1
			if next < len(m.steps) {
				m.current = next
				m.steps[next].status = statusRunning
				return m, runStep(next, checkFns[next])
			}
			m.done = true
			m.current = -1
		} else {
			m.steps[i].status = statusFail
			for j := i + 1; j < len(m.steps); j++ {
				m.steps[j].status = statusSkipped
			}
			m.done = true
			m.current = -1
			if i == 0 {
				m.adbMissing = true
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m checksModel) allPassed() bool {
	for _, s := range m.steps {
		if s.status != statusPass {
			return false
		}
	}
	return true
}

// Help
func (m checksModel) helpView() tea.View {
	lavColor := lipgloss.Color("#CBA6F7")
	blueColor := lipgloss.Color("#89B4FA")
	textColor := lipgloss.Color("#CDD6F4")
	dimColor := lipgloss.Color("#45475A")
	darkColor := lipgloss.Color("#1e1e2e")

	header := lipgloss.NewStyle().
		Bold(true).Foreground(darkColor).Background(lavColor).
		Padding(0, 4).
		Render("TROUBLESHOOTING GUIDE")

	section := func(icon, title, color string, fixes []string) string {
		badge := lipgloss.NewStyle().
			Background(lipgloss.Color(color)).
			Foreground(darkColor).
			Bold(true).
			Padding(0, 2).
			Render(icon)
		heading := lipgloss.NewStyle().
			Foreground(lipgloss.Color(color)).
			Bold(true).
			Render(title)
		titleRow := lipgloss.JoinHorizontal(lipgloss.Left, badge, "  ", heading)
		var lines []string
		lines = append(lines, titleRow)
		for _, f := range fixes {
			lines = append(lines, lipgloss.NewStyle().Foreground(textColor).Render("  → "+f))
		}
		return strings.Join(lines, "\n")
	}

	sep := lipgloss.NewStyle().Foreground(dimColor).Render(strings.Repeat("─", m.width-18))

	content := lipgloss.JoinVertical(lipgloss.Left,

		section("1", "ADB Not Installed", "#F38BA8", []string{
			"Download from: developer.android.com/studio/releases/platform-tools",
			"Extract the ZIP and add the platform-tools folder to your system PATH.",
			"Restart your terminal and run: adb version",
		}),
		"", sep, "",

		section("2", "ADB Daemon Not Running", "#FAB387", []string{
			"Run: adb kill-server  then  adb start-server",
			"Close Android Studio if it's open — it can conflict with adb.",
			"On Windows, try running the terminal as Administrator.",
		}),
		"", sep, "",

		section("3", "No Device Detected", "#89B4FA", []string{
			"Go to Settings → About Phone → tap Build Number 7 times → enable USB Debugging.",
			"Samsung: Settings → About Phone → Software Information → tap Build Number 7 times.",
			"Use a data cable (not charge-only) and try a different USB port.",
		}),
		"", sep, "",

		section("4", "Device Unauthorized", "#CBA6F7", []string{
			"Unlock your screen — the Allow prompt only shows when the screen is on.",
			"Tap 'Allow' on the USB Debugging prompt on your phone.",
			"Still stuck? Go to Developer Options → Revoke USB Authorizations → reconnect.",
		}),
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#313244")).
		Padding(1, 2).
		Width(m.width - 8).
		Render(content)

	closeBtn := lipgloss.NewStyle().
		Padding(0, 3).Bold(true).
		Background(dimColor).
		Foreground(textColor).
		Render("[any key] CLOSE")

	page := lipgloss.JoinVertical(lipgloss.Center,
		header, "\n", box, "\n", closeBtn,
	)

	container := lipgloss.NewStyle().
		Width(m.width-2).Height(m.height-2).
		Border(lipgloss.RoundedBorder()).
		BorderForegroundBlend(blueColor, lavColor).
		Align(lipgloss.Center, lipgloss.Top).
		Render(page)

	v := tea.NewView(container)
	v.AltScreen = true
	return v
}

// View
func (m checksModel) view() tea.View {
	if m.showHelp {
		return m.helpView()
	}

	passColor := lipgloss.Color("#A6E3A1")
	failColor := lipgloss.Color("#F38BA8")
	skipColor := lipgloss.Color("#45475A")
	runColor := lipgloss.Color("#89B4FA")
	pendColor := lipgloss.Color("#585B70")
	lavColor := lipgloss.Color("#CBA6F7")
	textColor := lipgloss.Color("#CDD6F4")
	orangeColor := lipgloss.Color("#FAB387")
	darkColor := lipgloss.Color("#1e1e2e")

	passStyle := lipgloss.NewStyle().Foreground(passColor)
	failStyle := lipgloss.NewStyle().Foreground(failColor)
	dimStyle := lipgloss.NewStyle().Foreground(skipColor)
	orangeStyle := lipgloss.NewStyle().Foreground(orangeColor)

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(darkColor).
		Background(runColor).
		Padding(0, 4).
		MarginBottom(1).
		Render("ADB STATUS CHECK")

	steps := m.steps
	spinnerView := m.spinner.View()

	var rows [][]string
	for _, step := range steps {
		var icon, statusText, detail string
		switch step.status {
		case statusPass:
			icon = "✔"
			statusText = "Pass"
			detail = step.detail
		case statusFail:
			icon = "✘"
			statusText = "Failed"
			detail = step.detail
		case statusRunning:
			icon = spinnerView
			statusText = "Running…"
			detail = ""
		case statusSkipped:
			icon = "─"
			statusText = "Skipped"
			detail = "—"
		default:
			icon = "○"
			statusText = "Pending"
			detail = "—"
		}
		rows = append(rows, []string{icon + "  " + step.label, statusText, detail})
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(lavColor).
		Bold(true).
		Align(lipgloss.Center).
		Padding(0, 2)

	baseCell := lipgloss.NewStyle().Padding(0, 2)

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(runColor)).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			if row < 0 || row >= len(steps) {
				return baseCell
			}
			switch steps[row].status {
			case statusPass:
				return baseCell.Foreground(passColor)
			case statusFail:
				return baseCell.Foreground(failColor)
			case statusRunning:
				return baseCell.Foreground(runColor).Bold(true)
			case statusSkipped:
				return baseCell.Foreground(skipColor).Italic(true)
			default:
				return baseCell.Foreground(pendColor)
			}
		}).
		Headers("CHECK", "STATUS", "DETAIL").
		Rows(rows...)

	tableStr := lipgloss.NewStyle().Foreground(textColor).Render(t.String())

	var installBanner string
	if m.adbMissing {
		installBanner = "\n" +
			orangeStyle.Bold(true).Render("ADB not installed.") +
			"  " +
			lipgloss.NewStyle().Foreground(runColor).Underline(true).Render(adbInstallURL) +
			"\n"
	}

	var statusLine string
	switch {
	case !m.done:
		statusLine = dimStyle.Italic(true).Render("Running checks…")
	case m.allPassed():
		statusLine = passStyle.Bold(true).Render("✔  All checks passed")
	default:
		statusLine = failStyle.Bold(true).Render("✘  One or more checks failed")
	}

	btnBase := lipgloss.NewStyle().Padding(0, 3).Bold(true).MarginTop(1)

	backBtn := btnBase.
		Background(skipColor).
		Foreground(textColor).
		Render("[b] BACK")

	refreshBtn := btnBase.
		Background(runColor).
		Foreground(darkColor).
		MarginLeft(2).
		Render("[r] REFRESH")

	quitBtn := btnBase.
		Background(failColor).
		Foreground(darkColor).
		MarginLeft(2).
		Render("[q] QUIT")

	installBtn := btnBase.
		Background(orangeColor).
		Foreground(darkColor).
		MarginLeft(2).
		Render("[o] INSTALL ADB")

	continueBtn := btnBase.
		Background(passColor).
		Foreground(darkColor).
		MarginLeft(2).
		Render("[enter] CONTINUE")

	helpBtn := btnBase.
		Background(lavColor).
		Foreground(darkColor).
		MarginLeft(2).
		Render("[h] HELP")

	var buttons string
	switch {
	case m.adbMissing:
		buttons = lipgloss.JoinHorizontal(lipgloss.Center,
			backBtn, refreshBtn, installBtn, helpBtn, quitBtn)
	case m.done && m.allPassed():
		buttons = lipgloss.JoinHorizontal(lipgloss.Center,
			backBtn, refreshBtn, continueBtn, helpBtn, quitBtn)
	default:
		buttons = lipgloss.JoinHorizontal(lipgloss.Center,
			backBtn, refreshBtn, helpBtn, quitBtn)
	}

	content := lipgloss.JoinVertical(lipgloss.Center,
		header,
		tableStr,
		installBanner,
		statusLine,
		"\n",
		buttons,
	)

	container := lipgloss.NewStyle().
		Width(m.width-2).
		Height(m.height-2).
		Border(lipgloss.RoundedBorder()).
		BorderForegroundBlend(lipgloss.Color("#89B4FA"), lipgloss.Color("#CBA6F7")).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)

	v := tea.NewView(container)
	v.AltScreen = true
	return v
}
