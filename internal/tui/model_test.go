package tui

import (
	"context"
	"fmt"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pzep1/lazycont/internal/compose"
	"github.com/pzep1/lazycont/internal/containercli"
)

type fakeClient struct {
	started           string
	pulled            string
	runImage          string
	runOptions        containercli.ContainerLaunchOptions
	createImage       string
	createOptions     containercli.ContainerLaunchOptions
	buildTag          string
	buildContext      string
	tagSource         string
	tagTarget         string
	pushed            string
	savedImage        string
	saveOutput        string
	loadedImage       string
	copySource        string
	copyDest          string
	exportID          string
	exportOutput      string
	restarted         string
	followLogsID      string
	followLogsCount   int
	logsRead          int
	execID            string
	execCommand       string
	topID             string
	commandArgs       []string
	commandCalls      [][]string
	machineLogsID     string
	machineLogsRead   int
	machineShellID    string
	machineImage      string
	machineName       string
	machineSetID      string
	machineSettings   []string
	defaultMachine    string
	stoppedMachine    string
	registryLogin     string
	registryUser      string
	registryLogout    string
	builderStarted    bool
	builderStopped    bool
	builderDeleted    bool
	systemLogsRead    bool
	systemLogsCount   int
	systemFollowed    bool
	systemStarted     bool
	systemStopped     bool
	deleted           string
	createdVolume     string
	volumeSize        string
	createdNetwork    string
	networkSubnet     string
	deletedVolume     string
	deletedNetwork    string
	deletedMachine    string
	pruned            string
	bulkStopAll       bool
	bulkKillAll       bool
	bulkDeleteAll     bool
	prunedImagesAll   bool
	bootLogsID        string
	machineBootLogsID string
	systemDNS         []containercli.SystemDNSDomain
	systemProperties  []containercli.SystemProperty
	failSystemDNS     bool
}

func (f *fakeClient) SystemDNS(context.Context) ([]containercli.SystemDNSDomain, error) {
	if f.failSystemDNS {
		return nil, fmt.Errorf("dns unavailable")
	}
	return f.systemDNS, nil
}

func (f *fakeClient) SystemProperties(context.Context) ([]containercli.SystemProperty, error) {
	return f.systemProperties, nil
}

func (f *fakeClient) SystemStatus(context.Context) (containercli.SystemStatus, error) {
	return containercli.SystemStatus{
		Status:           "running",
		APIServerAppName: "container-apiserver",
		APIServerBuild:   "release",
		APIServerCommit:  "ee848e3ebfd7c73b04dd419683be54fb450b8779",
		APIServerVersion: "container-apiserver version 1.0.0",
		AppRoot:          "/Users/example/Library/Application Support/com.apple.container/",
		InstallRoot:      "/usr/local/",
	}, nil
}

func (f *fakeClient) SystemDiskUsage(context.Context) (containercli.SystemDiskUsage, error) {
	return containercli.SystemDiskUsage{
		Containers: containercli.DiskUsageCategory{Total: 9, Active: 1, SizeInBytes: 6589943808, Reclaimable: 5833977856},
		Images:     containercli.DiskUsageCategory{Total: 6, Active: 4, SizeInBytes: 14597648384, Reclaimable: 2625372160},
		Volumes:    containercli.DiskUsageCategory{Total: 8, Active: 8, SizeInBytes: 16260087808},
	}, nil
}

func (f *fakeClient) SystemVersion(context.Context) ([]containercli.SystemVersion, error) {
	return []containercli.SystemVersion{{
		AppName:   "container",
		BuildType: "release",
		Commit:    "ee848e3ebfd7c73b04dd419683be54fb450b8779",
		Version:   "1.0.0",
	}, {
		AppName:   "container-apiserver",
		BuildType: "release",
		Commit:    "ee848e3ebfd7c73b04dd419683be54fb450b8779",
		Version:   "container-apiserver version 1.0.0",
	}}, nil
}

func (f *fakeClient) Containers(context.Context) ([]containercli.Container, error) {
	return []containercli.Container{{
		ID: "db",
		Configuration: containercli.ContainerConfiguration{
			ID: "db",
			Image: containercli.ImageRef{
				Reference: "docker.io/library/postgres:17",
			},
			Platform: containercli.Platform{OS: "linux", Architecture: "arm64"},
		},
		Status: containercli.ContainerStatus{State: "stopped"},
	}}, nil
}

func (f *fakeClient) Images(context.Context) ([]containercli.Image, error) {
	return []containercli.Image{{
		ID: "abc",
		Configuration: containercli.ImageConfiguration{
			Name: "docker.io/library/postgres:17",
		},
		Variants: []containercli.ImageVariant{{
			Platform: containercli.Platform{OS: "linux", Architecture: "arm64"},
			Size:     1024,
		}},
	}}, nil
}

func (f *fakeClient) Volumes(context.Context) ([]containercli.Volume, error) {
	return []containercli.Volume{{
		ID: "data",
		Configuration: containercli.VolumeConfiguration{
			Name:        "data",
			Driver:      "local",
			Format:      "ext4",
			SizeInBytes: 1024,
		},
	}}, nil
}

func (f *fakeClient) Networks(context.Context) ([]containercli.NetworkResource, error) {
	return []containercli.NetworkResource{{
		ID: "default",
		Configuration: containercli.NetworkConfiguration{
			Name:   "default",
			Mode:   "nat",
			Plugin: "container-network-vmnet",
		},
		Status: containercli.NetworkStatus{IPv4Subnet: "192.168.64.0/24"},
	}}, nil
}

func (f *fakeClient) Machines(context.Context) ([]containercli.Machine, error) {
	return []containercli.Machine{{
		ID:      "dev-machine",
		Default: true,
		Configuration: map[string]any{
			"image": map[string]any{"reference": "docker.io/library/alpine:3.22"},
			"resources": map[string]any{
				"cpus":          float64(2),
				"memoryInBytes": float64(2147483648),
			},
		},
		Status: map[string]any{"state": "running"},
	}}, nil
}

func (f *fakeClient) Registries(context.Context) ([]containercli.RegistryLogin, error) {
	return []containercli.RegistryLogin{{
		Server:   "ghcr.io",
		Username: "alice",
		Scheme:   "https",
	}}, nil
}

func (f *fakeClient) BuilderStatus(context.Context) (containercli.BuilderStatus, error) {
	return containercli.BuilderStatus{
		ID:         "buildkit",
		StateValue: "running",
		Present:    true,
		Configuration: map[string]any{
			"resources": map[string]any{
				"cpus":          float64(4),
				"memoryInBytes": float64(4294967296),
			},
		},
	}, nil
}

func (f *fakeClient) Stats(context.Context, ...string) ([]containercli.Stat, error) {
	return nil, nil
}

func (f *fakeClient) Logs(context.Context, string, int) (string, error) {
	f.logsRead++
	return "ready\n", nil
}

func (f *fakeClient) FollowLogsCommand(id string, _ int) (*exec.Cmd, error) {
	f.followLogsID = id
	f.followLogsCount++
	return exec.Command("printf", "container ready\n"), nil
}

func (f *fakeClient) MachineLogs(_ context.Context, id string, _ int) (string, error) {
	f.machineLogsID = id
	f.machineLogsRead++
	return "machine ready\n", nil
}

func (f *fakeClient) FollowMachineLogsCommand(id string, _ int) (*exec.Cmd, error) {
	f.machineLogsID = id
	return exec.Command("printf", "machine ready\n"), nil
}

func (f *fakeClient) SystemLogs(context.Context, string) (string, error) {
	f.systemLogsRead = true
	f.systemLogsCount++
	return "system ready\n", nil
}

func (f *fakeClient) FollowSystemLogsCommand(string) (*exec.Cmd, error) {
	f.systemFollowed = true
	return exec.Command("printf", "system ready\n"), nil
}

func (f *fakeClient) InspectContainer(context.Context, string) (string, error) {
	return `[{"id":"db"}]`, nil
}

func (f *fakeClient) InspectImage(context.Context, string) (string, error) {
	return `[{"id":"abc"}]`, nil
}

func (f *fakeClient) InspectVolume(context.Context, string) (string, error) {
	return `[{"id":"data"}]`, nil
}

func (f *fakeClient) InspectNetwork(context.Context, string) (string, error) {
	return `[{"id":"default"}]`, nil
}

func (f *fakeClient) InspectMachine(context.Context, string) (string, error) {
	return `[{"id":"dev-machine"}]`, nil
}

func (f *fakeClient) ShellCommand(string, string) (*exec.Cmd, error) {
	return exec.Command("true"), nil
}

func (f *fakeClient) Exec(_ context.Context, id string, command string) (string, error) {
	f.execID = id
	f.execCommand = command
	return "ok\n", nil
}

func (f *fakeClient) Top(_ context.Context, id string) (string, error) {
	f.topID = id
	return "UID PID PPID CMD\nroot 1 0 /bin/sh\n", nil
}

func (f *fakeClient) Command(_ context.Context, args []string) (string, error) {
	f.commandArgs = append([]string(nil), args...)
	f.commandCalls = append(f.commandCalls, append([]string(nil), args...))
	return "command output\n", nil
}

func (f *fakeClient) CommandProcess(args []string) (*exec.Cmd, error) {
	f.commandArgs = append([]string(nil), args...)
	return exec.Command("true"), nil
}

func (f *fakeClient) MachineShellCommand(id string) (*exec.Cmd, error) {
	f.machineShellID = id
	return exec.Command("true"), nil
}

func (f *fakeClient) CreateMachine(_ context.Context, image string, name string) error {
	f.machineImage = image
	f.machineName = name
	return nil
}

func (f *fakeClient) SetDefaultMachine(_ context.Context, id string) error {
	f.defaultMachine = id
	return nil
}

func (f *fakeClient) SetMachine(_ context.Context, id string, settings []string) error {
	f.machineSetID = id
	f.machineSettings = append([]string(nil), settings...)
	return nil
}

func (f *fakeClient) PullImage(_ context.Context, reference string) (string, error) {
	f.pulled = reference
	return "pull output\n", nil
}

func (f *fakeClient) RunImage(_ context.Context, image string, options containercli.ContainerLaunchOptions) error {
	f.runImage = image
	f.runOptions = options
	return nil
}

func (f *fakeClient) CreateContainer(_ context.Context, image string, options containercli.ContainerLaunchOptions) error {
	f.createImage = image
	f.createOptions = options
	return nil
}

func (f *fakeClient) BuildImage(_ context.Context, tag string, contextDir string) (string, error) {
	f.buildTag = tag
	f.buildContext = contextDir
	return "build output\n", nil
}

func (f *fakeClient) TagImage(_ context.Context, source string, target string) error {
	f.tagSource = source
	f.tagTarget = target
	return nil
}

func (f *fakeClient) PushImage(_ context.Context, reference string) (string, error) {
	f.pushed = reference
	return "push output\n", nil
}

func (f *fakeClient) SaveImage(_ context.Context, reference string, outputPath string) (string, error) {
	f.savedImage = reference
	f.saveOutput = outputPath
	return "save output\n", nil
}

