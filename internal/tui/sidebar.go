package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type viewLayout struct {
	bodyTop         int
	bodyHeight      int
	sidebarWidth    int
	panelX          int
	sidebarContentX int
	sidebarContentY int
	listFirstRowY   int
	listDataHeight  int
	// sidebar holds the rendered stacked-panel geometry so mouse hit-testing
	// shares one source of truth with the renderer.
	sidebar sidebarRender
}

// sidebarRender holds a built stacked-panel sidebar: the rendered content lines
// plus the geometry mouse hit-testing needs. Both renderSidebar and viewLayout
// build it so the visuals and click targets never drift apart.
type sidebarRender struct {
	lines          []string             // content lines (inside the box border/padding)
	headerRow      map[resourceKind]int // content-relative row of each section header
	listFirstRow   map[resourceKind]int // content-relative row of each section's first item
	listRows       map[resourceKind]int // item rows actually shown for each section
	listDataHeight map[resourceKind]int // paging capacity used to window each section
}

type sidebarSection struct {
	kind         resourceKind
	title        string
	label        string
	columns      string
	cursor       func(Model) int
	visibleCount func(Model) int
	countLabel   func(Model) string
	renderRows   func(Model, int, int) []string
}

func sidebarSections() []sidebarSection {
	return []sidebarSection{
		{
			kind:         resourceContainers,
			title:        "Containers",
			label:        "containers",
			columns:      "state / cpu / mem",
			cursor:       func(m Model) int { return m.containerCursor },
			visibleCount: func(m Model) int { return len(m.filteredContainerIndexes()) },
			countLabel:   func(m Model) string { return m.countLabel(len(m.filteredContainerIndexes()), len(m.containers)) },
			renderRows:   func(m Model, width int, height int) []string { return m.renderContainerList(width, height) },
		},
		{
			kind:         resourceServices,
			title:        "Services",
			label:        "services",
			columns:      "state / cpu / mem",
			cursor:       func(m Model) int { return m.serviceCursor },
			visibleCount: func(m Model) int { return len(m.filteredServiceIndexes()) },
			countLabel:   func(m Model) string { return m.countLabel(len(m.filteredServiceIndexes()), len(m.project.Services)) },
			renderRows:   func(m Model, width int, height int) []string { return m.renderServiceList(width, height) },
		},
		{
			kind:         resourceImages,
			title:        "Images",
			label:        "images",
			columns:      "size  used",
			cursor:       func(m Model) int { return m.imageCursor },
			visibleCount: func(m Model) int { return len(m.filteredImageIndexes()) },
			countLabel:   func(m Model) string { return m.countLabel(len(m.filteredImageIndexes()), len(m.images)) },
			renderRows:   func(m Model, width int, height int) []string { return m.renderImageList(width, height) },
		},
		{
			kind:         resourceBuilder,
			title:        "Builder",
			label:        "builder",
			columns:      "state",
			visibleCount: func(m Model) int { return m.filteredBuilderCount() },
			countLabel:   func(m Model) string { return emptyDash(m.builder.State()) },
			renderRows:   func(m Model, width int, height int) []string { return m.renderBuilderList(width, height) },
		},
		{
			kind:         resourceVolumes,
			title:        "Volumes",
			label:        "volumes",
			columns:      "size  used",
			cursor:       func(m Model) int { return m.volumeCursor },
			visibleCount: func(m Model) int { return len(m.filteredVolumeIndexes()) },
			countLabel:   func(m Model) string { return m.countLabel(len(m.filteredVolumeIndexes()), len(m.volumes)) },
			renderRows:   func(m Model, width int, height int) []string { return m.renderVolumeList(width, height) },
		},
		{
			kind:         resourceNetworks,
			title:        "Networks",
			label:        "networks",
			columns:      "mode  used",
			cursor:       func(m Model) int { return m.networkCursor },
			visibleCount: func(m Model) int { return len(m.filteredNetworkIndexes()) },
			countLabel:   func(m Model) string { return m.countLabel(len(m.filteredNetworkIndexes()), len(m.networks)) },
			renderRows:   func(m Model, width int, height int) []string { return m.renderNetworkList(width, height) },
		},
		{
			kind:         resourceMachines,
			title:        "Machines",
			label:        "machines",
			columns:      "state",
			cursor:       func(m Model) int { return m.machineCursor },
			visibleCount: func(m Model) int { return len(m.filteredMachineIndexes()) },
			countLabel:   func(m Model) string { return m.countLabel(len(m.filteredMachineIndexes()), len(m.machines)) },
			renderRows:   func(m Model, width int, height int) []string { return m.renderMachineList(width, height) },
		},
		{
			kind:         resourceRegistries,
			title:        "Registries",
			label:        "registries",
			columns:      "user",
			cursor:       func(m Model) int { return m.registryCursor },
			visibleCount: func(m Model) int { return len(m.filteredRegistryIndexes()) },
			countLabel:   func(m Model) string { return m.countLabel(len(m.filteredRegistryIndexes()), len(m.registries)) },
			renderRows:   func(m Model, width int, height int) []string { return m.renderRegistryList(width, height) },
		},
		{
			kind:         resourceSystem,
			title:        "System",
			label:        "system",
			columns:      "status",
			visibleCount: func(m Model) int { return m.filteredSystemCount() },
			countLabel:   func(m Model) string { return emptyDash(m.system.Status) },
			renderRows:   func(m Model, width int, height int) []string { return m.renderSystemList(width, height) },
		},
	}
}

