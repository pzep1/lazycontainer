package tui

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
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
	Volumes(context.Context) ([]containercli.Volume, error)
	Networks(context.Context) ([]containercli.NetworkResource, error)
	Machines(context.Context) ([]containercli.Machine, error)
	Stats(context.Context, ...string) ([]containercli.Stat, error)
	Logs(context.Context, string, int) (string, error)
	FollowLogsCommand(string, int) (*exec.Cmd, error)
	MachineLogs(context.Context, string, int) (string, error)
	FollowMachineLogsCommand(string, int) (*exec.Cmd, error)
	InspectContainer(context.Context, string) (string, error)
	InspectImage(context.Context, string) (string, error)
	InspectVolume(context.Context, string) (string, error)
	InspectNetwork(context.Context, string) (string, error)
	InspectMachine(context.Context, string) (string, error)
	ShellCommand(string, string) (*exec.Cmd, error)
	Exec(context.Context, string, string) (string, error)
	MachineShellCommand(string) (*exec.Cmd, error)
	CreateMachine(context.Context, string, string) error
	SetDefaultMachine(context.Context, string) error
	PullImage(context.Context, string) error
	RunImage(context.Context, string, string) error
	CreateContainer(context.Context, string, string) error
	BuildImage(context.Context, string, string) error
	TagImage(context.Context, string, string) error
	PushImage(context.Context, string) error
	SaveImage(context.Context, string, string) error
	LoadImage(context.Context, string) error
	Copy(context.Context, string, string) error
	ExportContainer(context.Context, string, string) error
	Start(context.Context, string) error
	Stop(context.Context, string) error
	Restart(context.Context, string) error
	StopMachine(context.Context, string) error
	Kill(context.Context, string) error
	DeleteContainer(context.Context, string, bool) error
	DeleteImage(context.Context, string, bool) error
	CreateVolume(context.Context, string, string) error
	CreateNetwork(context.Context, string, string) error
	DeleteVolume(context.Context, string) error
	DeleteNetwork(context.Context, string) error
	DeleteMachine(context.Context, string) error
	PruneContainers(context.Context) error
	PruneImages(context.Context, bool) error
	PruneVolumes(context.Context) error
	PruneNetworks(context.Context) error
}

type resourceKind int

const (
	resourceContainers resourceKind = iota
	resourceImages
	resourceVolumes
	resourceNetworks
	resourceMachines
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
	confirmPruneContainers
	confirmDeleteImage
	confirmPruneImages
	confirmDeleteVolume
	confirmDeleteNetwork
	confirmPruneVolumes
	confirmPruneNetworks
	confirmDeleteMachine
)

type promptKind int

const (
	promptNone promptKind = iota
	promptPullImage
	promptRunImage
	promptCreateContainer
	promptBuildImage
	promptTagImage
	promptCopy
	promptCreateMachine
	promptExportContainer
	promptExecCommand
	promptSaveImage
	promptLoadImage
	promptCreateVolume
	promptCreateNetwork
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
	volumeCursor    int
	networkCursor   int
	machineCursor   int

	containers []containercli.Container
	images     []containercli.Image
	volumes    []containercli.Volume
	networks   []containercli.NetworkResource
	machines   []containercli.Machine
	stats      []containercli.Stat
	system     containercli.SystemStatus

	panelMode    panelMode
	panelTitle   string
	panelBody    string
	panelOffset  int
	showHelp     bool
	filter       string
	filterInput  string
	filtering    bool
	prompt       promptKind
	promptInput  string
	promptTarget string

	busy        string
	statusLine  string
	err         error
	lastUpdated time.Time
	confirm     *pendingConfirm

	autoRefresh     bool
	refreshInterval time.Duration
}

