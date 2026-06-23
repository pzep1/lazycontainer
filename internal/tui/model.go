package tui

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pz/lazycont/internal/containercli"
)

type Client interface {
	SystemStatus(context.Context) (containercli.SystemStatus, error)
	Containers(context.Context) ([]containercli.Container, error)
	Images(context.Context) ([]containercli.Image, error)
	Stats(context.Context, ...string) ([]containercli.Stat, error)
	Logs(context.Context, string, int) (string, error)
	InspectContainer(context.Context, string) (string, error)
	InspectImage(context.Context, string) (string, error)
	Start(context.Context, string) error
	Stop(context.Context, string) error
	Kill(context.Context, string) error
	DeleteContainer(context.Context, string, bool) error
	DeleteImage(context.Context, string, bool) error
	PruneImages(context.Context, bool) error
}

type resourceKind int

const (
	resourceContainers resourceKind = iota
	resourceImages
)

type panelMode int

const (
	panelDetails panelMode = iota
	panelInspect
	panelLogs
)

type confirmAction int

const (
	confirmNone confirmAction = iota
	confirmDeleteContainer
	confirmDeleteImage
	confirmPruneImages
)

type pendingConfirm struct {
	action confirmAction
	target string
	label  string
}

type Model struct {
	client Client

	width  int
	height int

	active          resourceKind
	containerCursor int
	imageCursor     int

	containers []containercli.Container
	images     []containercli.Image
	stats      []containercli.Stat
	system     containercli.SystemStatus

	panelMode   panelMode
	panelTitle  string
	panelBody   string
	panelOffset int
	showHelp    bool

	busy        string
	statusLine  string
	err         error
	lastUpdated time.Time
	confirm     *pendingConfirm
}

type snapshotMsg struct {
	system     containercli.SystemStatus
	containers []containercli.Container
	images     []containercli.Image
	stats      []containercli.Stat
	err        error
	updated    time.Time
}

type outputMsg struct {
	title string
	body  string
	err   error
}

type actionDoneMsg struct {
	message string
	err     error
}

func New(client Client) Model {
	return Model{
		client:     client,
		panelMode:  panelDetails,
		panelTitle: "Details",
		statusLine: "starting",
	}
}

func (m Model) Init() tea.Cmd {
	return m.refreshCmd()
}

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case snapshotMsg:
		m.busy = ""
		m.system = msg.system
		m.containers = msg.containers
		m.images = msg.images
		m.stats = msg.stats
		m.err = msg.err
		m.lastUpdated = msg.updated
		m.clampCursors()
		if msg.err != nil {
			m.statusLine = msg.err.Error()
		} else {
			m.statusLine = "refreshed"
		}
		return m, nil
	case outputMsg:
		m.busy = ""
		m.err = msg.err
		if msg.err != nil {
			m.statusLine = msg.err.Error()
			return m, nil
		}
		m.panelTitle = msg.title
		m.panelBody = msg.body
		m.panelOffset = 0
		m.statusLine = "loaded " + strings.ToLower(msg.title)
		return m, nil
	case actionDoneMsg:
		m.busy = "refreshing"
		m.confirm = nil
		m.panelMode = panelDetails
		m.panelOffset = 0
		m.err = msg.err
		if msg.err != nil {
			m.busy = ""
			m.statusLine = msg.err.Error()
			return m, nil
		}
		m.statusLine = msg.message
		return m, m.refreshCmd()
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if m.confirm != nil {
		return m.handleConfirmKey(key)
	}

	switch key {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "?":
		m.showHelp = !m.showHelp
		return m, nil
	case "tab":
		if m.active == resourceContainers {
			m.active = resourceImages
		} else {
			m.active = resourceContainers
		}
		m.resetPanel()
		return m, nil
	case "r":
		m.busy = "refreshing"
		m.statusLine = "refreshing"
		return m, m.refreshCmd()
	case "up", "k":
		m.moveSelection(-1)
		return m, nil
	case "down", "j":
		m.moveSelection(1)
		return m, nil
	case "home":
		m.panelOffset = 0
		return m, nil
	case "end":
		m.panelOffset = m.maxPanelOffset()
		return m, nil
	case "pgup":
		m.scrollPanel(-m.panelPageSize())
		return m, nil
	case "pgdown":
		m.scrollPanel(m.panelPageSize())
		return m, nil
	case "enter", "i":
		return m.inspectSelected()
	case "l":
		return m.logsSelected()
	case "s":
		return m.lifecycleSelected("starting", "started", func(ctx context.Context, id string) error {
			return m.client.Start(ctx, id)
		})
	case "x":
		return m.lifecycleSelected("stopping", "stopped", func(ctx context.Context, id string) error {
			return m.client.Stop(ctx, id)
		})
	case "K":
		return m.lifecycleSelected("killing", "killed", func(ctx context.Context, id string) error {
			return m.client.Kill(ctx, id)
		})
	case "d":
		m.prepareDelete()
		return m, nil
	case "p":
		if m.active == resourceImages {
			m.confirm = &pendingConfirm{action: confirmPruneImages, label: "Prune unused images?"}
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleConfirmKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y", "Y", "enter":
		cmd := m.confirmCmd(*m.confirm)
		m.busy = "running"
		m.statusLine = "running action"
		return m, cmd
	case "n", "N", "esc", "q":
		m.confirm = nil
		m.statusLine = "cancelled"
		return m, nil
	}
	return m, nil
}

