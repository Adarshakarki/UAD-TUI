package main

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ADB device states
type AdbState int

const (
	StateMissingADB AdbState = iota
	StateNoDevice
	StateUnauthorized
	StateReady
)

func (s AdbState) String() string {
	switch s {
	case StateMissingADB:
		return "ADB not installed"
	case StateNoDevice:
		return "No device connected"
	case StateUnauthorized:
		return "Device unauthorized"
	case StateReady:
		return "Device ready"
	default:
		return "Unknown status"
	}
}

type adbCheckMsg struct {
	AdbInstalled     bool
	AdbServerRunning bool
	DeviceConnected  bool
	UsbAuthorized    bool
	ShellAccess      bool
	DeviceInfo       string
	AndroidVersion   string
	OverallMessage   string
}

func getSystemStatus() adbCheckMsg {
	_, err := exec.LookPath("adb")
	if err != nil {
		return adbCheckMsg{OverallMessage: "ADB was not found on your system path."}
	}

	msg := adbCheckMsg{AdbInstalled: true}
	_ = exec.Command("adb", "start-server").Run()

	cmd := exec.Command("adb", "devices")
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg.OverallMessage = "Failed to execute ADB command."
		return msg
	}

	output := string(out)
	if strings.Contains(output, "error: cannot connect to daemon") {
		msg.OverallMessage = "ADB server is not responding. Try refreshing."
		return msg
	}

	msg.AdbServerRunning = true
	lines := strings.Split(output, "\n")
	hasDevices := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "List of devices") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		hasDevices = true

		if strings.Contains(line, "unauthorized") {
			msg.DeviceConnected = true
			msg.UsbAuthorized = false
			msg.OverallMessage = "Device found but unauthorized. Check your phone for the authorization prompt."
			return msg
		}
		if parts[1] == "device" {
			serial := parts[0]
			msg.DeviceConnected = true
			msg.UsbAuthorized = true

			shellCmd := exec.Command("adb", "-s", serial, "shell", "echo", "shell_ok")
			shellOutput, shellErr := shellCmd.CombinedOutput()
			if shellErr != nil || !strings.Contains(strings.TrimSpace(string(shellOutput)), "shell_ok") {
				msg.ShellAccess = false
				msg.OverallMessage = "Device connected and authorized, but shell access failed."
				return msg
			}
			msg.ShellAccess = true

			info, ver := getDeviceInfo(serial)
			msg.DeviceInfo = info
			msg.AndroidVersion = ver
			msg.OverallMessage = "Device connected and ready."
			return msg
		}
	}

	if !hasDevices {
		msg.OverallMessage = "No devices detected. Connect your phone and enable USB Debugging."
	}
	return msg
}

func getDeviceInfo(serial string) (string, string) {
	m, _ := exec.Command("adb", "-s", serial, "shell", "getprop", "ro.product.model").Output()
	b, _ := exec.Command("adb", "-s", serial, "shell", "getprop", "ro.product.brand").Output()
	v, _ := exec.Command("adb", "-s", serial, "shell", "getprop", "ro.build.version.release").Output()
	return fmt.Sprintf("%s %s", strings.TrimSpace(string(b)), strings.TrimSpace(string(m))), strings.TrimSpace(string(v))
}

func checkAdb() tea.Cmd {
	return func() tea.Msg {
		return getSystemStatus()
	}
}

func restartAdbServer() {
	_ = exec.Command("adb", "kill-server").Run()
	_ = exec.Command("adb", "start-server").Run()
}