type snapshotMsg struct {
	system     containercli.SystemStatus
	containers []containercli.Container
	images     []containercli.Image
	volumes    []containercli.Volume
	networks   []containercli.NetworkResource
	machines   []containercli.Machine
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

type shellFinishedMsg struct {
	id  string
	err error
}

type followLogsFinishedMsg struct {
	id  string
	err error
}

type autoRefreshMsg time.Time

const defaultRefreshInterval = 5 * time.Second

func New(client Client) Model {
	return Model{
		client:          client,
		panelMode:       panelDetails,
		panelTitle:      "Details",
		statusLine:      "starting",
		autoRefresh:     true,
		refreshInterval: defaultRefreshInterval,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.refreshCmd(), m.autoRefreshCmd())
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
		m.volumes = msg.volumes
		m.networks = msg.networks
		m.machines = msg.machines
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
	case shellFinishedMsg:
		m.busy = ""
		m.err = msg.err
		if msg.err != nil {
			m.statusLine = msg.err.Error()
			return m, nil
		}
		m.statusLine = "shell exited " + msg.id
		return m, m.refreshCmd()
	case followLogsFinishedMsg:
		m.busy = ""
		m.err = msg.err
		if msg.err != nil {
			m.statusLine = msg.err.Error()
			return m, nil
		}
		m.statusLine = "log follow exited " + msg.id
		return m, m.refreshCmd()
	case autoRefreshMsg:
		return m.handleAutoRefresh()
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
	if m.prompt != promptNone {
		return m.handlePromptKey(msg)
	}
	if m.filtering {
		return m.handleFilterKey(msg)
	}

	switch key {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "/":
		return m.startFiltering(), nil
	case "esc":
		if m.filter != "" {
			m.filter = ""
			m.filterInput = ""
			m.clampCursors()
			m.resetPanel()
			m.statusLine = "filter cleared"
		}
		return m, nil
	}
	if keyRune(msg) == "/" {
		return m.startFiltering(), nil
	}

	switch key {
	case "?":
		m.showHelp = !m.showHelp
		return m, nil
	case "tab":
		m.active = (m.active + 1) % 5
		m.resetPanel()
		return m, nil
	case "r":
		m.busy = "refreshing"
		m.statusLine = "refreshing"
		return m, m.refreshCmd()
	case "ctrl+r":
		return m.restartSelected()
	case "u":
		m.autoRefresh = !m.autoRefresh
		if m.autoRefresh {
			m.statusLine = "auto-refresh on"
			return m, m.autoRefreshCmd()
		}
		m.statusLine = "auto-refresh off"
		return m, nil
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
	case "a":
		return m.startPullPrompt(), nil
	case "b":
		return m.startBuildPrompt(), nil
	case "R":
		return m.startRunPrompt()
	case "N":
		return m.startCreateContainerPrompt()
	case "t":
		return m.startTagPrompt()
	case "P":
		return m.pushSelectedImage()
	case "O":
		return m.startSaveImagePrompt()
	case "L":
		return m.startLoadImagePrompt()
	case "c":
		return m.startCopyPrompt()
	case "C":
		return m.startCreateResourcePrompt()
	case "E":
		return m.startExportPrompt()
	case "M":
		return m.startCreateMachinePrompt()
	case "S":
		return m.setDefaultMachine()
	case "l":
		return m.logsSelected()
	case "f":
		return m.followLogsSelected()
	case "e":
		return m.shellSelected()
	case "X":
		return m.startExecPrompt()
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
		switch m.active {
		case resourceContainers:
			m.confirm = &pendingConfirm{action: confirmPruneContainers, label: "Prune stopped containers?"}
		case resourceImages:
			m.confirm = &pendingConfirm{action: confirmPruneImages, label: "Prune unused images?"}
		case resourceVolumes:
			m.confirm = &pendingConfirm{action: confirmPruneVolumes, label: "Prune unused volumes?"}
		case resourceNetworks:
			m.confirm = &pendingConfirm{action: confirmPruneNetworks, label: "Prune unused networks?"}
		}
		return m, nil
	}
	return m, nil
}

func (m Model) startPullPrompt() Model {
	m.prompt = promptPullImage
	m.promptInput = ""
	m.promptTarget = ""
	m.statusLine = "pull image"
	return m
}

func (m Model) startRunPrompt() (tea.Model, tea.Cmd) {
	if m.active != resourceImages {
		return m, nil
	}
	image, ok := m.selectedImage()
	if !ok {
		return m, nil
	}
	m.prompt = promptRunImage
	m.promptInput = ""
	m.promptTarget = image.Name()
	m.statusLine = "run image " + image.Name()
	return m, nil
}

func (m Model) startCreateContainerPrompt() (tea.Model, tea.Cmd) {
	if m.active != resourceImages {
		return m, nil
	}
	image, ok := m.selectedImage()
	if !ok {
		return m, nil
	}
	m.prompt = promptCreateContainer
	m.promptInput = ""
	m.promptTarget = image.Name()
	m.statusLine = "create container from " + image.Name()
	return m, nil
}

func (m Model) startBuildPrompt() Model {
	m.prompt = promptBuildImage
	m.promptInput = ""
	m.promptTarget = ""
	m.statusLine = "build image"
	return m
}

func (m Model) startTagPrompt() (tea.Model, tea.Cmd) {
	if m.active != resourceImages {
		return m, nil
	}
	image, ok := m.selectedImage()
	if !ok {
		return m, nil
	}
	m.prompt = promptTagImage
	m.promptInput = ""
	m.promptTarget = image.Name()
	m.statusLine = "tag image " + image.Name()
	return m, nil
}

func (m Model) startSaveImagePrompt() (tea.Model, tea.Cmd) {
	if m.active != resourceImages {
		return m, nil
	}
	image, ok := m.selectedImage()
	if !ok {
		return m, nil
	}
	reference := image.Name()
	m.prompt = promptSaveImage
	m.promptInput = defaultImageArchivePath(reference)
	m.promptTarget = reference
	m.statusLine = "save image " + reference
	return m, nil
}

func (m Model) startLoadImagePrompt() (tea.Model, tea.Cmd) {
	if m.active != resourceImages {
		return m, nil
	}
	m.prompt = promptLoadImage
	m.promptInput = ""
	m.promptTarget = ""
	m.statusLine = "load image archive"
	return m, nil
}

func (m Model) startCopyPrompt() (tea.Model, tea.Cmd) {
	if m.active != resourceContainers {
		return m, nil
	}
	container, ok := m.selectedContainer()
	if !ok {
		return m, nil
	}
	id := container.Name()
	m.prompt = promptCopy
	m.promptInput = ""
	m.promptTarget = id
	m.statusLine = "copy files for " + id
	return m, nil
}

func (m Model) startExportPrompt() (tea.Model, tea.Cmd) {
	if m.active != resourceContainers {
		return m, nil
	}
	container, ok := m.selectedContainer()
	if !ok {
		return m, nil
	}
	id := container.Name()
	m.prompt = promptExportContainer
	m.promptInput = defaultContainerExportPath(id)
	m.promptTarget = id
	m.statusLine = "export container " + id
	return m, nil
}

func (m Model) startExecPrompt() (tea.Model, tea.Cmd) {
	if m.active != resourceContainers {
		return m, nil
	}
	container, ok := m.selectedContainer()
	if !ok {
		return m, nil
	}
	id := container.Name()
	if container.State() != "running" {
		m.statusLine = "start " + id + " before running commands"
		return m, nil
	}
	m.prompt = promptExecCommand
	m.promptInput = ""
	m.promptTarget = id
	m.statusLine = "exec command in " + id
	return m, nil
}

func (m Model) startCreateMachinePrompt() (tea.Model, tea.Cmd) {
	if m.active != resourceMachines {
		return m, nil
	}
	m.prompt = promptCreateMachine
	m.promptInput = ""
	m.promptTarget = ""
	m.statusLine = "create machine"
	return m, nil
}

func (m Model) startCreateResourcePrompt() (tea.Model, tea.Cmd) {
	switch m.active {
	case resourceVolumes:
		m.prompt = promptCreateVolume
		m.promptInput = ""
		m.promptTarget = ""
		m.statusLine = "create volume"
	case resourceNetworks:
		m.prompt = promptCreateNetwork
		m.promptInput = ""
		m.promptTarget = ""
		m.statusLine = "create network"
	}
	return m, nil
}

func (m Model) startFiltering() Model {
	m.filtering = true
	m.filterInput = m.filter
	m.statusLine = "filtering"
	return m
}

func keyRune(msg tea.KeyMsg) string {
	if msg.Type != tea.KeyRunes {
		return ""
	}
	return string(msg.Runes)
}

func (m Model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "enter":
		m.filtering = false
		m.filter = strings.TrimSpace(m.filterInput)
		m.clampCursors()
		m.resetPanel()
		if m.filter == "" {
			m.statusLine = "filter cleared"
		} else {
			m.statusLine = "filter applied"
		}
		return m, nil
	case "esc":
		m.filtering = false
		m.filterInput = m.filter
		m.statusLine = "filter cancelled"
		return m, nil
	case "backspace", "ctrl+h":
		if len(m.filterInput) > 0 {
			m.filterInput = m.filterInput[:len(m.filterInput)-1]
		}
		return m, nil
	case "ctrl+u":
		m.filterInput = ""
		return m, nil
	}

	if msg.Type == tea.KeyRunes {
		m.filterInput += string(msg.Runes)
	}
	return m, nil
}