func (m Model) refreshCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var errs []error
		msg := snapshotMsg{updated: time.Now()}

		if status, err := m.client.SystemStatus(ctx); err != nil {
			errs = append(errs, err)
		} else {
			msg.system = status
		}
		if containers, err := m.client.Containers(ctx); err != nil {
			errs = append(errs, err)
		} else {
			sort.Slice(containers, func(i, j int) bool {
				if containers[i].State() == containers[j].State() {
					return containers[i].Name() < containers[j].Name()
				}
				return containers[i].State() == "running"
			})
			msg.containers = containers
		}
		if images, err := m.client.Images(ctx); err != nil {
			errs = append(errs, err)
		} else {
			sort.Slice(images, func(i, j int) bool {
				return images[i].Name() < images[j].Name()
			})
			msg.images = images
		}
		if stats, err := m.client.Stats(ctx); err == nil {
			msg.stats = stats
		}
		msg.err = joinErrors(errs)
		return msg
	}
}

func (m Model) inspectSelected() (tea.Model, tea.Cmd) {
	switch m.active {
	case resourceContainers:
		container, ok := m.selectedContainer()
		if !ok {
			return m, nil
		}
		id := container.Name()
		m.busy = "inspecting"
		m.panelMode = panelInspect
		return m, func() tea.Msg {
			body, err := m.client.InspectContainer(context.Background(), id)
			return outputMsg{title: "Inspect " + id, body: body, err: err}
		}
	case resourceImages:
		image, ok := m.selectedImage()
		if !ok {
			return m, nil
		}
		name := image.Name()
		m.busy = "inspecting"
		m.panelMode = panelInspect
		return m, func() tea.Msg {
			body, err := m.client.InspectImage(context.Background(), name)
			return outputMsg{title: "Inspect " + name, body: body, err: err}
		}
	}
	return m, nil
}

func (m Model) logsSelected() (tea.Model, tea.Cmd) {
	container, ok := m.selectedContainer()
	if m.active != resourceContainers || !ok {
		return m, nil
	}
	id := container.Name()
	m.busy = "loading logs"
	m.panelMode = panelLogs
	return m, func() tea.Msg {
		body, err := m.client.Logs(context.Background(), id, 200)
		if strings.TrimSpace(body) == "" && err == nil {
			body = "No logs returned."
		}
		return outputMsg{title: "Logs " + id, body: body, err: err}
	}
}

func (m Model) lifecycleSelected(busy string, done string, action func(context.Context, string) error) (tea.Model, tea.Cmd) {
	if m.active != resourceContainers {
		return m, nil
	}
	container, ok := m.selectedContainer()
	if !ok {
		return m, nil
	}
	id := container.Name()
	m.busy = busy
	m.statusLine = busy + " " + id
	return m, func() tea.Msg {
		err := action(context.Background(), id)
		return actionDoneMsg{message: done + " " + id, err: err}
	}
}