func (f *fakeClient) LoadImage(_ context.Context, inputPath string) (string, error) {
	f.loadedImage = inputPath
	return "load output\n", nil
}

func (f *fakeClient) RegistryLoginCommand(server string, username string) (*exec.Cmd, error) {
	f.registryLogin = server
	f.registryUser = username
	return exec.Command("true"), nil
}

func (f *fakeClient) LogoutRegistry(_ context.Context, registry string) error {
	f.registryLogout = registry
	return nil
}

func (f *fakeClient) StartBuilder(context.Context) error {
	f.builderStarted = true
	return nil
}

func (f *fakeClient) StopBuilder(context.Context) error {
	f.builderStopped = true
	return nil
}

func (f *fakeClient) DeleteBuilder(context.Context, bool) error {
	f.builderDeleted = true
	return nil
}

func (f *fakeClient) StartSystem(context.Context) error {
	f.systemStarted = true
	return nil
}

func (f *fakeClient) StopSystem(context.Context) error {
	f.systemStopped = true
	return nil
}

func (f *fakeClient) Copy(_ context.Context, source string, destination string) error {
	f.copySource = source
	f.copyDest = destination
	return nil
}

func (f *fakeClient) ExportContainer(_ context.Context, id string, outputPath string) error {
	f.exportID = id
	f.exportOutput = outputPath
	return nil
}

func (f *fakeClient) Start(_ context.Context, id string) error {
	f.started = id
	return nil
}

func (f *fakeClient) Stop(context.Context, string) error {
	return nil
}

func (f *fakeClient) Restart(_ context.Context, id string) error {
	f.restarted = id
	return nil
}

func (f *fakeClient) StopMachine(_ context.Context, id string) error {
	f.stoppedMachine = id
	return nil
}

func (f *fakeClient) BootLogs(_ context.Context, id string, _ int) (string, error) {
	f.bootLogsID = id
	return "vm boot: kernel started\nvm boot: init running", nil
}

func (f *fakeClient) MachineBootLogs(_ context.Context, id string, _ int) (string, error) {
	f.machineBootLogsID = id
	return "machine boot: kernel started", nil
}

func (f *fakeClient) Kill(context.Context, string) error {
	return nil
}

func (f *fakeClient) StopAll(context.Context) error {
	f.bulkStopAll = true
	return nil
}

func (f *fakeClient) KillAll(context.Context) error {
	f.bulkKillAll = true
	return nil
}

func (f *fakeClient) DeleteAllContainers(_ context.Context, _ bool) error {
	f.bulkDeleteAll = true
	return nil
}

func (f *fakeClient) DeleteContainer(_ context.Context, id string, _ bool) error {
	f.deleted = id
	return nil
}

func (f *fakeClient) DeleteImage(context.Context, string, bool) error {
	return nil
}

func (f *fakeClient) CreateVolume(_ context.Context, name string, size string) error {
	f.createdVolume = name
	f.volumeSize = size
	return nil
}

func (f *fakeClient) CreateNetwork(_ context.Context, name string, subnet string) error {
	f.createdNetwork = name
	f.networkSubnet = subnet
	return nil
}

func (f *fakeClient) DeleteVolume(_ context.Context, volume string) error {
	f.deletedVolume = volume
	return nil
}

func (f *fakeClient) DeleteNetwork(_ context.Context, network string) error {
	f.deletedNetwork = network
	return nil
}

func (f *fakeClient) DeleteMachine(_ context.Context, id string) error {
	f.deletedMachine = id
	return nil
}

func (f *fakeClient) PruneContainers(context.Context) error {
	f.pruned = "containers"
	return nil
}

func (f *fakeClient) PruneImages(_ context.Context, all bool) error {
	if all {
		f.prunedImagesAll = true
	}
	return nil
}

func (f *fakeClient) PruneVolumes(context.Context) error {
	return nil
}

func (f *fakeClient) PruneNetworks(context.Context) error {
	return nil
}

func TestModelLoadsSnapshotIntoView(t *testing.T) {
	model := New(&fakeClient{})
	msg := model.refreshCmd()().(snapshotMsg)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	updated, _ = updated.Update(msg)
	view := updated.View()

	if !strings.Contains(view, "● running") {
		t.Fatalf("view did not include system status:\n%s", view)
	}
	if !strings.Contains(view, "db") {
		t.Fatalf("view did not include container:\n%s", view)
	}
	if !strings.Contains(view, "Containers (1)") {
		t.Fatalf("view did not include container count:\n%s", view)
	}
	if !strings.Contains(view, "Builder (running)") || !strings.Contains(view, "Volumes (1)") || !strings.Contains(view, "Networks (1)") || !strings.Contains(view, "Machines (1)") || !strings.Contains(view, "Registries (1)") || !strings.Contains(view, "System (running)") {
		t.Fatalf("view did not include secondary resource counts:\n%s", view)
	}
}

func TestMouseClickSelectsVisibleContainerRow(t *testing.T) {
	model := New(&fakeClient{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	updated, _ = updated.Update(snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{
			testContainer("db", "docker.io/library/postgres:17"),
			testContainer("api", "docker.io/library/nginx:latest"),
			testContainer("cache", "docker.io/library/redis:7"),
		},
	})
	state := updated.(Model)
	layout, ok := state.viewLayout()
	if !ok {
		t.Fatalf("expected test layout")
	}

	updated, cmd := state.Update(tea.MouseMsg{
		X:      layout.sidebarContentX + 2,
		Y:      layout.listFirstRowY + 1,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	if cmd != nil {
		t.Fatalf("expected no command")
	}
	state = updated.(Model)
	if state.containerCursor != 1 {
		t.Fatalf("container cursor mismatch: %d", state.containerCursor)
	}
	if !strings.Contains(state.View(), "ID:       api") {
		t.Fatalf("view did not show clicked container details:\n%s", state.View())
	}
}

func TestMouseClickSelectsResourceTab(t *testing.T) {
	model := New(&fakeClient{})
	msg := model.refreshCmd()().(snapshotMsg)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	updated, _ = updated.Update(msg)
	state := updated.(Model)
	x, y := tabClickPoint(t, state, resourceNetworks)

	updated, cmd := state.Update(tea.MouseMsg{
		X:      x,
		Y:      y,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	if cmd != nil {
		t.Fatalf("expected no command")
	}
	state = updated.(Model)
	if state.active != resourceNetworks {
		t.Fatalf("active resource mismatch: %v", state.active)
	}
	if !strings.Contains(state.View(), "Name:       default") {
		t.Fatalf("view did not show clicked tab details:\n%s", state.View())
	}
}

func TestNumberKeyFocusesResourceSection(t *testing.T) {
	model := New(&fakeClient{})
	msg := model.refreshCmd()().(snapshotMsg)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 28})
	updated, _ = updated.Update(msg)

	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	if cmd != nil {
		t.Fatalf("expected no command")
	}
	state := updated.(Model)
	if state.active != resourceVolumes {
		t.Fatalf("active resource mismatch: got %v want volumes", state.active)
	}
	if !strings.Contains(state.View(), "▌ Volumes") {
		t.Fatalf("view did not focus volumes section:\n%s", state.View())
	}
}

func TestCompactSidebarShowsListAtShortHeight(t *testing.T) {
	model := New(&fakeClient{})
	msg := model.refreshCmd()().(snapshotMsg)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 12})
	updated, _ = updated.Update(msg)

	view := updated.View()
	for _, want := range []string{"▌ Containers", "db"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view did not include %q at short height:\n%s", want, view)
		}
	}
}

func TestMouseWheelScrollsPanel(t *testing.T) {
	model := New(&fakeClient{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 16})
	state := updated.(Model)
	state.bufferKind = bufOutput
	state.panelTitle = "Long output"
	lines := make([]string, 40)
	for idx := range lines {
		lines[idx] = fmt.Sprintf("line %02d", idx)
	}
	state.panelBody = strings.Join(lines, "\n")
	layout, ok := state.viewLayout()
	if !ok {
		t.Fatalf("expected test layout")
	}

	updated, cmd := state.Update(tea.MouseMsg{
		X:      layout.panelX + 2,
		Y:      layout.bodyTop + 2,
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	})
	if cmd != nil {
		t.Fatalf("expected no command")
	}
	state = updated.(Model)
	if state.panelOffset != 3 {
		t.Fatalf("panel offset mismatch: %d", state.panelOffset)
	}
}

// withTab returns the model with the given resource's main-panel tab selected.
func withTab(state Model, kind resourceKind, tab mainTab) Model {
	for i, t := range tabsFor(kind) {
		if t == tab {
			state.tabIndex[kind] = i
		}
	}
	return state
}

func TestContainerDetailsShowMetricSummary(t *testing.T) {
	model := New(&fakeClient{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 34})
	updated, _ = updated.Update(snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{
			testContainerWithState("web", "docker.io/library/nginx:latest", "running"),
		},
		stats: []containercli.Stat{{
			"id":               "web",
			"memoryUsageBytes": float64(47431680),
			"memoryLimitBytes": float64(1073741824),
			"cpuUsageUsec":     float64(1234567),
			"networkRxBytes":   float64(1289011),
			"networkTxBytes":   float64(876544),
			"blockReadBytes":   float64(4718592),
			"blockWriteBytes":  float64(2202009),
			"numProcesses":     float64(3),
		}},
	})

	view := withTab(updated.(Model), resourceContainers, tabStats).View()
	for _, want := range []string{"Stats", "CPU time: 1.2s", "Memory:", "[#---------------]", "Network:", "PIDs:     3"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view did not include %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "memoryUsageBytes") {
		t.Fatalf("view rendered raw stats instead of summary:\n%s", view)
	}
}

func TestContainerListShowsMetricSummary(t *testing.T) {
	model := New(&fakeClient{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 130, Height: 28})
	updated, _ = updated.Update(snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{
			testContainerWithState("web", "docker.io/library/nginx:latest", "running"),
		},
		stats: []containercli.Stat{{
			"id":               "web",
			"cpuPercent":       float64(12.34),
			"memoryUsageBytes": float64(47431680),
			"memoryLimitBytes": float64(1073741824),
		}},
	})

	view := updated.View()
	for _, want := range []string{"state / cpu / mem", "12.3% cpu", "4.4% mem"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view did not include %q:\n%s", want, view)
		}
	}
}

