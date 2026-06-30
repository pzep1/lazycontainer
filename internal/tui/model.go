package tui

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pzep1/lazycont/internal/appmeta"
	"github.com/pzep1/lazycont/internal/compose"
	"github.com/pzep1/lazycont/internal/containercli"
)

type Client interface {
	SystemStatus(context.Context) (containercli.SystemStatus, error)
	SystemDiskUsage(context.Context) (containercli.SystemDiskUsage, error)
	SystemVersion(context.Context) ([]containercli.SystemVersion, error)
	SystemDNS(context.Context) ([]containercli.SystemDNSDomain, error)
	SystemProperties(context.Context) ([]containercli.SystemProperty, error)
	Containers(context.Context) ([]containercli.Container, error)
	Images(context.Context) ([]containercli.Image, error)
	Volumes(context.Context) ([]containercli.Volume, error)
	Networks(context.Context) ([]containercli.NetworkResource, error)
	Machines(context.Context) ([]containercli.Machine, error)
	Registries(context.Context) ([]containercli.RegistryLogin, error)
	BuilderStatus(context.Context) (containercli.BuilderStatus, error)
	Stats(context.Context, ...string) ([]containercli.Stat, error)
	Logs(context.Context, string, int) (string, error)
	FollowLogsCommand(string, int) (*exec.Cmd, error)
	BootLogs(context.Context, string, int) (string, error)
	MachineBootLogs(context.Context, string, int) (string, error)
	MachineLogs(context.Context, string, int) (string, error)
	FollowMachineLogsCommand(string, int) (*exec.Cmd, error)
	SystemLogs(context.Context, string) (string, error)
	FollowSystemLogsCommand(string) (*exec.Cmd, error)
	InspectContainer(context.Context, string) (string, error)
	InspectImage(context.Context, string) (string, error)
	InspectVolume(context.Context, string) (string, error)
	InspectNetwork(context.Context, string) (string, error)
	InspectMachine(context.Context, string) (string, error)
	ShellCommand(string, string) (*exec.Cmd, error)
	Exec(context.Context, string, string) (string, error)
	Top(context.Context, string) (string, error)
	Command(context.Context, []string) (string, error)
	CommandProcess([]string) (*exec.Cmd, error)
	MachineShellCommand(string) (*exec.Cmd, error)
	CreateMachine(context.Context, string, string) error
	SetDefaultMachine(context.Context, string) error
	SetMachine(context.Context, string, []string) error
	PullImage(context.Context, string) (string, error)
	RunImage(context.Context, string, containercli.ContainerLaunchOptions) error
	CreateContainer(context.Context, string, containercli.ContainerLaunchOptions) error
	BuildImage(context.Context, string, string) (string, error)
	TagImage(context.Context, string, string) error
	PushImage(context.Context, string) (string, error)
	SaveImage(context.Context, string, string) (string, error)
	LoadImage(context.Context, string) (string, error)
	RegistryLoginCommand(string, string) (*exec.Cmd, error)
	LogoutRegistry(context.Context, string) error
	StartBuilder(context.Context) error
	StopBuilder(context.Context) error
	DeleteBuilder(context.Context, bool) error
	StartSystem(context.Context) error
	StopSystem(context.Context) error
	Copy(context.Context, string, string) error
	ExportContainer(context.Context, string, string) error
	Start(context.Context, string) error
	Stop(context.Context, string) error
	Restart(context.Context, string) error
	StopMachine(context.Context, string) error
	Kill(context.Context, string) error
	StopAll(context.Context) error
	KillAll(context.Context) error
	DeleteAllContainers(context.Context, bool) error
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
	resourceServices
	resourceImages
	resourceBuilder
	resourceVolumes
	resourceNetworks
	resourceMachines
	resourceRegistries
	resourceSystem
	resourceCount
)

type mainTab int

const (
	tabConfig mainTab = iota
	tabLogs
	tabStats
	tabEnv
	tabPorts
	tabMounts
	tabHealth
	tabTop
	tabInspect
)

// fetched reports whether a tab's content is loaded asynchronously by a
// one-shot command (as opposed to rendered from snapshot data or streamed). The
// Logs tab is handled separately via a live follow stream.
func (t mainTab) fetched() bool {
	return t == tabTop || t == tabInspect
}

// screenMode controls how much width the main panel occupies relative to the
// sidebar, cycled with + / _.
type screenMode int

const (
	screenNormal screenMode = iota
	screenHalf
	screenFull
	screenModeCount
)

func (s screenMode) label() string {
	switch s {
	case screenHalf:
		return "half"
	case screenFull:
		return "fullscreen"
	default:
		return "normal"
	}
}

// bufferKind tracks what the panel's text buffer (panelTitle/panelBody)
// currently holds.
type bufferKind int

const (
	bufNone   bufferKind = iota // panel renders the active tab from snapshot data
	bufTab                      // panel renders a fetched tab's buffered content
	bufOutput                   // panel renders transient command output
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
	promptContainerCommand
	promptCustomCommand
	promptSaveImage
	promptLoadImage
	promptCreateVolume
	promptCreateNetwork
	promptRegistryLogin
	promptSetMachine
)

// pendingConfirm is a queued destructive action awaiting a y/n confirmation.
// run performs the action and returns a status message; defining it at each
// action's call site keeps confirmation execution with the feature that owns it
// rather than in a central switch.
type pendingConfirm struct {
	label string
	run   func(context.Context, Model) (string, error)
}

type Options struct {
	CustomCommands     []CustomCommand
	ConfigPath         string
	OpenConfigCommand  func(string) (*exec.Cmd, error)
	LoadConfigCommands func() ([]CustomCommand, error)
	// ReloadConfig, if set, returns a fresh Options after the config file is
	// edited so commands, theme, and gui/log settings can be reapplied without
	// a restart. It takes precedence over LoadConfigCommands.
	ReloadConfig    func() (Options, error)
	OpenLinkCommand func(string) (*exec.Cmd, error)
	StartupWarning  string

	// Appearance and behaviour (from config gui/logs/refreshIntervalMs).
	ScreenMode      string
	SidePanelWidth  float64
	BorderStyle     string
	ActiveColor     string
	SelectedBgColor string
	LogsTail        int
	LogsSince       string
	RefreshInterval time.Duration
	// Ignore hides resources whose name contains any of these substrings.
	Ignore []string
	// LoadProject discovers and parses a Compose file (typically in the working
	// directory) so the Services panel can orchestrate a multi-container stack.
	// A missing file returns a zero Project and nil error; a parse failure
	// returns the error. It is re-invoked on each refresh so edits apply live.
	LoadProject func() (compose.Project, error)
}

type CustomCommand struct {
	Name   string
	Args   []string
	Attach bool
}

type Model struct {
	client Client

	width  int
	height int

	active          resourceKind
	containerCursor int
	serviceCursor   int
	imageCursor     int
	volumeCursor    int
	networkCursor   int
	machineCursor   int
	registryCursor  int

	containers       []containercli.Container
	images           []containercli.Image
	volumes          []containercli.Volume
	networks         []containercli.NetworkResource
	machines         []containercli.Machine
	registries       []containercli.RegistryLogin
	builder          containercli.BuilderStatus
	systemUsage      containercli.SystemDiskUsage
	systemVersions   []containercli.SystemVersion
	systemDNS        []containercli.SystemDNSDomain
	systemProperties []containercli.SystemProperty
	project          compose.Project
	projectErr       error
	stats            []containercli.Stat
	statHistory      map[string][]statHistorySample
	system           containercli.SystemStatus

	tabIndex     [resourceCount]int
	bufferKind   bufferKind
	bufferTab    mainTab
	bufferKey    string
	panelTitle   string
	panelBody    string
	panelOffset  int
	stream       *logStream
	streamGen    int
	logLines     []string
	logKey       string
	logFollow    bool
	menu         *actionMenu
	bulkMenu     *bulkMenuState
	showHelp     bool
	helpOffset   int
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
	customCommands  []CustomCommand
	configPath      string
	openConfig      func(string) (*exec.Cmd, error)
	loadConfig      func() ([]CustomCommand, error)
	reloadConfig    func() (Options, error)
	openLink        func(string) (*exec.Cmd, error)
	loadProject     func() (compose.Project, error)
	logsTail        int
	logsSince       string
	screenMode      screenMode
	sidePanelWidth  float64
	ignore          []string
}

type snapshotMsg struct {
	system           containercli.SystemStatus
	containers       []containercli.Container
	images           []containercli.Image
	volumes          []containercli.Volume
	networks         []containercli.NetworkResource
	machines         []containercli.Machine
	registries       []containercli.RegistryLogin
	builder          containercli.BuilderStatus
	systemUsage      containercli.SystemDiskUsage
	systemVersions   []containercli.SystemVersion
	systemDNS        []containercli.SystemDNSDomain
	systemProperties []containercli.SystemProperty
	// systemDNSOK / systemPropsOK record whether the best-effort fetch
	// succeeded, so a transient failure leaves the last-known values in place
	// rather than blanking the System pane.
	systemDNSOK   bool
	systemPropsOK bool
	project       compose.Project
	projectErr    error
	stats         []containercli.Stat
	err           error
	updated       time.Time
}

