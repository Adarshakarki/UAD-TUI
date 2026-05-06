package main

import (
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/lipgloss"
)

// ADB states
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
	state AdbState
	msg   string
}

// Check Adb status
func getSystemStatus() (AdbState, string) {
	_, err := exec.LookPath("adb")
	if err != nil {
		return StateMissingADB, "ADB was not found on your system path."
	}

	cmd := exec.Command("adb", "devices")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return StateMissingADB, "Failed to execute ADB command."
	}

	lines := strings.Split(string(out), "\n")
	hasDevices := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "List of devices") {
			continue
		}
		hasDevices = true
		if strings.Contains(line, "unauthorized") {
			return StateUnauthorized, "Device found but unauthorized. Check your phone."
		}
		if strings.Contains(line, "device") {
			return StateReady, "Device connected and authorized."
		}
	}

	if !hasDevices {
		return StateNoDevice, "No Android devices detected."
	}
	return StateNoDevice, "No Android devices detected."
}

func checkAdb() tea.Cmd {
	return func() tea.Msg {
		state, msg := getSystemStatus()
		return adbCheckMsg{state, msg}
	}
}

// Kills and restarts the ADB server
func restartAdbServer() { 
	_ = exec.Command("adb", "kill-server").Run()
	_ = exec.Command("adb", "start-server").Run()
}

// Renders the ADB status view
func (m model) adbView() string { 
	const (
		base    = "#1e1e2e" // Catppuccin Mocha Base
		surface = "#313244" // Catppuccin Mocha Surface0
		teal    = "#94e2d5" // Catppuccin Mocha Teal
		peach   = "#fab387" // Catppuccin Mocha Peach
		red     = "#f38ba8" // Catppuccin Mocha Red
	)

	statusColor := red
	if m.adbState == StateReady {
		statusColor = teal
	} else if m.adbState == StateUnauthorized {
		statusColor = peach
	}

	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color(teal)).
		Bold(true).
		Padding(0, 1).
		Render(" ADB DEVICE MANAGER ")

	status := lipgloss.NewStyle().
		Foreground(lipgloss.Color(statusColor)).
		Render("Status: " + m.adbMsg)

	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6adc8")).
		Render("r: refresh  •  esc: back  •  q: quit")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		header,
		"",
		status,
		"",
		hint,
	)

	card := lipgloss.NewStyle().
		Background(lipgloss.Color(surface)).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(teal)).
		Padding(2, 6).
		Render(content)

	// Center layout
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		card,
		lipgloss.WithWhitespaceBackground(lipgloss.Color(base)),
	)
}