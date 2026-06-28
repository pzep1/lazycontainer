package tui

import (
	"os/exec"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pzep1/lazycont/internal/containercli"
)

func TestActionsMenuOpensAndDispatchesSelection(t *testing.T) {
	model := New(&fakeClient{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 28})
	updated, _ = updated.Update(snapshotMsg{
		system:     containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{testContainer("db", "docker.io/library/postgres:17")},
	})

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	state := updated.(Model)
	if state.menu == nil {
		t.Fatalf("expected actions menu to open")
	}
	if !strings.Contains(state.View(), "Container actions") {
		t.Fatalf("menu overlay not rendered:\n%s", state.View())
	}

	// Point the cursor at the Inspect action and select it.
	idx := -1
	for i, item := range state.menu.items {
		if item.key == "i" {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatalf("inspect action missing from menu")
	}
	state.menu.cursor = idx
	updated, cmd := state.Update(tea.KeyMsg{Type: tea.KeyEnter})
	state = updated.(Model)
	if state.menu != nil {
		t.Fatalf("expected menu to close after selection")
	}
	if state.activeMainTab() != tabInspect {
		t.Fatalf("expected inspect tab after menu selection, got %v", state.activeMainTab())
	}
	if cmd == nil {
		t.Fatalf("expected inspect fetch command")
	}
}

func TestActionsMenuShortcutKeyDispatches(t *testing.T) {
	model := New(&fakeClient{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 28})
	updated, _ = updated.Update(snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{
			testContainerWithState("web", "docker.io/library/nginx:latest", "running"),
		},
	})

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	state := updated.(Model)
	if state.menu == nil {
		t.Fatalf("expected actions menu to open")
	}

	updated, cmd := state.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	state = updated.(Model)
	if state.menu != nil {
		t.Fatalf("expected menu to close after shortcut key")
	}
	if cmd == nil {
		t.Fatalf("expected stop command from menu shortcut x")
	}
}

func TestActionsMenuEscCloses(t *testing.T) {
	model := New(&fakeClient{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 28})
	updated, _ = updated.Update(snapshotMsg{
		system:     containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{testContainer("db", "docker.io/library/postgres:17")},
	})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if updated.(Model).menu != nil {
		t.Fatalf("expected menu closed on esc")
	}
}

func TestAttachCustomCommandRunsAsSubprocess(t *testing.T) {
	client := &fakeClient{}
	model := NewWithOptions(client, Options{
		CustomCommands: []CustomCommand{{
			Name:   "Shell",
			Args:   []string{"exec", "{container}", "sh"},
			Attach: true,
		}},
	})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 28})
	updated, _ = updated.Update(snapshotMsg{
		system:     containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{testContainer("db", "docker.io/library/postgres:17")},
	})

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{';'}})
	for _, r := range "Shell" {
		updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	_, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected attach subprocess command")
	}
	want := []string{"exec", "db", "sh"}
	if !reflect.DeepEqual(client.commandArgs, want) {
		t.Fatalf("attach command args = %#v, want %#v", client.commandArgs, want)
	}
}

func TestOpenInBrowserUsesFirstPublishedPort(t *testing.T) {
	var opened string
	model := NewWithOptions(&fakeClient{}, Options{
		OpenLinkCommand: func(url string) (*exec.Cmd, error) {
			opened = url
			return exec.Command("true"), nil
		},
	})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 28})
	c := testContainer("web", "docker.io/library/nginx:latest")
	c.Configuration.PublishedPorts = []containercli.Port{{HostPort: 8080, ContainerPort: 80, Proto: "tcp"}}
	updated, _ = updated.Update(snapshotMsg{
		system:     containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{c},
	})

	_, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	if cmd == nil {
		t.Fatalf("expected open-in-browser command")
	}
	cmd()
	if opened != "http://localhost:8080" {
		t.Fatalf("opened %q, want http://localhost:8080", opened)
	}
}

func TestHelpOverlayOpensScrollsAndCloses(t *testing.T) {
	model := New(&fakeClient{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 28})

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	state := updated.(Model)
	if !state.showHelp {
		t.Fatalf("expected help open")
	}
	view := state.View()
	for _, want := range []string{"keybindings", "Global", "Navigation"} {
		if !strings.Contains(view, want) {
			t.Fatalf("help overlay missing %q:\n%s", want, view)
		}
	}

	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyDown})
	if updated.(Model).helpOffset != 1 {
		t.Fatalf("expected help to scroll, offset %d", updated.(Model).helpOffset)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if updated.(Model).showHelp {
		t.Fatalf("expected help to close on second ?")
	}
}