type outputMsg struct {
	title   string
	body    string
	err     error
	refresh bool
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

type configEditedMsg struct {
	path string
	err  error
}

type autoRefreshMsg time.Time

const defaultRefreshInterval = 5 * time.Second
const maxStatHistorySamples = 120

type statHistorySample struct {
	at time.Time

	// Derived instantaneous values, computed from deltas against the previous
	// sample. Apple's `container stats` reports cumulative counters
	// (cpuUsageUsec, networkRx/TxBytes, blockRead/WriteBytes), so a live
	// CPU%/throughput only exists once two samples can be differenced.
	cpuPercent  float64
	hasCPU      bool
	networkRate float64 // bytes/sec
	hasNetwork  bool
	blockRate   float64 // bytes/sec
	hasBlock    bool

	// memoryBytes is an instantaneous gauge in the raw stat, used as-is.
	memoryBytes float64
	hasMemory   bool

	// Raw cumulative counters retained so the next sample can difference them.
	cpuTimeUsec  float64
	hasCPUTime   bool
	networkTotal float64
	hasNetTotal  bool
	blockTotal   float64
	hasBlkTotal  bool
}

func New(client Client) Model {
	return NewWithOptions(client, Options{})
}

func NewWithOptions(client Client, opts Options) Model {
	commands := append([]CustomCommand(nil), opts.CustomCommands...)
	statusLine := "starting"
	if strings.TrimSpace(opts.StartupWarning) != "" {
		statusLine = strings.TrimSpace(opts.StartupWarning)
	}
	applyTheme(opts.BorderStyle, opts.ActiveColor, opts.SelectedBgColor)
	refresh := defaultRefreshInterval
	if opts.RefreshInterval > 0 {
		refresh = opts.RefreshInterval
	}
	// Load the Compose project once up front so the Services panel is populated
	// before the first refresh lands.
	var project compose.Project
	var projectErr error
	if opts.LoadProject != nil {
		project, projectErr = opts.LoadProject()
	}
	return Model{
		client:          client,
		statusLine:      statusLine,
		autoRefresh:     true,
		refreshInterval: refresh,
		project:         project,
		projectErr:      projectErr,
		loadProject:     opts.LoadProject,
		customCommands:  commands,
		configPath:      opts.ConfigPath,
		openConfig:      opts.OpenConfigCommand,
		loadConfig:      opts.LoadConfigCommands,
		reloadConfig:    opts.ReloadConfig,
		openLink:        opts.OpenLinkCommand,
		logsTail:        opts.LogsTail,
		logsSince:       opts.LogsSince,
		screenMode:      parseScreenMode(opts.ScreenMode),
		sidePanelWidth:  opts.SidePanelWidth,
		ignore:          normalizeIgnore(opts.Ignore),
	}
}

// normalizeIgnore lowercases and trims the configured ignore patterns once (and
// drops empties), so isIgnored can match without re-normalizing per render.
func normalizeIgnore(patterns []string) []string {
	out := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		if pattern = strings.ToLower(strings.TrimSpace(pattern)); pattern != "" {
			out = append(out, pattern)
		}
	}
	return out
}

// applyReloadedOptions reapplies commands, theme, and gui/log settings after
// the config file is edited in-session.
func (m *Model) applyReloadedOptions(opts Options) {
	m.customCommands = append([]CustomCommand(nil), opts.CustomCommands...)
	applyTheme(opts.BorderStyle, opts.ActiveColor, opts.SelectedBgColor)
	m.screenMode = parseScreenMode(opts.ScreenMode)
	m.sidePanelWidth = opts.SidePanelWidth
	m.logsTail = opts.LogsTail
	m.logsSince = opts.LogsSince
	m.ignore = normalizeIgnore(opts.Ignore)
	if opts.RefreshInterval > 0 {
		m.refreshInterval = opts.RefreshInterval
	}
}