func TestContainerDetailsShowMetricHistory(t *testing.T) {
	model := New(&fakeClient{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	updated, _ = updated.Update(snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{
			testContainerWithState("web", "docker.io/library/nginx:latest", "running"),
		},
		stats: []containercli.Stat{{
			"id":               "web",
			"cpuPercent":       float64(10),
			"memoryUsageBytes": float64(10 * 1024 * 1024),
			"memoryLimitBytes": float64(100 * 1024 * 1024),
			"networkRxBytes":   float64(1000),
			"networkTxBytes":   float64(2000),
			"blockReadBytes":   float64(3000),
			"blockWriteBytes":  float64(4000),
		}},
	})
	updated, _ = updated.Update(snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{
			testContainerWithState("web", "docker.io/library/nginx:latest", "running"),
		},
		stats: []containercli.Stat{{
			"id":               "web",
			"cpuPercent":       float64(80),
			"memoryUsageBytes": float64(60 * 1024 * 1024),
			"memoryLimitBytes": float64(100 * 1024 * 1024),
			"networkRxBytes":   float64(5000),
			"networkTxBytes":   float64(7000),
			"blockReadBytes":   float64(9000),
			"blockWriteBytes":  float64(11000),
		}},
	})

	view := withTab(updated.(Model), resourceContainers, tabStats).View()
	for _, want := range []string{"CPU %", "cur 80.0%", "max 100.0%", "Memory", "█", "Current"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view did not include %q:\n%s", want, view)
		}
	}
}

// TestDerivesLiveCPUPercentFromCumulativeUsec verifies the headline fix: Apple's
// `container stats` reports a cumulative cpuUsageUsec counter (no ready
// percentage), so a live CPU% must be differenced from successive samples and
// surfaced in the list, the current summary, and the CPU% graph.
func TestDerivesLiveCPUPercentFromCumulativeUsec(t *testing.T) {
	model := New(&fakeClient{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 130, Height: 64})
	base := time.Unix(1_700_000_000, 0)

	// cpuUsageUsec rises 1.0s of CPU time over 2s wall (50%), then 0.6s over 2s
	// (30%); networkRx/Tx and blockRead/Write climb so throughput is derivable.
	sample := func(at time.Time, cpuUsec, net, blk float64) tea.Msg {
		return snapshotMsg{
			updated: at,
			system:  containercli.SystemStatus{Status: "running"},
			containers: []containercli.Container{
				testContainerWithState("web", "docker.io/library/nginx:latest", "running"),
			},
			stats: []containercli.Stat{{
				"id":               "web",
				"cpuUsageUsec":     cpuUsec,
				"memoryUsageBytes": float64(52428800),  // 50 MB
				"memoryLimitBytes": float64(104857600), // 100 MB
				"networkRxBytes":   net,
				"networkTxBytes":   float64(0),
				"blockReadBytes":   blk,
				"blockWriteBytes":  float64(0),
				"numProcesses":     float64(3),
			}},
		}
	}

	updated, _ = updated.Update(sample(base, 1_000_000, 0, 0))
	updated, _ = updated.Update(sample(base.Add(2*time.Second), 2_000_000, 500_000, 400_000))
	updated, _ = updated.Update(sample(base.Add(4*time.Second), 2_600_000, 900_000, 600_000))

	// The container list row shows the derived live percentage, not raw usec.
	listView := updated.View()
	for _, want := range []string{"30.0% cpu", "50.0% mem"} {
		if !strings.Contains(listView, want) {
			t.Fatalf("list view did not include derived %q:\n%s", want, listView)
		}
	}
	if strings.Contains(listView, "cpu time") || strings.Contains(listView, "CPU time") {
		t.Fatalf("list should show derived CPU%%, not cumulative CPU time:\n%s", listView)
	}

	// The Stats tab renders CPU%, Network, and Block IO graphs plus a live
	// "Current" CPU summary line.
	statsView := withTab(updated.(Model), resourceContainers, tabStats).View()
	for _, want := range []string{"CPU %", "cur 30.0%", "Network", "Block IO", "Current", "CPU:"} {
		if !strings.Contains(statsView, want) {
			t.Fatalf("stats view did not include %q:\n%s", want, statsView)
		}
	}
	if strings.Contains(statsView, "CPU time:") {
		t.Fatalf("stats summary should show live CPU%%, not cumulative CPU time:\n%s", statsView)
	}
}

func TestBootLogsShowVMOutput(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	updated, _ = updated.Update(snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{
			testContainerWithState("web", "docker.io/library/nginx:latest", "running"),
		},
	})

	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyCtrlB})
	if cmd == nil {
		t.Fatalf("expected a boot-logs command")
	}
	updated, _ = updated.Update(cmd().(outputMsg))
	if client.bootLogsID != "web" {
		t.Fatalf("expected boot logs for web, got %q", client.bootLogsID)
	}
	if view := updated.View(); !strings.Contains(view, "Boot logs web") || !strings.Contains(view, "kernel started") {
		t.Fatalf("boot logs not displayed:\n%s", view)
	}
}

func TestSystemPaneShowsDNSAndProperties(t *testing.T) {
	client := &fakeClient{
		systemDNS:        []containercli.SystemDNSDomain{{Name: "myapp.test"}},
		systemProperties: []containercli.SystemProperty{{ID: "build.rosetta", Value: "true"}},
	}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	updated, _ = updated.Update(model.refreshCmd()().(snapshotMsg))

	// Jump to the System pane and read its detail.
	view := switchToResource(t, updated, resourceSystem).View()
	for _, want := range []string{"DNS domains", "myapp.test", "Properties", "build.rosetta: true"} {
		if !strings.Contains(view, want) {
			t.Fatalf("system detail did not include %q:\n%s", want, view)
		}
	}
}

func TestIgnoreListHidesMatchingResources(t *testing.T) {
	model := NewWithOptions(&fakeClient{}, Options{Ignore: []string{"infra"}})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	updated, _ = updated.Update(snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{
			testContainerWithState("web", "docker.io/library/nginx:latest", "running"),
			testContainerWithState("infra-proxy", "docker.io/library/envoy:latest", "running"),
		},
	})

	state := updated.(Model)
	if got := state.activeVisibleCount(); got != 1 {
		t.Fatalf("expected ignore to hide 1 of 2 containers, visible=%d", got)
	}
	view := state.View()
	if !strings.Contains(view, "web") {
		t.Fatalf("expected web container visible:\n%s", view)
	}
	if strings.Contains(view, "infra-proxy") {
		t.Fatalf("expected infra-proxy hidden by ignore list:\n%s", view)
	}
}

func TestNumberKeysJumpToResourcePane(t *testing.T) {
	model := New(&fakeClient{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	updated, _ = updated.Update(model.refreshCmd()().(snapshotMsg))

	cases := []struct {
		key  rune
		want resourceKind
	}{
		{'2', resourceServices},
		{'3', resourceImages},
		{'5', resourceVolumes},
		{'7', resourceMachines},
		{'9', resourceSystem},
		{'1', resourceContainers},
	}
	for _, tc := range cases {
		updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tc.key}})
		if got := updated.(Model).active; got != tc.want {
			t.Fatalf("key %q: active = %v, want %v", tc.key, got, tc.want)
		}
	}
}

// TestDerivationPrefersCumulativeUsecOverZeroFilledPercent guards against a
// runtime that emits a literal cpuPercent:0 alongside a rising cpuUsageUsec —
// the cumulative counter must win so a busy container isn't pinned to 0%.
func TestDerivationPrefersCumulativeUsecOverZeroFilledPercent(t *testing.T) {
	model := New(&fakeClient{})
	base := time.Unix(1_700_000_000, 0)
	model.recordStatHistory([]containercli.Stat{{
		"id": "web", "cpuPercent": float64(0), "cpuUsageUsec": float64(1_000_000),
	}}, base)
	model.recordStatHistory([]containercli.Stat{{
		"id": "web", "cpuPercent": float64(0), "cpuUsageUsec": float64(2_000_000),
	}}, base.Add(2*time.Second))

	pct, ok := model.derivedCPUPercent("web")
	if !ok || pct < 49 || pct > 51 {
		t.Fatalf("expected ~50%% derived from usec despite cpuPercent:0, got %v ok=%v", pct, ok)
	}
}

// TestDerivationSkipsTinyIntervals guards against rapid back-to-back refreshes
// (e.g. an auto-refresh immediately followed by `r`) spiking CPU% by dividing a
// counter delta by a near-zero elapsed time.
func TestDerivationSkipsTinyIntervals(t *testing.T) {
	model := New(&fakeClient{})
	base := time.Unix(1_700_000_000, 0)
	model.recordStatHistory([]containercli.Stat{{"id": "web", "cpuUsageUsec": float64(1_000_000)}}, base)
	// 20ms later with a large delta — must NOT produce a derived percentage.
	model.recordStatHistory([]containercli.Stat{{"id": "web", "cpuUsageUsec": float64(1_500_000)}}, base.Add(20*time.Millisecond))
	if pct, ok := model.derivedCPUPercent("web"); ok {
		t.Fatalf("expected no derived CPU%% for a sub-0.5s interval, got %v", pct)
	}

	// A full interval against the stored sample derives normally (Δ1e6 / 2s = 50%).
	model.recordStatHistory([]containercli.Stat{{"id": "web", "cpuUsageUsec": float64(2_500_000)}}, base.Add(2020*time.Millisecond))
	pct, ok := model.derivedCPUPercent("web")
	if !ok || pct < 49 || pct > 51 {
		t.Fatalf("expected ~50%% after a full interval, got %v ok=%v", pct, ok)
	}
}

// TestSystemDNSPreservedOnTransientError verifies a failed best-effort DNS fetch
// keeps the last-known domains instead of blanking the System pane.
func TestSystemDNSPreservedOnTransientError(t *testing.T) {
	client := &fakeClient{systemDNS: []containercli.SystemDNSDomain{{Name: "myapp.test"}}}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	updated, _ = updated.Update(model.refreshCmd()().(snapshotMsg))
	if got := len(updated.(Model).systemDNS); got != 1 {
		t.Fatalf("expected 1 DNS domain after first refresh, got %d", got)
	}

	// Next refresh fails the DNS fetch; the previously-known domain must remain.
	client.failSystemDNS = true
	updated, _ = updated.Update(updated.(Model).refreshCmd()().(snapshotMsg))
	state := updated.(Model)
	if got := len(state.systemDNS); got != 1 {
		t.Fatalf("expected DNS domain preserved on transient error, got %d", got)
	}
	if !strings.Contains(switchToSystemView(t, state), "myapp.test") {
		t.Fatalf("expected myapp.test still visible after transient error")
	}
}

func switchToSystemView(t *testing.T, state Model) string {
	t.Helper()
	return switchToResource(t, state, resourceSystem).View()
}

func composeTestProject() compose.Project {
	return compose.Project{
		Name: "shop",
		File: "compose.yaml",
		Services: []compose.Service{
			{Name: "web", Image: "nginx:latest", Ports: []string{"8080:80"}},
			{Name: "db", Image: "postgres:17"},
		},
	}
}

func TestServicesPaneListsServicesWithState(t *testing.T) {
	client := &fakeClient{}
	model := NewWithOptions(client, Options{LoadProject: func() (compose.Project, error) {
		return composeTestProject(), nil
	}})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	// db's container is running; web has none yet.
	updated, _ = updated.Update(snapshotMsg{
		system:     containercli.SystemStatus{Status: "running"},
		project:    composeTestProject(),
		containers: []containercli.Container{testContainerWithState("shop-db", "postgres:17", "running")},
	})

	view := switchToResource(t, updated, resourceServices).View()
	for _, want := range []string{"Services", "web", "db", "running"} {
		if !strings.Contains(view, want) {
			t.Fatalf("services pane missing %q:\n%s", want, view)
		}
	}
}

