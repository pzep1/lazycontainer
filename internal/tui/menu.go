package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// menuItem is one selectable action in the actions menu, paired with the
// keybinding that performs it.
type menuItem struct {
	key   string
	label string
}

// actionMenu is the context-aware overlay listing the actions available for the
// active resource.
type actionMenu struct {
	title  string
	items  []menuItem
	cursor int
}

// keyMsgFor synthesises a tea.KeyMsg for a stored keybinding so a menu
// selection can be dispatched through the normal key handler.
func keyMsgFor(key string) tea.KeyMsg {
	switch key {
	case "ctrl+r":
		return tea.KeyMsg{Type: tea.KeyCtrlR}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
}

// actionMenuItems returns the menu title and actions for the active resource.
func (m Model) actionMenuItems() (string, []menuItem) {
	switch m.active {
	case resourceContainers:
		return "Container actions", []menuItem{
			{"l", "View logs (stream)"},
			{"f", "Follow logs (full screen)"},
			{"e", "Exec shell"},
			{"X", "Run command"},
			{"i", "Inspect"},
			{"w", "Open in browser"},
			{"s", "Start"},
			{"x", "Stop"},
			{"ctrl+r", "Restart"},
			{"K", "Kill"},
			{"c", "Copy files"},
			{"E", "Export filesystem"},
			{"d", "Delete"},
			{"p", "Prune stopped"},
		}
	case resourceImages:
		return "Image actions", []menuItem{
			{"a", "Pull image"},
			{"b", "Build image"},
			{"R", "Run image"},
			{"N", "Create container"},
			{"t", "Tag"},
			{"P", "Push"},
			{"O", "Save archive"},
			{"L", "Load archive"},
			{"i", "Inspect"},
			{"d", "Delete"},
			{"p", "Prune unused"},
		}
	case resourceVolumes:
		return "Volume actions", []menuItem{
			{"C", "Create volume"},
			{"i", "Inspect"},
			{"d", "Delete"},
			{"p", "Prune unused"},
		}
	case resourceNetworks:
		return "Network actions", []menuItem{
			{"C", "Create network"},
			{"i", "Inspect"},
			{"d", "Delete"},
			{"p", "Prune unused"},
		}
	case resourceMachines:
		return "Machine actions", []menuItem{
			{"M", "Create machine"},
			{"m", "Configure"},
			{"S", "Set default"},
			{"l", "Logs"},
			{"e", "Shell"},
			{"i", "Inspect"},
			{"x", "Stop"},
			{"d", "Delete"},
		}
	case resourceRegistries:
		return "Registry actions", []menuItem{
			{"g", "Log in"},
			{"i", "Inspect"},
			{"d", "Log out"},
		}
	case resourceBuilder:
		return "Builder actions", []menuItem{
			{"s", "Start"},
			{"x", "Stop"},
			{"d", "Delete"},
		}
	case resourceSystem:
		return "System actions", []menuItem{
			{"l", "Logs"},
			{"f", "Follow logs"},
			{"s", "Start services"},
			{"x", "Stop services"},
			{"i", "Inspect"},
		}
	}
	return "Actions", nil
}

func (m Model) openActionMenu() (tea.Model, tea.Cmd) {
	title, items := m.actionMenuItems()
	if len(items) == 0 {
		m.statusLine = "no actions for " + resourceLabel(m.active)
		return m, nil
	}
	m.menu = &actionMenu{title: title, items: items}
	m.statusLine = "actions menu"
	return m, nil
}

func (m Model) handleMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.stopStream()
		return m, tea.Quit
	case "esc", "q", " ", "space":
		m.menu = nil
		m.statusLine = "menu closed"
		return m, nil
	case "up", "k":
		if m.menu.cursor > 0 {
			m.menu.cursor--
		}
		return m, nil
	case "down", "j":
		if m.menu.cursor < len(m.menu.items)-1 {
			m.menu.cursor++
		}
		return m, nil
	case "enter":
		if len(m.menu.items) == 0 {
			m.menu = nil
			return m, nil
		}
		item := m.menu.items[m.menu.cursor]
		m.menu = nil
		return m.handleKey(keyMsgFor(item.key))
	}
	if item, ok := m.menuItemForKey(msg); ok {
		m.menu = nil
		return m.handleKey(keyMsgFor(item.key))
	}
	return m, nil
}

func (m Model) menuItemForKey(msg tea.KeyMsg) (menuItem, bool) {
	if m.menu == nil {
		return menuItem{}, false
	}
	key := msg.String()
	for _, item := range m.menu.items {
		if item.key == key {
			return item, true
		}
	}
	return menuItem{}, false
}

func (m Model) renderMenuOverlay() string {
	width := m.width / 2
	if width < 32 {
		width = 32
	}
	if width > m.width-4 {
		width = m.width - 4
	}
	lines := make([]string, 0, len(m.menu.items)+1)
	lines = append(lines, mutedStyle.Render("↑/↓ move · enter or key · esc close"))
	for i, item := range m.menu.items {
		key := item.key
		row := padRight(key, 7) + item.label
		if i == m.menu.cursor {
			row = selectedStyle.Width(width - 2).Render(truncate("▶ "+row, width-2))
		} else {
			row = truncate("  "+row, width-2)
		}
		lines = append(lines, row)
	}
	box := activePanelStyle.Width(width - 2).Render(
		tabActiveStyle.Render(m.menu.title) + "\n\n" + strings.Join(lines, "\n"),
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func padRight(value string, width int) string {
	if len(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-len(value))
}
