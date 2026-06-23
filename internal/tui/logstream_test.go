package tui

import (
	"os/exec"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pzep1/lazycont/internal/containercli"
)

func TestStartStreamReadsLinesInOrder(t *testing.T) {
	cmd := exec.Command("printf", "alpha\nbeta\ngamma\n")
	stream, readCmd := startStream(1, cmd)
	if stream == nil {
		t.Fatalf("expected a stream")
	}
	defer stream.stop()

	var lines []string
	for i := 0; readCmd != nil && i < 50; i++ {
		msg, ok := readCmd().(logStreamMsg)
		if !ok {
			t.Fatalf("expected logStreamMsg")
		}
		if msg.done {
			break
		}
		lines = append(lines, msg.line)
		readCmd = readStream(msg.gen, stream.ch)
	}

	want := []string{"alpha", "beta", "gamma"}
	if len(lines) != len(want) {
		t.Fatalf("got %v want %v", lines, want)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("line %d: got %q want %q", i, lines[i], want[i])
		}
	}
}

func TestLeavingLogsTabStopsStream(t *testing.T) {
	model := New(&fakeClient{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	updated, _ = updated.Update(snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{
			testContainerWithState("web", "docker.io/library/nginx:latest", "running"),
		},
	})
	// Enter the Logs tab: a stream starts.
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if cmd == nil {
		t.Fatalf("expected stream command")
	}
	updated = drainStream(t, updated, cmd)
	if updated.(Model).stream == nil {
		t.Fatalf("expected an active stream on the Logs tab")
	}
	// Move to the next tab: the stream is torn down.
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	if updated.(Model).stream != nil {
		t.Fatalf("expected stream stopped after leaving Logs tab")
	}
}

func TestScrollDetachesAndEndReattachesAutoscroll(t *testing.T) {
	model := New(&fakeClient{})
	state := withTab(model, resourceContainers, tabLogs)
	state.logFollow = true
	state.logLines = make([]string, 200)
	for i := range state.logLines {
		state.logLines[i] = "line"
	}
	state.width = 100
	state.height = 24

	state.scrollPanel(-5)
	if state.logFollow {
		t.Fatalf("scrolling up should detach autoscroll")
	}
	updated, _ := state.Update(tea.KeyMsg{Type: tea.KeyEnd})
	if !updated.(Model).logFollow {
		t.Fatalf("End should re-enable autoscroll")
	}
}