func (m Model) handlePromptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "enter":
		return m.applyPrompt()
	case "esc":
		m.prompt = promptNone
		m.promptInput = ""
		m.promptTarget = ""
		m.statusLine = "cancelled"
		return m, nil
	case "backspace", "ctrl+h":
		if len(m.promptInput) > 0 {
			m.promptInput = m.promptInput[:len(m.promptInput)-1]
		}
		return m, nil
	case "ctrl+u":
		m.promptInput = ""
		return m, nil
	}

	if msg.Type == tea.KeyRunes {
		m.promptInput += string(msg.Runes)
	}
	return m, nil
}

func (m Model) applyPrompt() (tea.Model, tea.Cmd) {
	switch m.prompt {
	case promptPullImage:
		reference := strings.TrimSpace(m.promptInput)
		m.prompt = promptNone
		m.promptInput = ""
		if reference == "" {
			m.statusLine = "pull cancelled"
			return m, nil
		}
		m.busy = "pulling " + reference
		m.statusLine = "pulling " + reference
		return m, func() tea.Msg {
			err := m.client.PullImage(context.Background(), reference)
			return actionDoneMsg{message: "pulled image " + reference, err: err}
		}
	case promptRunImage:
		image := m.promptTarget
		name := strings.TrimSpace(m.promptInput)
		m.prompt = promptNone
		m.promptInput = ""
		m.promptTarget = ""
		if strings.TrimSpace(image) == "" {
			m.statusLine = "run cancelled"
			return m, nil
		}
		m.busy = "running " + image
		m.statusLine = "running " + image
		return m, func() tea.Msg {
			err := m.client.RunImage(context.Background(), image, name)
			message := "started container from " + image
			if name != "" {
				message = "started " + name
			}
			return actionDoneMsg{message: message, err: err}
		}
	case promptCreateContainer:
		image := m.promptTarget
		name := strings.TrimSpace(m.promptInput)
		m.prompt = promptNone
		m.promptInput = ""
		m.promptTarget = ""
		if strings.TrimSpace(image) == "" {
			m.statusLine = "container create cancelled"
			return m, nil
		}
		m.busy = "creating container from " + image
		m.statusLine = "creating container from " + image
		return m, func() tea.Msg {
			err := m.client.CreateContainer(context.Background(), image, name)
			message := "created container from " + image
			if name != "" {
				message = "created container " + name
			}
			return actionDoneMsg{message: message, err: err}
		}
	case promptBuildImage:
		tag, contextDir := parseBuildImageInput(m.promptInput)
		m.prompt = promptNone
		m.promptInput = ""
		m.promptTarget = ""
		if tag == "" {
			m.statusLine = "build cancelled"
			return m, nil
		}
		m.busy = "building " + tag
		m.statusLine = "building " + tag
		return m, func() tea.Msg {
			err := m.client.BuildImage(context.Background(), tag, contextDir)
			return actionDoneMsg{message: "built image " + tag, err: err}
		}
	case promptTagImage:
		source := m.promptTarget
		target := strings.TrimSpace(m.promptInput)
		m.prompt = promptNone
		m.promptInput = ""
		m.promptTarget = ""
		if strings.TrimSpace(source) == "" || target == "" {
			m.statusLine = "tag cancelled"
			return m, nil
		}
		m.busy = "tagging " + source
		m.statusLine = "tagging " + source
		return m, func() tea.Msg {
			err := m.client.TagImage(context.Background(), source, target)
			return actionDoneMsg{message: "tagged image " + target, err: err}
		}
	case promptCopy:
		container := m.promptTarget
		source, destination, ok := parseCopyInput(m.promptInput, container)
		m.prompt = promptNone
		m.promptInput = ""
		m.promptTarget = ""
		if strings.TrimSpace(container) == "" || !ok {
			m.statusLine = "copy cancelled"
			return m, nil
		}
		m.busy = "copying files for " + container
		m.statusLine = "copying files for " + container
		return m, func() tea.Msg {
			err := m.client.Copy(context.Background(), source, destination)
			return actionDoneMsg{message: "copied files for " + container, err: err}
		}
	case promptExportContainer:
		container := m.promptTarget
		outputPath := strings.TrimSpace(m.promptInput)
		m.prompt = promptNone
		m.promptInput = ""
		m.promptTarget = ""
		if strings.TrimSpace(container) == "" || outputPath == "" {
			m.statusLine = "export cancelled"
			return m, nil
		}
		m.busy = "exporting " + container
		m.statusLine = "exporting " + container
		return m, func() tea.Msg {
			err := m.client.ExportContainer(context.Background(), container, outputPath)
			return actionDoneMsg{message: "exported " + container + " to " + outputPath, err: err}
		}
	case promptExecCommand:
		container := m.promptTarget
		command := strings.TrimSpace(m.promptInput)
		m.prompt = promptNone
		m.promptInput = ""
		m.promptTarget = ""
		if strings.TrimSpace(container) == "" || command == "" {
			m.statusLine = "exec cancelled"
			return m, nil
		}
		m.busy = "running command in " + container
		m.statusLine = "running command in " + container
		m.panelMode = panelInspect
		return m, func() tea.Msg {
			body, err := m.client.Exec(context.Background(), container, command)
			if strings.TrimSpace(body) == "" && err == nil {
				body = "Command completed with no output."
			}
			return outputMsg{title: "Exec " + container, body: body, err: err}
		}
	case promptSaveImage:
		reference := m.promptTarget
		outputPath := strings.TrimSpace(m.promptInput)
		m.prompt = promptNone
		m.promptInput = ""
		m.promptTarget = ""
		if strings.TrimSpace(reference) == "" || outputPath == "" {
			m.statusLine = "image save cancelled"
			return m, nil
		}
		m.busy = "saving image " + reference
		m.statusLine = "saving image " + reference
		return m, func() tea.Msg {
			err := m.client.SaveImage(context.Background(), reference, outputPath)
			return actionDoneMsg{message: "saved image " + reference + " to " + outputPath, err: err}
		}
	case promptLoadImage:
		inputPath := strings.TrimSpace(m.promptInput)
		m.prompt = promptNone
		m.promptInput = ""
		m.promptTarget = ""
		if inputPath == "" {
			m.statusLine = "image load cancelled"
			return m, nil
		}
		m.busy = "loading image archive"
		m.statusLine = "loading image archive"
		return m, func() tea.Msg {
			err := m.client.LoadImage(context.Background(), inputPath)
			return actionDoneMsg{message: "loaded image archive " + inputPath, err: err}
		}
	case promptCreateVolume:
		name, size, ok := parseCreateResourceInput(m.promptInput)
		m.prompt = promptNone
		m.promptInput = ""
		m.promptTarget = ""
		if !ok {
			m.statusLine = "volume create cancelled"
			return m, nil
		}
		m.busy = "creating volume " + name
		m.statusLine = "creating volume " + name
		return m, func() tea.Msg {
			err := m.client.CreateVolume(context.Background(), name, size)
			return actionDoneMsg{message: "created volume " + name, err: err}
		}
	case promptCreateNetwork:
		name, subnet, ok := parseCreateResourceInput(m.promptInput)
		m.prompt = promptNone
		m.promptInput = ""
		m.promptTarget = ""
		if !ok {
			m.statusLine = "network create cancelled"
			return m, nil
		}
		m.busy = "creating network " + name
		m.statusLine = "creating network " + name
		return m, func() tea.Msg {
			err := m.client.CreateNetwork(context.Background(), name, subnet)
			return actionDoneMsg{message: "created network " + name, err: err}
		}
	case promptCreateMachine:
		image, name, ok := parseCreateMachineInput(m.promptInput)
		m.prompt = promptNone
		m.promptInput = ""
		m.promptTarget = ""
		if !ok {
			m.statusLine = "machine create cancelled"
			return m, nil
		}
		m.busy = "creating machine"
		m.statusLine = "creating machine from " + image
		return m, func() tea.Msg {
			err := m.client.CreateMachine(context.Background(), image, name)
			message := "created machine from " + image
			if name != "" {
				message = "created machine " + name
			}
			return actionDoneMsg{message: message, err: err}
		}
	default:
		return m, nil
	}
}

