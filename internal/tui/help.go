package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pzep1/lazycont/internal/appmeta"
)

// helpLines returns the grouped keybinding reference shown in the help overlay.
func helpLines() []string {
	groups := []struct {
		title string
		keys  [][2]string
	}{
		{"Global", [][2]string{
			{"tab / shift+tab", "switch resource pane"},
			{"←/→ or h", "previous / next resource pane"},
			{"1-9", "jump to resource pane"},
			{"[ / ]", "previous / next main-panel tab"},
			{"+ / _", "cycle screen mode (normal/half/fullscreen)"},
			{"/", "filter list"},
			{"esc", "clear filter / close output"},
			{":", "run ad-hoc container command"},
			{";", "run named custom command"},
			{"o", "open config in editor"},
			{"r", "refresh"},
			{"u", "toggle auto-refresh"},
			{"space", "open actions menu"},
			{"B", "open bulk actions menu"},
			{"?", "toggle this help"},
			{"q / ctrl+c", "quit"},
		}},
		{"Navigation & panel", [][2]string{
			{"up/k down/j", "move selection"},
			{"pgup/pgdn", "scroll panel"},
			{"home / end", "top / bottom (end re-enables log autoscroll)"},
			{"mouse", "click tabs/rows, wheel to scroll"},
		}},
		{"Main-panel tabs", [][2]string{
			{"i / enter", "Inspect tab"},
			{"l", "Logs tab (live stream)"},
			{"f", "follow logs full-screen"},
			{"ctrl+b", "VM boot logs (containers, machines)"},
		}},
		{"Containers", [][2]string{
			{"s x ctrl+r K", "start / stop / restart / kill"},
			{"e X", "exec shell / run command"},
			{"c E", "copy files / export filesystem"},
			{"w", "open first port in browser"},
			{"d p B", "delete / prune stopped / bulk actions"},
		}},
		{"Services (compose)", [][2]string{
			{"u U", "up service / up whole project"},
			{"d D", "down service / down whole project"},
			{"R", "recreate service (down + up)"},
			{"s x ctrl+r", "start / stop / restart its container"},
			{"l e i", "logs / shell / inspect its container"},
		}},
		{"Images", [][2]string{
			{"a b", "pull / build"},
			{"R N", "run / create container"},
			{"t P", "tag / push"},
			{"O L", "save / load archive"},
			{"d p", "delete / prune unused"},
		}},
		{"Volumes & networks", [][2]string{
			{"C", "create volume / network"},
			{"d p", "delete / prune unused"},
		}},
		{"Machines", [][2]string{
			{"M m S", "create / configure / set default"},
			{"l e", "logs / shell"},
			{"x d", "stop / delete"},
		}},
		{"Registries", [][2]string{
			{"g d", "log in / log out"},
		}},
		{"Builder & system", [][2]string{
			{"s x", "start / stop"},
			{"d", "delete builder"},
			{"l f", "system logs / follow"},
		}},
	}

	var lines []string
	for _, group := range groups {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, tabActiveStyle.Render(group.title))
		for _, kv := range group.keys {
			lines = append(lines, "  "+padRight(kv[0], 18)+kv[1])
		}
	}
	return lines
}

func (m Model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.stopStream()
		return m, tea.Quit
	case "?", "esc", "q", " ", "space":
		m.showHelp = false
		m.helpOffset = 0
		return m, nil
	case "up", "k":
		if m.helpOffset > 0 {
			m.helpOffset--
		}
		return m, nil
	case "down", "j":
		m.helpOffset++
		return m, nil
	case "pgup":
		m.helpOffset -= 10
		if m.helpOffset < 0 {
			m.helpOffset = 0
		}
		return m, nil
	case "pgdown":
		m.helpOffset += 10
		return m, nil
	case "home":
		m.helpOffset = 0
		return m, nil
	}
	return m, nil
}

func (m Model) renderHelpOverlay() string {
	top := topStyle.Width(m.width).Render(appmeta.Name + " help — keybindings")
	footer := footerStyle.Width(m.width).Render("↑/↓ scroll · ? or esc to close")
	bodyHeight := m.height - lipgloss.Height(top) - lipgloss.Height(footer)
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	contentWidth := m.width - 4
	textHeight := bodyHeight - 2
	if textHeight < 1 {
		textHeight = 1
	}
	body := strings.Join(helpLines(), "\n")
	rendered := renderTextWindow(body, contentWidth, textHeight, &m.helpOffset)
	box := panelStyle.Width(m.width - 2).Height(bodyHeight - 2).Render(strings.Join(rendered, "\n"))
	return lipgloss.JoinVertical(lipgloss.Left, top, box, footer)
}