func sidebarSectionFor(kind resourceKind) (sidebarSection, bool) {
	for _, section := range sidebarSections() {
		if section.kind == kind {
			return section, true
		}
	}
	return sidebarSection{}, false
}

func sidebarOrder() []resourceKind {
	sections := sidebarSections()
	order := make([]resourceKind, 0, len(sections))
	for _, section := range sections {
		order = append(order, section.kind)
	}
	return order
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.confirm != nil || m.prompt != promptNone || m.filtering {
		return m, nil
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if m.mouseInPanel(msg) {
			m.scrollPanel(-3)
			return m, nil
		} else if m.mouseInSidebar(msg) {
			m.moveSelection(-1)
			cmd := m.ensureMainPanelCmd()
			return m, cmd
		}
		return m, nil
	case tea.MouseButtonWheelDown:
		if m.mouseInPanel(msg) {
			m.scrollPanel(3)
			return m, nil
		} else if m.mouseInSidebar(msg) {
			m.moveSelection(1)
			cmd := m.ensureMainPanelCmd()
			return m, cmd
		}
		return m, nil
	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		if tab, ok := m.mainTabAt(msg.X, msg.Y); ok {
			cmd := m.activateTab(tab)
			return m, cmd
		}
		if kind, ok := m.tabAt(msg.X, msg.Y); ok {
			if m.active != kind {
				m.active = kind
				m.clampCursors()
				m.resetPanel()
				m.statusLine = "selected " + resourceLabel(kind)
				cmd := m.ensureMainPanelCmd()
				return m, cmd
			}
			return m, nil
		}
		if kind, index, ok := m.listIndexAt(msg.X, msg.Y); ok {
			if m.active != kind {
				m.active = kind
				m.clampCursors()
			}
			if m.setActiveVisibleIndex(index) {
				m.resetPanel()
				m.statusLine = "selected " + resourceLabel(m.active)
				cmd := m.ensureMainPanelCmd()
				return m, cmd
			}
			return m, nil
		}
	}
	return m, nil
}

func (m Model) mouseInSidebar(msg tea.MouseMsg) bool {
	layout, ok := m.viewLayout()
	return ok &&
		msg.X >= 0 && msg.X < layout.sidebarWidth &&
		msg.Y >= layout.bodyTop && msg.Y < layout.bodyTop+layout.bodyHeight
}

func (m Model) mouseInPanel(msg tea.MouseMsg) bool {
	layout, ok := m.viewLayout()
	return ok &&
		msg.X >= layout.panelX && msg.X < m.width &&
		msg.Y >= layout.bodyTop && msg.Y < layout.bodyTop+layout.bodyHeight
}