func TestServicesPaneEmptyStateWhenNoComposeFile(t *testing.T) {
	model := New(&fakeClient{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	updated, _ = updated.Update(snapshotMsg{system: containercli.SystemStatus{Status: "running"}})
	view := switchToResource(t, updated, resourceServices).View()
	if !strings.Contains(view, "No compose.yaml found") {
		t.Fatalf("expected empty-state guidance:\n%s", view)
	}
}

func TestServiceInspectWithoutContainerShowsEmptyState(t *testing.T) {
	client := &fakeClient{}
	model := NewWithOptions(client, Options{LoadProject: func() (compose.Project, error) {
		return composeTestProject(), nil
	}})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	// No containers exist for any service.
	updated, _ = updated.Update(snapshotMsg{system: containercli.SystemStatus{Status: "running"}, project: composeTestProject()})
	updated = switchToResource(t, updated, resourceServices)

	// Open the Inspect tab for a service with no backing container.
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	view := updated.View()
	if !strings.Contains(view, "No container yet") {
		t.Fatalf("expected empty-state message, not a perpetual Loading:\n%s", view)
	}
	if strings.Contains(view, "Loading inspect") {
		t.Fatalf("inspect should not hang on Loading for a containerless service:\n%s", view)
	}
}

func TestUpServiceRunsContainerRun(t *testing.T) {
	client := &fakeClient{}
	model := NewWithOptions(client, Options{LoadProject: func() (compose.Project, error) {
		return composeTestProject(), nil
	}})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	updated, _ = updated.Update(snapshotMsg{system: containercli.SystemStatus{Status: "running"}, project: composeTestProject()})
	updated = switchToResource(t, updated, resourceServices)

	// Cursor starts on "web"; press u to bring it up.
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if cmd == nil {
		t.Fatalf("expected an up command")
	}
	updated.Update(cmd().(actionDoneMsg))

	want := []string{"run", "--detach", "--name", "shop-web", "--publish", "8080:80", "nginx:latest"}
	if !reflect.DeepEqual(client.commandArgs, want) {
		t.Fatalf("up service command mismatch\nwant: %#v\n got: %#v", want, client.commandArgs)
	}
}

func TestDownServiceRequiresConfirmationThenStopsAndRemoves(t *testing.T) {
	client := &fakeClient{}
	model := NewWithOptions(client, Options{LoadProject: func() (compose.Project, error) {
		return composeTestProject(), nil
	}})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	updated, _ = updated.Update(snapshotMsg{system: containercli.SystemStatus{Status: "running"}, project: composeTestProject()})
	updated = switchToResource(t, updated, resourceServices)

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if len(client.commandCalls) != 0 {
		t.Fatalf("down ran before confirmation: %#v", client.commandCalls)
	}
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatalf("expected confirmation command")
	}
	updated.Update(cmd().(actionDoneMsg))

	want := [][]string{{"stop", "shop-web"}, {"delete", "--force", "shop-web"}}
	if !reflect.DeepEqual(client.commandCalls, want) {
		t.Fatalf("down service commands mismatch\nwant: %#v\n got: %#v", want, client.commandCalls)
	}
}

func TestTabCycleSwitchesMainPanel(t *testing.T) {
	model := New(&fakeClient{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 28})
	updated, _ = updated.Update(snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{
			testContainerWithState("web", "docker.io/library/nginx:latest", "running"),
		},
	})
	state := updated.(Model)
	if state.activeMainTab() != tabConfig {
		t.Fatalf("expected default tab Config, got %v", state.activeMainTab())
	}
	updated, cmd := state.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	state = updated.(Model)
	if state.activeMainTab() != tabLogs {
		t.Fatalf("expected Logs tab after ], got %v", state.activeMainTab())
	}
	if cmd == nil {
		t.Fatalf("expected a fetch command when entering Logs tab")
	}
	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	state = updated.(Model)
	if state.activeMainTab() != tabConfig {
		t.Fatalf("expected Config tab after [, got %v", state.activeMainTab())
	}
}

func TestInspectTabFetchesContainerJSON(t *testing.T) {
	model := New(&fakeClient{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 28})
	updated, _ = updated.Update(snapshotMsg{
		system:     containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{testContainer("db", "docker.io/library/postgres:17")},
	})
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if cmd == nil {
		t.Fatalf("expected inspect fetch command")
	}
	state := updated.(Model)
	if state.activeMainTab() != tabInspect {
		t.Fatalf("expected Inspect tab, got %v", state.activeMainTab())
	}
	msg := cmd().(tabFetchedMsg)
	updated, _ = state.Update(msg)
	if !strings.Contains(updated.View(), `"id":"db"`) {
		t.Fatalf("inspect tab did not show JSON:\n%s", updated.View())
	}
}

func TestEnvTabShowsContainerEnvironment(t *testing.T) {
	c := testContainer("web", "docker.io/library/nginx:latest")
	c.Configuration.InitProcess.Environment = []string{"PATH=/usr/bin", "FOO=bar"}
	model := New(&fakeClient{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 28})
	updated, _ = updated.Update(snapshotMsg{
		system:     containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{c},
	})
	view := withTab(updated.(Model), resourceContainers, tabEnv).View()
	for _, want := range []string{"PATH=/usr/bin", "FOO=bar"} {
		if !strings.Contains(view, want) {
			t.Fatalf("env tab missing %q:\n%s", want, view)
		}
	}
}

func TestTopTabFetchesProcesses(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 28})
	updated, _ = updated.Update(snapshotMsg{
		system:     containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{testContainerWithState("web", "docker.io/library/nginx:latest", "running")},
	})
	state := withTab(updated.(Model), resourceContainers, tabTop)
	cmd := state.ensureBufferCmd()
	if cmd == nil {
		t.Fatalf("expected top fetch command")
	}
	msg := cmd().(tabFetchedMsg)
	updated, _ = state.Update(msg)
	if client.topID != "web" {
		t.Fatalf("expected top target web, got %q", client.topID)
	}
	if !strings.Contains(updated.View(), "PID") {
		t.Fatalf("top tab did not show processes:\n%s", updated.View())
	}
}

func TestScreenModeCyclesAndFullscreenHidesSidebar(t *testing.T) {
	model := New(&fakeClient{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	updated, _ = updated.Update(snapshotMsg{
		system:     containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{testContainer("web", "docker.io/library/nginx:latest")},
	})
	state := updated.(Model)
	if state.sidebarWidth() <= 0 {
		t.Fatalf("expected a sidebar in normal mode")
	}
	if !strings.Contains(state.View(), "Volumes") {
		t.Fatalf("expected sidebar labels in normal mode")
	}

	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}})
	state = updated.(Model)
	if state.screenMode != screenHalf {
		t.Fatalf("expected half mode, got %v", state.screenMode)
	}

	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}})
	state = updated.(Model)
	if state.screenMode != screenFull {
		t.Fatalf("expected fullscreen, got %v", state.screenMode)
	}
	if state.sidebarWidth() != 0 {
		t.Fatalf("expected hidden sidebar in fullscreen, got width %d", state.sidebarWidth())
	}
	if strings.Contains(state.View(), "Volumes") {
		t.Fatalf("expected sidebar hidden in fullscreen:\n%s", state.View())
	}

	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}})
	if updated.(Model).screenMode != screenNormal {
		t.Fatalf("expected wrap back to normal")
	}

	// _ cycles backwards.
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'_'}})
	if updated.(Model).screenMode != screenFull {
		t.Fatalf("expected _ to step back to fullscreen, got %v", updated.(Model).screenMode)
	}
}

func TestConfigEditReloadsThemeAndSettings(t *testing.T) {
	model := NewWithOptions(&fakeClient{}, Options{
		ReloadConfig: func() (Options, error) {
			return Options{
				CustomCommands:  []CustomCommand{{Name: "Ver", Args: []string{"version"}}},
				ScreenMode:      "fullscreen",
				SidePanelWidth:  0.5,
				LogsTail:        999,
				LogsSince:       "1h",
				RefreshInterval: 3 * time.Second,
			}, nil
		},
	})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	updated, _ = updated.Update(configEditedMsg{path: "/tmp/config.json"})
	state := updated.(Model)
	if state.screenMode != screenFull {
		t.Fatalf("expected screen mode reloaded, got %v", state.screenMode)
	}
	if state.logsTail != 999 || state.logsSince != "1h" {
		t.Fatalf("expected logs settings reloaded: tail=%d since=%q", state.logsTail, state.logsSince)
	}
	if state.refreshInterval != 3*time.Second {
		t.Fatalf("expected refresh interval reloaded, got %v", state.refreshInterval)
	}
	if len(state.customCommands) != 1 || state.customCommands[0].Name != "Ver" {
		t.Fatalf("expected commands reloaded, got %#v", state.customCommands)
	}
}

func TestActionErrorKeepsCachedTabContent(t *testing.T) {
	model := New(&fakeClient{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 28})
	updated, _ = updated.Update(snapshotMsg{
		system:     containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{testContainer("db", "docker.io/library/postgres:17")},
	})
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	updated, _ = updated.Update(cmd().(tabFetchedMsg))
	state := updated.(Model)
	if state.bufferKind != bufTab {
		t.Fatalf("expected inspect content loaded")
	}

	updated, _ = state.Update(actionDoneMsg{message: "x", err: fmt.Errorf("boom")})
	state = updated.(Model)
	if state.bufferKind != bufTab {
		t.Fatalf("action error should keep cached tab content, got bufferKind %v", state.bufferKind)
	}
	if !strings.Contains(state.View(), `"id":"db"`) {
		t.Fatalf("expected cached inspect still visible after error:\n%s", state.View())
	}
}

func TestStatHistoryIsBounded(t *testing.T) {
	model := New(&fakeClient{})
	base := time.Unix(1_700_000_000, 0)
	for idx := 0; idx < maxStatHistorySamples+5; idx++ {
		model.recordStatHistory([]containercli.Stat{{
			"id":         "web",
			"cpuPercent": float64(idx),
		}}, base.Add(time.Duration(idx)*time.Second))
	}

	if got := len(model.statHistory["web"]); got != maxStatHistorySamples {
		t.Fatalf("history length = %d, want %d", got, maxStatHistorySamples)
	}
}

