package main

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"os/exec"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// UAD types
type uadPackage struct {
	ID           string   `json:"id"`
	List         string   `json:"list"`
	Description  string   `json:"description"`
	Dependencies []string `json:"dependencies"`
	NeededBy     []string `json:"neededBy"`
	Labels       []string `json:"labels"`
	Removal      string   `json:"removal"`
}

// Filter enums and cycles
type removalFilter int

const (
	rfAll removalFilter = iota
	rfRecommended
	rfAdvanced
	rfExpert
	rfUnknown
)

var removalCycle = []struct{ label, value string }{
	{"All", ""},
	{"Recommended", "Recommended"},
	{"Advanced", "Advanced"},
	{"Expert", "Expert"},
	{"Unknown", "Unknown"},
}

type listFilter int

const (
	lfAll listFilter = iota
	lfOem
	lfAosp
	lfGoogle
	lfMisc
	lfCarrier
	lfUnknown
)

var listCycle = []struct{ label, value string }{
	{"All", ""},
	{"OEM", "Oem"},
	{"AOSP", "Aosp"},
	{"Google", "Google"},
	{"Misc", "Misc"},
	{"Carrier", "Carrier"},
	{"Unknown", "Unknown"},
}

type pkgSortMode int

const (
	sortByName pkgSortMode = iota
	sortByPackage
	sortByRemoval
)

var sortCycle = []string{"Name A→Z", "Package A→Z", "Removal"}

// Messages
type packagesLoadedMsg struct {
	packages []uadPackage
	err      error
}

type startUninstallMsg struct {
	packages []string
}

// Load packages from device using adb, correlate with UAD list for metadata
func loadPackages() tea.Cmd {
	return func() tea.Msg {
		// UAD list for metadata only — missing file is not fatal.
		uadMap := map[string]uadPackage{}
		if data, err := os.ReadFile("uad_lists.json"); err == nil {
			var all []uadPackage
			if json.Unmarshal(data, &all) == nil {
				for _, p := range all {
					uadMap[p.ID] = p
				}
			}
		}

		out, err := exec.Command("adb", "shell", "pm", "list", "packages").Output()
		if err != nil {
			return packagesLoadedMsg{err: fmt.Errorf("adb failed — is device connected? (%w)", err)}
		}

		var result []uadPackage
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimRight(line, "\r\n")
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "package:") {
				continue
			}
			id := strings.TrimPrefix(line, "package:")
			if id == "" {
				continue
			}
			if p, ok := uadMap[id]; ok {
				result = append(result, p)
			} else {
				result = append(result, uadPackage{
					ID:      id,
					List:    "Unknown",
					Removal: "Unknown",
				})
			}
		}

		if len(result) == 0 {
			return packagesLoadedMsg{err: fmt.Errorf("no packages returned by adb — check USB debugging")}
		}
		return packagesLoadedMsg{packages: result}
	}
}

// Icons
var avatarPalette = []string{
	"#F38BA8", "#FAB387", "#F9E2AF", "#A6E3A1",
	"#94E2D5", "#89B4FA", "#CBA6F7", "#F5C2E7",
	"#7ED8A8", "#89DCF3",
}

func packageAvatar(id string) string {
	name := friendlyName(id)
	initials := "??"
	words := strings.Fields(name)
	switch {
	case len(words) >= 2:
		a, b := []rune(words[0]), []rune(words[1])
		if len(a) > 0 && len(b) > 0 {
			initials = strings.ToUpper(string(a[0])) + strings.ToUpper(string(b[0]))
		}
	case len(words) == 1 && len([]rune(words[0])) >= 2:
		r := []rune(words[0])
		initials = strings.ToUpper(string(r[:2]))
	}
	h := fnv.New32a()
	h.Write([]byte(id))
	bg := avatarPalette[int(h.Sum32())%len(avatarPalette)]
	return lipgloss.NewStyle().
		Background(lipgloss.Color(bg)).
		Foreground(lipgloss.Color("#1e1e2e")).
		Bold(true).
		Padding(0, 1).
		Render(initials)
}