// tabAt maps a click to the resource section whose header sits on that row.
// Each stacked panel header is one row, so a click anywhere along it focuses
// that section.
func (m Model) tabAt(x int, y int) (resourceKind, bool) {
	layout, ok := m.viewLayout()
	if !ok || layout.sidebarWidth <= 0 || x < layout.sidebarContentX || x >= layout.sidebarWidth-1 {
		return resourceContainers, false
	}
	row := y - layout.sidebarContentY
	if row < 0 {
		return resourceContainers, false
	}
	for kind, headerRow := range layout.sidebar.headerRow {
		if row == headerRow {
			return kind, true
		}
	}
	return resourceContainers, false
}

// listIndexAt maps a click to the section and row under it. With every section's
// list visible at once, it scans all of them (not just the focused one) so a
// click in any panel both focuses it and selects the row.
func (m Model) listIndexAt(x int, y int) (resourceKind, int, bool) {
	layout, ok := m.viewLayout()
	if !ok || x < layout.sidebarContentX || x >= layout.sidebarWidth-1 {
		return resourceContainers, 0, false
	}
	contentRow := y - layout.sidebarContentY
	if contentRow < 0 {
		return resourceContainers, 0, false
	}
	for _, section := range sidebarSections() {
		kind := section.kind
		n := layout.sidebar.listRows[kind]
		if n == 0 {
			continue
		}
		first := layout.sidebar.listFirstRow[kind]
		if contentRow < first || contentRow >= first+n {
			continue
		}
		total := m.visibleCount(kind)
		if total == 0 {
			return resourceContainers, 0, false
		}
		start := visibleStart(m.cursorFor(kind), layout.sidebar.listDataHeight[kind], total)
		index := start + (contentRow - first)
		if index < 0 || index >= total {
			return resourceContainers, 0, false
		}
		return kind, index, true
	}
	return resourceContainers, 0, false
}

// mainTabAt maps a click on the main panel's tab strip to its tab. It mirrors
// renderPanelHeader's layout exactly (labels separated by one space, dropped
// once they overflow the content width) so the click targets line up with what
// is drawn. It reports false when transient output replaces the tab strip.
func (m Model) mainTabAt(x int, y int) (mainTab, bool) {
	if m.bufferKind == bufOutput {
		return tabConfig, false
	}
	layout, ok := m.viewLayout()
	if !ok {
		return tabConfig, false
	}
	// The tab strip is the first content row of the panel box: one border row
	// sits above it, and the panel starts at panelX with a 1-col border + 1-col
	// padding before its content.
	if y != layout.bodyTop+1 {
		return tabConfig, false
	}
	contentX := layout.panelX + 2
	width := m.width - layout.panelX - 4
	col := x - contentX
	if col < 0 || col >= width {
		return tabConfig, false
	}
	used := 0
	for i, t := range m.activeTabs() {
		label := t.label()
		extra := len(label)
		if i > 0 {
			extra++ // separating space
		}
		if i > 0 && used+extra > width {
			break
		}
		start := used
		if i > 0 {
			start++ // skip the separator space
		}
		used += extra
		if col >= start && col < used {
			return t, true
		}
	}
	return tabConfig, false
}

func (m Model) viewLayout() (viewLayout, bool) {
	if m.width < 60 || m.height < 8 {
		return viewLayout{}, false
	}
	// The optional overview strip sits between the top bar and the body, so it
	// pushes the body (and every sidebar hit-test row) down by its height.
	overviewHeight := 0
	if ov := m.renderOverview(); ov != "" {
		overviewHeight = lipgloss.Height(ov)
	}
	bodyTop := 1 + overviewHeight
	bodyHeight := m.height - 2 - overviewHeight
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	sidebarWidth := m.sidebarWidth()
	// Header rows live inside the sidebar box: one border row + one (zero)
	// padding row sit above the first content line, so content starts one row
	// into the body box.
	contentY := bodyTop + 1
	sidebar := m.buildSidebar(sidebarWidth, bodyHeight-2)
	return viewLayout{
		bodyTop:         bodyTop,
		bodyHeight:      bodyHeight,
		sidebarWidth:    sidebarWidth,
		panelX:          sidebarWidth,
		sidebarContentX: 2,
		sidebarContentY: contentY,
		listFirstRowY:   contentY + sidebar.listFirstRow[m.active],
		listDataHeight:  sidebar.listDataHeight[m.active],
		sidebar:         sidebar,
	}, true
}