func TestImageDetailsShowLayerHistory(t *testing.T) {
	model := New(&fakeClient{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 130, Height: 30})
	updated, _ = updated.Update(snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		images: []containercli.Image{{
			ID: "abc",
			Configuration: containercli.ImageConfiguration{
				Name: "docker.io/library/alpine:latest",
			},
			Variants: []containercli.ImageVariant{{
				Digest:   "sha256:def",
				Platform: containercli.Platform{OS: "linux", Architecture: "arm64", Variant: "v8"},
				Size:     4203982,
				Config: containercli.ImageVariantConfig{
					History: []containercli.ImageHistory{
						{CreatedBy: "ADD alpine-minirootfs-3.24.0-aarch64.tar.gz / # buildkit"},
						{CreatedBy: "CMD [\"/bin/sh\"]", EmptyLayer: true},
					},
					RootFS: containercli.ImageRootFS{
						DiffIDs: []string{"sha256:375591c23c8de111a75382d674cf6688f56adecb5e3018d29ada57c10135db5e"},
						Type:    "layers",
					},
				},
			}},
		}},
	})
	updated = switchToResource(t, updated, resourceImages)

	view := updated.View()
	for _, want := range []string{"Layer history", "linux/arm64/v8", "375591c23c8d", "metadata", "CMD [\"/bin/sh\"]"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view did not include %q:\n%s", want, view)
		}
	}
}

func TestBuilderPaneShowsStatusAndLifecycleActions(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	msg := model.refreshCmd()().(snapshotMsg)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 28})
	updated, _ = updated.Update(msg)
	updated = switchToBuilder(t, updated)

	view := updated.View()
	for _, want := range []string{"Builder (running)", "buildkit", "running", "CPUs:    4", "Memory:  4.0 GB"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view did not include %q:\n%s", want, view)
		}
	}

	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if cmd == nil {
		t.Fatalf("expected start builder command")
	}
	done := cmd().(actionDoneMsg)
	updated, refresh := updated.Update(done)
	if refresh == nil {
		t.Fatalf("expected refresh after start builder")
	}
	if !client.builderStarted {
		t.Fatalf("expected builder start call")
	}

	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd == nil {
		t.Fatalf("expected stop builder command")
	}
	done = cmd().(actionDoneMsg)
	updated, refresh = updated.Update(done)
	if refresh == nil {
		t.Fatalf("expected refresh after stop builder")
	}
	if !client.builderStopped {
		t.Fatalf("expected builder stop call")
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	_, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatalf("expected delete builder confirmation command")
	}
	done = cmd().(actionDoneMsg)
	updated, refresh = updated.Update(done)
	if refresh == nil {
		t.Fatalf("expected refresh after delete builder")
	}
	if !client.builderDeleted {
		t.Fatalf("expected builder delete call")
	}
}

func TestSystemPaneShowsDiagnosticsLogsAndLifecycleActions(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	msg := model.refreshCmd()().(snapshotMsg)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 130, Height: 32})
	updated, _ = updated.Update(msg)
	updated = switchToSystem(t, updated)

	view := updated.View()
	for _, want := range []string{"System (running)", "running", "Disk usage", "Containers: 9 total", "Versions", "container: 1.0.0"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view did not include %q:\n%s", want, view)
		}
	}

	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if cmd == nil {
		t.Fatalf("expected system logs stream command")
	}
	if !client.systemFollowed {
		t.Fatalf("expected system logs follow stream")
	}
	updated = drainStream(t, updated, cmd)
	if !strings.Contains(updated.View(), "system ready") {
		t.Fatalf("view did not show streamed system logs:\n%s", updated.View())
	}

	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if cmd == nil {
		t.Fatalf("expected follow system logs command")
	}
	if !client.systemFollowed {
		t.Fatalf("expected follow system logs call")
	}

	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if cmd == nil {
		t.Fatalf("expected start system command")
	}
	done := cmd().(actionDoneMsg)
	updated, refresh := updated.Update(done)
	if refresh == nil {
		t.Fatalf("expected refresh after start system")
	}
	if !client.systemStarted {
		t.Fatalf("expected system start call")
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	_, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatalf("expected confirmed stop system command")
	}
	done = cmd().(actionDoneMsg)
	updated, refresh = updated.Update(done)
	if refresh == nil {
		t.Fatalf("expected refresh after stop system")
	}
	if !client.systemStopped {
		t.Fatalf("expected system stop call")
	}
}

func TestDeleteRequiresConfirmation(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	msg := model.refreshCmd()().(snapshotMsg)
	updated, _ := model.Update(msg)
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if client.deleted != "" {
		t.Fatalf("delete ran before confirmation")
	}

	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatalf("expected confirmation command")
	}
	done := cmd().(actionDoneMsg)
	updated, _ = updated.Update(done)

	if client.deleted != "db" {
		t.Fatalf("expected confirmed delete for db, got %q", client.deleted)
	}
}

func TestPruneContainersRequiresConfirmation(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	msg := model.refreshCmd()().(snapshotMsg)
	updated, _ := model.Update(msg)
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if client.pruned != "" {
		t.Fatalf("prune ran before confirmation")
	}

	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatalf("expected confirmation command")
	}
	done := cmd().(actionDoneMsg)
	updated, _ = updated.Update(done)

	if client.pruned != "containers" {
		t.Fatalf("expected confirmed container prune, got %q", client.pruned)
	}
}

func TestBulkMenuStopAllRequiresConfirmation(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	updated, _ = updated.Update(model.refreshCmd()().(snapshotMsg))

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'B'}})
	if view := updated.View(); !strings.Contains(view, "Bulk container actions") || !strings.Contains(view, "Stop all containers") {
		t.Fatalf("bulk menu not shown:\n%s", view)
	}

	// Selecting the first item arms a confirmation but must not run yet.
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if client.bulkStopAll {
		t.Fatalf("stop-all ran before confirmation")
	}

	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatalf("expected confirmation command")
	}
	updated.Update(cmd().(actionDoneMsg))
	if !client.bulkStopAll {
		t.Fatalf("expected StopAll after confirmation")
	}
}

func TestBulkMenuRemoveAllContainersForce(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	updated, _ = updated.Update(model.refreshCmd()().(snapshotMsg))

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'B'}})
	// Move to "Remove ALL containers (force)" (4th item) and run it.
	for i := 0; i < 3; i++ {
		updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatalf("expected confirmation command")
	}
	updated.Update(cmd().(actionDoneMsg))
	if !client.bulkDeleteAll {
		t.Fatalf("expected DeleteAllContainers after confirmation")
	}
}

func TestFilterNarrowsContainersAndActionsUseVisibleSelection(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	snapshot := snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{
			testContainer("api-service", "docker.io/library/alpine:latest"),
			testContainer("db", "docker.io/library/postgres:17"),
		},
	}
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(snapshot)
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "postgres" {
		updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})

	view := updated.View()
	if !strings.Contains(view, "Containers (1/2)") {
		t.Fatalf("view did not show filtered container count:\n%s", view)
	}
	if strings.Contains(view, "api-service") {
		t.Fatalf("view included filtered-out container:\n%s", view)
	}
	if !strings.Contains(view, "db") {
		t.Fatalf("view did not include matching container:\n%s", view)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	_, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatalf("expected delete command for filtered row")
	}
	done := cmd().(actionDoneMsg)
	updated, _ = updated.Update(done)

	if client.deleted != "db" {
		t.Fatalf("expected filtered delete target db, got %q", client.deleted)
	}
}

func TestEscapeClearsFilter(t *testing.T) {
	model := New(&fakeClient{})
	snapshot := snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{
			testContainer("api-service", "docker.io/library/alpine:latest"),
			testContainer("db", "docker.io/library/postgres:17"),
		},
	}
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(snapshot)
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})

	view := updated.View()
	if strings.Contains(view, "Containers (1/2)") {
		t.Fatalf("filter count remained after escape:\n%s", view)
	}
	if !strings.Contains(view, "api-service") || !strings.Contains(view, "db") {
		t.Fatalf("view did not restore all containers:\n%s", view)
	}
}

func TestShellRequiresRunningContainer(t *testing.T) {
	model := New(&fakeClient{})
	msg := model.refreshCmd()().(snapshotMsg)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	updated, _ = updated.Update(msg)
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd != nil {
		t.Fatalf("expected no shell command for stopped container")
	}
	view := updated.View()
	if !strings.Contains(view, "start db before opening a shell") {
		t.Fatalf("view did not explain shell guard:\n%s", view)
	}
}

func TestExecCommandRequiresRunningContainer(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	msg := model.refreshCmd()().(snapshotMsg)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	updated, _ = updated.Update(msg)
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
	if cmd != nil {
		t.Fatalf("expected no exec prompt command for stopped container")
	}
	if client.execID != "" {
		t.Fatalf("exec ran for stopped container")
	}
	view := updated.View()
	if !strings.Contains(view, "start db before running commands") {
		t.Fatalf("view did not explain exec guard:\n%s", view)
	}
}

func TestExecCommandShowsSelectedContainerOutput(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{
			testContainerWithState("api-service", "docker.io/library/alpine:latest", "running"),
			testContainerWithState("db", "docker.io/library/postgres:17", "running"),
		},
	})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
	for _, r := range "cat /etc/os-release" {
		updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected exec command")
	}
	output := cmd().(outputMsg)
	updated, _ = updated.Update(output)

	if client.execID != "db" {
		t.Fatalf("expected exec target db, got %q", client.execID)
	}
	if client.execCommand != "cat /etc/os-release" {
		t.Fatalf("expected exec command, got %q", client.execCommand)
	}
	view := updated.View()
	if !strings.Contains(view, "Exec db") || !strings.Contains(view, "ok") {
		t.Fatalf("view did not show exec output:\n%s", view)
	}
}

func TestContainerCommandPromptRunsAdHocContainerCommand(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	updated, _ = updated.Update(snapshotMsg{system: containercli.SystemStatus{Status: "running"}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	for _, r := range `image list --filter "name=postgres latest"` {
		updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected container command")
	}
	output := cmd().(outputMsg)
	updated, _ = updated.Update(output)

	wantArgs := []string{"image", "list", "--filter", "name=postgres latest"}
	if !reflect.DeepEqual(client.commandArgs, wantArgs) {
		t.Fatalf("command args mismatch\nwant: %#v\n got: %#v", wantArgs, client.commandArgs)
	}
	view := updated.View()
	if !strings.Contains(view, "container image list --filter name=postgres latest") || !strings.Contains(view, "command output") {
		t.Fatalf("view did not show container command output:\n%s", view)
	}
}

func TestCustomCommandPromptRunsConfiguredCommandByNumber(t *testing.T) {
	client := &fakeClient{}
	model := NewWithOptions(client, Options{
		CustomCommands: []CustomCommand{{
			Name: "Images",
			Args: []string{"image", "list", "--format", "json"},
		}},
	})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{';'}})
	if !strings.Contains(updated.View(), "1=Images") {
		t.Fatalf("view did not show custom command choices:\n%s", updated.View())
	}
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected custom command")
	}
	output := cmd().(outputMsg)
	updated, _ = updated.Update(output)

	wantArgs := []string{"image", "list", "--format", "json"}
	if !reflect.DeepEqual(client.commandArgs, wantArgs) {
		t.Fatalf("command args mismatch\nwant: %#v\n got: %#v", wantArgs, client.commandArgs)
	}
	if !strings.Contains(updated.View(), "Custom Images") || !strings.Contains(updated.View(), "command output") {
		t.Fatalf("view did not show custom command output:\n%s", updated.View())
	}
}