// Sort
var removalOrder = map[string]int{
	"Recommended": 0, "Advanced": 1, "Expert": 2, "Unknown": 3,
}

func sortPackages(pkgs []uadPackage, mode pkgSortMode) []uadPackage {
	cp := make([]uadPackage, len(pkgs))
	copy(cp, pkgs)
	switch mode {
	case sortByName:
		sort.Slice(cp, func(i, j int) bool {
			return strings.ToLower(friendlyName(cp[i].ID)) < strings.ToLower(friendlyName(cp[j].ID))
		})
	case sortByPackage:
		sort.Slice(cp, func(i, j int) bool { return cp[i].ID < cp[j].ID })
	case sortByRemoval:
		sort.Slice(cp, func(i, j int) bool {
			oi, oj := removalOrder[cp[i].Removal], removalOrder[cp[j].Removal]
			if oi != oj {
				return oi < oj
			}
			return strings.ToLower(friendlyName(cp[i].ID)) < strings.ToLower(friendlyName(cp[j].ID))
		})
	}
	return cp
}

// Model
type overlay int

const (
	overlayNone      overlay = iota
	overlayDetails           // package details
	overlayShortcuts         // keyboard shortcuts
)

type packagesModel struct {
	width         int
	height        int
	loading       bool
	loadErr       string
	packages      []uadPackage
	filtered      []uadPackage
	selected      map[string]bool
	cursor        int
	search        string
	searchActive  bool
	offset        int
	activeOverlay overlay
	status        string
	activeRemoval removalFilter
	activeList    listFilter
	sortMode      pkgSortMode
}

func newPackagesModel() packagesModel {
	return packagesModel{selected: map[string]bool{}}
}

func (m packagesModel) init() (packagesModel, tea.Cmd) {
	m.loading = true
	m.loadErr = ""
	m.activeOverlay = overlayNone
	return m, loadPackages()
}

func (m packagesModel) visibleRows() int {
	rows := (m.height - 11) / 2
	if rows < 2 {
		rows = 2
	}
	return rows
}

func (m packagesModel) applyFilter() []uadPackage {
	rfVal := removalCycle[m.activeRemoval].value
	lfVal := listCycle[m.activeList].value
	q := strings.ToLower(m.search)
	var out []uadPackage
	for _, p := range m.packages {
		if rfVal != "" && p.Removal != rfVal {
			continue
		}
		if lfVal != "" && !strings.EqualFold(p.List, lfVal) {
			continue
		}
		if q != "" {
			inID := strings.Contains(strings.ToLower(p.ID), q)
			inName := strings.Contains(strings.ToLower(friendlyName(p.ID)), q)
			inDesc := strings.Contains(strings.ToLower(p.Description), q)
			if !inID && !inName && !inDesc {
				continue
			}
		}
		out = append(out, p)
	}
	return sortPackages(out, m.sortMode)
}

func (m packagesModel) refilter() packagesModel {
	m.filtered = m.applyFilter()
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	return m.clampOffset()
}