func (m model) adbView() string {
	const (
		base    = "#1e1e2e"
		surface = "#313244"
		overlay = "#45475a"
		mantle  = "#181825"
		teal    = "#94e2d5"
		peach   = "#fab387"
		red     = "#f38ba8"
		mauve   = "#cba6f7"
		green   = "#a6e3a1"
		text    = "#cdd6f4"
		subtext = "#a6adc8"
	)

	isChecking := m.adbMsg == "Scanning..."
	isReady := m.adbState == StateReady

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(base)).
		Background(lipgloss.Color(teal)).
		Bold(true).
		Padding(0, 2)

	statusDot := "◉"
	statusColor := red
	statusText := "CHECKING"
	if isReady {
		statusDot = "◉"
		statusColor = green
		statusText = "READY"
	} else if isChecking {
		statusDot = "◎"
		statusColor = peach
		statusText = "SCANNING"
	}

	dotStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor))
	header := lipgloss.JoinHorizontal(lipgloss.Top,
		headerStyle.Render(" ADB SYSTEM CHECK "),
		"  ",
		dotStyle.Render(statusDot+" "+statusText),
	)

	// Checklist steps
	type step struct {
		num   string
		label string
		ok    bool
	}
	steps := []step{
		{"01", "ADB Binary Found", m.adbInstalled},
		{"02", "ADB Daemon Running", m.adbServerRunning},
		{"03", "USB Connection", m.deviceConnected},
		{"04", "Device Authorized", m.usbAuthorized},
		{"05", "Shell Access", m.shellAccess},
	}

	numStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(overlay)).
		Width(4)
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(text)).
		Width(22)
	checkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(teal)).Bold(true)
	crossStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(red))

	var rows []string
	for _, s := range steps {
		var icon string
		switch {
		case s.ok:
			icon = checkStyle.Render("✓ PASS")
		case isChecking:
			icon = lipgloss.NewStyle().Foreground(lipgloss.Color(peach)).Render(m.spinner.View() + " WAIT")
		default:
			icon = crossStyle.Render("✗ FAIL")
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top,
			numStyle.Render(s.num),
			labelStyle.Render(s.label),
			icon,
		)
		rows = append(rows, row)
	}

	checklistBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(overlay)).
		Padding(1, 2).
		Width(38).
		Background(lipgloss.Color(mantle)).
		Render(strings.Join(rows, "\n"))

	// Detail / Info panel
	infoStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Width(38).
		MarginTop(1)

	var detailBox string
	switch {
	case !m.adbInstalled:
		detailBox = infoStyle.
			BorderForeground(lipgloss.Color(red)).
			Background(lipgloss.Color(mantle)).
			Render(lipgloss.JoinVertical(lipgloss.Left,
				lipgloss.NewStyle().Foreground(lipgloss.Color(red)).Bold(true).Render("ADB NOT FOUND"),
				"",
				lipgloss.NewStyle().Foreground(lipgloss.Color(subtext)).Render(
					"1. Download SDK Platform-Tools\n"+
						"   from developer.android.com\n"+
						"2. Add folder to system PATH\n"+
						"3. Restart this application",
				),
			))

	case isReady:
		detailBox = infoStyle.
			BorderForeground(lipgloss.Color(teal)).
			Background(lipgloss.Color(mantle)).
			Render(lipgloss.JoinVertical(lipgloss.Left,
				lipgloss.NewStyle().Foreground(lipgloss.Color(teal)).Bold(true).Render("DEVICE CONNECTED"),
				"",
				lipgloss.NewStyle().Foreground(lipgloss.Color(mauve)).Bold(true).
					Render("  "+m.adbInfo),
				lipgloss.NewStyle().Foreground(lipgloss.Color(subtext)).
					Render(fmt.Sprintf("  Android %s", m.adbAndroid)),
			))

	case m.adbState == StateUnauthorized:
		detailBox = infoStyle.
			BorderForeground(lipgloss.Color(peach)).
			Background(lipgloss.Color(mantle)).
			Render(lipgloss.JoinVertical(lipgloss.Left,
				lipgloss.NewStyle().Foreground(lipgloss.Color(peach)).Bold(true).Render("ACTION REQUIRED"),
				"",
				lipgloss.NewStyle().Foreground(lipgloss.Color(subtext)).Render(
					"Your phone is asking for permission.\n\n"+
						"Look at your device screen and tap\n"+
						"\"Allow USB Debugging\" to continue.",
				),
			))

	case m.adbState == StateNoDevice:
		detailBox = infoStyle.
			BorderForeground(lipgloss.Color(overlay)).
			Background(lipgloss.Color(mantle)).
			Render(lipgloss.JoinVertical(lipgloss.Left,
				lipgloss.NewStyle().Foreground(lipgloss.Color(subtext)).Bold(true).Render("WAITING FOR DEVICE"),
				"",
				lipgloss.NewStyle().Foreground(lipgloss.Color(subtext)).Render(
					"1. Connect phone via USB cable\n"+
						"2. Enable USB Debugging\n"+
						"   Settings → Developer Options\n"+
						"3. Press [r] to refresh",
				),
			))

	default:
		detailBox = infoStyle.
			BorderForeground(lipgloss.Color(overlay)).
			Background(lipgloss.Color(mantle)).
			Render(lipgloss.NewStyle().Foreground(lipgloss.Color(subtext)).Render(m.adbMsg))
	}

	// Action Button
	var btn string
	if isReady {
		btn = lipgloss.NewStyle().
			Foreground(lipgloss.Color(base)).
			Background(lipgloss.Color(teal)).
			Bold(true).
			Padding(0, 4).
			MarginTop(1).
			Render("  CONTINUE  →  (enter)")
	} else if !m.adbInstalled {
		btn = lipgloss.NewStyle().
			Foreground(lipgloss.Color(base)).
			Background(lipgloss.Color(red)).
			Bold(true).
			Padding(0, 4).
			MarginTop(1).
			Render("  FIX ADB FIRST")
	} else {
		btn = lipgloss.NewStyle().
			Foreground(lipgloss.Color(subtext)).
			Background(lipgloss.Color(overlay)).
			Padding(0, 4).
			MarginTop(1).
			Render("  WAITING...")
	}

	// Footer hints
	kStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(mauve)).Bold(true)
	dStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(subtext))
	sep := dStyle.Render("  •  ")

	footer := lipgloss.NewStyle().MarginTop(1).Render(
		lipgloss.JoinHorizontal(lipgloss.Top,
			kStyle.Render("r"), dStyle.Render(" refresh"),
			sep,
			kStyle.Render("esc"), dStyle.Render(" home"),
			sep,
			kStyle.Render("q"), dStyle.Render(" quit"),
		),
	)

	// Assemble card
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		header,
		"",
		checklistBox,
		detailBox,
		btn,
		footer,
	)

	card := lipgloss.NewStyle().
		Background(lipgloss.Color(surface)).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(teal)).
		Padding(1, 4).
		Align(lipgloss.Center).
		Render(content)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		card,
		lipgloss.WithWhitespaceBackground(lipgloss.Color(base)),
	)
}