func TestCustomCommandPromptRunsConfiguredCommandByUniqueName(t *testing.T) {
	client := &fakeClient{}
	model := NewWithOptions(client, Options{
		CustomCommands: []CustomCommand{{
			Name: "System disk",
			Args: []string{"system", "df"},
		}, {
			Name: "Image list",
			Args: []string{"image", "list"},
		}},
	})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{';'}})
	for _, r := range "disk" {
		updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	_, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected custom command")
	}
	_ = cmd().(outputMsg)

	wantArgs := []string{"system", "df"}
	if !reflect.DeepEqual(client.commandArgs, wantArgs) {
		t.Fatalf("command args mismatch\nwant: %#v\n got: %#v", wantArgs, client.commandArgs)
	}
}

func TestCustomCommandExpandsSelectedContainerPlaceholder(t *testing.T) {
	client := &fakeClient{}
	model := NewWithOptions(client, Options{
		CustomCommands: []CustomCommand{{
			Name: "Container logs",
			Args: []string{"logs", "--tail", "50", "{container}"},
		}},
	})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	updated, _ = updated.Update(snapshotMsg{
		system:     containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{testContainer("db", "docker.io/library/postgres:17")},
	})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{';'}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	_, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected custom command")
	}
	_ = cmd().(outputMsg)

	wantArgs := []string{"logs", "--tail", "50", "db"}
	if !reflect.DeepEqual(client.commandArgs, wantArgs) {
		t.Fatalf("command args mismatch\nwant: %#v\n got: %#v", wantArgs, client.commandArgs)
	}
}

func TestCustomCommandExpandsActiveResourcePlaceholder(t *testing.T) {
	client := &fakeClient{}
	model := NewWithOptions(client, Options{
		CustomCommands: []CustomCommand{{
			Name: "Inspect resource",
			Args: []string{"image", "inspect", "{resource}"},
		}},
	})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	updated, _ = updated.Update(snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		images: []containercli.Image{{
			Configuration: containercli.ImageConfiguration{Name: "docker.io/library/postgres:17"},
		}},
	})
	updated = switchToResource(t, updated, resourceImages)
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{';'}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	_, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected custom command")
	}
	_ = cmd().(outputMsg)

	wantArgs := []string{"image", "inspect", "docker.io/library/postgres:17"}
	if !reflect.DeepEqual(client.commandArgs, wantArgs) {
		t.Fatalf("command args mismatch\nwant: %#v\n got: %#v", wantArgs, client.commandArgs)
	}
}

func TestCustomCommandReportsMissingSelectedPlaceholder(t *testing.T) {
	client := &fakeClient{}
	model := NewWithOptions(client, Options{
		CustomCommands: []CustomCommand{{
			Name: "Container logs",
			Args: []string{"logs", "{container}"},
		}},
	})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{';'}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("expected no command")
	}
	state := updated.(Model)
	if state.statusLine != "custom command needs selected container" {
		t.Fatalf("statusLine = %q", state.statusLine)
	}
	if len(client.commandArgs) != 0 {
		t.Fatalf("command unexpectedly ran: %#v", client.commandArgs)
	}
}

func TestCustomCommandPromptRequiresConfiguredCommands(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{';'}})
	if cmd != nil {
		t.Fatalf("expected no command")
	}
	state := updated.(Model)
	if state.prompt != promptNone {
		t.Fatalf("prompt = %v, want none", state.prompt)
	}
	if state.statusLine != "no custom commands configured" {
		t.Fatalf("statusLine = %q", state.statusLine)
	}
}

func TestOpenConfigUsesConfiguredEditorCommand(t *testing.T) {
	client := &fakeClient{}
	var openedPath string
	model := NewWithOptions(client, Options{
		ConfigPath: "/tmp/lazycontainer/config.json",
		OpenConfigCommand: func(path string) (*exec.Cmd, error) {
			openedPath = path
			return exec.Command("true"), nil
		},
		LoadConfigCommands: func() ([]CustomCommand, error) {
			return []CustomCommand{{
				Name: "Images",
				Args: []string{"image", "list"},
			}}, nil
		},
	})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	if cmd == nil {
		t.Fatalf("expected config editor command")
	}
	if openedPath != "/tmp/lazycontainer/config.json" {
		t.Fatalf("opened path = %q", openedPath)
	}
	if updated.(Model).busy != "editing config" {
		t.Fatalf("busy = %q", updated.(Model).busy)
	}
	if updated.(Model).statusLine != "editing config" {
		t.Fatalf("statusLine = %q", updated.(Model).statusLine)
	}

	updated, cmd = updated.Update(configEditedMsg{path: "/tmp/lazycontainer/config.json"})
	if cmd != nil {
		t.Fatalf("expected no command after config reload")
	}
	state := updated.(Model)
	if state.statusLine != "edited config /tmp/lazycontainer/config.json" {
		t.Fatalf("statusLine = %q", state.statusLine)
	}
	if len(state.customCommands) != 1 || state.customCommands[0].Name != "Images" {
		t.Fatalf("custom commands were not reloaded: %#v", state.customCommands)
	}
}

func TestOpenConfigRequiresEditorCommand(t *testing.T) {
	client := &fakeClient{}
	model := NewWithOptions(client, Options{ConfigPath: "/tmp/lazycontainer/config.json"})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	if cmd != nil {
		t.Fatalf("expected no command")
	}
	if updated.(Model).statusLine != "config editor unavailable" {
		t.Fatalf("statusLine = %q", updated.(Model).statusLine)
	}
}

func TestRestartRequiresRunningContainer(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	msg := model.refreshCmd()().(snapshotMsg)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	updated, _ = updated.Update(msg)
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd != nil {
		t.Fatalf("expected no restart command for stopped container")
	}
	if client.restarted != "" {
		t.Fatalf("restart ran for stopped container")
	}
	view := updated.View()
	if !strings.Contains(view, "start db before restarting") {
		t.Fatalf("view did not explain restart guard:\n%s", view)
	}
}

func TestRestartUsesSelectedRunningContainer(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{
			testContainerWithState("api-service", "docker.io/library/alpine:latest", "running"),
			testContainerWithState("db", "docker.io/library/postgres:17", "running"),
		},
	})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd == nil {
		t.Fatalf("expected restart command")
	}
	done := cmd().(actionDoneMsg)
	updated, refresh := updated.Update(done)
	if refresh == nil {
		t.Fatalf("expected refresh after restart")
	}
	if client.restarted != "db" {
		t.Fatalf("expected selected container db, got %q", client.restarted)
	}
}

func TestInitStartsRefreshAndAutoRefreshTimer(t *testing.T) {
	model := New(&fakeClient{})
	cmd := model.Init()
	if cmd == nil {
		t.Fatalf("expected init command")
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected batch init message, got %T", msg)
	}
	if len(batch) != 2 {
		t.Fatalf("expected refresh and auto-refresh commands, got %d", len(batch))
	}
}

func TestAutoRefreshTickRefreshesWhenIdleAndReschedules(t *testing.T) {
	model := New(&fakeClient{})
	model.refreshInterval = time.Millisecond
	updated, cmd := model.Update(autoRefreshMsg(time.Now()))
	if cmd == nil {
		t.Fatalf("expected auto-refresh batch command")
	}
	if updated.(Model).busy != "refreshing" {
		t.Fatalf("expected refreshing busy state, got %q", updated.(Model).busy)
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected refresh and next tick batch, got %T", msg)
	}
	if len(batch) != 2 {
		t.Fatalf("expected refresh and next tick commands, got %d", len(batch))
	}
}

func TestAutoRefreshTickSkipsDuringPrompt(t *testing.T) {
	model := New(&fakeClient{})
	model.prompt = promptPullImage
	model.refreshInterval = time.Millisecond
	updated, cmd := model.Update(autoRefreshMsg(time.Now()))
	if cmd == nil {
		t.Fatalf("expected next tick command")
	}
	if updated.(Model).busy != "" {
		t.Fatalf("expected no refresh while prompting, got busy %q", updated.(Model).busy)
	}
	if _, ok := cmd().(autoRefreshMsg); !ok {
		t.Fatalf("expected a rescheduled auto-refresh tick")
	}
}

func TestAutoRefreshToggle(t *testing.T) {
	model := New(&fakeClient{})
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if cmd != nil {
		t.Fatalf("expected no command when disabling auto-refresh")
	}
	if updated.(Model).autoRefresh {
		t.Fatalf("expected auto-refresh disabled")
	}
	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if cmd == nil {
		t.Fatalf("expected tick command when enabling auto-refresh")
	}
	if !updated.(Model).autoRefresh {
		t.Fatalf("expected auto-refresh enabled")
	}
}

func TestFollowLogsUsesSelectedContainer(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	updated, _ = updated.Update(snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{
			testContainer("api-service", "docker.io/library/alpine:latest"),
			testContainer("db", "docker.io/library/postgres:17"),
		},
	})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	_, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if cmd == nil {
		t.Fatalf("expected follow logs exec command")
	}
	if client.followLogsID != "db" {
		t.Fatalf("expected selected container db, got %q", client.followLogsID)
	}
}

func TestLogsTabStreamsSelectedContainer(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	updated, _ = updated.Update(snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{
			testContainer("api-service", "docker.io/library/alpine:latest"),
		},
	})
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if cmd == nil {
		t.Fatalf("expected log stream command")
	}
	if client.followLogsID != "api-service" {
		t.Fatalf("expected stream target api-service, got %q", client.followLogsID)
	}
	if client.followLogsCount != 1 {
		t.Fatalf("expected one follow stream, got %d", client.followLogsCount)
	}
	updated = drainStream(t, updated, cmd)
	if !strings.Contains(updated.View(), "container ready") {
		t.Fatalf("view did not show streamed logs:\n%s", updated.View())
	}

	// Manual refresh keeps the existing stream rather than restarting it.
	_, rcmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if rcmd == nil {
		t.Fatalf("expected refresh command")
	}
	if client.followLogsCount != 1 {
		t.Fatalf("expected stream not to restart on refresh, got %d", client.followLogsCount)
	}
}

func TestAutoRefreshKeepsLogStreamRunning(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	model.refreshInterval = time.Nanosecond
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	updated, _ = updated.Update(snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{
			testContainer("api-service", "docker.io/library/alpine:latest"),
		},
	})
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if cmd == nil {
		t.Fatalf("expected log stream command")
	}
	updated = drainStream(t, updated, cmd)
	if client.followLogsCount != 1 {
		t.Fatalf("expected one follow stream, got %d", client.followLogsCount)
	}

	_, cmd = updated.Update(autoRefreshMsg(time.Now()))
	if cmd == nil {
		t.Fatalf("expected auto-refresh command")
	}
	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected refresh/tick batch, got %T", cmd())
	}
	// Snapshot + next tick only; the live stream is not re-fetched on refresh.
	if len(batch) != 2 {
		t.Fatalf("expected snapshot and tick commands, got %d", len(batch))
	}
	if client.followLogsCount != 1 {
		t.Fatalf("expected stream not to restart on auto-refresh, got %d", client.followLogsCount)
	}
}

