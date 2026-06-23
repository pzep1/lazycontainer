package tui

import (
	"bufio"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// maxLogLines bounds the in-memory streamed log buffer.
const maxLogLines = 5000

// logStream owns a running `… logs --follow` process and the channel its output
// lines are delivered on. Each stream is tagged with a generation so stale
// messages from a previous stream are dropped after the user moves on.
type logStream struct {
	gen  int
	ch   chan logStreamMsg
	stop func()
}

// logStreamMsg carries one streamed log line, or a terminal done/err marker.
type logStreamMsg struct {
	gen  int
	line string
	done bool
	err  error
}

// startStream starts cmd, pipes its combined stdout+stderr through a scanner,
// and returns a handle plus the initial read command. The caller is responsible
// for invoking the handle's stop func when the stream is no longer needed.
func startStream(gen int, cmd *exec.Cmd) (*logStream, tea.Cmd) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, doneStreamCmd(gen, err)
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return nil, doneStreamCmd(gen, err)
	}

	ch := make(chan logStreamMsg, 256)
	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			ch <- logStreamMsg{gen: gen, line: scanner.Text()}
		}
		_ = cmd.Wait()
		close(ch)
	}()

	stop := func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}
	return &logStream{gen: gen, ch: ch, stop: stop}, readStream(gen, ch)
}

// readStream returns a command that blocks for the next line on ch.
func readStream(gen int, ch chan logStreamMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return logStreamMsg{gen: gen, done: true}
		}
		return msg
	}
}

func doneStreamCmd(gen int, err error) tea.Cmd {
	return func() tea.Msg {
		return logStreamMsg{gen: gen, done: true, err: err}
	}
}

// ensureMainPanelCmd reconciles the main panel's asynchronous content with the
// current selection and tab: it starts/refreshes the log stream on the Logs
// tab, tears the stream down elsewhere, and lazily fetches one-shot tabs.
func (m *Model) ensureMainPanelCmd() tea.Cmd {
	if m.bufferKind == bufOutput {
		m.stopStream()
		return nil
	}
	if m.activeMainTab() == tabLogs {
		return m.ensureStreamCmd()
	}
	m.stopStream()
	return m.ensureBufferCmd()
}

// stopStream tears down any running log stream.
func (m *Model) stopStream() {
	if m.stream != nil {
		m.stream.stop()
		m.stream = nil
	}
	m.logKey = ""
}

// ensureStreamCmd starts (or restarts) the follow stream for the currently
// selected log resource when needed.
func (m *Model) ensureStreamCmd() tea.Cmd {
	key := m.currentBufferKey(tabLogs)
	if key == "" {
		m.stopStream()
		m.logLines = nil
		return nil
	}
	if m.stream != nil && m.logKey == key {
		return nil
	}
	return m.startLogStreamCmd(key)
}

func (m *Model) startLogStreamCmd(key string) tea.Cmd {
	m.stopStream()
	_, cmd, err := m.followLogsCommandForSelection()
	if err != nil {
		m.logLines = []string{"Logs unavailable: " + err.Error()}
		m.logKey = key
		return nil
	}
	if cmd == nil {
		m.logLines = nil
		m.logKey = key
		return nil
	}
	m.streamGen++
	gen := m.streamGen
	m.logLines = nil
	m.logKey = key
	m.logFollow = true
	m.panelOffset = 0
	stream, readCmd := startStream(gen, cmd)
	m.stream = stream
	if stream == nil {
		m.logKey = ""
	}
	return readCmd
}

// handleLogStream appends a streamed line (or handles stream termination),
// keeping the view pinned to the bottom while autoscroll is enabled.
func (m Model) handleLogStream(msg logStreamMsg) (tea.Model, tea.Cmd) {
	if m.stream == nil || msg.gen != m.streamGen {
		return m, nil
	}
	if msg.done {
		if msg.err != nil {
			m.statusLine = "log stream error: " + msg.err.Error()
		}
		return m, nil
	}
	m.logLines = append(m.logLines, msg.line)
	if len(m.logLines) > maxLogLines {
		m.logLines = m.logLines[len(m.logLines)-maxLogLines:]
	}
	if m.logFollow {
		m.panelOffset = m.maxPanelOffset()
	}
	return m, readStream(msg.gen, m.stream.ch)
}

// logsTabContent renders the streamed log buffer for the Logs tab.
func (m Model) logsTabContent() (string, string) {
	title := "Logs"
	if m.active == resourceSystem {
		title = "System logs"
	} else if name := m.fetchTargetName(); name != "" {
		title = "Logs " + name
	}
	if len(m.logLines) == 0 {
		return title, "Following logs… (waiting for output)"
	}
	return title, strings.Join(m.logLines, "\n")
}