func (m packagesModel) clampOffset() packagesModel {
	vis := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+vis {
		m.offset = m.cursor - vis + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
	return m
}

func (m packagesModel) moveCursor(delta int) packagesModel {
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	return m.clampOffset()
}

func (m packagesModel) resetAndRefilter() packagesModel {
	m.cursor = 0
	m.offset = 0
	return m.refilter()
}

func (m packagesModel) update(msg tea.Msg) (packagesModel, tea.Cmd) {
	switch msg := msg.(type) {

	case packagesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.loadErr = msg.err.Error()
			return m, nil
		}
		m.packages = msg.packages
		m = m.refilter()
		return m, nil

	// Mouse scrolling for list navigation
	case tea.MouseMsg:
		m2 := msg.Mouse()
		s := m2.String()
		if m.activeOverlay == overlayNone {
			if strings.Contains(s, "up") {
				m = m.moveCursor(-3)
			} else if strings.Contains(s, "down") {
				m = m.moveCursor(3)
			}
		}

	case tea.KeyPressMsg:
		key := msg.String()

		// Overlay keys have highest priority — they capture all input until dismissed
		if m.activeOverlay != overlayNone {
			switch key {
			case "esc", "q", "?", "d", "b":
				m.activeOverlay = overlayNone
			}
			return m, nil
		}

		// Search
		if m.searchActive {
			switch key {
			case "esc":
				m.searchActive = false
				m.search = ""
				m = m.resetAndRefilter()
			case "enter":
				m.searchActive = false
			case "backspace":
				r := []rune(m.search)
				if len(r) > 0 {
					m.search = string(r[:len(r)-1])
					m = m.refilter()
				}
			case "up":
				m = m.moveCursor(-1)
			case "down":
				m = m.moveCursor(1)
			default:
				r := []rune(key)
				if len(r) == 1 && r[0] >= 32 && r[0] != 127 {
					m.search += string(r[0])
					m = m.refilter()
				}
			}
			return m, nil
		}

		switch key {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "b":
			return m, navigate(pageChecks)
		case "/":
			m.searchActive = true
		case "esc":
			if m.search != "" {
				m.search = ""
				m = m.resetAndRefilter()
			}
		case "?":
			m.activeOverlay = overlayShortcuts
		case "d":
			if len(m.filtered) > 0 {
				m.activeOverlay = overlayDetails
			}

		// Navigation
		case "up":
			m = m.moveCursor(-1)
		case "down":
			m = m.moveCursor(1)
		case "left":
			m = m.moveCursor(-m.visibleRows())
		case "right":
			m = m.moveCursor(m.visibleRows())

		// Selection
		case " ", "space":
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				id := m.filtered[m.cursor].ID
				m.selected[id] = !m.selected[id]
			}

		case "a":
			for _, p := range m.filtered {
				m.selected[p.ID] = true
			}
		case "n":
			m.selected = map[string]bool{}

		// Filters and sort — these reset cursor and offset to avoid out-of-bounds after refiltering
		case "tab":
			m.activeRemoval = removalFilter((int(m.activeRemoval) + 1) % len(removalCycle))
			m = m.resetAndRefilter()
		case "l":
			m.activeList = listFilter((int(m.activeList) + 1) % len(listCycle))
			m = m.resetAndRefilter()
		case "s":
			m.sortMode = pkgSortMode((int(m.sortMode) + 1) % len(sortCycle))
			m = m.resetAndRefilter()

		// Refresh packages from device
		case "ctrl+r":
			m.loading = true
			m.loadErr = ""
			m.cursor = 0
			m.offset = 0
			m.selected = map[string]bool{}
			return m, loadPackages()

		// Uninstall selected packages
		case "u":
			if countSelected(m.selected) == 0 {
				m.status = "No packages selected — press [space] to select"
				return m, nil
			}
			var pkgs []string
			for id, on := range m.selected {
				if on {
					pkgs = append(pkgs, id)
				}
			}
			return m, func() tea.Msg { return startUninstallMsg{packages: pkgs} }
		}
	}

	return m, nil
}

