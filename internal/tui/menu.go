package tui

import (
	"context"
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
	case "ctrl+b":
		return tea.KeyMsg{Type: tea.KeyCtrlB}
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
			{"ctrl+b", "Boot logs (VM)"},
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
			{"B", "Bulk actions"},
		}
	case resourceServices:
		return "Service actions", []menuItem{
			{"u", "Up service"},
			{"U", "Up project (all)"},
			{"R", "Recreate service"},
			{"s", "Start"},
			{"x", "Stop"},
			{"ctrl+r", "Restart"},
			{"l", "Logs"},
			{"ctrl+b", "Boot logs (VM)"},
			{"e", "Shell"},
			{"i", "Inspect"},
			{"d", "Down service"},
			{"D", "Down project (all)"},
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
			{"B", "Bulk actions"},
		}
	case resourceVolumes:
		return "Volume actions", []menuItem{
			{"C", "Create volume"},
			{"i", "Inspect"},
			{"d", "Delete"},
			{"p", "Prune unused"},
			{"B", "Bulk actions"},
		}
	case resourceNetworks:
		return "Network actions", []menuItem{
			{"C", "Create network"},
			{"i", "Inspect"},
			{"d", "Delete"},
			{"p", "Prune unused"},
			{"B", "Bulk actions"},
		}
	case resourceMachines:
		return "Machine actions", []menuItem{
			{"M", "Create machine"},
			{"m", "Configure"},
			{"S", "Set default"},
			{"l", "Logs"},
			{"ctrl+b", "Boot logs (VM)"},
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

// bulkMenuItem is one entry in the bulk-commands menu. Each owns its own
// execution closure, run after the shared confirmation prompt, so the bulk
// feature defines its behavior here rather than in a central switch.
type bulkMenuItem struct {
	label  string
	prompt string
	run    func(context.Context, Model) (string, error)
}

// bulkMenuState is the overlay listing bulk actions for the active resource —
// operations that act on every item in the pane at once (lazydocker's `b`).
type bulkMenuState struct {
	title  string
	items  []bulkMenuItem
	cursor int
}

// bulkMenuItems returns the menu title and bulk actions for the active resource.
// Apple's `container` CLI backs these with `stop --all`, `kill --all`,
// `delete --all`, and the various `prune` subcommands.
func (m Model) bulkMenuItems() (string, []bulkMenuItem) {
	switch m.active {
	case resourceContainers:
		return "Bulk container actions", []bulkMenuItem{
			{"Stop all containers", "Stop all running containers?", func(ctx context.Context, m Model) (string, error) {
				return "stopped all containers", m.client.StopAll(ctx)
			}},
			{"Kill all containers", "Kill all running containers?", func(ctx context.Context, m Model) (string, error) {
				return "killed all containers", m.client.KillAll(ctx)
			}},
			{"Remove stopped containers", "Prune stopped containers?", func(ctx context.Context, m Model) (string, error) {
				return "pruned stopped containers", m.client.PruneContainers(ctx)
			}},
			{"Remove ALL containers (force)", "Force-remove ALL containers?", func(ctx context.Context, m Model) (string, error) {
				return "removed all containers", m.client.DeleteAllContainers(ctx, true)
			}},
		}
	case resourceImages:
		return "Bulk image actions", []bulkMenuItem{
			{"Prune unused images", "Prune unused images?", func(ctx context.Context, m Model) (string, error) {
				return "pruned unused images", m.client.PruneImages(ctx, false)
			}},
			{"Prune ALL images", "Prune ALL images, including base layers?", func(ctx context.Context, m Model) (string, error) {
				return "pruned all images", m.client.PruneImages(ctx, true)
			}},
		}
	case resourceVolumes:
		return "Bulk volume actions", []bulkMenuItem{
			{"Prune unused volumes", "Prune unused volumes?", func(ctx context.Context, m Model) (string, error) {
				return "pruned unused volumes", m.client.PruneVolumes(ctx)
			}},
		}
	case resourceNetworks:
		return "Bulk network actions", []bulkMenuItem{
			{"Prune unused networks", "Prune unused networks?", func(ctx context.Context, m Model) (string, error) {
				return "pruned unused networks", m.client.PruneNetworks(ctx)
			}},
		}
	}
	return "", nil
}

func (m Model) openBulkMenu() (tea.Model, tea.Cmd) {
	title, items := m.bulkMenuItems()
	if len(items) == 0 {
		m.statusLine = "no bulk actions for " + resourceLabel(m.active)
		return m, nil
	}
	m.bulkMenu = &bulkMenuState{title: title, items: items}
	m.statusLine = "bulk actions menu"
	return m, nil
}

func (m Model) handleBulkMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.stopStream()
		return m, tea.Quit
	case "esc", "q", "B", " ", "space":
		m.bulkMenu = nil
		m.statusLine = "menu closed"
		return m, nil
	case "up", "k":
		if m.bulkMenu.cursor > 0 {
			m.bulkMenu.cursor--
		}
		return m, nil
	case "down", "j":
		if m.bulkMenu.cursor < len(m.bulkMenu.items)-1 {
			m.bulkMenu.cursor++
		}
		return m, nil
	case "enter":
		if len(m.bulkMenu.items) == 0 {
			m.bulkMenu = nil
			return m, nil
		}
		item := m.bulkMenu.items[m.bulkMenu.cursor]
		m.bulkMenu = nil
		m.confirm = &pendingConfirm{label: item.prompt, run: item.run}
		m.statusLine = item.prompt
		return m, nil
	}
	return m, nil
}

func (m Model) renderBulkMenuOverlay() string {
	width := m.width / 2
	if width < 32 {
		width = 32
	}
	if width > m.width-4 {
		width = m.width - 4
	}
	lines := make([]string, 0, len(m.bulkMenu.items)+1)
	lines = append(lines, mutedStyle.Render("↑/↓ move · enter run · esc close"))
	for i, item := range m.bulkMenu.items {
		row := "  " + item.label
		if i == m.bulkMenu.cursor {
			row = selectedStyle.Width(width - 2).Render(truncate("▶ "+item.label, width-2))
		} else {
			row = truncate(row, width-2)
		}
		lines = append(lines, row)
	}
	box := activePanelStyle.Width(width - 2).Render(
		tabActiveStyle.Render(m.bulkMenu.title) + "\n\n" + strings.Join(lines, "\n"),
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func padRight(value string, width int) string {
	if len(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-len(value))
}