func (m *Model) prepareDelete() {
	switch m.active {
	case resourceContainers:
		container, ok := m.selectedContainer()
		if ok {
			id := container.Name()
			m.confirm = &pendingConfirm{action: confirmDeleteContainer, target: id, label: "Delete container " + id + "?"}
		}
	case resourceImages:
		image, ok := m.selectedImage()
		if ok {
			name := image.Name()
			m.confirm = &pendingConfirm{action: confirmDeleteImage, target: name, label: "Delete image " + name + "?"}
		}
	}
}

func (m Model) confirmCmd(confirm pendingConfirm) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		switch confirm.action {
		case confirmDeleteContainer:
			err := m.client.DeleteContainer(ctx, confirm.target, false)
			return actionDoneMsg{message: "deleted container " + confirm.target, err: err}
		case confirmDeleteImage:
			err := m.client.DeleteImage(ctx, confirm.target, false)
			return actionDoneMsg{message: "deleted image " + confirm.target, err: err}
		case confirmPruneImages:
			err := m.client.PruneImages(ctx, false)
			return actionDoneMsg{message: "pruned unused images", err: err}
		default:
			return actionDoneMsg{err: errors.New("unknown action")}
		}
	}
}

func (m *Model) moveSelection(delta int) {
	switch m.active {
	case resourceContainers:
		m.containerCursor += delta
	case resourceImages:
		m.imageCursor += delta
	}
	m.clampCursors()
	m.resetPanel()
}

func (m *Model) clampCursors() {
	m.containerCursor = clamp(m.containerCursor, 0, len(m.containers)-1)
	m.imageCursor = clamp(m.imageCursor, 0, len(m.images)-1)
}

func (m *Model) resetPanel() {
	m.panelMode = panelDetails
	m.panelTitle = "Details"
	m.panelBody = ""
	m.panelOffset = 0
}

func (m *Model) scrollPanel(delta int) {
	m.panelOffset += delta
	m.clampPanelOffset()
}

func (m Model) selectedContainer() (containercli.Container, bool) {
	if len(m.containers) == 0 || m.containerCursor < 0 || m.containerCursor >= len(m.containers) {
		return containercli.Container{}, false
	}
	return m.containers[m.containerCursor], true
}

func (m Model) selectedImage() (containercli.Image, bool) {
	if len(m.images) == 0 || m.imageCursor < 0 || m.imageCursor >= len(m.images) {
		return containercli.Image{}, false
	}
	return m.images[m.imageCursor], true
}

func joinErrors(errs []error) error {
	filtered := make([]string, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			filtered = append(filtered, err.Error())
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return errors.New(strings.Join(filtered, "; "))
}

func clamp(value int, min int, max int) int {
	if max < min {
		return min
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "lazycont"
	}
	if m.height < 8 || m.width < 60 {
		return "lazycont needs a terminal of at least 60x8"
	}

	top := m.renderTopBar()
	footer := m.renderFooter()
	bodyHeight := m.height - lipgloss.Height(top) - lipgloss.Height(footer)
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	sidebarWidth := m.width / 2
	if sidebarWidth < 44 {
		sidebarWidth = 44
	}
	if sidebarWidth > 72 {
		sidebarWidth = 72
	}
	if sidebarWidth > m.width-28 {
		sidebarWidth = m.width - 28
	}
	panelWidth := m.width - sidebarWidth

	sidebar := m.renderSidebar(sidebarWidth, bodyHeight)
	panel := m.renderPanel(panelWidth, bodyHeight)
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, panel)

	return lipgloss.JoinVertical(lipgloss.Left, top, body, footer)
}

var (
	colorText       = lipgloss.Color("252")
	colorMuted      = lipgloss.Color("244")
	colorPanel      = lipgloss.Color("238")
	colorActive     = lipgloss.Color("39")
	colorGreen      = lipgloss.Color("42")
	colorRed        = lipgloss.Color("203")
	colorYellow     = lipgloss.Color("214")
	colorBackground = lipgloss.Color("235")

	topStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorBackground).
			Padding(0, 1)
	footerStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Background(colorBackground).
			Padding(0, 1)
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorPanel).
			Padding(0, 1)
	activePanelStyle = panelStyle.Copy().
				BorderForeground(colorActive)
	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("57")).
			Bold(true)
	mutedStyle   = lipgloss.NewStyle().Foreground(colorMuted)
	errorStyle   = lipgloss.NewStyle().Foreground(colorRed)
	runningStyle = lipgloss.NewStyle().Foreground(colorGreen)
	stoppedStyle = lipgloss.NewStyle().Foreground(colorYellow)
)