// Shortcuts overlay
func (m packagesModel) shortcutsView() tea.View {
	lavColor := lipgloss.Color("#CBA6F7")
	blueColor := lipgloss.Color("#89B4FA")
	textColor := lipgloss.Color("#CDD6F4")
	darkColor := lipgloss.Color("#1e1e2e")

	descStyle := lipgloss.NewStyle().Foreground(textColor)

	keyColors := []string{
		"#89B4FA", // blue
		"#CBA6F7", // lavender
		"#A6E3A1", // green
		"#FAB387", // peach
		"#94E2D5", // teal
		"#F38BA8", // red
		"#F9E2AF", // yellow
		"#89DCF3", // sky
	}

	keyBadge := func(key string, colorIdx int) string {
		c := keyColors[colorIdx%len(keyColors)]
		return lipgloss.NewStyle().
			Background(lipgloss.Color(c)).
			Foreground(darkColor).
			Bold(true).
			Padding(0, 1).
			Render(key)
	}

	header := lipgloss.NewStyle().
		Bold(true).Foreground(darkColor).Background(lavColor).
		Padding(0, 4).MarginBottom(1).
		Render("KEYBOARD SHORTCUTS")

	type shortcut struct{ key, desc string }
	shortcuts := []shortcut{
		{"/", "activate search"},
		{"esc", "cancel search / clear"},
		{"↑ ↓", "navigate list"},
		{"← →", "page up / down"},
		{"space", "toggle selection"},
		{"a", "select all visible"},
		{"n", "deselect all"},
		{"tab", "cycle removal filter"},
		{"l", "cycle list filter"},
		{"s", "cycle sort order"},
		{"d", "open app details"},
		{"?", "toggle shortcuts"},
		{"ctrl+r", "refresh from device"},
		{"u", "uninstall selected"},
		{"b", "back to checks"},
		{"q", "quit"},
	}

	colW := (m.width - 16) / 2

	var rows []string
	for i := 0; i < len(shortcuts); i += 2 {
		left := keyBadge(shortcuts[i].key, i) + "  " + descStyle.Render(shortcuts[i].desc)
		right := ""
		if i+1 < len(shortcuts) {
			right = keyBadge(shortcuts[i+1].key, i+1) + "  " + descStyle.Render(shortcuts[i+1].desc)
		}
		row := lipgloss.JoinHorizontal(lipgloss.Left,
			lipgloss.NewStyle().Width(colW).Render(left),
			right,
		)
		rows = append(rows, row)
		rows = append(rows, "")
	}

	if len(rows) > 0 && rows[len(rows)-1] == "" {
		rows = rows[:len(rows)-1]
	}

	bodyBox := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#313244")).
		Padding(1, 3).
		Width(m.width - 10).
		Render(strings.Join(rows, "\n"))

	closeBtn := lipgloss.NewStyle().
		Padding(0, 3).Bold(true).
		Background(lipgloss.Color("#45475A")).
		Foreground(textColor).
		Render("[?] or [esc] CLOSE")

	content := lipgloss.JoinVertical(lipgloss.Center,
		header, bodyBox, "\n", closeBtn,
	)

	container := lipgloss.NewStyle().
		Width(m.width-2).Height(m.height-2).
		Border(lipgloss.RoundedBorder()).
		BorderForegroundBlend(lavColor, blueColor).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)

	v := tea.NewView(container)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeAllMotion
	return v
}

