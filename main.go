package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

type page int

const (
	pageWelcome page = iota
	pageChecks
	pagePackages
	pageProgress
)

func navigate(p page) tea.Cmd {
	return func() tea.Msg { return p }
}

type rootModel struct {
	current  page
	width    int
	height   int
	welcome  welcomeModel
	checks   checksModel
	packages packagesModel
	progress progressModel
}

func newRootModel() rootModel {
	return rootModel{
		current:  pageWelcome,
		welcome:  newWelcomeModel(),
		checks:   newChecksModel(),
		packages: newPackagesModel(),
		progress: newProgressModel(),
	}
}

func (r rootModel) Init() tea.Cmd {
	return r.welcome.init()
}

func (r rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		r.width, r.height = msg.Width, msg.Height
		r.welcome.width, r.welcome.height = msg.Width, msg.Height
		r.checks.width, r.checks.height = msg.Width, msg.Height
		r.packages.width, r.packages.height = msg.Width, msg.Height
		r.progress.width, r.progress.height = msg.Width, msg.Height
		return r, nil

	case page:
		r.current = msg
		switch msg {
		case pageChecks:
			return r, r.checks.init()
		case pagePackages:
			var cmd tea.Cmd
			r.packages, cmd = r.packages.init()
			return r, cmd
		case pageProgress:
			return r, r.progress.init()
		}
		return r, nil

	case startUninstallMsg:
		r.current = pageProgress
		var cmd tea.Cmd
		r.progress, cmd = r.progress.update(msg)
		return r, cmd
	}

	var cmd tea.Cmd
	switch r.current {
	case pageWelcome:
		r.welcome, cmd = r.welcome.update(msg)
	case pageChecks:
		r.checks, cmd = r.checks.update(msg)
	case pagePackages:
		r.packages, cmd = r.packages.update(msg)
	case pageProgress:
		r.progress, cmd = r.progress.update(msg)
	}
	return r, cmd
}

func (r rootModel) View() tea.View {
	switch r.current {
	case pageChecks:
		return r.checks.view()
	case pagePackages:
		return r.packages.view()
	case pageProgress:
		return r.progress.view()
	default:
		return r.welcome.view()
	}
}

func main() {
	if _, err := tea.NewProgram(newRootModel()).Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