func parseBuildImageInput(input string) (string, string) {
	trimmed := strings.TrimSpace(input)
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return "", ""
	}
	tag := fields[0]
	contextDir := strings.TrimSpace(strings.TrimPrefix(trimmed, tag))
	if contextDir == "" {
		contextDir = "."
	}
	return tag, contextDir
}

func parseCopyInput(input string, container string) (string, string, bool) {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) != 2 || strings.TrimSpace(container) == "" {
		return "", "", false
	}
	return expandSelectedContainerPath(fields[0], container), expandSelectedContainerPath(fields[1], container), true
}

func expandSelectedContainerPath(path string, container string) string {
	if strings.HasPrefix(path, ":") {
		return strings.TrimSpace(container) + path
	}
	return path
}

func parseCreateMachineInput(input string) (string, string, bool) {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) == 0 || len(fields) > 2 {
		return "", "", false
	}
	name := ""
	if len(fields) == 2 {
		name = fields[1]
	}
	return fields[0], name, true
}

func parseCreateResourceInput(input string) (string, string, bool) {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) == 0 || len(fields) > 2 {
		return "", "", false
	}
	option := ""
	if len(fields) == 2 {
		option = fields[1]
	}
	return fields[0], option, true
}

func defaultContainerExportPath(id string) string {
	name := strings.TrimSpace(id)
	if name == "" {
		return "container.tar"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", " ", "_")
	return replacer.Replace(name) + ".tar"
}

func defaultImageArchivePath(reference string) string {
	name := strings.TrimSpace(reference)
	if name == "" {
		return "image.tar"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", " ", "_", "@", "_")
	return replacer.Replace(name) + ".tar"
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
		if volumes, err := m.client.Volumes(ctx); err != nil {
			errs = append(errs, err)
		} else {
			sort.Slice(volumes, func(i, j int) bool {
				return volumes[i].Name() < volumes[j].Name()
			})
			msg.volumes = volumes
		}
		if networks, err := m.client.Networks(ctx); err != nil {
			errs = append(errs, err)
		} else {
			sort.Slice(networks, func(i, j int) bool {
				return networks[i].Name() < networks[j].Name()
			})
			msg.networks = networks
		}
		if machines, err := m.client.Machines(ctx); err != nil {
			errs = append(errs, err)
		} else {
			sort.Slice(machines, func(i, j int) bool {
				if machines[i].Default != machines[j].Default {
					return machines[i].Default
				}
				if machines[i].State() != machines[j].State() {
					return machines[i].State() == "running"
				}
				return machines[i].Name() < machines[j].Name()
			})
			msg.machines = machines
		}
		if stats, err := m.client.Stats(ctx); err == nil {
			msg.stats = stats
		}
		msg.err = joinErrors(errs)
		return msg
	}
}

func (m Model) autoRefreshCmd() tea.Cmd {
	if !m.autoRefresh {
		return nil
	}
	interval := m.refreshInterval
	if interval <= 0 {
		interval = defaultRefreshInterval
	}
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return autoRefreshMsg(t)
	})
}