// Details overlay
func (m packagesModel) detailsView() tea.View {
	if m.cursor >= len(m.filtered) {
		return tea.NewView("")
	}
	p := m.filtered[m.cursor]

	lavColor := lipgloss.Color("#CBA6F7")
	passColor := lipgloss.Color("#A6E3A1")
	failColor := lipgloss.Color("#F38BA8")
	dimColor := lipgloss.Color("#45475A")
	textColor := lipgloss.Color("#CDD6F4")
	blueColor := lipgloss.Color("#89B4FA")
	orangeColor := lipgloss.Color("#FAB387")
	darkColor := lipgloss.Color("#1e1e2e")

	var removalColored string
	switch p.Removal {
	case "Recommended":
		removalColored = lipgloss.NewStyle().Foreground(passColor).Bold(true).Render("● Recommended")
	case "Advanced":
		removalColored = lipgloss.NewStyle().Foreground(orangeColor).Bold(true).Render("● Advanced")
	case "Expert":
		removalColored = lipgloss.NewStyle().Foreground(failColor).Bold(true).Render("● Expert")
	default:
		removalColored = lipgloss.NewStyle().Foreground(dimColor).Render("● " + p.Removal)
	}

	field := func(label, value string) string {
		return lipgloss.NewStyle().Foreground(lavColor).Bold(true).Render(label+": ") +
			lipgloss.NewStyle().Foreground(textColor).Render(value)
	}

	desc := strings.TrimSpace(p.Description)
	if desc == "" {
		desc = lipgloss.NewStyle().Foreground(dimColor).Italic(true).Render("No description available.")
	}

	deps, neededBy, labels := "none", "none", "none"
	if len(p.Dependencies) > 0 {
		deps = strings.Join(p.Dependencies, ", ")
	}
	if len(p.NeededBy) > 0 {
		neededBy = strings.Join(p.NeededBy, ", ")
	}
	if len(p.Labels) > 0 {
		labels = strings.Join(p.Labels, ", ")
	}

	header := lipgloss.NewStyle().
		Bold(true).Foreground(darkColor).Background(lavColor).
		Padding(0, 4).MarginBottom(1).
		Render("PACKAGE DETAILS")

	titleRow := lipgloss.JoinHorizontal(lipgloss.Center,
		packageAvatar(p.ID), "  ",
		lipgloss.NewStyle().Foreground(lavColor).Bold(true).Render(friendlyName(p.ID)),
	)

	body := lipgloss.JoinVertical(lipgloss.Left,
		titleRow,
		field("Package", p.ID),
		field("List", p.List),
		"  "+removalColored,
		"",
		lipgloss.NewStyle().Foreground(blueColor).Bold(true).Render("Description:"),
		lipgloss.NewStyle().Foreground(textColor).Render(desc),
		"",
		field("Dependencies", deps),
		field("Needed By", neededBy),
		field("Labels", labels),
	)

	bodyBox := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#313244")).
		Padding(1, 2).Width(m.width - 10).
		Render(body)

	closeBtn := lipgloss.NewStyle().
		Padding(0, 3).Bold(true).
		Background(lipgloss.Color("#45475A")).
		Foreground(textColor).
		Render("[d] or [esc] CLOSE")

	content := lipgloss.JoinVertical(lipgloss.Center,
		header, bodyBox, "\n", closeBtn,
	)

	container := lipgloss.NewStyle().
		Width(m.width-2).Height(m.height-2).
		Border(lipgloss.RoundedBorder()).
		BorderForegroundBlend(lavColor, blueColor).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)

	v := tea.NewView(container)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeAllMotion
	return v
}