func sidebarWidthFor(width int) int {
	// Prefer a sidebar in the [44, 72] band, but never starve the main panel:
	// it must keep at least 28 columns. On narrow terminals the panel minimum
	// wins, so the sidebar may shrink below 44 (down to a 1-column floor).
	sidebarWidth := width / 2
	if sidebarWidth > 72 {
		sidebarWidth = 72
	}
	if sidebarWidth < 44 {
		sidebarWidth = 44
	}
	if sidebarWidth > width-28 {
		sidebarWidth = width - 28
	}
	if sidebarWidth < 1 {
		sidebarWidth = 1
	}
	return sidebarWidth
}

// sidebarWidth is the current sidebar width for the active screen mode.
// Fullscreen hides the sidebar; half mode narrows it to widen the main panel.
func (m Model) sidebarWidth() int {
	switch m.screenMode {
	case screenFull:
		return 0
	case screenHalf:
		return clampSidebar(m.width/4, m.width, 24)
	default:
		if m.sidePanelWidth > 0 && m.sidePanelWidth < 1 {
			return clampSidebar(int(float64(m.width)*m.sidePanelWidth), m.width, 30)
		}
		return sidebarWidthFor(m.width)
	}
}

func clampSidebar(w int, total int, min int) int {
	if w < min {
		w = min
	}
	if w > total-28 {
		w = total - 28
	}
	if w < 0 {
		w = 0
	}
	return w
}

func (m Model) activeCursor() int {
	return m.cursorFor(m.active)
}

// cursorFor is the list cursor for any resource section.
func (m Model) cursorFor(kind resourceKind) int {
	section, ok := sidebarSectionFor(kind)
	if !ok || section.cursor == nil {
		return 0
	}
	return section.cursor(m)
}

func (m Model) activeVisibleCount() int {
	return m.visibleCount(m.active)
}

// visibleCount is the number of (filter-respecting) rows a resource section has.
func (m Model) visibleCount(kind resourceKind) int {
	section, ok := sidebarSectionFor(kind)
	if !ok || section.visibleCount == nil {
		return 0
	}
	return section.visibleCount(m)
}

// buildSidebar lays out the lazydocker-style stacked sidebar: every resource is
// a titled section header and, unlike the old accordion, every section shows its
// list at once. Vertical space is split by sidebarRowBudgets, which gives the
// focused section the largest share. contentHeight is the room inside the
// sidebar box (border + padding excluded).
func (m Model) buildSidebar(width int, contentHeight int) sidebarRender {
	sections := sidebarSections()
	innerWidth := width - 4 // box border (2 cols) + horizontal padding (2 cols)
	if innerWidth < 1 {
		innerWidth = 1
	}
	if contentHeight < 1 {
		contentHeight = 1
	}
	budgets := m.sidebarRowBudgets(sections, contentHeight)
	out := sidebarRender{
		headerRow:      make(map[resourceKind]int, len(sections)),
		listFirstRow:   make(map[resourceKind]int, len(sections)),
		listRows:       make(map[resourceKind]int, len(sections)),
		listDataHeight: make(map[resourceKind]int, len(sections)),
	}
	for _, section := range sections {
		rows, shown := budgets[section.kind]
		if !shown {
			continue // no room for this section on a very short terminal
		}
		out.headerRow[section.kind] = len(out.lines)
		out.lines = append(out.lines, m.renderSidebarHeader(section, innerWidth, rows))
		out.listDataHeight[section.kind] = rows
		out.listFirstRow[section.kind] = len(out.lines)
		if rows < 1 {
			continue
		}
		items := section.renderRows(m, innerWidth, rows)
		out.listRows[section.kind] = len(items)
		out.lines = append(out.lines, items...)
	}
	// Never overflow the box height (extreme small terminals).
	if len(out.lines) > contentHeight {
		out.lines = out.lines[:contentHeight]
	}
	return out
}