func (m Model) handleAutoRefresh() (tea.Model, tea.Cmd) {
	nextTick := m.autoRefreshCmd()
	if !m.autoRefresh {
		return m, nil
	}
	if m.busy != "" || m.confirm != nil || m.prompt != promptNone || m.filtering {
		return m, nextTick
	}
	m.busy = "refreshing"
	m.statusLine = "auto refreshing"
	return m, tea.Batch(m.refreshCmd(), nextTick)
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
	case resourceVolumes:
		volume, ok := m.selectedVolume()
		if !ok {
			return m, nil
		}
		name := volume.Name()
		m.busy = "inspecting"
		m.panelMode = panelInspect
		return m, func() tea.Msg {
			body, err := m.client.InspectVolume(context.Background(), name)
			return outputMsg{title: "Inspect " + name, body: body, err: err}
		}
	case resourceNetworks:
		network, ok := m.selectedNetwork()
		if !ok {
			return m, nil
		}
		name := network.Name()
		m.busy = "inspecting"
		m.panelMode = panelInspect
		return m, func() tea.Msg {
			body, err := m.client.InspectNetwork(context.Background(), name)
			return outputMsg{title: "Inspect " + name, body: body, err: err}
		}
	case resourceMachines:
		machine, ok := m.selectedMachine()
		if !ok {
			return m, nil
		}
		id := machine.Name()
		m.busy = "inspecting"
		m.panelMode = panelInspect
		return m, func() tea.Msg {
			body, err := m.client.InspectMachine(context.Background(), id)
			return outputMsg{title: "Inspect " + id, body: body, err: err}
		}
	}
	return m, nil
}

func (m Model) logsSelected() (tea.Model, tea.Cmd) {
	switch m.active {
	case resourceContainers:
		container, ok := m.selectedContainer()
		if !ok {
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
	case resourceMachines:
		machine, ok := m.selectedMachine()
		if !ok {
			return m, nil
		}
		id := machine.Name()
		m.busy = "loading logs"
		m.panelMode = panelLogs
		return m, func() tea.Msg {
			body, err := m.client.MachineLogs(context.Background(), id, 200)
			if strings.TrimSpace(body) == "" && err == nil {
				body = "No machine logs returned."
			}
			return outputMsg{title: "Machine logs " + id, body: body, err: err}
		}
	}
	return m, nil
}

func (m Model) followLogsSelected() (tea.Model, tea.Cmd) {
	id, cmd, err := m.followLogsCommandForSelection()
	if err != nil {
		m.err = err
		m.statusLine = err.Error()
		return m, nil
	}
	if cmd == nil {
		return m, nil
	}
	m.busy = "following logs " + id
	m.statusLine = "following logs " + id
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return followLogsFinishedMsg{id: id, err: err}
	})
}

func (m Model) followLogsCommandForSelection() (string, *exec.Cmd, error) {
	switch m.active {
	case resourceContainers:
		container, ok := m.selectedContainer()
		if !ok {
			return "", nil, nil
		}
		id := container.Name()
		cmd, err := m.client.FollowLogsCommand(id, 200)
		return id, cmd, err
	case resourceMachines:
		machine, ok := m.selectedMachine()
		if !ok {
			return "", nil, nil
		}
		id := machine.Name()
		cmd, err := m.client.FollowMachineLogsCommand(id, 200)
		return id, cmd, err
	default:
		return "", nil, nil
	}
}

func (m Model) shellSelected() (tea.Model, tea.Cmd) {
	if m.active == resourceMachines {
		return m.machineShellSelected()
	}
	if m.active != resourceContainers {
		return m, nil
	}
	container, ok := m.selectedContainer()
	if !ok {
		return m, nil
	}
	id := container.Name()
	if container.State() != "running" {
		m.statusLine = "start " + id + " before opening a shell"
		return m, nil
	}
	cmd, err := m.client.ShellCommand(id, "/bin/sh")
	if err != nil {
		m.err = err
		m.statusLine = err.Error()
		return m, nil
	}
	m.busy = "shell " + id
	m.statusLine = "opening shell " + id
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return shellFinishedMsg{id: id, err: err}
	})
}

func (m Model) machineShellSelected() (tea.Model, tea.Cmd) {
	machine, ok := m.selectedMachine()
	if !ok {
		return m, nil
	}
	id := machine.Name()
	cmd, err := m.client.MachineShellCommand(id)
	if err != nil {
		m.err = err
		m.statusLine = err.Error()
		return m, nil
	}
	m.busy = "machine shell " + id
	m.statusLine = "opening machine shell " + id
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return shellFinishedMsg{id: id, err: err}
	})
}

func (m Model) restartSelected() (tea.Model, tea.Cmd) {
	if m.active != resourceContainers {
		return m, nil
	}
	container, ok := m.selectedContainer()
	if !ok {
		return m, nil
	}
	id := container.Name()
	if container.State() != "running" {
		m.statusLine = "start " + id + " before restarting"
		return m, nil
	}
	m.busy = "restarting"
	m.statusLine = "restarting " + id
	return m, func() tea.Msg {
		err := m.client.Restart(context.Background(), id)
		return actionDoneMsg{message: "restarted " + id, err: err}
	}
}

func (m Model) pushSelectedImage() (tea.Model, tea.Cmd) {
	if m.active != resourceImages {
		return m, nil
	}
	image, ok := m.selectedImage()
	if !ok {
		return m, nil
	}
	reference := strings.TrimSpace(image.Name())
	if reference == "" {
		m.statusLine = "push cancelled"
		return m, nil
	}
	m.busy = "pushing " + reference
	m.statusLine = "pushing " + reference
	return m, func() tea.Msg {
		err := m.client.PushImage(context.Background(), reference)
		return actionDoneMsg{message: "pushed image " + reference, err: err}
	}
}

