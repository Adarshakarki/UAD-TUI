package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Different states of the application
const (
	stateHome         = "home"
	stateAdbCheck     = "adb"
	statePackages     = "packages"
	stateUninstalling = "uninstalling"
	stateResults      = "results"
)

// model holds the application's state
type model struct {
	state    string
	adbState AdbState
	adbMsg   string

	// Package List State
	packages    []pkgItem
	cursor      int
	uadMetadata map[string]string

	// Results
	results []string

	// Dynamic layout
	width  int
	height int
}

func (m model) Init() tea.Cmd {
	return nil
}

// Initializes the model with ADB metadata and default states.
func initialModel() model {
	metadata := make(map[string]string)
	data, err := os.ReadFile("uad_lists.json")
	if err == nil {
		var items []struct {
			ID          string `json:"id"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal(data, &items); err == nil {
			for _, item := range items {
				name := strings.Split(item.Description, "\n")[0]
				if name == "" {
					name = item.ID
				}
				metadata[item.ID] = name
			}
		}
	}

	return model{
		state:       stateHome,
		adbState:    StateNoDevice,
		adbMsg:      "Scanning...",
		uadMetadata: metadata,
	}
}

// Update handles messages and updates the model's state.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// Update stored window dimensions on resize.
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	// Handle ADB check results and transition to package listing if ready.
	case adbCheckMsg:
		m.adbState = msg.state
		m.adbMsg = msg.msg
		if m.adbState == StateReady && m.state == stateAdbCheck {
			return m, fetchPackages(m.uadMetadata)
		}
		return m, nil
	// Populate package list and transition to package view.
	case packagesLoadedMsg:
		m.packages = msg
		m.state = statePackages
		m.cursor = 0
		return m, nil
	// Display results after batch uninstall.
	case uninstallFinishedMsg:
		m.results = msg
		m.state = stateResults
		return m, nil
	// Handle keyboard input for navigation and actions.
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		// Move cursor up in lists.
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		// Move cursor down in lists.
		case "down", "j":
			if m.cursor < len(m.packages)-1 {
				m.cursor++
			}
		// Toggle package selection.
		case " ":
			if m.state == statePackages {
				m.packages[m.cursor].selected = !m.packages[m.cursor].selected
			}
		// Progress through states or initiate actions.
		case "enter":
			if m.state == stateHome {
				m.state = stateAdbCheck
				return m, checkAdb()
			}
			if m.state == statePackages {
				m.state = stateUninstalling
				return m, runUninstall(m.packages)
			}
		// Refresh ADB
		case "r":

			if m.state == stateAdbCheck {
				restartAdbServer()
				return m, checkAdb()
			}

		// Navigation back
		case "b", "esc":
			if m.state == stateResults || m.state == statePackages || m.state == stateAdbCheck {
				m.state = stateHome
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	switch m.state {
	case stateHome:
		return m.homeView()
	case stateAdbCheck:
		return m.adbView()
	case statePackages:
		return m.packageListView()
	case stateUninstalling:
		return m.processingView()
	case stateResults:
		return m.resultsView()
	}
	return "Unknown state"
}

// render home screen
func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if err := p.Start(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