// sidebarRowBudgets splits contentHeight across the stacked sections. Every
// shown section costs one header row; the remainder is distributed as list rows
// with the focused section weighted double. On terminals too short to show all
// sections, the focused section is guaranteed a few rows first and the rest get
// header-only billing in order until the height runs out.
func (m Model) sidebarRowBudgets(sections []sidebarSection, contentHeight int) map[resourceKind]int {
	rows := make(map[resourceKind]int, len(sections))
	shown := make(map[resourceKind]bool, len(sections))
	budget := contentHeight
	focused := m.active

	if budget > 0 {
		budget-- // focused header
		shown[focused] = true
		const focusedReserve = 4
		give := m.sidebarWant(focused)
		if give > focusedReserve {
			give = focusedReserve
		}
		if give > budget {
			give = budget
		}
		rows[focused] = give
		budget -= give
	}
	for _, section := range sections {
		kind := section.kind
		if kind == focused || budget <= 0 {
			continue
		}
		budget--
		shown[kind] = true
	}
	for budget > 0 {
		progressed := false
		for _, section := range sections {
			kind := section.kind
			if !shown[kind] {
				continue
			}
			step := 1
			if kind == focused {
				step = 2
			}
			for s := 0; s < step && budget > 0 && rows[kind] < m.sidebarWant(kind); s++ {
				rows[kind]++
				budget--
				progressed = true
			}
		}
		if !progressed {
			break
		}
	}

	out := make(map[resourceKind]int, len(sections))
	for _, section := range sections {
		if shown[section.kind] {
			out[section.kind] = rows[section.kind]
		}
	}
	return out
}

// sidebarWant is how many list rows a section could usefully fill: its visible
// item count, or one row for the focused-but-empty case (to show the "no items"
// message). Unfocused empty sections want zero rows (header only).
func (m Model) sidebarWant(kind resourceKind) int {
	count := m.visibleCount(kind)
	if count == 0 {
		if kind == m.active {
			return 1
		}
		return 0
	}
	return count
}

func (m Model) renderSidebar(width int, height int) string {
	style := activePanelStyle.Width(width - 2).Height(height - 2)
	sb := m.buildSidebar(width, height-2)
	return style.Render(strings.Join(sb.lines, "\n"))
}

// renderSidebarHeader draws one stacked-panel title bar. The focused section
// gets an accent bar + bold accent title; the rest stay muted. When the section
// shows rows (rows > 0) a muted column hint is right-aligned on the header so
// the dense rows below have labels without spending a whole row on them.
func (m Model) renderSidebarHeader(section sidebarSection, width int, rows int) string {
	if width < 4 {
		return truncate(section.title, width)
	}
	title := section.title
	if section.countLabel != nil {
		if count := section.countLabel(m); count != "" {
			title += " (" + count + ")"
		}
	}
	title = truncate(title, width-2)

	leftPlainWidth := 2 + len(title) // "▌ "/"  " marker + title
	var left string
	if section.kind == m.active {
		left = sidebarBarStyle.Render("▌ ") + sidebarActiveStyle.Render(title)
	} else {
		left = "  " + sidebarHeaderStyle.Render(title)
	}

	if rows < 1 || section.columns == "" {
		return left
	}
	gap := width - leftPlainWidth - len(section.columns)
	if gap < 1 {
		return left // no room for the column hint
	}
	return left + strings.Repeat(" ", gap) + mutedStyle.Render(section.columns)
}

func resourceLabel(kind resourceKind) string {
	if section, ok := sidebarSectionFor(kind); ok {
		return section.label
	}
	return "resource"
}

func visibleStart(cursor int, height int, total int) int {
	if height <= 0 || total <= height {
		return 0
	}
	start := cursor - height/2
	if start < 0 {
		return 0
	}
	if start+height > total {
		return total - height
	}
	return start
}