func (m Model) setDefaultMachine() (tea.Model, tea.Cmd) {
	if m.active != resourceMachines {
		return m, nil
	}
	machine, ok := m.selectedMachine()
	if !ok {
		return m, nil
	}
	id := machine.Name()
	if strings.TrimSpace(id) == "" {
		m.statusLine = "set default cancelled"
		return m, nil
	}
	m.busy = "setting default machine"
	m.statusLine = "setting default machine " + id
	return m, func() tea.Msg {
		err := m.client.SetDefaultMachine(context.Background(), id)
		return actionDoneMsg{message: "set default machine " + id, err: err}
	}
}

func (m Model) lifecycleSelected(busy string, done string, action func(context.Context, string) error) (tea.Model, tea.Cmd) {
	if m.active == resourceMachines && busy == "stopping" {
		machine, ok := m.selectedMachine()
		if !ok {
			return m, nil
		}
		id := machine.Name()
		m.busy = busy
		m.statusLine = busy + " " + id
		return m, func() tea.Msg {
			err := m.client.StopMachine(context.Background(), id)
			return actionDoneMsg{message: "stopped machine " + id, err: err}
		}
	}
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
	case resourceVolumes:
		volume, ok := m.selectedVolume()
		if ok {
			name := volume.Name()
			m.confirm = &pendingConfirm{action: confirmDeleteVolume, target: name, label: "Delete volume " + name + "?"}
		}
	case resourceNetworks:
		network, ok := m.selectedNetwork()
		if ok {
			name := network.Name()
			m.confirm = &pendingConfirm{action: confirmDeleteNetwork, target: name, label: "Delete network " + name + "?"}
		}
	case resourceMachines:
		machine, ok := m.selectedMachine()
		if ok {
			id := machine.Name()
			m.confirm = &pendingConfirm{action: confirmDeleteMachine, target: id, label: "Delete machine " + id + "?"}
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
		case confirmPruneContainers:
			err := m.client.PruneContainers(ctx)
			return actionDoneMsg{message: "pruned stopped containers", err: err}
		case confirmDeleteImage:
			err := m.client.DeleteImage(ctx, confirm.target, false)
			return actionDoneMsg{message: "deleted image " + confirm.target, err: err}
		case confirmPruneImages:
			err := m.client.PruneImages(ctx, false)
			return actionDoneMsg{message: "pruned unused images", err: err}
		case confirmDeleteVolume:
			err := m.client.DeleteVolume(ctx, confirm.target)
			return actionDoneMsg{message: "deleted volume " + confirm.target, err: err}
		case confirmDeleteNetwork:
			err := m.client.DeleteNetwork(ctx, confirm.target)
			return actionDoneMsg{message: "deleted network " + confirm.target, err: err}
		case confirmPruneVolumes:
			err := m.client.PruneVolumes(ctx)
			return actionDoneMsg{message: "pruned unused volumes", err: err}
		case confirmPruneNetworks:
			err := m.client.PruneNetworks(ctx)
			return actionDoneMsg{message: "pruned unused networks", err: err}
		case confirmDeleteMachine:
			err := m.client.DeleteMachine(ctx, confirm.target)
			return actionDoneMsg{message: "deleted machine " + confirm.target, err: err}
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
	case resourceVolumes:
		m.volumeCursor += delta
	case resourceNetworks:
		m.networkCursor += delta
	case resourceMachines:
		m.machineCursor += delta
	}
	m.clampCursors()
	m.resetPanel()
}

func (m *Model) clampCursors() {
	m.containerCursor = clamp(m.containerCursor, 0, len(m.filteredContainerIndexes())-1)
	m.imageCursor = clamp(m.imageCursor, 0, len(m.filteredImageIndexes())-1)
	m.volumeCursor = clamp(m.volumeCursor, 0, len(m.filteredVolumeIndexes())-1)
	m.networkCursor = clamp(m.networkCursor, 0, len(m.filteredNetworkIndexes())-1)
	m.machineCursor = clamp(m.machineCursor, 0, len(m.filteredMachineIndexes())-1)
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
	indexes := m.filteredContainerIndexes()
	if len(indexes) == 0 || m.containerCursor < 0 || m.containerCursor >= len(indexes) {
		return containercli.Container{}, false
	}
	return m.containers[indexes[m.containerCursor]], true
}

func (m Model) selectedImage() (containercli.Image, bool) {
	indexes := m.filteredImageIndexes()
	if len(indexes) == 0 || m.imageCursor < 0 || m.imageCursor >= len(indexes) {
		return containercli.Image{}, false
	}
	return m.images[indexes[m.imageCursor]], true
}

func (m Model) selectedVolume() (containercli.Volume, bool) {
	indexes := m.filteredVolumeIndexes()
	if len(indexes) == 0 || m.volumeCursor < 0 || m.volumeCursor >= len(indexes) {
		return containercli.Volume{}, false
	}
	return m.volumes[indexes[m.volumeCursor]], true
}

func (m Model) selectedNetwork() (containercli.NetworkResource, bool) {
	indexes := m.filteredNetworkIndexes()
	if len(indexes) == 0 || m.networkCursor < 0 || m.networkCursor >= len(indexes) {
		return containercli.NetworkResource{}, false
	}
	return m.networks[indexes[m.networkCursor]], true
}

func (m Model) selectedMachine() (containercli.Machine, bool) {
	indexes := m.filteredMachineIndexes()
	if len(indexes) == 0 || m.machineCursor < 0 || m.machineCursor >= len(indexes) {
		return containercli.Machine{}, false
	}
	return m.machines[indexes[m.machineCursor]], true
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
	if m.prompt != promptNone {
		return footerStyle.Width(m.width).Foreground(colorActive).Render(truncate(m.promptLine(), m.width-2))
	}
	if m.filtering {
		value := m.filterInput
		if value == "" {
			value = " "
		}
		line := "/ " + value + "  enter apply, esc cancel, ctrl+u clear"
		return footerStyle.Width(m.width).Foreground(colorActive).Render(truncate(line, m.width-2))
	}
	if m.showHelp {
		help := "tab switch | / filter | r refresh | u auto-refresh | a pull image | b build image | t tag image | P push image | O save image | L load image | R run image | N create container | C create volume/network | M create machine | S default machine | i inspect | c copy files | E export container | l logs | f follow logs | e shell | X exec command | s start | ctrl+r restart | x stop | K kill | d delete | p prune | q quit"
		return footerStyle.Width(m.width).Render(truncate(help, m.width-2))
	}
	status := m.statusLine
	if status == "" {
		status = "? help"
	}
	if activeFilter(m.filter) != "" {
		status = status + " | filter: " + m.filter
	}
	if m.err != nil {
		return footerStyle.Width(m.width).Foreground(colorRed).Render(truncate(status, m.width-2))
	}
	return footerStyle.Width(m.width).Render(truncate(status+" | "+m.autoRefreshLabel()+" | f follow | ? help", m.width-2))
}