func (m Model) renderTopBar() string {
	status := m.system.Status
	if status == "" {
		status = "unknown"
	}
	left := fmt.Sprintf("lazycont | apple container: %s", status)
	if m.busy != "" {
		left += " | " + m.busy
	}
	right := ""
	if !m.lastUpdated.IsZero() {
		right = "updated " + m.lastUpdated.Format("15:04:05")
	}
	line := fitColumns(left, right, m.width-2)
	return topStyle.Width(m.width).Render(line)
}

func (m Model) renderFooter() string {
	if m.confirm != nil {
		return topStyle.Width(m.width).Foreground(colorYellow).Render(m.confirm.label + "  y/enter confirm, n/esc cancel")
	}
	if m.showHelp {
		help := "tab switch | r refresh | i inspect | l logs | s start | x stop | K kill | d delete | p prune images | pgup/pgdown scroll | q quit"
		return footerStyle.Width(m.width).Render(truncate(help, m.width-2))
	}
	status := m.statusLine
	if status == "" {
		status = "? help"
	}
	if m.err != nil {
		return footerStyle.Width(m.width).Foreground(colorRed).Render(truncate(status, m.width-2))
	}
	return footerStyle.Width(m.width).Render(truncate(status+" | ? help", m.width-2))
}

func (m Model) renderSidebar(width int, height int) string {
	style := activePanelStyle.Width(width - 2).Height(height - 2)
	title := m.renderTabs()
	var lines []string
	lines = append(lines, title, "")
	if m.active == resourceContainers {
		lines = append(lines, m.renderContainerList(width-4, height-5)...)
	} else {
		lines = append(lines, m.renderImageList(width-4, height-5)...)
	}
	return style.Render(strings.Join(lines, "\n"))
}

func (m Model) renderTabs() string {
	containers := fmt.Sprintf("containers %d", len(m.containers))
	images := fmt.Sprintf("images %d", len(m.images))
	if m.active == resourceContainers {
		containers = selectedStyle.Render(" " + containers + " ")
		images = mutedStyle.Render(" " + images + " ")
	} else {
		containers = mutedStyle.Render(" " + containers + " ")
		images = selectedStyle.Render(" " + images + " ")
	}
	return containers + " " + images
}

func (m Model) renderContainerList(width int, height int) []string {
	if len(m.containers) == 0 {
		return []string{mutedStyle.Render("No containers found.")}
	}
	rows := []string{mutedStyle.Render(fitColumns("name", "state", width))}
	start := visibleStart(m.containerCursor, height-1, len(m.containers))
	end := start + height - 1
	if end > len(m.containers) {
		end = len(m.containers)
	}
	now := effectiveNow(m.lastUpdated)
	for idx := start; idx < end; idx++ {
		container := m.containers[idx]
		name := truncate(container.Name(), 22)
		meta := fmt.Sprintf("%s  %s", container.State(), container.CreatedAgo(now))
		line := fitColumns(name, meta, width)
		if idx == m.containerCursor {
			line = selectedStyle.Width(width).Render(truncate(line, width))
		} else {
			line = colorState(line, container.State())
		}
		rows = append(rows, line)
	}
	return rows
}

func (m Model) renderImageList(width int, height int) []string {
	if len(m.images) == 0 {
		return []string{mutedStyle.Render("No images found.")}
	}
	rows := []string{mutedStyle.Render(fitColumns("image", "size", width))}
	start := visibleStart(m.imageCursor, height-1, len(m.images))
	end := start + height - 1
	if end > len(m.images) {
		end = len(m.images)
	}
	for idx := start; idx < end; idx++ {
		image := m.images[idx]
		name := truncate(image.Name(), 34)
		line := fitColumns(name, image.Size(), width)
		if idx == m.imageCursor {
			line = selectedStyle.Width(width).Render(truncate(line, width))
		}
		rows = append(rows, line)
	}
	return rows
}

func (m Model) renderPanel(width int, height int) string {
	style := panelStyle.Width(width - 2).Height(height - 2)
	title, body := m.panelContent()
	contentWidth := width - 4
	textHeight := panelTextHeight(height)

	lines := []string{title, ""}
	renderedBody := renderTextWindow(body, contentWidth, textHeight, &m.panelOffset)
	lines = append(lines, renderedBody...)
	return style.Render(strings.Join(lines, "\n"))
}