func parseScreenMode(value string) screenMode {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "half":
		return screenHalf
	case "full", "fullscreen":
		return screenFull
	default:
		return screenNormal
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
		m.registries = msg.registries
		m.builder = msg.builder
		m.systemUsage = msg.systemUsage
		m.systemVersions = msg.systemVersions
		// Keep the last-known DNS/properties when a best-effort fetch failed, so
		// a transient error doesn't flicker the System pane to empty.
		if msg.systemDNSOK {
			m.systemDNS = msg.systemDNS
		}
		if msg.systemPropsOK {
			m.systemProperties = msg.systemProperties
		}
		m.project = msg.project
		m.projectErr = msg.projectErr
		m.stats = msg.stats
		m.recordStatHistory(msg.stats, msg.updated)
		m.pruneStatHistory(msg.containers)
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
		m.bufferKind = bufOutput
		m.panelTitle = msg.title
		m.panelBody = msg.body
		m.panelOffset = 0
		m.statusLine = "loaded " + strings.ToLower(msg.title)
		if msg.refresh {
			m.busy = "refreshing"
			return m, m.refreshCmd()
		}
		return m, nil
	case tabFetchedMsg:
		return m.handleTabFetched(msg)
	case logStreamMsg:
		return m.handleLogStream(msg)
	case actionDoneMsg:
		m.confirm = nil
		m.err = msg.err
		if msg.err != nil {
			// Keep any cached tab content visible and recover the active fetched
			// tab / log stream rather than stranding it on "Loading…".
			m.busy = ""
			m.statusLine = msg.err.Error()
			return m, m.ensureMainPanelCmd()
		}
		m.busy = "refreshing"
		m.bufferKind = bufNone
		m.panelOffset = 0
		m.statusLine = msg.message
		return m, m.refreshWithActiveTab(nil)
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
	case configEditedMsg:
		m.busy = ""
		m.err = msg.err
		if msg.err != nil {
			m.statusLine = msg.err.Error()
			return m, nil
		}
		if m.reloadConfig != nil {
			opts, err := m.reloadConfig()
			if err != nil {
				m.err = err
				m.statusLine = "config reload failed: " + err.Error()
				return m, nil
			}
			m.applyReloadedOptions(opts)
		} else if m.loadConfig != nil {
			commands, err := m.loadConfig()
			if err != nil {
				m.err = err
				m.statusLine = "config reload failed: " + err.Error()
				return m, nil
			}
			m.customCommands = append([]CustomCommand(nil), commands...)
		}
		m.statusLine = "edited config " + msg.path
		return m, nil
	case autoRefreshMsg:
		return m.handleAutoRefresh()
	case tea.MouseMsg:
		return m.handleMouse(msg)
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) setActiveVisibleIndex(index int) bool {
	switch m.active {
	case resourceContainers:
		if index < 0 || index >= len(m.filteredContainerIndexes()) {
			return false
		}
		m.containerCursor = index
	case resourceServices:
		if index < 0 || index >= len(m.filteredServiceIndexes()) {
			return false
		}
		m.serviceCursor = index
	case resourceImages:
		if index < 0 || index >= len(m.filteredImageIndexes()) {
			return false
		}
		m.imageCursor = index
	case resourceBuilder:
		return index == 0 && m.builderMatchesFilter()
	case resourceVolumes:
		if index < 0 || index >= len(m.filteredVolumeIndexes()) {
			return false
		}
		m.volumeCursor = index
	case resourceNetworks:
		if index < 0 || index >= len(m.filteredNetworkIndexes()) {
			return false
		}
		m.networkCursor = index
	case resourceMachines:
		if index < 0 || index >= len(m.filteredMachineIndexes()) {
			return false
		}
		m.machineCursor = index
	case resourceRegistries:
		if index < 0 || index >= len(m.filteredRegistryIndexes()) {
			return false
		}
		m.registryCursor = index
	case resourceSystem:
		return index == 0 && m.systemMatchesFilter()
	default:
		return false
	}
	return true
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if m.menu != nil {
		return m.handleMenuKey(msg)
	}
	if m.bulkMenu != nil {
		return m.handleBulkMenuKey(msg)
	}
	if m.showHelp {
		return m.handleHelpKey(msg)
	}
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
		m.stopStream()
		return m, tea.Quit
	case "/":
		return m.startFiltering(), nil
	case "esc":
		if m.bufferKind == bufOutput {
			m.bufferKind = bufNone
			m.panelOffset = 0
			m.statusLine = "closed output"
			cmd := m.ensureMainPanelCmd()
			return m, cmd
		}
		if m.filter != "" {
			m.filter = ""
			m.filterInput = ""
			m.clampCursors()
			m.resetPanel()
			m.statusLine = "filter cleared"
			cmd := m.ensureMainPanelCmd()
			return m, cmd
		}
		return m, nil
	}
	if keyRune(msg) == "/" {
		return m.startFiltering(), nil
	}

	// The Services pane owns a handful of keys (lifecycle, up/down, recreate);
	// delegate to its handler first and let everything else fall through to the
	// global keys below.
	if m.active == resourceServices {
		if model, cmd, handled := m.handleServiceKey(key); handled {
			return model, cmd
		}
	}

	switch key {
	case "?":
		m.showHelp = true
		m.helpOffset = 0
		return m, nil
	case " ", "space":
		return m.openActionMenu()
	case "B":
		return m.openBulkMenu()
	case ":":
		return m.startContainerCommandPrompt(), nil
	case ";":
		return m.startCustomCommandPrompt()
	case "tab", "right":
		m.goToResource((m.active + 1) % resourceCount)
		cmd := m.ensureMainPanelCmd()
		return m, cmd
	case "shift+tab", "left", "h":
		m.goToResource((m.active - 1 + resourceCount) % resourceCount)
		cmd := m.ensureMainPanelCmd()
		return m, cmd
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		m.goToResource(resourceKind(key[0] - '1'))
		cmd := m.ensureMainPanelCmd()
		return m, cmd
	case "[":
		cmd := m.cycleTab(-1)
		return m, cmd
	case "]":
		cmd := m.cycleTab(1)
		return m, cmd
	case "+", "=":
		m.screenMode = (m.screenMode + 1) % screenModeCount
		m.statusLine = "screen: " + m.screenMode.label()
		return m, nil
	case "_", "-":
		m.screenMode = (m.screenMode - 1 + screenModeCount) % screenModeCount
		m.statusLine = "screen: " + m.screenMode.label()
		return m, nil
	case "r":
		m.busy = "refreshing"
		m.statusLine = "refreshing"
		return m, m.refreshWithActiveTab(nil)
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
	case "o":
		return m.openConfigSelected()
	case "up", "k":
		m.moveSelection(-1)
		cmd := m.ensureMainPanelCmd()
		return m, cmd
	case "down", "j":
		m.moveSelection(1)
		cmd := m.ensureMainPanelCmd()
		return m, cmd
	case "home":
		m.panelOffset = 0
		m.logFollow = false
		return m, nil
	case "end":
		m.panelOffset = m.maxPanelOffset()
		m.logFollow = true
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
	case "g":
		return m.startRegistryLoginPrompt()
	case "c":
		return m.startCopyPrompt()
	case "C":
		return m.startCreateResourcePrompt()
	case "E":
		return m.startExportPrompt()
	case "M":
		return m.startCreateMachinePrompt()
	case "m":
		return m.startSetMachinePrompt()
	case "S":
		return m.setDefaultMachine()
	case "l":
		return m.logsSelected()
	case "f":
		return m.followLogsSelected()
	case "ctrl+b":
		return m.bootLogsSelected()
	case "e":
		return m.shellSelected()
	case "w":
		return m.openInBrowser()
	case "X":
		return m.startExecPrompt()
	case "s":
		if m.active == resourceSystem {
			return m.startSystem()
		}
		if m.active == resourceBuilder {
			return m.startBuilder()
		}
		return m.lifecycleSelected("starting", "started", func(ctx context.Context, id string) error {
			return m.client.Start(ctx, id)
		})
	case "x":
		if m.active == resourceSystem {
			m.confirm = &pendingConfirm{label: "Stop all container services?", run: func(ctx context.Context, m Model) (string, error) {
				return "stopped container services", m.client.StopSystem(ctx)
			}}
			return m, nil
		}
		if m.active == resourceBuilder {
			return m.stopBuilder()
		}
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
			m.confirm = &pendingConfirm{label: "Prune stopped containers?", run: func(ctx context.Context, m Model) (string, error) {
				return "pruned stopped containers", m.client.PruneContainers(ctx)
			}}
		case resourceImages:
			m.confirm = &pendingConfirm{label: "Prune unused images?", run: func(ctx context.Context, m Model) (string, error) {
				return "pruned unused images", m.client.PruneImages(ctx, false)
			}}
		case resourceVolumes:
			m.confirm = &pendingConfirm{label: "Prune unused volumes?", run: func(ctx context.Context, m Model) (string, error) {
				return "pruned unused volumes", m.client.PruneVolumes(ctx)
			}}
		case resourceNetworks:
			m.confirm = &pendingConfirm{label: "Prune unused networks?", run: func(ctx context.Context, m Model) (string, error) {
				return "pruned unused networks", m.client.PruneNetworks(ctx)
			}}
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

func (m Model) startRegistryLoginPrompt() (tea.Model, tea.Cmd) {
	if m.active != resourceRegistries {
		return m, nil
	}
	m.prompt = promptRegistryLogin
	m.promptInput = ""
	m.promptTarget = ""
	m.statusLine = "registry login"
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

func (m Model) startContainerCommandPrompt() Model {
	m.prompt = promptContainerCommand
	m.promptInput = ""
	m.promptTarget = ""
	m.statusLine = "container command"
	return m
}

func (m Model) startCustomCommandPrompt() (tea.Model, tea.Cmd) {
	if len(m.customCommands) == 0 {
		m.statusLine = "no custom commands configured"
		return m, nil
	}
	m.prompt = promptCustomCommand
	m.promptInput = ""
	m.promptTarget = ""
	m.statusLine = "custom command"
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

func (m Model) startSetMachinePrompt() (tea.Model, tea.Cmd) {
	if m.active != resourceMachines {
		return m, nil
	}
	machine, ok := m.selectedMachine()
	if !ok {
		return m, nil
	}
	id := machine.Name()
	m.prompt = promptSetMachine
	m.promptInput = ""
	m.promptTarget = id
	m.statusLine = "configure machine " + id
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

// bootLogsSelected shows the VM boot logs for the selected container or
// machine in the output buffer — an Apple-specific diagnostic since each
// container/machine boots its own lightweight VM.
func (m Model) bootLogsSelected() (tea.Model, tea.Cmd) {
	lines := m.logTail()
	switch m.active {
	case resourceContainers:
		container, ok := m.selectedContainer()
		if !ok {
			return m, nil
		}
		id := container.Name()
		m.busy = "loading boot logs for " + id
		m.statusLine = m.busy
		return m, m.containerBootLogsCmd(id, lines)
	case resourceServices:
		service, ok := m.selectedService()
		if !ok {
			return m, nil
		}
		container, ok := m.serviceContainer(service)
		if !ok {
			m.statusLine = service.Name + " has no container yet — press u to bring it up"
			return m, nil
		}
		id := container.Name()
		m.busy = "loading boot logs for " + id
		m.statusLine = m.busy
		return m, m.containerBootLogsCmd(id, lines)
	case resourceMachines:
		machine, ok := m.selectedMachine()
		if !ok {
			return m, nil
		}
		id := machine.Name()
		m.busy = "loading boot logs for " + id
		m.statusLine = m.busy
		return m, func() tea.Msg {
			body, err := m.client.MachineBootLogs(context.Background(), id, lines)
			if strings.TrimSpace(body) == "" && err == nil {
				body = "No boot log output."
			}
			return outputMsg{title: "Boot logs " + id, body: body, err: err}
		}
	}
	return m, nil
}

// containerBootLogsCmd fetches a container's VM boot logs into the output buffer.
func (m Model) containerBootLogsCmd(id string, lines int) tea.Cmd {
	return func() tea.Msg {
		body, err := m.client.BootLogs(context.Background(), id, lines)
		if strings.TrimSpace(body) == "" && err == nil {
			body = "No boot log output."
		}
		return outputMsg{title: "Boot logs " + id, body: body, err: err}
	}
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
			body, err := m.client.PullImage(context.Background(), reference)
			return outputMsg{title: "Pull " + reference, body: commandOutputBody(body), err: err, refresh: true}
		}
	case promptRunImage:
		image := m.promptTarget
		options, ok := parseContainerLaunchInput(m.promptInput)
		m.prompt = promptNone
		m.promptInput = ""
		m.promptTarget = ""
		if strings.TrimSpace(image) == "" || !ok {
			m.statusLine = "run cancelled"
			return m, nil
		}
		m.busy = "running " + image
		m.statusLine = "running " + image
		return m, func() tea.Msg {
			err := m.client.RunImage(context.Background(), image, options)
			message := "started container from " + image
			if name := containerLaunchName(options); name != "" {
				message = "started " + name
			}
			return actionDoneMsg{message: message, err: err}
		}
	case promptCreateContainer:
		image := m.promptTarget
		options, ok := parseContainerLaunchInput(m.promptInput)
		m.prompt = promptNone
		m.promptInput = ""
		m.promptTarget = ""
		if strings.TrimSpace(image) == "" || !ok {
			m.statusLine = "container create cancelled"
			return m, nil
		}
		m.busy = "creating container from " + image
		m.statusLine = "creating container from " + image
		return m, func() tea.Msg {
			err := m.client.CreateContainer(context.Background(), image, options)
			message := "created container from " + image
			if name := containerLaunchName(options); name != "" {
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
			body, err := m.client.BuildImage(context.Background(), tag, contextDir)
			return outputMsg{title: "Build " + tag, body: commandOutputBody(body), err: err, refresh: true}
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
		return m, func() tea.Msg {
			body, err := m.client.Exec(context.Background(), container, command)
			if strings.TrimSpace(body) == "" && err == nil {
				body = "Command completed with no output."
			}
			return outputMsg{title: "Exec " + container, body: body, err: err}
		}
	case promptContainerCommand:
		input := m.promptInput
		args, ok := splitPromptArgs(input)
		m.prompt = promptNone
		m.promptInput = ""
		m.promptTarget = ""
		if !ok || len(args) == 0 {
			m.statusLine = "container command cancelled"
			return m, nil
		}
		title := "container " + strings.Join(args, " ")
		m.busy = "running " + title
		m.statusLine = "running " + title
		return m, func() tea.Msg {
			body, err := m.client.Command(context.Background(), args)
			if strings.TrimSpace(body) == "" && err == nil {
				body = "Command completed with no output."
			}
			return outputMsg{title: title, body: body, err: err}
		}
	case promptCustomCommand:
		input := m.promptInput
		m.prompt = promptNone
		m.promptInput = ""
		m.promptTarget = ""
		command, ok := m.customCommandByInput(input)
		if !ok {
			m.statusLine = "custom command not found"
			return m, nil
		}
		title := "Custom " + command.Name
		args, missing, ok := m.expandCustomCommandArgs(command.Args)
		if !ok {
			m.statusLine = "custom command needs selected " + missing
			return m, nil
		}
		m.busy = "running " + title
		m.statusLine = "running " + title
		if command.Attach {
			cmd, err := m.client.CommandProcess(args)
			if err != nil {
				m.busy = ""
				m.err = err
				m.statusLine = err.Error()
				return m, nil
			}
			return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
				return actionDoneMsg{message: "ran " + title, err: err}
			})
		}
		return m, func() tea.Msg {
			body, err := m.client.Command(context.Background(), args)
			return outputMsg{title: title, body: commandOutputBody(body), err: err}
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
			body, err := m.client.SaveImage(context.Background(), reference, outputPath)
			return outputMsg{title: "Save " + reference, body: commandOutputBody(body), err: err, refresh: true}
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
			body, err := m.client.LoadImage(context.Background(), inputPath)
			return outputMsg{title: "Load " + inputPath, body: commandOutputBody(body), err: err, refresh: true}
		}
	case promptRegistryLogin:
		server, username, ok := parseRegistryLoginInput(m.promptInput)
		m.prompt = promptNone
		m.promptInput = ""
		m.promptTarget = ""
		if !ok {
			m.statusLine = "registry login cancelled"
			return m, nil
		}
		cmd, err := m.client.RegistryLoginCommand(server, username)
		if err != nil {
			m.err = err
			m.statusLine = err.Error()
			return m, nil
		}
		m.busy = "logging in to registry " + server
		m.statusLine = "logging in to registry " + server
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
			return actionDoneMsg{message: "logged in registry " + server, err: err}
		})
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
	case promptSetMachine:
		machine := m.promptTarget
		settings := parseMachineSettingsInput(m.promptInput)
		m.prompt = promptNone
		m.promptInput = ""
		m.promptTarget = ""
		if strings.TrimSpace(machine) == "" || len(settings) == 0 {
			m.statusLine = "machine configure cancelled"
			return m, nil
		}
		m.busy = "configuring machine " + machine
		m.statusLine = "configuring machine " + machine
		return m, func() tea.Msg {
			err := m.client.SetMachine(context.Background(), machine, settings)
			return actionDoneMsg{message: "configured machine " + machine, err: err}
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

func parseContainerLaunchInput(input string) (containercli.ContainerLaunchOptions, bool) {
	tokens, ok := splitPromptArgs(input)
	if !ok {
		return containercli.ContainerLaunchOptions{}, false
	}
	options := containercli.ContainerLaunchOptions{}
	for idx := 0; idx < len(tokens); idx++ {
		token := tokens[idx]
		if token == "--" {
			options.Arguments = append(options.Arguments, tokens[idx+1:]...)
			return options, true
		}
		if name, value, ok := splitLaunchAssignment(token); ok {
			if name == "name" {
				options.Name = value
				continue
			}
			if flag, ok := launchAssignmentFlag(name); ok {
				options.Flags = append(options.Flags, flag, value)
				continue
			}
		}
		if strings.Contains(token, "=") && !strings.HasPrefix(token, "-") {
			return containercli.ContainerLaunchOptions{}, false
		}
		if flag, ok := launchBooleanFlag(token); ok {
			options.Flags = append(options.Flags, flag)
			continue
		}
		if launchFlagNeedsValue(token) {
			if idx+1 >= len(tokens) || tokens[idx+1] == "--" {
				return containercli.ContainerLaunchOptions{}, false
			}
			options.Flags = append(options.Flags, token, tokens[idx+1])
			idx++
			continue
		}
		if strings.HasPrefix(token, "-") {
			options.Flags = append(options.Flags, token)
			continue
		}
		if options.Name == "" {
			options.Name = token
			continue
		}
		return containercli.ContainerLaunchOptions{}, false
	}
	return options, true
}

func splitPromptArgs(input string) ([]string, bool) {
	var tokens []string
	var current strings.Builder
	var quote rune
	escaped := false
	inToken := false

	for _, r := range input {
		if escaped {
			current.WriteRune(r)
			escaped = false
			inToken = true
			continue
		}
		if r == '\\' {
			escaped = true
			inToken = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
				continue
			}
			current.WriteRune(r)
			inToken = true
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
			inToken = true
		case ' ', '\t', '\n', '\r':
			if inToken {
				tokens = append(tokens, current.String())
				current.Reset()
				inToken = false
			}
		default:
			current.WriteRune(r)
			inToken = true
		}
	}
	if escaped {
		current.WriteRune('\\')
	}
	if quote != 0 {
		return nil, false
	}
	if inToken {
		tokens = append(tokens, current.String())
	}
	return tokens, true
}

func splitLaunchAssignment(token string) (string, string, bool) {
	if strings.HasPrefix(token, "-") {
		return "", "", false
	}
	name, value, ok := strings.Cut(token, "=")
	if !ok || strings.TrimSpace(name) == "" || strings.TrimSpace(value) == "" {
		return "", "", false
	}
	return strings.ToLower(strings.TrimSpace(name)), value, true
}

func launchAssignmentFlag(name string) (string, bool) {
	flags := map[string]string{
		"a":                        "--arch",
		"arch":                     "--arch",
		"c":                        "--cpus",
		"cap-add":                  "--cap-add",
		"cap-drop":                 "--cap-drop",
		"cidfile":                  "--cidfile",
		"cpus":                     "--cpus",
		"cwd":                      "--workdir",
		"dns":                      "--dns",
		"dns-domain":               "--dns-domain",
		"dns-option":               "--dns-option",
		"dns-search":               "--dns-search",
		"e":                        "--env",
		"entrypoint":               "--entrypoint",
		"env":                      "--env",
		"env-file":                 "--env-file",
		"gid":                      "--gid",
		"init-image":               "--init-image",
		"k":                        "--kernel",
		"kernel":                   "--kernel",
		"l":                        "--label",
		"label":                    "--label",
		"m":                        "--memory",
		"max-concurrent-downloads": "--max-concurrent-downloads",
		"memory":                   "--memory",
		"mount":                    "--mount",
		"network":                  "--network",
		"os":                       "--os",
		"p":                        "--publish",
		"platform":                 "--platform",
		"progress":                 "--progress",
		"publish":                  "--publish",
		"publish-socket":           "--publish-socket",
		"runtime":                  "--runtime",
		"scheme":                   "--scheme",
		"shm-size":                 "--shm-size",
		"tmpfs":                    "--tmpfs",
		"u":                        "--user",
		"uid":                      "--uid",
		"ulimit":                   "--ulimit",
		"user":                     "--user",
		"v":                        "--volume",
		"volume":                   "--volume",
		"w":                        "--workdir",
		"workdir":                  "--workdir",
	}
	flag, ok := flags[name]
	return flag, ok
}

func launchBooleanFlag(token string) (string, bool) {
	flags := map[string]string{
		"detach":         "--detach",
		"init":           "--init",
		"interactive":    "--interactive",
		"no-dns":         "--no-dns",
		"read-only":      "--read-only",
		"remove":         "--remove",
		"rm":             "--rm",
		"rosetta":        "--rosetta",
		"ssh":            "--ssh",
		"tty":            "--tty",
		"virtualization": "--virtualization",
	}
	flag, ok := flags[strings.ToLower(token)]
	return flag, ok
}

func launchFlagNeedsValue(token string) bool {
	if strings.Contains(token, "=") {
		return false
	}
	flags := map[string]struct{}{
		"-a":                         {},
		"-c":                         {},
		"-e":                         {},
		"-k":                         {},
		"-l":                         {},
		"-m":                         {},
		"-p":                         {},
		"-u":                         {},
		"-v":                         {},
		"-w":                         {},
		"--arch":                     {},
		"--cap-add":                  {},
		"--cap-drop":                 {},
		"--cidfile":                  {},
		"--cpus":                     {},
		"--cwd":                      {},
		"--dns":                      {},
		"--dns-domain":               {},
		"--dns-option":               {},
		"--dns-search":               {},
		"--entrypoint":               {},
		"--env":                      {},
		"--env-file":                 {},
		"--gid":                      {},
		"--init-image":               {},
		"--kernel":                   {},
		"--label":                    {},
		"--max-concurrent-downloads": {},
		"--memory":                   {},
		"--mount":                    {},
		"--name":                     {},
		"--network":                  {},
		"--os":                       {},
		"--platform":                 {},
		"--progress":                 {},
		"--publish":                  {},
		"--publish-socket":           {},
		"--runtime":                  {},
		"--scheme":                   {},
		"--shm-size":                 {},
		"--tmpfs":                    {},
		"--uid":                      {},
		"--ulimit":                   {},
		"--user":                     {},
		"--volume":                   {},
		"--workdir":                  {},
	}
	_, ok := flags[token]
	return ok
}

func containerLaunchName(options containercli.ContainerLaunchOptions) string {
	for idx, flag := range options.Flags {
		if flag == "--name" && idx+1 < len(options.Flags) {
			return strings.TrimSpace(options.Flags[idx+1])
		}
		if strings.HasPrefix(flag, "--name=") {
			return strings.TrimSpace(strings.TrimPrefix(flag, "--name="))
		}
	}
	if strings.TrimSpace(options.Name) != "" {
		return strings.TrimSpace(options.Name)
	}
	return ""
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

func parseMachineSettingsInput(input string) []string {
	fields := strings.Fields(strings.TrimSpace(input))
	settings := make([]string, 0, len(fields))
	for _, field := range fields {
		if strings.Contains(field, "=") {
			settings = append(settings, field)
		}
	}
	return settings
}

func parseRegistryLoginInput(input string) (string, string, bool) {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) == 0 || len(fields) > 2 {
		return "", "", false
	}
	username := ""
	if len(fields) == 2 {
		username = fields[1]
	}
	return fields[0], username, true
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

func commandOutputBody(body string) string {
	if strings.TrimSpace(body) == "" {
		return "Command completed with no output."
	}
	return body
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
		if usage, err := m.client.SystemDiskUsage(ctx); err != nil {
			errs = append(errs, err)
		} else {
			msg.systemUsage = usage
		}
		if versions, err := m.client.SystemVersion(ctx); err != nil {
			errs = append(errs, err)
		} else {
			msg.systemVersions = versions
		}
		// DNS domains and system properties are best-effort: they are newer
		// subcommands, so an error (older CLI) must not surface as a failure. The
		// OK flags let the snapshot handler keep the last-known values on a
		// transient error instead of clobbering them with an empty slice.
		if domains, err := m.client.SystemDNS(ctx); err == nil {
			msg.systemDNS = domains
			msg.systemDNSOK = true
		}
		if properties, err := m.client.SystemProperties(ctx); err == nil {
			msg.systemProperties = properties
			msg.systemPropsOK = true
		}
		// Re-read the Compose file each refresh so edits apply live.
		if m.loadProject != nil {
			msg.project, msg.projectErr = m.loadProject()
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
		if registries, err := m.client.Registries(ctx); err != nil {
			errs = append(errs, err)
		} else {
			sort.Slice(registries, func(i, j int) bool {
				return registries[i].Name() < registries[j].Name()
			})
			msg.registries = registries
		}
		if builder, err := m.client.BuilderStatus(ctx); err != nil {
			errs = append(errs, err)
		} else {
			msg.builder = builder
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

func (m Model) refreshWithActiveTab(nextTick tea.Cmd) tea.Cmd {
	cmds := []tea.Cmd{m.refreshCmd()}
	if cmd := m.forceFetchActiveTabCmd(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	if nextTick != nil {
		cmds = append(cmds, nextTick)
	}
	return tea.Batch(cmds...)
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
	return m, m.refreshWithActiveTab(nextTick)
}

// inspectSelected jumps the main panel to the Inspect tab (falling back to the
// Config tab for resources without a dedicated inspect command).
func (m Model) inspectSelected() (tea.Model, tea.Cmd) {
	cmd := m.activateTab(tabInspect)
	return m, cmd
}

// logsSelected jumps the main panel to the Logs tab when the resource supports
// it.
func (m Model) logsSelected() (tea.Model, tea.Cmd) {
	if !hasTab(m.active, tabLogs) {
		m.statusLine = "no logs for " + resourceLabel(m.active)
		return m, nil
	}
	cmd := m.activateTab(tabLogs)
	return m, cmd
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
	case resourceServices:
		service, ok := m.selectedService()
		if !ok {
			return "", nil, nil
		}
		container, ok := m.serviceContainer(service)
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
	case resourceSystem:
		if !m.systemMatchesFilter() {
			return "", nil, nil
		}
		cmd, err := m.client.FollowSystemLogsCommand("5m")
		return "system", cmd, err
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
	if container.State() != "running" {
		m.statusLine = "start " + container.Name() + " before opening a shell"
		return m, nil
	}
	return m.shellForContainer(container.Name())
}

// shellForContainer opens an interactive /bin/sh in the named container,
// suspending the TUI until the shell exits. Shared by container and service shells.
func (m Model) shellForContainer(id string) (tea.Model, tea.Cmd) {
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

// openInBrowser opens the selected container's first published port in the
// system browser (fire-and-forget, without suspending the TUI).
func (m Model) openInBrowser() (tea.Model, tea.Cmd) {
	if m.active != resourceContainers {
		return m, nil
	}
	if m.openLink == nil {
		m.statusLine = "open-in-browser unavailable"
		return m, nil
	}
	container, ok := m.selectedContainer()
	if !ok {
		return m, nil
	}
	url, ok := container.FirstPublishedURL()
	if !ok {
		m.statusLine = "no published ports for " + container.Name()
		return m, nil
	}
	openLink := m.openLink
	m.statusLine = "opening " + url
	return m, func() tea.Msg {
		cmd, err := openLink(url)
		if err == nil && cmd != nil {
			err = cmd.Start()
		}
		return actionDoneMsg{message: "opened " + url, err: err}
	}
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

func (m Model) openConfigSelected() (tea.Model, tea.Cmd) {
	if m.openConfig == nil {
		m.statusLine = "config editor unavailable"
		return m, nil
	}
	if strings.TrimSpace(m.configPath) == "" {
		m.statusLine = "config path unavailable"
		return m, nil
	}
	cmd, err := m.openConfig(m.configPath)
	if err != nil {
		m.err = err
		m.statusLine = err.Error()
		return m, nil
	}
	m.busy = "editing config"
	m.statusLine = "editing config"
	path := m.configPath
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return configEditedMsg{path: path, err: err}
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
		body, err := m.client.PushImage(context.Background(), reference)
		return outputMsg{title: "Push " + reference, body: commandOutputBody(body), err: err, refresh: true}
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

func (m Model) startBuilder() (tea.Model, tea.Cmd) {
	if m.active != resourceBuilder {
		return m, nil
	}
	m.busy = "starting builder"
	m.statusLine = "starting builder"
	return m, func() tea.Msg {
		err := m.client.StartBuilder(context.Background())
		return actionDoneMsg{message: "started builder", err: err}
	}
}

func (m Model) stopBuilder() (tea.Model, tea.Cmd) {
	if m.active != resourceBuilder {
		return m, nil
	}
	if !m.builder.Present {
		m.statusLine = "builder is not present"
		return m, nil
	}
	m.busy = "stopping builder"
	m.statusLine = "stopping builder"
	return m, func() tea.Msg {
		err := m.client.StopBuilder(context.Background())
		return actionDoneMsg{message: "stopped builder", err: err}
	}
}

func (m Model) startSystem() (tea.Model, tea.Cmd) {
	if m.active != resourceSystem {
		return m, nil
	}
	m.busy = "starting system"
	m.statusLine = "starting container services"
	return m, func() tea.Msg {
		err := m.client.StartSystem(context.Background())
		return actionDoneMsg{message: "started container services", err: err}
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
			m.confirm = &pendingConfirm{label: "Delete container " + id + "?", run: func(ctx context.Context, m Model) (string, error) {
				return "deleted container " + id, m.client.DeleteContainer(ctx, id, false)
			}}
		}
	case resourceImages:
		image, ok := m.selectedImage()
		if ok {
			name := image.Name()
			m.confirm = &pendingConfirm{label: "Delete image " + name + "?", run: func(ctx context.Context, m Model) (string, error) {
				return "deleted image " + name, m.client.DeleteImage(ctx, name, false)
			}}
		}
	case resourceBuilder:
		if m.builder.Present {
			m.confirm = &pendingConfirm{label: "Delete builder?", run: func(ctx context.Context, m Model) (string, error) {
				return "deleted builder", m.client.DeleteBuilder(ctx, false)
			}}
		} else {
			m.statusLine = "builder is not present"
		}
	case resourceVolumes:
		volume, ok := m.selectedVolume()
		if ok {
			name := volume.Name()
			m.confirm = &pendingConfirm{label: "Delete volume " + name + "?", run: func(ctx context.Context, m Model) (string, error) {
				return "deleted volume " + name, m.client.DeleteVolume(ctx, name)
			}}
		}
	case resourceNetworks:
		network, ok := m.selectedNetwork()
		if ok {
			name := network.Name()
			m.confirm = &pendingConfirm{label: "Delete network " + name + "?", run: func(ctx context.Context, m Model) (string, error) {
				return "deleted network " + name, m.client.DeleteNetwork(ctx, name)
			}}
		}
	case resourceMachines:
		machine, ok := m.selectedMachine()
		if ok {
			id := machine.Name()
			m.confirm = &pendingConfirm{label: "Delete machine " + id + "?", run: func(ctx context.Context, m Model) (string, error) {
				return "deleted machine " + id, m.client.DeleteMachine(ctx, id)
			}}
		}
	case resourceRegistries:
		registry, ok := m.selectedRegistry()
		if ok {
			name := registry.Name()
			m.confirm = &pendingConfirm{label: "Log out from registry " + name + "?", run: func(ctx context.Context, m Model) (string, error) {
				return "logged out registry " + name, m.client.LogoutRegistry(ctx, name)
			}}
		}
	}
}

// confirmCmd runs a confirmed action's closure against the current model. Each
// action defines its own execution where it is created, so this dispatcher has
// no per-action knowledge.
func (m Model) confirmCmd(confirm pendingConfirm) tea.Cmd {
	run := confirm.run
	model := m
	return func() tea.Msg {
		if run == nil {
			return actionDoneMsg{err: errors.New("no action to confirm")}
		}
		message, err := run(context.Background(), model)
		return actionDoneMsg{message: message, err: err}
	}
}

func (m *Model) goToResource(kind resourceKind) {
	if kind < 0 || kind >= resourceCount {
		return
	}
	m.active = kind
	m.clampCursors()
	m.resetPanel()
	m.statusLine = "selected " + resourceLabel(kind)
}

func (m *Model) moveSelection(delta int) {
	switch m.active {
	case resourceContainers:
		m.containerCursor += delta
	case resourceServices:
		m.serviceCursor += delta
	case resourceImages:
		m.imageCursor += delta
	case resourceVolumes:
		m.volumeCursor += delta
	case resourceNetworks:
		m.networkCursor += delta
	case resourceMachines:
		m.machineCursor += delta
	case resourceRegistries:
		m.registryCursor += delta
	}
	m.clampCursors()
	m.resetPanel()
}

func (m *Model) clampCursors() {
	m.containerCursor = clamp(m.containerCursor, 0, len(m.filteredContainerIndexes())-1)
	m.serviceCursor = clamp(m.serviceCursor, 0, len(m.filteredServiceIndexes())-1)
	m.imageCursor = clamp(m.imageCursor, 0, len(m.filteredImageIndexes())-1)
	m.volumeCursor = clamp(m.volumeCursor, 0, len(m.filteredVolumeIndexes())-1)
	m.networkCursor = clamp(m.networkCursor, 0, len(m.filteredNetworkIndexes())-1)
	m.machineCursor = clamp(m.machineCursor, 0, len(m.filteredMachineIndexes())-1)
	m.registryCursor = clamp(m.registryCursor, 0, len(m.filteredRegistryIndexes())-1)
}

func (m *Model) resetPanel() {
	m.bufferKind = bufNone
	m.panelTitle = ""
	m.panelBody = ""
	m.panelOffset = 0
	m.clampActiveTab()
}

func (m *Model) scrollPanel(delta int) {
	m.panelOffset += delta
	m.clampPanelOffset()
	// On the Logs tab, manual scrolling away from the bottom detaches autoscroll;
	// scrolling back to the bottom re-enables it.
	m.logFollow = m.panelOffset >= m.maxPanelOffset()
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

func (m Model) selectedRegistry() (containercli.RegistryLogin, bool) {
	indexes := m.filteredRegistryIndexes()
	if len(indexes) == 0 || m.registryCursor < 0 || m.registryCursor >= len(indexes) {
		return containercli.RegistryLogin{}, false
	}
	return m.registries[indexes[m.registryCursor]], true
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
		return appmeta.Name
	}
	if m.height < 8 || m.width < 60 {
		return appmeta.Name + " needs a terminal of at least 60x8"
	}
	if m.showHelp {
		return m.renderHelpOverlay()
	}
	if m.menu != nil {
		return m.renderMenuOverlay()
	}
	if m.bulkMenu != nil {
		return m.renderBulkMenuOverlay()
	}

	top := m.renderTopBar()
	footer := m.renderFooter()
	overview := m.renderOverview()
	overviewHeight := 0
	if overview != "" {
		overviewHeight = lipgloss.Height(overview)
	}
	bodyHeight := m.height - lipgloss.Height(top) - lipgloss.Height(footer) - overviewHeight
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	sidebarWidth := m.sidebarWidth()
	panelWidth := m.width - sidebarWidth

	panel := m.renderPanel(panelWidth, bodyHeight)
	var body string
	if sidebarWidth <= 0 {
		body = panel
	} else {
		sidebar := m.renderSidebar(sidebarWidth, bodyHeight)
		body = lipgloss.JoinHorizontal(lipgloss.Top, sidebar, panel)
	}

	parts := []string{top}
	if overview != "" {
		parts = append(parts, overview)
	}
	parts = append(parts, body, footer)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// fleetStats aggregates the latest per-container samples into a fleet mean CPU%
// and total memory in use. n is how many containers contributed a sample.
func (m Model) fleetStats() (cpuMean float64, memTotal float64, n int) {
	var cpuSum float64
	now := effectiveNow(m.lastUpdated)
	for _, st := range m.stats {
		var sample statHistorySample
		var ok bool
		if id := statIdentifier(st); id != "" {
			sample, ok = m.latestStatSample(id)
		}
		if !ok {
			sample, ok = statHistorySampleFromStat(st, now)
		}
		if !ok {
			continue
		}
		if sample.hasCPU {
			cpuSum += sample.cpuPercent
		}
		if sample.hasMemory {
			memTotal += sample.memoryBytes
		}
		n++
	}
	if n > 0 {
		cpuMean = cpuSum / float64(n)
	}
	return cpuMean, memTotal, n
}

// renderOverview draws the pinned, read-only fleet summary strip shown between
// the top bar and the body. It returns "" when the terminal is too short or an
// overlay is open, so callers can omit the row and reclaim its height.
func (m Model) renderOverview() string {
	if m.width < 60 || m.height < 18 || m.menu != nil || m.showHelp {
		return ""
	}
	running := 0
	for _, c := range m.containers {
		if c.State() == "running" {
			running++
		}
	}
	segs := []string{
		fmt.Sprintf("%d ctr (%d up)", len(m.containers), running),
		fmt.Sprintf("%d img", len(m.images)),
	}
	if cpuMean, memTotal, n := m.fleetStats(); n > 0 {
		segs = append(segs, fmt.Sprintf("cpu %5.1f%%", cpuMean), "mem "+containercli.FormatBytes(int64(memTotal)))
	}
	segs = append(segs, "disk "+m.systemUsage.TotalSize()+" / "+m.systemUsage.TotalReclaimable()+" reclaim")
	if state := m.builder.State(); state != "" {
		segs = append(segs, "builder "+state)
	}
	line := "FLEET  " + strings.Join(segs, " · ")
	return overviewStyle.Width(m.width).Render(truncate(line, m.width-2))
}

// applyTheme overrides panel borders and accent colours from config. It mutates
// package-level styles and is invoked once at startup.
func applyTheme(border string, activeColor string, selectedBg string) {
	if b, ok := borderForName(border); ok {
		panelStyle = panelStyle.Border(b)
		activePanelStyle = panelStyle.BorderForeground(colorActive)
	}
	if strings.TrimSpace(activeColor) != "" {
		colorActive = lipgloss.Color(activeColor)
		activePanelStyle = activePanelStyle.BorderForeground(colorActive)
		tabActiveStyle = tabActiveStyle.Foreground(colorActive)
		sidebarBarStyle = sidebarBarStyle.Foreground(colorActive)
		sidebarActiveStyle = sidebarActiveStyle.Foreground(colorActive)
		keyHintStyle = keyHintStyle.Foreground(colorActive)
	}
	if strings.TrimSpace(selectedBg) != "" {
		selectedStyle = selectedStyle.Background(lipgloss.Color(selectedBg))
	}
}

func borderForName(name string) (lipgloss.Border, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "rounded":
		return lipgloss.RoundedBorder(), true
	case "single", "normal":
		return lipgloss.NormalBorder(), true
	case "double":
		return lipgloss.DoubleBorder(), true
	case "hidden", "none":
		return lipgloss.HiddenBorder(), true
	}
	return lipgloss.Border{}, false
}

var (
	colorText   = lipgloss.Color("252")
	colorMuted  = lipgloss.Color("244")
	colorPanel  = lipgloss.Color("238")
	colorActive = lipgloss.Color("39")
	colorGreen  = lipgloss.Color("42")
	colorRed    = lipgloss.Color("203")
	colorYellow = lipgloss.Color("214")

	topStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Bold(true).
			Padding(0, 1)
	footerStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)
	overviewStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Background(lipgloss.Color("236")).
			Padding(0, 1)
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPanel).
			Padding(0, 1)
	activePanelStyle = panelStyle.Copy().
				BorderForeground(colorActive)
	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("57")).
			Bold(true)
	// cursorRestStyle marks the selected row of an UNfocused section: bold but
	// without the bright background, so each panel shows its current item
	// without competing with the focused panel's highlight.
	cursorRestStyle = lipgloss.NewStyle().Foreground(colorText).Bold(true)
	mutedStyle      = lipgloss.NewStyle().Foreground(colorMuted)
	errorStyle      = lipgloss.NewStyle().Foreground(colorRed)

	runningStyle = lipgloss.NewStyle().Foreground(colorGreen)
	stoppedStyle = lipgloss.NewStyle().Foreground(colorYellow)

	tabActiveStyle = lipgloss.NewStyle().Foreground(colorActive).Bold(true).Underline(true)

	// Sidebar section headers (lazydocker-style stacked panels). The focused
	// section gets an accent bar and a bold accent title; the rest are muted.
	sidebarBarStyle    = lipgloss.NewStyle().Foreground(colorActive).Bold(true)
	sidebarActiveStyle = lipgloss.NewStyle().Foreground(colorActive).Bold(true)
	sidebarHeaderStyle = lipgloss.NewStyle().Foreground(colorText).Bold(true)
	keyHintStyle       = lipgloss.NewStyle().Foreground(colorActive).Bold(true)
	topNameStyle       = lipgloss.NewStyle().Foreground(colorActive).Bold(true)
)

// statusDot returns a colored ● reflecting a resource/service state.
func statusDot(state string) string {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "running", "ready", "active":
		return runningStyle.Render("●")
	case "stopped", "exited", "not running", "inactive", "down":
		return errorStyle.Render("●")
	case "", "unknown":
		return mutedStyle.Render("●")
	default:
		return stoppedStyle.Render("●")
	}
}

func (m Model) renderTopBar() string {
	status := m.system.Status
	if status == "" {
		status = "unknown"
	}
	left := topNameStyle.Render(appmeta.Name) + "  " + statusDot(status) + " " + mutedStyle.Render(status)
	leftWidth := len(appmeta.Name) + 2 + 1 + 1 + len(status)
	if m.busy != "" {
		left += mutedStyle.Render("  ·  ") + stoppedStyle.Render(m.busy)
		leftWidth += 5 + len(m.busy)
	}
	right, rightWidth := "", 0
	if !m.lastUpdated.IsZero() {
		text := "updated " + m.lastUpdated.Format("15:04:05")
		right, rightWidth = mutedStyle.Render(text), len(text)
	}
	inner := m.width - 2
	gap := inner - leftWidth - rightWidth
	if gap < 1 {
		gap, right = 1, ""
	}
	return topStyle.Width(m.width).Render(left + strings.Repeat(" ", gap) + right)
}

func (m Model) renderFooter() string {
	if m.confirm != nil {
		line := stoppedStyle.Render(m.confirm.label) + mutedStyle.Render("  ") + keyHints([][2]string{{"y/enter", "confirm"}, {"n/esc", "cancel"}})
		return footerStyle.Width(m.width).Render(truncate(line, m.width-2))
	}
	if m.prompt != promptNone {
		return footerStyle.Width(m.width).Foreground(colorActive).Render(truncate(m.promptLine(), m.width-2))
	}
	if m.filtering {
		value := m.filterInput
		if value == "" {
			value = " "
		}
		line := keyHintStyle.Render("/") + " " + value + "  " + keyHints([][2]string{{"enter", "apply"}, {"esc", "cancel"}, {"ctrl+u", "clear"}})
		return footerStyle.Width(m.width).Render(truncate(line, m.width-2))
	}

	status := m.statusLine
	if status == "" {
		status = "ready"
	}
	if activeFilter(m.filter) != "" {
		status = status + " · filter: " + truncate(m.filter, 18)
	}
	status = status + " · " + m.autoRefreshLabel()

	inner := m.width - 2
	// Reserve a stable slot for the status text so the hint strip doesn't
	// reflow every time a routine message ("refreshed" → "loaded images")
	// changes length. Short messages all share the reserved width, keeping the
	// hints anchored; only an unusually long message (a guard or error) is
	// allowed to eat into the hint space, dropping the lowest-priority hints.
	const statusReserve = 30
	statusFootprint := len(status)
	if statusFootprint < statusReserve {
		statusFootprint = statusReserve
	}
	hints, hintsWidth := fitKeyHints(m.footerKeyHints(), inner-statusFootprint-1)
	statusText := truncate(status, inner-hintsWidth-1)
	statusStyled := mutedStyle.Render(statusText)
	if m.err != nil {
		statusStyled = errorStyle.Render(statusText)
	}
	gap := inner - len(statusText) - hintsWidth
	if gap < 1 {
		gap = 1
	}
	return footerStyle.Width(m.width).Render(statusStyled + strings.Repeat(" ", gap) + hints)
}

// footerKeyHints returns the key/label pairs shown at the bottom-right. The
// always-available globals come first so they survive when space is tight; the
// focused resource's primary actions follow.
func (m Model) footerKeyHints() [][2]string {
	pairs := [][2]string{{"space", "menu"}, {"?", "help"}, {"q", "quit"}}
	switch m.active {
	case resourceContainers:
		pairs = append(pairs, [2]string{"s", "start"}, [2]string{"x", "stop"}, [2]string{"l", "logs"}, [2]string{"e", "shell"})
	case resourceServices:
		pairs = append(pairs, [2]string{"u", "up"}, [2]string{"d", "down"}, [2]string{"R", "recreate"}, [2]string{"l", "logs"})
	case resourceImages:
		pairs = append(pairs, [2]string{"a", "pull"}, [2]string{"R", "run"}, [2]string{"b", "build"})
	case resourceVolumes, resourceNetworks:
		pairs = append(pairs, [2]string{"C", "create"}, [2]string{"d", "delete"})
	case resourceMachines:
		pairs = append(pairs, [2]string{"M", "new"}, [2]string{"e", "shell"}, [2]string{"S", "default"})
	case resourceRegistries:
		pairs = append(pairs, [2]string{"g", "login"}, [2]string{"d", "logout"})
	case resourceBuilder, resourceSystem:
		pairs = append(pairs, [2]string{"s", "start"}, [2]string{"x", "stop"})
	}
	return pairs
}

// keyHints renders "key label · key label …" with accented keys and muted
// labels.
func keyHints(pairs [][2]string) string {
	segments := make([]string, 0, len(pairs))
	for _, p := range pairs {
		segments = append(segments, keyHintStyle.Render(p[0])+" "+mutedStyle.Render(p[1]))
	}
	return strings.Join(segments, mutedStyle.Render(" · "))
}

// fitKeyHints renders as many leading pairs as fit in maxWidth and returns the
// rendered string with its visible width.
func fitKeyHints(pairs [][2]string, maxWidth int) (string, int) {
	if maxWidth <= 0 {
		return "", 0
	}
	width, kept := 0, 0
	for i, p := range pairs {
		add := len(p[0]) + 1 + len(p[1])
		if i > 0 {
			add += 3 // " · "
		}
		if width+add > maxWidth {
			break
		}
		width += add
		kept++
	}
	return keyHints(pairs[:kept]), width
}

func (m Model) autoRefreshLabel() string {
	if m.autoRefresh {
		return "u auto:on "
	}
	return "u auto:off"
}

func (m Model) promptLine() string {
	switch m.prompt {
	case promptPullImage:
		return "pull image: " + m.promptInput + "  enter pull, esc cancel"
	case promptRunImage:
		return "run opts for " + m.promptTarget + ": " + m.promptInput + "  name=web p=8080:80 env=K=V -- cmd, esc cancel"
	case promptCreateContainer:
		return "create opts for " + m.promptTarget + ": " + m.promptInput + "  name=web p=8080:80 env=K=V -- cmd, esc cancel"
	case promptBuildImage:
		return "build image tag [context-dir]: " + m.promptInput + "  enter build, context defaults ., esc cancel"
	case promptTagImage:
		return "new tag for " + m.promptTarget + ": " + m.promptInput + "  enter tag, esc cancel"
	case promptCopy:
		return "copy for " + m.promptTarget + " src dest (:path is selected container): " + m.promptInput + "  enter copy, esc cancel"
	case promptCreateMachine:
		return "machine image [name]: " + m.promptInput + "  enter create, name optional, esc cancel"
	case promptSetMachine:
		return "machine settings for " + m.promptTarget + ": " + m.promptInput + "  enter set, e.g. cpus=4 memory=8G home-mount=ro, esc cancel"
	case promptExportContainer:
		return "export " + m.promptTarget + " to tar path: " + m.promptInput + "  enter export, ctrl+u clear, esc cancel"
	case promptExecCommand:
		return "exec in " + m.promptTarget + ": " + m.promptInput + "  enter run, esc cancel"
	case promptContainerCommand:
		return "container command: " + m.promptInput + "  enter run, e.g. image list --format json, esc cancel"
	case promptCustomCommand:
		return "custom command: " + m.promptInput + "  enter run by number/name, " + m.customCommandChoices() + ", esc cancel"
	case promptSaveImage:
		return "save " + m.promptTarget + " to tar path: " + m.promptInput + "  enter save, ctrl+u clear, esc cancel"
	case promptLoadImage:
		return "load image archive path: " + m.promptInput + "  enter load, esc cancel"
	case promptCreateVolume:
		return "volume name [size]: " + m.promptInput + "  enter create, esc cancel"
	case promptCreateNetwork:
		return "network name [subnet]: " + m.promptInput + "  enter create, esc cancel"
	case promptRegistryLogin:
		return "registry server [username]: " + m.promptInput + "  enter login, esc cancel"
	default:
		return ""
	}
}

func (m Model) customCommandChoices() string {
	if len(m.customCommands) == 0 {
		return "none configured"
	}
	choices := make([]string, 0, len(m.customCommands))
	for idx, command := range m.customCommands {
		choices = append(choices, fmt.Sprintf("%d=%s", idx+1, command.Name))
	}
	return strings.Join(choices, ", ")
}

func (m Model) customCommandByInput(input string) (CustomCommand, bool) {
	query := strings.ToLower(strings.TrimSpace(input))
	if query == "" {
		return CustomCommand{}, false
	}
	if number, err := strconv.Atoi(query); err == nil {
		index := number - 1
		if index >= 0 && index < len(m.customCommands) {
			return m.customCommands[index], true
		}
		return CustomCommand{}, false
	}

	for _, command := range m.customCommands {
		if strings.ToLower(command.Name) == query {
			return command, true
		}
	}

	var match CustomCommand
	matches := 0
	for _, command := range m.customCommands {
		if strings.Contains(strings.ToLower(command.Name), query) {
			match = command
			matches++
		}
	}
	return match, matches == 1
}

func (m Model) expandCustomCommandArgs(args []string) ([]string, string, bool) {
	values := m.customCommandPlaceholderValues()
	out := make([]string, len(args))
	for idx, arg := range args {
		expanded := arg
		for _, entry := range customCommandPlaceholders() {
			placeholder := entry.placeholder
			value := values[placeholder]
			if strings.Contains(expanded, placeholder) {
				if value == "" {
					return nil, entry.label, false
				}
				expanded = strings.ReplaceAll(expanded, placeholder, value)
			}
		}
		out[idx] = expanded
	}
	return out, "", true
}

func (m Model) customCommandPlaceholderValues() map[string]string {
	values := map[string]string{
		"{container}": "",
		"{image}":     "",
		"{volume}":    "",
		"{network}":   "",
		"{machine}":   "",
		"{registry}":  "",
		"{resource}":  "",
	}
	if container, ok := m.selectedContainer(); ok {
		values["{container}"] = container.Name()
	}
	if image, ok := m.selectedImage(); ok {
		values["{image}"] = image.Name()
	}
	if volume, ok := m.selectedVolume(); ok {
		values["{volume}"] = volume.Name()
	}
	if network, ok := m.selectedNetwork(); ok {
		values["{network}"] = network.Name()
	}
	if machine, ok := m.selectedMachine(); ok {
		values["{machine}"] = machine.Name()
	}
	if registry, ok := m.selectedRegistry(); ok {
		values["{registry}"] = registry.Name()
	}
	if resource := m.selectedResourceName(); resource != "" {
		values["{resource}"] = resource
	}
	return values
}

func (m Model) selectedResourceName() string {
	switch m.active {
	case resourceContainers:
		if container, ok := m.selectedContainer(); ok {
			return container.Name()
		}
	case resourceServices:
		if service, ok := m.selectedService(); ok {
			// Prefer the backing container name so {resource}/{container}
			// placeholders target something runnable.
			if container, ok := m.serviceContainer(service); ok {
				return container.Name()
			}
			return m.project.ContainerNameFor(service)
		}
	case resourceImages:
		if image, ok := m.selectedImage(); ok {
			return image.Name()
		}
	case resourceBuilder:
		if m.builderMatchesFilter() {
			return m.builder.Name()
		}
	case resourceVolumes:
		if volume, ok := m.selectedVolume(); ok {
			return volume.Name()
		}
	case resourceNetworks:
		if network, ok := m.selectedNetwork(); ok {
			return network.Name()
		}
	case resourceMachines:
		if machine, ok := m.selectedMachine(); ok {
			return machine.Name()
		}
	case resourceRegistries:
		if registry, ok := m.selectedRegistry(); ok {
			return registry.Name()
		}
	case resourceSystem:
		if m.systemMatchesFilter() {
			return "system"
		}
	}
	return ""
}

func customCommandPlaceholders() []struct {
	placeholder string
	label       string
} {
	return []struct {
		placeholder string
		label       string
	}{
		{"{container}", "container"},
		{"{image}", "image"},
		{"{volume}", "volume"},
		{"{network}", "network"},
		{"{machine}", "machine"},
		{"{registry}", "registry"},
		{"{resource}", "resource"},
	}
}

func (m Model) renderPanel(width int, height int) string {
	style := panelStyle.Width(width - 2).Height(height - 2)
	title, body := m.panelContent()
	contentWidth := width - 4
	textHeight := panelTextHeight(height)

	header := m.renderPanelHeader(title, contentWidth)
	lines := []string{header, ""}
	renderedBody := renderTextWindow(body, contentWidth, textHeight, &m.panelOffset)
	lines = append(lines, renderedBody...)
	return style.Render(strings.Join(lines, "\n"))
}

// renderPanelHeader renders the main-panel tab strip, or the output title when
// transient command output is displayed.
func (m Model) renderPanelHeader(title string, width int) string {
	if m.bufferKind == bufOutput {
		return truncate(title, width)
	}
	tabs := m.activeTabs()
	active := m.activeMainTab()
	segments := make([]string, 0, len(tabs)+1)
	used := 0
	for i, t := range tabs {
		label := t.label()
		extra := len(label)
		if i > 0 {
			extra++
		}
		if i > 0 && used+extra > width {
			segments = append(segments, "…")
			break
		}
		used += extra
		if t == active {
			segments = append(segments, tabActiveStyle.Render(label))
		} else {
			segments = append(segments, mutedStyle.Render(label))
		}
	}
	return strings.Join(segments, " ")
}

func (m Model) panelContent() (string, string) {
	if m.bufferKind == bufOutput {
		return m.panelTitle, m.panelBody
	}
	now := effectiveNow(m.lastUpdated)
	return m.tabContent(now)
}

// minDerivationInterval is the smallest sample gap that yields a meaningful
// rate. Two refreshes close together (e.g. an auto-refresh immediately followed
// by a manual `r`) land a few milliseconds apart, and dividing a counter delta
// by that tiny interval would spike CPU%/throughput with pure timing noise.
const minDerivationInterval = 500 * time.Millisecond

type jsonNumber interface {
	Float64() (float64, error)
}

func (m Model) filteredContainerIndexes() []int {
	filter := activeFilter(m.filter)
	indexes := make([]int, 0, len(m.containers))
	for idx, container := range m.containers {
		if m.isIgnored(container.Name(), container.ImageName()) {
			continue
		}
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
		if m.isIgnored(image.Name()) {
			continue
		}
		if filter == "" || matchFields(filter, image.Name(), image.Digest(), image.Platforms(), image.Size()) {
			indexes = append(indexes, idx)
		}
	}
	return indexes
}

func (m Model) filteredBuilderCount() int {
	if m.builderMatchesFilter() {
		return 1
	}
	return 0
}

func (m Model) builderMatchesFilter() bool {
	filter := activeFilter(m.filter)
	return filter == "" || matchFields(filter, "builder", m.builder.Name(), m.builder.State(), m.builder.CPUs(), m.builder.Memory())
}

func (m Model) filteredVolumeIndexes() []int {
	filter := activeFilter(m.filter)
	indexes := make([]int, 0, len(m.volumes))
	for idx, volume := range m.volumes {
		if m.isIgnored(volume.Name()) {
			continue
		}
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
		if m.isIgnored(network.Name()) {
			continue
		}
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
		if m.isIgnored(machine.Name()) {
			continue
		}
		if filter == "" || matchFields(filter, machine.Name(), machine.State(), machine.Image(), machine.CPUs(), machine.Memory()) {
			indexes = append(indexes, idx)
		}
	}
	return indexes
}

func (m Model) filteredRegistryIndexes() []int {
	filter := activeFilter(m.filter)
	indexes := make([]int, 0, len(m.registries))
	for idx, registry := range m.registries {
		if m.isIgnored(registry.Name()) {
			continue
		}
		if filter == "" || matchFields(filter, registry.Name(), registry.User(), registry.RegistryScheme()) {
			indexes = append(indexes, idx)
		}
	}
	return indexes
}

func (m Model) filteredSystemCount() int {
	if m.systemMatchesFilter() {
		return 1
	}
	return 0
}

func (m Model) systemMatchesFilter() bool {
	filter := activeFilter(m.filter)
	versionFields := make([]string, 0, len(m.systemVersions)*3)
	for _, version := range m.systemVersions {
		versionFields = append(versionFields, version.AppName, version.Version, version.BuildType)
	}
	fields := []string{
		"system",
		m.system.Status,
		m.system.APIServerAppName,
		m.system.APIServerVersion,
		m.system.AppRoot,
		m.system.InstallRoot,
		m.systemUsage.TotalSize(),
		m.systemUsage.TotalReclaimable(),
	}
	fields = append(fields, versionFields...)
	return filter == "" || matchFields(filter, fields...)
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

// isIgnored reports whether any of a resource's fields contains a configured
// `ignore` substring, hiding noisy resources from every list (lazydocker's
// `ignore`). Patterns are pre-normalized (lowercased, trimmed) by
// normalizeIgnore, so only each field is lowercased here.
func (m Model) isIgnored(fields ...string) bool {
	if len(m.ignore) == 0 {
		return false
	}
	for _, field := range fields {
		field = strings.ToLower(field)
		for _, pattern := range m.ignore {
			if strings.Contains(field, pattern) {
				return true
			}
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

// padLeft right-aligns value in a field of width columns (space-padded on the
// left) so right-aligned, live-updating columns keep a constant width as their
// contents gain or lose characters. It never truncates.
func padLeft(value string, width int) string {
	if len(value) >= width {
		return value
	}
	return strings.Repeat(" ", width-len(value)) + value
}

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}