func (m Model) autoRefreshLabel() string {
	if m.autoRefresh {
		return "u auto:on"
	}
	return "u auto:off"
}

func (m Model) promptLine() string {
	switch m.prompt {
	case promptPullImage:
		return "pull image: " + m.promptInput + "  enter pull, esc cancel"
	case promptRunImage:
		return "container name for " + m.promptTarget + ": " + m.promptInput + "  enter run, blank auto, esc cancel"
	case promptCreateContainer:
		return "container name for " + m.promptTarget + ": " + m.promptInput + "  enter create stopped, blank auto, esc cancel"
	case promptBuildImage:
		return "build image tag [context-dir]: " + m.promptInput + "  enter build, context defaults ., esc cancel"
	case promptTagImage:
		return "new tag for " + m.promptTarget + ": " + m.promptInput + "  enter tag, esc cancel"
	case promptCopy:
		return "copy for " + m.promptTarget + " src dest (:path is selected container): " + m.promptInput + "  enter copy, esc cancel"
	case promptCreateMachine:
		return "machine image [name]: " + m.promptInput + "  enter create, name optional, esc cancel"
	case promptExportContainer:
		return "export " + m.promptTarget + " to tar path: " + m.promptInput + "  enter export, ctrl+u clear, esc cancel"
	case promptExecCommand:
		return "exec in " + m.promptTarget + ": " + m.promptInput + "  enter run, esc cancel"
	case promptSaveImage:
		return "save " + m.promptTarget + " to tar path: " + m.promptInput + "  enter save, ctrl+u clear, esc cancel"
	case promptLoadImage:
		return "load image archive path: " + m.promptInput + "  enter load, esc cancel"
	case promptCreateVolume:
		return "volume name [size]: " + m.promptInput + "  enter create, esc cancel"
	case promptCreateNetwork:
		return "network name [subnet]: " + m.promptInput + "  enter create, esc cancel"
	default:
		return ""
	}
}

func (m Model) renderSidebar(width int, height int) string {
	style := activePanelStyle.Width(width - 2).Height(height - 2)
	var lines []string
	lines = append(lines, strings.Split(m.renderTabs(), "\n")...)
	lines = append(lines, "")
	listHeight := height - len(lines) - 2
	if listHeight < 1 {
		listHeight = 1
	}
	switch m.active {
	case resourceContainers:
		lines = append(lines, m.renderContainerList(width-4, listHeight)...)
	case resourceImages:
		lines = append(lines, m.renderImageList(width-4, listHeight)...)
	case resourceVolumes:
		lines = append(lines, m.renderVolumeList(width-4, listHeight)...)
	case resourceNetworks:
		lines = append(lines, m.renderNetworkList(width-4, listHeight)...)
	case resourceMachines:
		lines = append(lines, m.renderMachineList(width-4, listHeight)...)
	}
	return style.Render(strings.Join(lines, "\n"))
}

func (m Model) renderTabs() string {
	containers := m.tabLabel("containers", len(m.filteredContainerIndexes()), len(m.containers))
	images := m.tabLabel("images", len(m.filteredImageIndexes()), len(m.images))
	volumes := m.tabLabel("volumes", len(m.filteredVolumeIndexes()), len(m.volumes))
	networks := m.tabLabel("networks", len(m.filteredNetworkIndexes()), len(m.networks))
	machines := m.tabLabel("machines", len(m.filteredMachineIndexes()), len(m.machines))
	tabs := []string{containers, images, volumes, networks, machines}
	for idx := range tabs {
		label := " " + tabs[idx] + " "
		if resourceKind(idx) == m.active {
			tabs[idx] = selectedStyle.Render(label)
		} else {
			tabs[idx] = mutedStyle.Render(label)
		}
	}
	return strings.Join(tabs[:2], " ") + "\n" + strings.Join(tabs[2:], " ")
}

func (m Model) tabLabel(name string, filtered int, total int) string {
	if activeFilter(m.filter) == "" || filtered == total {
		return fmt.Sprintf("%s %d", name, total)
	}
	return fmt.Sprintf("%s %d/%d", name, filtered, total)
}