func (m Model) panelContent() (string, string) {
	if m.panelMode != panelDetails {
		return m.panelTitle, m.panelBody
	}
	now := effectiveNow(m.lastUpdated)
	switch m.active {
	case resourceContainers:
		container, ok := m.selectedContainer()
		if !ok {
			return "Details", "No container selected."
		}
		lines := container.DetailLines(now)
		if statLines := m.statLines(container.Name()); len(statLines) > 0 {
			lines = append(lines, "", "Stats")
			lines = append(lines, statLines...)
		}
		return "Details " + container.Name(), strings.Join(lines, "\n")
	case resourceImages:
		image, ok := m.selectedImage()
		if !ok {
			return "Details", "No image selected."
		}
		return "Details " + image.Name(), strings.Join(image.DetailLines(now), "\n")
	default:
		return "Details", ""
	}
}

func (m Model) statLines(containerID string) []string {
	for _, stat := range m.stats {
		if !statMatches(stat, containerID) {
			continue
		}
		keys := make([]string, 0, len(stat))
		for key := range stat {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		lines := make([]string, 0, len(keys))
		for _, key := range keys {
			lines = append(lines, fmt.Sprintf("  %s: %v", key, stat[key]))
		}
		return lines
	}
	return nil
}

func statMatches(stat containercli.Stat, containerID string) bool {
	if containerID == "" {
		return false
	}
	for _, key := range []string{"id", "ID", "container", "Container", "containerID", "containerId", "name", "Name"} {
		value, ok := stat[key]
		if !ok {
			continue
		}
		if strings.Contains(fmt.Sprint(value), containerID) || strings.Contains(containerID, fmt.Sprint(value)) {
			return true
		}
	}
	return false
}

func renderTextWindow(body string, width int, height int, offset *int) []string {
	rawLines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		lines = append(lines, truncate(strings.ReplaceAll(line, "\t", "    "), width))
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	maxOffset := len(lines) - height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if *offset > maxOffset {
		*offset = maxOffset
	}
	if *offset < 0 {
		*offset = 0
	}
	end := *offset + height
	if end > len(lines) {
		end = len(lines)
	}
	visible := append([]string(nil), lines[*offset:end]...)
	for len(visible) < height {
		visible = append(visible, "")
	}
	return visible
}

func (m Model) panelPageSize() int {
	return panelTextHeight(m.height - 2)
}

func (m *Model) clampPanelOffset() {
	maxOffset := m.maxPanelOffset()
	if m.panelOffset < 0 {
		m.panelOffset = 0
	}
	if m.panelOffset > maxOffset {
		m.panelOffset = maxOffset
	}
}

func (m Model) maxPanelOffset() int {
	_, body := m.panelContent()
	lineCount := len(strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n"))
	maxOffset := lineCount - m.panelPageSize()
	if maxOffset < 0 {
		return 0
	}
	return maxOffset
}

func panelTextHeight(panelHeight int) int {
	height := panelHeight - 6
	if height < 1 {
		return 1
	}
	return height
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

func effectiveNow(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now()
	}
	return value
}

func colorState(line string, state string) string {
	switch state {
	case "running":
		return runningStyle.Render(line)
	case "stopped", "exited":
		return stoppedStyle.Render(line)
	default:
		return line
	}
}

func fitColumns(left string, right string, width int) string {
	if width <= 0 {
		return ""
	}
	left = stripNewline(left)
	right = stripNewline(right)
	if len(left)+len(right)+1 > width {
		remaining := width - len(right) - 1
		if remaining < width/3 {
			remaining = width / 3
		}
		left = truncate(left, remaining)
	}
	spaces := width - len(left) - len(right)
	if spaces < 1 {
		spaces = 1
	}
	line := left + strings.Repeat(" ", spaces) + right
	return truncate(line, width)
}

func truncate(value string, width int) string {
	value = stripNewline(value)
	if width <= 0 {
		return ""
	}
	if len(value) <= width {
		return value
	}
	if width <= 3 {
		return value[:width]
	}
	return value[:width-3] + "..."
}

func stripNewline(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	return value
}