// Main view
func (m packagesModel) view() tea.View {
	switch m.activeOverlay {
	case overlayDetails:
		return m.detailsView()
	case overlayShortcuts:
		return m.shortcutsView()
	}

	// Header badge
	passColor := lipgloss.Color("#A6E3A1")
	failColor := lipgloss.Color("#F38BA8")
	dimColor := lipgloss.Color("#45475A")
	blueColor := lipgloss.Color("#89B4FA")
	lavColor := lipgloss.Color("#CBA6F7")
	textColor := lipgloss.Color("#CDD6F4")
	orangeColor := lipgloss.Color("#FAB387")
	darkColor := lipgloss.Color("#1e1e2e")

	passStyle := lipgloss.NewStyle().Foreground(passColor)
	failStyle := lipgloss.NewStyle().Foreground(failColor)
	dimStyle := lipgloss.NewStyle().Foreground(dimColor)
	orangeStyle := lipgloss.NewStyle().Foreground(orangeColor)
	lavStyle := lipgloss.NewStyle().Foreground(lavColor)
	blueStyle := lipgloss.NewStyle().Foreground(blueColor)
	textStyle := lipgloss.NewStyle().Foreground(textColor)

	// Package counts by removal type for stats summary in header
	var nRec, nAdv, nExp, nUnk int
	for _, p := range m.packages {
		switch p.Removal {
		case "Recommended":
			nRec++
		case "Advanced":
			nAdv++
		case "Expert":
			nExp++
		default:
			nUnk++
		}
	}
	sel := countSelected(m.selected)

	// Header
	pkgBadge := lipgloss.NewStyle().
		Bold(true).
		Foreground(darkColor).
		Background(lavColor).
		Padding(0, 2).
		Render("PACKAGES")

	statsText := fmt.Sprintf("  %s  %s  %s  %s  %s  total: %d",
		passStyle.Render(fmt.Sprintf("● %d rec", nRec)),
		orangeStyle.Render(fmt.Sprintf("● %d adv", nAdv)),
		failStyle.Render(fmt.Sprintf("● %d exp", nExp)),
		dimStyle.Render(fmt.Sprintf("● %d unk", nUnk)),
		lavStyle.Bold(true).Render(fmt.Sprintf("%d selected", sel)),
		len(m.packages),
	)

	headerRow := lipgloss.JoinHorizontal(lipgloss.Left, pkgBadge, statsText)

	filterBtn := func(label, value, key, labelBg, valueBg, valueFg string) string {
		lbl := lipgloss.NewStyle().
			Background(lipgloss.Color(labelBg)).
			Foreground(darkColor).
			Bold(true).
			Padding(0, 1).
			Render(label + ":")
		val := lipgloss.NewStyle().
			Background(lipgloss.Color(valueBg)).
			Foreground(lipgloss.Color(valueFg)).
			Bold(true).
			Padding(0, 1).
			Render(value)
		hint := dimStyle.Render(" " + key + "  ")
		return lipgloss.JoinHorizontal(lipgloss.Left, lbl, " ", val, hint)
	}

	filterRow := lipgloss.JoinHorizontal(lipgloss.Left,
		filterBtn("Removal", removalCycle[m.activeRemoval].label, "[tab]", "#F38BA8", "#45475A", "#CBA6F7"),
		filterBtn("List", listCycle[m.activeList].label, "[l]", "#89B4FA", "#45475A", "#89DCF3"),
		filterBtn("Sort", sortCycle[m.sortMode], "[s]", "#A6E3A1", "#45475A", "#CDD6F4"),
		lipgloss.NewStyle().
			Background(lipgloss.Color("#CBA6F7")).
			Foreground(lipgloss.Color("#1e1e2e")).
			Bold(true).Padding(0, 2).
			Render("Shortcuts")+dimStyle.Render(" [?]"),
	)

	header := lipgloss.JoinVertical(lipgloss.Left, headerRow, filterRow)

	// Loading and error states
	if m.loading {
		body := lipgloss.JoinVertical(lipgloss.Center,
			header, "\n", dimStyle.Render("Loading packages from device…"))
		v := tea.NewView(wrapContainer(body, m.width, m.height))
		v.AltScreen = true
		v.MouseMode = tea.MouseModeAllMotion
		return v
	}
	if m.loadErr != "" {
		body := lipgloss.JoinVertical(lipgloss.Center,
			header, "\n",
			failStyle.Render("Error: "+m.loadErr), "\n",
			dimStyle.Render("[b] back  [ctrl+r] retry  [q] quit"),
		)
		v := tea.NewView(wrapContainer(body, m.width, m.height))
		v.AltScreen = true
		v.MouseMode = tea.MouseModeAllMotion
		return v
	}

	// Search box
	var searchContent string
	searchBorderColor := blueColor
	if m.searchActive {
		searchBorderColor = lavColor
		searchContent = blueStyle.Render("🔍︎ "+m.search+"▋") +
			dimStyle.Render(fmt.Sprintf("  %d found  [esc] cancel  [enter] done", len(m.filtered)))
	} else if m.search != "" {
		searchContent = blueStyle.Render("🔍︎ "+m.search) +
			dimStyle.Render(fmt.Sprintf("  %d found  [esc] clear", len(m.filtered)))
	} else {
		searchContent = dimStyle.Render("[/] search by name or package ID…")
	}
	searchBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(searchBorderColor).
		Padding(0, 1).
		Width(m.width - 6).
		Render(searchContent)

	// Package list
	vis := m.visibleRows()
	end := m.offset + vis
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	sep := dimStyle.Render(strings.Repeat("─", m.width-6))

	var rowsBuf strings.Builder
	for i := m.offset; i < end; i++ {
		p := m.filtered[i]
		isCursor := i == m.cursor

		var chk string
		if m.selected[p.ID] {
			chk = passStyle.Bold(true).Render("[✔]")
		} else {
			chk = dimStyle.Render("[ ]")
		}

		// Removal badge
		var remBadge string
		switch p.Removal {
		case "Recommended":
			remBadge = passStyle.Render("Rec")
		case "Advanced":
			remBadge = orangeStyle.Render("Adv")
		case "Expert":
			remBadge = failStyle.Render("Exp")
		default:
			remBadge = dimStyle.Render("Unk")
		}

		name := friendlyName(p.ID)
		maxName := m.width / 3
		if maxName < 8 {
			maxName = 8
		}
		if len([]rune(name)) > maxName {
			r := []rune(name)
			name = string(r[:maxName-1]) + "…"
		}
		pkgName := p.ID
		maxPkg := m.width/2 - 4
		if maxPkg < 8 {
			maxPkg = 8
		}
		if len(pkgName) > maxPkg {
			pkgName = pkgName[:maxPkg-1] + "…"
		}

		var row string
		if isCursor {
			row = fmt.Sprintf("▶ %s %s  %s  %s  %s",
				chk,
				packageAvatar(p.ID),
				lavStyle.Bold(true).Render(name),
				blueStyle.Render(pkgName),
				remBadge,
			)
		} else {
			row = fmt.Sprintf("  %s %s  %s  %s  %s",
				chk,
				packageAvatar(p.ID),
				textStyle.Render(name),
				dimStyle.Render(pkgName),
				remBadge,
			)
		}

		fmt.Fprintln(&rowsBuf, row)
		if i < end-1 {
			fmt.Fprintln(&rowsBuf, sep)
		}
	}

	if len(m.filtered) == 0 {
		rowsBuf.Reset()
		fmt.Fprintln(&rowsBuf, dimStyle.Italic(true).Render("  No packages match the current filters."))
	}

	// Scroll hint
	from := 0
	if len(m.filtered) > 0 {
		from = m.offset + 1
	}
	scrollHint := dimStyle.Render(fmt.Sprintf(
		"%d–%d of %d  [↑↓ j k] navigate  [ctrl+u/d] page",
		from, end, len(m.filtered),
	))

	// Description strip
	var descStrip string
	if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
		line := firstLine(m.filtered[m.cursor].Description)
		if line == "" {
			line = "No description — press [d] for details"
		}
		maxW := m.width - 8
		if len([]rune(line)) > maxW {
			r := []rune(line)
			line = string(r[:maxW-1]) + "…"
		}
		descStrip = dimStyle.Render("ℹ  ") + textStyle.Render(line)
	}

	// Status line
	var statusLine string
	if m.status != "" {
		statusLine = orangeStyle.Render(m.status)
	}

	// Compose
	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		searchBox,
		rowsBuf.String(),
		scrollHint,
		descStrip,
		statusLine,
	)

	// Outer container
	container := lipgloss.NewStyle().
		Width(m.width-2).Height(m.height-2).
		Border(lipgloss.RoundedBorder()).
		BorderForegroundBlend(lavColor, blueColor).
		Align(lipgloss.Left, lipgloss.Top).
		Render(content)

	v := tea.NewView(container)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeAllMotion
	return v
}

// Simple wrapper to ensure consistent container styling for overlays
func wrapContainer(content string, w, h int) string {
	return lipgloss.NewStyle().
		Width(w-2).Height(h-2).
		Border(lipgloss.RoundedBorder()).
		BorderForegroundBlend(lipgloss.Color("#CBA6F7"), lipgloss.Color("#89B4FA")).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)
}

func firstLine(desc string) string {
	for _, l := range strings.Split(desc, "\n") {
		l = strings.TrimSpace(l)
		if l != "" {
			return l
		}
	}
	return ""
}

func friendlyName(id string) string {
	parts := strings.Split(id, ".")
	last := parts[len(parts)-1]
	words := strings.FieldsFunc(last, func(r rune) bool {
		return r == '_' || r == '-'
	})
	var out []string
	for _, w := range words {
		if len(w) > 0 {
			out = append(out, strings.ToUpper(w[:1])+w[1:])
		}
	}
	if len(out) == 0 && len(parts) > 1 {
		w := parts[len(parts)-2]
		if len(w) > 0 {
			return strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(out, " ")
}

func countSelected(sel map[string]bool) int {
	n := 0
	for _, v := range sel {
		if v {
			n++
		}
	}
	return n
}
