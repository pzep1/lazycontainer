package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

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
		if index, ok := m.listIndexAt(msg.X, msg.Y); ok {
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
func (m Model) listIndexAt(x int, y int) (int, bool) {
	layout, ok := m.viewLayout()
	if !ok || x < layout.sidebarContentX || x >= layout.sidebarWidth-1 {
		return 0, false
	}
	row := y - layout.listFirstRowY
	if row < 0 || row >= layout.sidebar.listRows {
		return 0, false
	}

	total := m.activeVisibleCount()
	if total == 0 {
		return 0, false
	}
	start := visibleStart(m.activeCursor(), layout.listDataHeight, total)
	index := start + row
	if index < 0 || index >= total {
		return 0, false
	}
	return index, true
}
func (m Model) viewLayout() (viewLayout, bool) {
	if m.width < 60 || m.height < 8 {
		return viewLayout{}, false
	}
	bodyHeight := m.height - 2
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	sidebarWidth := m.sidebarWidth()
	// Header rows live inside the sidebar box: one border row + one (zero)
	// padding row sit above the first content line, so content starts at Y=2.
	const contentY = 2
	sidebar := m.buildSidebar(sidebarWidth, bodyHeight-2)
	return viewLayout{
		bodyTop:         1,
		bodyHeight:      bodyHeight,
		sidebarWidth:    sidebarWidth,
		panelX:          sidebarWidth,
		sidebarContentX: 2,
		sidebarContentY: contentY,
		listFirstRowY:   contentY + sidebar.listFirstRow,
		listDataHeight:  sidebar.listDataHeight,
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
func sidebarOrder() []resourceKind {
	return []resourceKind{
		resourceContainers, resourceServices, resourceImages, resourceBuilder,
		resourceVolumes, resourceNetworks, resourceMachines, resourceRegistries,
		resourceSystem,
	}
}

// buildSidebar lays out the lazydocker-style stacked sidebar: every resource is
// a titled section header, and the focused section expands to show its list.
// contentHeight is the room inside the sidebar box (border + padding excluded).
func (m Model) buildSidebar(width int, contentHeight int) sidebarRender {
	order := sidebarOrder()
	innerWidth := width - 4 // box border (2 cols) + horizontal padding (2 cols)
	if innerWidth < 1 {
		innerWidth = 1
	}
	if contentHeight < 1 {
		contentHeight = 1
	}
	// Every section costs one header row; the focused section gets the rest.
	listRegion := contentHeight - len(order)
	if listRegion < 0 {
		listRegion = 0
	}
	out := sidebarRender{
		headerRow:      make(map[resourceKind]int, len(order)),
		listFirstRow:   -1,
		listDataHeight: listRegion - 1,
	}
	if out.listDataHeight < 0 {
		out.listDataHeight = 0
	}
	for _, kind := range order {
		out.headerRow[kind] = len(out.lines)
		out.lines = append(out.lines, m.renderSidebarHeader(kind, innerWidth))
		if kind != m.active || listRegion < 1 {
			continue
		}
		items := m.renderActiveList(kind, innerWidth, listRegion)
		for idx, row := range items {
			if idx == 1 {
				out.listFirstRow = len(out.lines)
			}
			out.lines = append(out.lines, row)
		}
		if len(items) > 1 {
			out.listRows = len(items) - 1
		}
		if out.listFirstRow == -1 {
			out.listFirstRow = len(out.lines)
		}
	}
	if out.listFirstRow == -1 {
		out.listFirstRow = len(out.lines)
	}
	// Never overflow the box height (extreme small terminals).
	if len(out.lines) > contentHeight {
		out.lines = out.lines[:contentHeight]
	}
	return out
}
func (m Model) renderSidebar(width int, height int) string {
	style := activePanelStyle.Width(width - 2).Height(height - 2)
	sb := m.buildSidebar(width, height-2)
	return style.Render(strings.Join(sb.lines, "\n"))
}

// renderSidebarHeader draws one stacked-panel title bar. The focused section
// gets an accent bar + bold accent title; the rest stay muted.
func (m Model) renderSidebarHeader(kind resourceKind, width int) string {
	if width < 4 {
		return truncate(sidebarTitle(kind), width)
	}
	title := sidebarTitle(kind)
	if count := m.sidebarCountLabel(kind); count != "" {
		title += " (" + count + ")"
	}
	body := truncate(title, width-2)
	if kind == m.active {
		return sidebarBarStyle.Render("▌ ") + sidebarActiveStyle.Render(body)
	}
	return "  " + sidebarHeaderStyle.Render(body)
}
func sidebarTitle(kind resourceKind) string {
	switch kind {
	case resourceContainers:
		return "Containers"
	case resourceServices:
		return "Services"
	case resourceImages:
		return "Images"
	case resourceBuilder:
		return "Builder"
	case resourceVolumes:
		return "Volumes"
	case resourceNetworks:
		return "Networks"
	case resourceMachines:
		return "Machines"
	case resourceRegistries:
		return "Registries"
	case resourceSystem:
		return "System"
	default:
		return "Resource"
	}
}

// sidebarCountLabel is the parenthetical shown beside a section title: a count
// (filtered/total when a filter hides rows) for list resources, or a state for
// the singleton builder/system sections.
func (m Model) sidebarCountLabel(kind resourceKind) string {
	switch kind {
	case resourceContainers:
		return m.countLabel(len(m.filteredContainerIndexes()), len(m.containers))
	case resourceServices:
		return m.countLabel(len(m.filteredServiceIndexes()), len(m.project.Services))
	case resourceImages:
		return m.countLabel(len(m.filteredImageIndexes()), len(m.images))
	case resourceBuilder:
		return emptyDash(m.builder.State())
	case resourceVolumes:
		return m.countLabel(len(m.filteredVolumeIndexes()), len(m.volumes))
	case resourceNetworks:
		return m.countLabel(len(m.filteredNetworkIndexes()), len(m.networks))
	case resourceMachines:
		return m.countLabel(len(m.filteredMachineIndexes()), len(m.machines))
	case resourceRegistries:
		return m.countLabel(len(m.filteredRegistryIndexes()), len(m.registries))
	case resourceSystem:
		return emptyDash(m.system.Status)
	default:
		return ""
	}
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
