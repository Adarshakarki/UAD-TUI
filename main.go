package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Application screens
const (
	stateHome         = "home"
	stateAdbCheck     = "adb"
	statePackages     = "packages"
	stateUninstalling = "uninstalling"
	stateResults      = "results"
)

// Package filter modes
const (
	filterAll      = 0
	filterSystem   = 1
	filterUser     = 2
	filterSelected = 3
)

// model holds all application state
type model struct {
	state      string
	adbState   AdbState
	adbMsg     string
	adbInfo    string
	adbAndroid string

	adbInstalled     bool
	adbServerRunning bool
	deviceConnected  bool
	usbAuthorized    bool
	shellAccess      bool

	spinner spinner.Model

	packages    []pkgItem
	cursor      int
	uadMetadata map[string]string

	// Search & filter
	searchQuery string
	searchMode  bool
	filterMode  int

	results []string

	width  int
	height int
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

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

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#94e2d5"))

	return model{
		state:       stateHome,
		adbState:    StateNoDevice,
		adbMsg:      "Scanning...",
		spinner:     s,
		uadMetadata: metadata,
		filterMode:  filterAll,
	}
}

// filteredPackages returns packages matching current search query and filter mode.
func (m model) filteredPackages() []pkgItem {
	q := strings.ToLower(m.searchQuery)
	var result []pkgItem
	for _, p := range m.packages {
		// Search filter
		if q != "" {
			if !strings.Contains(strings.ToLower(p.id), q) &&
				!strings.Contains(strings.ToLower(p.name), q) {
				continue
			}
		}
		// Tab filter
		switch m.filterMode {
		case filterSelected:
			if !p.selected {
				continue
			}
		case filterSystem:
			if !isSystemPackage(p.id) {
				continue
			}
		case filterUser:
			if isSystemPackage(p.id) {
				continue
			}
		}
		result = append(result, p)
	}
	return result
}

// isSystemPackage heuristically identifies Android system packages by prefix.
func isSystemPackage(id string) bool {
	systemPrefixes := []string{
		"com.android", "com.google", "com.samsung", "com.sec",
		"com.qualcomm", "com.mediatek", "com.huawei", "com.miui",
		"com.oneplus", "com.lge", "com.motorola", "com.sony",
		"android", "com.qti", "com.qcom",
	}
	for _, prefix := range systemPrefixes {
		if strings.HasPrefix(id, prefix) {
			return true
		}
	}
	return false
}

// Update handles all incoming messages and keyboard events.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case adbCheckMsg:
		m.adbInstalled = msg.AdbInstalled
		m.adbServerRunning = msg.AdbServerRunning
		m.deviceConnected = msg.DeviceConnected
		m.usbAuthorized = msg.UsbAuthorized
		m.shellAccess = msg.ShellAccess
		m.adbInfo = msg.DeviceInfo
		m.adbAndroid = msg.AndroidVersion
		m.adbMsg = msg.OverallMessage

		if !m.adbInstalled {
			m.adbState = StateMissingADB
		} else if !m.deviceConnected {
			m.adbState = StateNoDevice
		} else if !m.usbAuthorized || !m.shellAccess {
			m.adbState = StateUnauthorized
		} else {
			m.adbState = StateReady
		}
		return m, nil

	case packagesLoadedMsg:
		m.packages = msg
		m.state = statePackages
		m.cursor = 0
		m.filterMode = filterAll
		m.searchQuery = ""
		m.searchMode = false
		return m, nil

	case uninstallFinishedMsg:
		m.results = msg
		m.state = stateResults
		return m, nil

	case tea.KeyMsg:
		// Intercept all keystrokes when search mode is active
		if m.searchMode && m.state == statePackages {
			switch msg.String() {
			case "esc":
				m.searchMode = false
				m.searchQuery = ""
				m.cursor = 0
			case "enter":
				m.searchMode = false
				m.cursor = 0
			case "backspace":
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
					m.cursor = 0
				}
			default:
				// Accept printable characters
				if len(msg.String()) == 1 {
					m.searchQuery += msg.String()
					m.cursor = 0
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			filtered := m.filteredPackages()
			if m.cursor < len(filtered)-1 {
				m.cursor++
			}

		// Cycle through filter tabs
		case "tab":
			if m.state == statePackages {
				m.filterMode = (m.filterMode + 1) % 4
				m.cursor = 0
			}

		// Enter search mode
		case "/":
			if m.state == statePackages {
				m.searchMode = true
			}

		// Toggle package selection (operates on filtered view)
		case " ":
			if m.state == statePackages {
				filtered := m.filteredPackages()
				if m.cursor < len(filtered) {
					targetID := filtered[m.cursor].id
					for i := range m.packages {
						if m.packages[i].id == targetID {
							m.packages[i].selected = !m.packages[i].selected
							break
						}
					}
				}
			}

		case "enter":
			if m.state == stateHome {
				m.state = stateAdbCheck
				return m, checkAdb()
			}
			if m.state == stateAdbCheck && m.adbState == StateReady {
				m.adbMsg = "Scanning..."
				return m, fetchPackages(m.uadMetadata)
			}
			if m.state == statePackages {
				m.state = stateUninstalling
				return m, runUninstall(m.packages)
			}

		case "r":
			if m.state == stateAdbCheck {
				m.adbMsg = "Scanning..."
				restartAdbServer()
				return m, checkAdb()
			}

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

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