func TestMachinePaneShowsAndActionsUseSelectedMachine(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	msg := model.refreshCmd()().(snapshotMsg)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(msg)
	updated = switchToMachines(t, updated)

	view := updated.View()
	if !strings.Contains(view, "Machines (1)") {
		t.Fatalf("view did not include machine count:\n%s", view)
	}
	if !strings.Contains(view, "dev-machine") || !strings.Contains(view, "running") {
		t.Fatalf("view did not include selected machine:\n%s", view)
	}

	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if cmd == nil {
		t.Fatalf("expected machine logs stream command")
	}
	if client.machineLogsID != "dev-machine" {
		t.Fatalf("expected machine log target dev-machine, got %q", client.machineLogsID)
	}
	updated = drainStream(t, updated, cmd)
	if !strings.Contains(updated.View(), "machine ready") {
		t.Fatalf("view did not show streamed machine logs:\n%s", updated.View())
	}

	client.machineLogsID = ""
	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if cmd == nil {
		t.Fatalf("expected follow machine logs exec command")
	}
	if client.machineLogsID != "dev-machine" {
		t.Fatalf("expected follow machine log target dev-machine, got %q", client.machineLogsID)
	}

	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd == nil {
		t.Fatalf("expected machine shell exec command")
	}
	if client.machineShellID != "dev-machine" {
		t.Fatalf("expected machine shell target dev-machine, got %q", client.machineShellID)
	}

	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd == nil {
		t.Fatalf("expected stop machine command")
	}
	stopDone := cmd().(actionDoneMsg)
	updated, _ = updated.Update(stopDone)
	if client.stoppedMachine != "dev-machine" {
		t.Fatalf("expected stopped machine dev-machine, got %q", client.stoppedMachine)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	_, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatalf("expected delete machine confirmation command")
	}
	deleteDone := cmd().(actionDoneMsg)
	updated, _ = updated.Update(deleteDone)
	if client.deletedMachine != "dev-machine" {
		t.Fatalf("expected deleted machine dev-machine, got %q", client.deletedMachine)
	}
}

func TestRegistryPaneShowsAndLogoutUsesSelectedRegistry(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	msg := model.refreshCmd()().(snapshotMsg)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(msg)
	updated = switchToRegistries(t, updated)

	view := updated.View()
	if !strings.Contains(view, "Registries (1)") {
		t.Fatalf("view did not include registry count:\n%s", view)
	}
	if !strings.Contains(view, "ghcr.io") || !strings.Contains(view, "alice") {
		t.Fatalf("view did not include selected registry:\n%s", view)
	}

	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if cmd != nil {
		t.Fatalf("expected registry inspect to use local details")
	}
	if !strings.Contains(updated.View(), "Registry login") {
		t.Fatalf("view did not show registry details:\n%s", updated.View())
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	_, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatalf("expected registry logout confirmation command")
	}
	done := cmd().(actionDoneMsg)
	updated, refresh := updated.Update(done)
	if refresh == nil {
		t.Fatalf("expected refresh after registry logout")
	}
	if client.registryLogout != "ghcr.io" {
		t.Fatalf("expected registry logout ghcr.io, got %q", client.registryLogout)
	}
}

func TestRegistryLoginPromptRunsInteractiveCommand(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(snapshotMsg{system: containercli.SystemStatus{Status: "running"}})
	updated = switchToRegistries(t, updated)

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	for _, r := range "registry.example.com alice" {
		updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected registry login exec command")
	}
	if client.registryLogin != "registry.example.com" {
		t.Fatalf("expected registry login server, got %q", client.registryLogin)
	}
	if client.registryUser != "alice" {
		t.Fatalf("expected registry login user, got %q", client.registryUser)
	}
}

func TestCreateMachinePromptUsesImageAndOptionalName(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(snapshotMsg{system: containercli.SystemStatus{Status: "running"}})
	updated = switchToMachines(t, updated)

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'M'}})
	for _, r := range "alpine:3.22 dev-machine" {
		updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected create machine command")
	}
	done := cmd().(actionDoneMsg)
	updated, refresh := updated.Update(done)
	if refresh == nil {
		t.Fatalf("expected refresh after create machine")
	}

	if client.machineImage != "alpine:3.22" {
		t.Fatalf("expected machine image alpine:3.22, got %q", client.machineImage)
	}
	if client.machineName != "dev-machine" {
		t.Fatalf("expected machine name dev-machine, got %q", client.machineName)
	}
}

func TestSetMachinePromptUsesSelectedMachineAndSettings(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	msg := model.refreshCmd()().(snapshotMsg)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(msg)
	updated = switchToMachines(t, updated)

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	for _, r := range "cpus=4 memory=8G home-mount=ro ignored" {
		updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected set machine command")
	}
	done := cmd().(actionDoneMsg)
	updated, refresh := updated.Update(done)
	if refresh == nil {
		t.Fatalf("expected refresh after set machine")
	}

	if client.machineSetID != "dev-machine" {
		t.Fatalf("expected machine set target dev-machine, got %q", client.machineSetID)
	}
	wantSettings := []string{"cpus=4", "memory=8G", "home-mount=ro"}
	if strings.Join(client.machineSettings, " ") != strings.Join(wantSettings, " ") {
		t.Fatalf("settings mismatch\nwant: %#v\n got: %#v", wantSettings, client.machineSettings)
	}
}

func TestSetDefaultMachineUsesSelectedMachine(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		machines: []containercli.Machine{{
			ID: "dev-machine",
			Status: map[string]any{
				"state": "stopped",
			},
		}},
	})
	updated = switchToMachines(t, updated)

	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	if cmd == nil {
		t.Fatalf("expected set default machine command")
	}
	done := cmd().(actionDoneMsg)
	updated, refresh := updated.Update(done)
	if refresh == nil {
		t.Fatalf("expected refresh after set default")
	}

	if client.defaultMachine != "dev-machine" {
		t.Fatalf("expected default machine dev-machine, got %q", client.defaultMachine)
	}
}

func TestPullImagePromptRunsPullAndRefreshes(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(snapshotMsg{system: containercli.SystemStatus{Status: "running"}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	for _, r := range "docker.io/library/alpine:latest" {
		updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected pull command")
	}
	output := cmd().(outputMsg)
	if !strings.Contains(output.body, "pull output") {
		t.Fatalf("expected pull output body, got %q", output.body)
	}
	updated, refresh := updated.Update(output)
	if refresh == nil {
		t.Fatalf("expected refresh after pull")
	}

	if client.pulled != "docker.io/library/alpine:latest" {
		t.Fatalf("expected pulled image reference, got %q", client.pulled)
	}
}

func TestRunSelectedImagePromptsForLaunchOptions(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		images: []containercli.Image{{
			ID: "abc",
			Configuration: containercli.ImageConfiguration{
				Name: "docker.io/library/alpine:latest",
			},
		}},
	})
	updated = switchToResource(t, updated, resourceImages)
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	for _, r := range `name=web p=127.0.0.1:8080:80 env=APP_ENV=dev v=cache:/cache network=frontend -- npm start` {
		updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected run image command")
	}
	done := cmd().(actionDoneMsg)
	updated, refresh := updated.Update(done)
	if refresh == nil {
		t.Fatalf("expected refresh after run")
	}

	if client.runImage != "docker.io/library/alpine:latest" {
		t.Fatalf("expected run image target, got %q", client.runImage)
	}
	if client.runOptions.Name != "web" {
		t.Fatalf("expected container name web, got %q", client.runOptions.Name)
	}
	wantFlags := []string{
		"--publish", "127.0.0.1:8080:80",
		"--env", "APP_ENV=dev",
		"--volume", "cache:/cache",
		"--network", "frontend",
	}
	if !reflect.DeepEqual(client.runOptions.Flags, wantFlags) {
		t.Fatalf("run flags mismatch\nwant: %#v\n got: %#v", wantFlags, client.runOptions.Flags)
	}
	wantArgs := []string{"npm", "start"}
	if !reflect.DeepEqual(client.runOptions.Arguments, wantArgs) {
		t.Fatalf("run arguments mismatch\nwant: %#v\n got: %#v", wantArgs, client.runOptions.Arguments)
	}
}

func TestCreateContainerFromSelectedImagePromptsForLaunchOptions(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		images: []containercli.Image{{
			ID: "abc",
			Configuration: containercli.ImageConfiguration{
				Name: "docker.io/library/alpine:latest",
			},
		}},
	})
	updated = switchToResource(t, updated, resourceImages)
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
	for _, r := range `--name worker --env "GREETING=hello world" -- /bin/sh -lc "echo ready"` {
		updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected create container command")
	}
	done := cmd().(actionDoneMsg)
	updated, refresh := updated.Update(done)
	if refresh == nil {
		t.Fatalf("expected refresh after create container")
	}

	if client.createImage != "docker.io/library/alpine:latest" {
		t.Fatalf("expected create image target, got %q", client.createImage)
	}
	wantFlags := []string{"--name", "worker", "--env", "GREETING=hello world"}
	if !reflect.DeepEqual(client.createOptions.Flags, wantFlags) {
		t.Fatalf("create flags mismatch\nwant: %#v\n got: %#v", wantFlags, client.createOptions.Flags)
	}
	wantArgs := []string{"/bin/sh", "-lc", "echo ready"}
	if !reflect.DeepEqual(client.createOptions.Arguments, wantArgs) {
		t.Fatalf("create arguments mismatch\nwant: %#v\n got: %#v", wantArgs, client.createOptions.Arguments)
	}
}

func TestCreateContainerStillAcceptsBareName(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		images: []containercli.Image{{
			ID: "abc",
			Configuration: containercli.ImageConfiguration{
				Name: "docker.io/library/alpine:latest",
			},
		}},
	})
	updated = switchToResource(t, updated, resourceImages)
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
	for _, r := range "scratch" {
		updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	_, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected create container command")
	}
	_ = cmd().(actionDoneMsg)

	if client.createOptions.Name != "scratch" {
		t.Fatalf("expected bare name scratch, got %q", client.createOptions.Name)
	}
}

func TestParseContainerLaunchInputRejectsUnknownAssignment(t *testing.T) {
	if _, ok := parseContainerLaunchInput("name=web madeup=value"); ok {
		t.Fatalf("expected unknown launch assignment to be rejected")
	}
}

func TestParseContainerLaunchInputKeepsProgressValue(t *testing.T) {
	options, ok := parseContainerLaunchInput("--progress plain --name web")
	if !ok {
		t.Fatalf("expected launch options to parse")
	}
	wantFlags := []string{"--progress", "plain", "--name", "web"}
	if !reflect.DeepEqual(options.Flags, wantFlags) {
		t.Fatalf("flags mismatch\nwant: %#v\n got: %#v", wantFlags, options.Flags)
	}
}