func (m Model) renderContainerList(width int, height int) []string {
	indexes := m.filteredContainerIndexes()
	if len(indexes) == 0 {
		return []string{mutedStyle.Render(m.emptyListMessage("containers"))}
	}
	rows := []string{mutedStyle.Render(fitColumns("name", "state", width))}
	start := visibleStart(m.containerCursor, height-1, len(indexes))
	end := start + height - 1
	if end > len(indexes) {
		end = len(indexes)
	}
	now := effectiveNow(m.lastUpdated)
	for idx := start; idx < end; idx++ {
		container := m.containers[indexes[idx]]
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
	indexes := m.filteredImageIndexes()
	if len(indexes) == 0 {
		return []string{mutedStyle.Render(m.emptyListMessage("images"))}
	}
	rows := []string{mutedStyle.Render(fitColumns("image", "size", width))}
	start := visibleStart(m.imageCursor, height-1, len(indexes))
	end := start + height - 1
	if end > len(indexes) {
		end = len(indexes)
	}
	for idx := start; idx < end; idx++ {
		image := m.images[indexes[idx]]
		name := truncate(image.Name(), 34)
		line := fitColumns(name, image.Size(), width)
		if idx == m.imageCursor {
			line = selectedStyle.Width(width).Render(truncate(line, width))
		}
		rows = append(rows, line)
	}
	return rows
}

func (m Model) renderVolumeList(width int, height int) []string {
	indexes := m.filteredVolumeIndexes()
	if len(indexes) == 0 {
		return []string{mutedStyle.Render(m.emptyListMessage("volumes"))}
	}
	rows := []string{mutedStyle.Render(fitColumns("volume", "size", width))}
	start := visibleStart(m.volumeCursor, height-1, len(indexes))
	end := start + height - 1
	if end > len(indexes) {
		end = len(indexes)
	}
	now := effectiveNow(m.lastUpdated)
	for idx := start; idx < end; idx++ {
		volume := m.volumes[indexes[idx]]
		name := truncate(volume.Name(), 34)
		meta := fmt.Sprintf("%s  %s", volume.Size(), volume.CreatedAgo(now))
		line := fitColumns(name, meta, width)
		if idx == m.volumeCursor {
			line = selectedStyle.Width(width).Render(truncate(line, width))
		}
		rows = append(rows, line)
	}
	return rows
}

func (m Model) renderNetworkList(width int, height int) []string {
	indexes := m.filteredNetworkIndexes()
	if len(indexes) == 0 {
		return []string{mutedStyle.Render(m.emptyListMessage("networks"))}
	}
	rows := []string{mutedStyle.Render(fitColumns("network", "mode", width))}
	start := visibleStart(m.networkCursor, height-1, len(indexes))
	end := start + height - 1
	if end > len(indexes) {
		end = len(indexes)
	}
	for idx := start; idx < end; idx++ {
		network := m.networks[indexes[idx]]
		name := truncate(network.Name(), 34)
		meta := emptyDash(network.Configuration.Mode)
		line := fitColumns(name, meta, width)
		if idx == m.networkCursor {
			line = selectedStyle.Width(width).Render(truncate(line, width))
		}
		rows = append(rows, line)
	}
	return rows
}

func (m Model) renderMachineList(width int, height int) []string {
	indexes := m.filteredMachineIndexes()
	if len(indexes) == 0 {
		return []string{mutedStyle.Render(m.emptyListMessage("machines"))}
	}
	rows := []string{mutedStyle.Render(fitColumns("machine", "state", width))}
	start := visibleStart(m.machineCursor, height-1, len(indexes))
	end := start + height - 1
	if end > len(indexes) {
		end = len(indexes)
	}
	now := effectiveNow(m.lastUpdated)
	for idx := start; idx < end; idx++ {
		machine := m.machines[indexes[idx]]
		name := truncate(machine.Name(), 26)
		meta := fmt.Sprintf("%s  %s", machine.State(), machine.CreatedAgo(now))
		if machine.Default {
			name = "* " + name
		}
		line := fitColumns(name, meta, width)
		if idx == m.machineCursor {
			line = selectedStyle.Width(width).Render(truncate(line, width))
		} else if machine.State() == "running" {
			line = runningStyle.Render(line)
		}
		rows = append(rows, line)
	}
	return rows
}

func (m Model) emptyListMessage(kind string) string {
	if activeFilter(m.filter) == "" {
		return "No " + kind + " found."
	}
	return "No " + kind + " match " + fmt.Sprintf("%q.", m.filter)
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
	case resourceVolumes:
		volume, ok := m.selectedVolume()
		if !ok {
			return "Details", "No volume selected."
		}
		return "Details " + volume.Name(), strings.Join(volume.DetailLines(now), "\n")
	case resourceNetworks:
		network, ok := m.selectedNetwork()
		if !ok {
			return "Details", "No network selected."
		}
		return "Details " + network.Name(), strings.Join(network.DetailLines(now), "\n")
	case resourceMachines:
		machine, ok := m.selectedMachine()
		if !ok {
			return "Details", "No machine selected."
		}
		return "Details " + machine.Name(), strings.Join(machine.DetailLines(now), "\n")
	default:
		return "Details", ""
	}
}

func (m Model) statLines(containerID string) []string {
	for _, stat := range m.stats {
		if !statMatches(stat, containerID) {
			continue
		}
		if lines := stat.SummaryLines(); len(lines) > 0 {
			return lines
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

func (m Model) filteredContainerIndexes() []int {
	filter := activeFilter(m.filter)
	indexes := make([]int, 0, len(m.containers))
	for idx, container := range m.containers {
		if filter == "" || matchFields(filter, container.Name(), container.State(), container.ImageName(), container.Ports(), container.Platform()) {
			indexes = append(indexes, idx)
		}
	}
	return indexes
}

func (m Model) filteredImageIndexes() []int {
	filter := activeFilter(m.filter)
	indexes := make([]int, 0, len(m.images))
	for idx, image := range m.images {
		if filter == "" || matchFields(filter, image.Name(), image.Digest(), image.Platforms(), image.Size()) {
			indexes = append(indexes, idx)
		}
	}
	return indexes
}

func (m Model) filteredVolumeIndexes() []int {
	filter := activeFilter(m.filter)
	indexes := make([]int, 0, len(m.volumes))
	for idx, volume := range m.volumes {
		if filter == "" || matchFields(filter, volume.Name(), volume.Configuration.Driver, volume.Configuration.Format, volume.Configuration.Source, volume.Size()) {
			indexes = append(indexes, idx)
		}
	}
	return indexes
}

func (m Model) filteredNetworkIndexes() []int {
	filter := activeFilter(m.filter)
	indexes := make([]int, 0, len(m.networks))
	for idx, network := range m.networks {
		if filter == "" || matchFields(filter, network.Name(), network.Configuration.Mode, network.Configuration.Plugin, network.Status.IPv4Gateway, network.Status.IPv4Subnet, network.Status.IPv6Subnet) {
			indexes = append(indexes, idx)
		}
	}
	return indexes
}

func (m Model) filteredMachineIndexes() []int {
	filter := activeFilter(m.filter)
	indexes := make([]int, 0, len(m.machines))
	for idx, machine := range m.machines {
		if filter == "" || matchFields(filter, machine.Name(), machine.State(), machine.Image(), machine.CPUs(), machine.Memory()) {
			indexes = append(indexes, idx)
		}
	}
	return indexes
}

func activeFilter(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func matchFields(filter string, fields ...string) bool {
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), filter) {
			return true
		}
	}
	return false
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

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}