func TestBuildImagePromptBuildsWithDefaultContext(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(snapshotMsg{system: containercli.SystemStatus{Status: "running"}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	for _, r := range "registry.example.com/app:dev" {
		updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected build command")
	}
	output := cmd().(outputMsg)
	if !strings.Contains(output.body, "build output") {
		t.Fatalf("expected build output body, got %q", output.body)
	}
	updated, refresh := updated.Update(output)
	if refresh == nil {
		t.Fatalf("expected refresh after build")
	}

	if client.buildTag != "registry.example.com/app:dev" {
		t.Fatalf("expected build tag, got %q", client.buildTag)
	}
	if client.buildContext != "." {
		t.Fatalf("expected default build context, got %q", client.buildContext)
	}
}

func TestBuildImagePromptBuildsWithProvidedContext(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(snapshotMsg{system: containercli.SystemStatus{Status: "running"}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	for _, r := range "registry.example.com/app:dev ./services/api" {
		updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected build command")
	}
	output := cmd().(outputMsg)
	if !strings.Contains(output.body, "build output") {
		t.Fatalf("expected build output body, got %q", output.body)
	}
	updated, refresh := updated.Update(output)
	if refresh == nil {
		t.Fatalf("expected refresh after build")
	}

	if client.buildTag != "registry.example.com/app:dev" {
		t.Fatalf("expected build tag, got %q", client.buildTag)
	}
	if client.buildContext != "./services/api" {
		t.Fatalf("expected provided build context, got %q", client.buildContext)
	}
}

func TestTagSelectedImagePromptsForTargetReference(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		images: []containercli.Image{{
			ID: "abc",
			Configuration: containercli.ImageConfiguration{
				Name: "docker.io/library/alpine:latest",
			},
		}},
	})
	updated = switchToResource(t, updated, resourceImages)
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	for _, r := range "registry.example.com/alpine:dev" {
		updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected tag image command")
	}
	done := cmd().(actionDoneMsg)
	updated, refresh := updated.Update(done)
	if refresh == nil {
		t.Fatalf("expected refresh after tag")
	}

	if client.tagSource != "docker.io/library/alpine:latest" {
		t.Fatalf("expected selected image source, got %q", client.tagSource)
	}
	if client.tagTarget != "registry.example.com/alpine:dev" {
		t.Fatalf("expected target image reference, got %q", client.tagTarget)
	}
}

func TestPushSelectedImageUsesImageReference(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		images: []containercli.Image{{
			ID: "abc",
			Configuration: containercli.ImageConfiguration{
				Name: "registry.example.com/alpine:dev",
			},
		}},
	})
	updated = switchToResource(t, updated, resourceImages)
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}})
	if cmd == nil {
		t.Fatalf("expected push image command")
	}
	output := cmd().(outputMsg)
	if !strings.Contains(output.body, "push output") {
		t.Fatalf("expected push output body, got %q", output.body)
	}
	updated, refresh := updated.Update(output)
	if refresh == nil {
		t.Fatalf("expected refresh after push")
	}

	if client.pushed != "registry.example.com/alpine:dev" {
		t.Fatalf("expected pushed image reference, got %q", client.pushed)
	}
}

func TestSaveSelectedImageUsesDefaultArchivePath(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(snapshotMsg{
		system: containercli.SystemStatus{Status: "running"},
		images: []containercli.Image{{
			ID: "abc",
			Configuration: containercli.ImageConfiguration{
				Name: "docker.io/library/alpine:latest",
			},
		}},
	})
	updated = switchToResource(t, updated, resourceImages)
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'O'}})
	if cmd != nil {
		t.Fatalf("expected save prompt, got command")
	}
	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected save image command")
	}
	output := cmd().(outputMsg)
	if !strings.Contains(output.body, "save output") {
		t.Fatalf("expected save output body, got %q", output.body)
	}
	updated, refresh := updated.Update(output)
	if refresh == nil {
		t.Fatalf("expected refresh after save")
	}

	if client.savedImage != "docker.io/library/alpine:latest" {
		t.Fatalf("expected saved image reference, got %q", client.savedImage)
	}
	if client.saveOutput != "docker.io_library_alpine_latest.tar" {
		t.Fatalf("expected default image archive path, got %q", client.saveOutput)
	}
}

func TestLoadImageArchivePromptUsesInputPath(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(snapshotMsg{system: containercli.SystemStatus{Status: "running"}})
	updated = switchToResource(t, updated, resourceImages)
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	for _, r := range "./archives/alpine.tar" {
		updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected load image command")
	}
	output := cmd().(outputMsg)
	if !strings.Contains(output.body, "load output") {
		t.Fatalf("expected load output body, got %q", output.body)
	}
	updated, refresh := updated.Update(output)
	if refresh == nil {
		t.Fatalf("expected refresh after load")
	}

	if client.loadedImage != "./archives/alpine.tar" {
		t.Fatalf("expected loaded image archive path, got %q", client.loadedImage)
	}
}

func TestCopySelectedContainerExpandsContainerSource(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(snapshotMsg{
		system:     containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{testContainer("db", "docker.io/library/postgres:17")},
	})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	for _, r := range ":/etc/hosts ./hosts" {
		updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected copy command")
	}
	done := cmd().(actionDoneMsg)
	updated, refresh := updated.Update(done)
	if refresh == nil {
		t.Fatalf("expected refresh after copy")
	}

	if client.copySource != "db:/etc/hosts" {
		t.Fatalf("expected selected container source, got %q", client.copySource)
	}
	if client.copyDest != "./hosts" {
		t.Fatalf("expected local destination, got %q", client.copyDest)
	}
}

func TestCopySelectedContainerExpandsContainerDestination(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(snapshotMsg{
		system:     containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{testContainer("db", "docker.io/library/postgres:17")},
	})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	for _, r := range "./config.json :/app/config.json" {
		updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected copy command")
	}
	done := cmd().(actionDoneMsg)
	updated, refresh := updated.Update(done)
	if refresh == nil {
		t.Fatalf("expected refresh after copy")
	}

	if client.copySource != "./config.json" {
		t.Fatalf("expected local source, got %q", client.copySource)
	}
	if client.copyDest != "db:/app/config.json" {
		t.Fatalf("expected selected container destination, got %q", client.copyDest)
	}
}

func TestExportSelectedContainerUsesDefaultTarPath(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(snapshotMsg{
		system:     containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{testContainer("db", "docker.io/library/postgres:17")},
	})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'E'}})
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected export command")
	}
	done := cmd().(actionDoneMsg)
	updated, refresh := updated.Update(done)
	if refresh == nil {
		t.Fatalf("expected refresh after export")
	}

	if client.exportID != "db" {
		t.Fatalf("expected exported container db, got %q", client.exportID)
	}
	if client.exportOutput != "db.tar" {
		t.Fatalf("expected default export path db.tar, got %q", client.exportOutput)
	}
}

func TestCreateVolumePromptUsesNameAndOptionalSize(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(snapshotMsg{system: containercli.SystemStatus{Status: "running"}})
	updated = switchToVolumes(t, updated)

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})
	for _, r := range "cache 10G" {
		updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected create volume command")
	}
	done := cmd().(actionDoneMsg)
	updated, refresh := updated.Update(done)
	if refresh == nil {
		t.Fatalf("expected refresh after create volume")
	}

	if client.createdVolume != "cache" {
		t.Fatalf("expected created volume cache, got %q", client.createdVolume)
	}
	if client.volumeSize != "10G" {
		t.Fatalf("expected volume size 10G, got %q", client.volumeSize)
	}
}

func TestCreateNetworkPromptUsesNameAndOptionalSubnet(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(snapshotMsg{system: containercli.SystemStatus{Status: "running"}})
	updated = switchToNetworks(t, updated)

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})
	for _, r := range "frontend 192.168.90.0/24" {
		updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected create network command")
	}
	done := cmd().(actionDoneMsg)
	updated, refresh := updated.Update(done)
	if refresh == nil {
		t.Fatalf("expected refresh after create network")
	}

	if client.createdNetwork != "frontend" {
		t.Fatalf("expected created network frontend, got %q", client.createdNetwork)
	}
	if client.networkSubnet != "192.168.90.0/24" {
		t.Fatalf("expected network subnet, got %q", client.networkSubnet)
	}
}

func switchToBuilder(t *testing.T, model tea.Model) tea.Model {
	t.Helper()
	return switchToResource(t, model, resourceBuilder)
}

func switchToVolumes(t *testing.T, model tea.Model) tea.Model {
	t.Helper()
	return switchToResource(t, model, resourceVolumes)
}

func switchToNetworks(t *testing.T, model tea.Model) tea.Model {
	t.Helper()
	return switchToResource(t, model, resourceNetworks)
}

func switchToMachines(t *testing.T, model tea.Model) tea.Model {
	t.Helper()
	return switchToResource(t, model, resourceMachines)
}

func switchToRegistries(t *testing.T, model tea.Model) tea.Model {
	t.Helper()
	return switchToResource(t, model, resourceRegistries)
}

func switchToSystem(t *testing.T, model tea.Model) tea.Model {
	t.Helper()
	return switchToResource(t, model, resourceSystem)
}

// switchToResource tabs forward until the target pane is focused, so tests stay
// correct regardless of pane ordering.
func switchToResource(t *testing.T, model tea.Model, kind resourceKind) tea.Model {
	t.Helper()
	updated := model
	for i := 0; i < int(resourceCount); i++ {
		if updated.(Model).active == kind {
			return updated
		}
		updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
	}
	if updated.(Model).active != kind {
		t.Fatalf("could not switch to resource %d", kind)
	}
	return updated
}

// drainStream pumps log-stream read commands until the stream ends, returning
// the model with all streamed lines applied.
func drainStream(t *testing.T, model tea.Model, cmd tea.Cmd) tea.Model {
	t.Helper()
	for i := 0; cmd != nil && i < 200; i++ {
		msg := cmd()
		lsm, ok := msg.(logStreamMsg)
		if !ok {
			t.Fatalf("expected logStreamMsg, got %T", msg)
		}
		var next tea.Cmd
		model, next = model.Update(lsm)
		cmd = next
		if lsm.done {
			break
		}
	}
	return model
}

func tabClickPoint(t *testing.T, model Model, target resourceKind) (int, int) {
	t.Helper()
	layout, ok := model.viewLayout()
	if !ok {
		t.Fatalf("expected test layout")
	}
	row, ok := layout.sidebar.headerRow[target]
	if !ok {
		t.Fatalf("resource section not found: %v", target)
	}
	return layout.sidebarContentX + 1, layout.sidebarContentY + row
}

func testContainer(id string, image string) containercli.Container {
	return testContainerWithState(id, image, "stopped")
}

func testContainerWithState(id string, image string, state string) containercli.Container {
	return containercli.Container{
		ID: id,
		Configuration: containercli.ContainerConfiguration{
			ID: id,
			Image: containercli.ImageRef{
				Reference: image,
			},
			Platform: containercli.Platform{OS: "linux", Architecture: "arm64"},
		},
		Status: containercli.ContainerStatus{State: state},
	}
}
