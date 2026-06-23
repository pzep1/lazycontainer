package tui

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pz/lazycont/internal/containercli"
)

type fakeClient struct {
	started        string
	pulled         string
	runImage       string
	runName        string
	buildTag       string
	buildContext   string
	tagSource      string
	tagTarget      string
	pushed         string
	copySource     string
	copyDest       string
	restarted      string
	followLogsID   string
	machineLogsID  string
	machineShellID string
	stoppedMachine string
	deleted        string
	deletedVolume  string
	deletedNetwork string
	deletedMachine string
	pruned         string
}

func (f *fakeClient) SystemStatus(context.Context) (containercli.SystemStatus, error) {
	return containercli.SystemStatus{Status: "running"}, nil
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

func (f *fakeClient) Stats(context.Context, ...string) ([]containercli.Stat, error) {
	return nil, nil
}

func (f *fakeClient) Logs(context.Context, string, int) (string, error) {
	return "ready\n", nil
}

func (f *fakeClient) FollowLogsCommand(id string, _ int) (*exec.Cmd, error) {
	f.followLogsID = id
	return exec.Command("true"), nil
}

func (f *fakeClient) MachineLogs(_ context.Context, id string, _ int) (string, error) {
	f.machineLogsID = id
	return "machine ready\n", nil
}

func (f *fakeClient) FollowMachineLogsCommand(id string, _ int) (*exec.Cmd, error) {
	f.machineLogsID = id
	return exec.Command("true"), nil
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

func (f *fakeClient) MachineShellCommand(id string) (*exec.Cmd, error) {
	f.machineShellID = id
	return exec.Command("true"), nil
}

func (f *fakeClient) PullImage(_ context.Context, reference string) error {
	f.pulled = reference
	return nil
}

func (f *fakeClient) RunImage(_ context.Context, image string, name string) error {
	f.runImage = image
	f.runName = name
	return nil
}

func (f *fakeClient) BuildImage(_ context.Context, tag string, contextDir string) error {
	f.buildTag = tag
	f.buildContext = contextDir
	return nil
}

func (f *fakeClient) TagImage(_ context.Context, source string, target string) error {
	f.tagSource = source
	f.tagTarget = target
	return nil
}

func (f *fakeClient) PushImage(_ context.Context, reference string) error {
	f.pushed = reference
	return nil
}

func (f *fakeClient) Copy(_ context.Context, source string, destination string) error {
	f.copySource = source
	f.copyDest = destination
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

func (f *fakeClient) Kill(context.Context, string) error {
	return nil
}

func (f *fakeClient) DeleteContainer(_ context.Context, id string, _ bool) error {
	f.deleted = id
	return nil
}

func (f *fakeClient) DeleteImage(context.Context, string, bool) error {
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

func (f *fakeClient) PruneImages(context.Context, bool) error {
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

	if !strings.Contains(view, "apple container: running") {
		t.Fatalf("view did not include system status:\n%s", view)
	}
	if !strings.Contains(view, "db") {
		t.Fatalf("view did not include container:\n%s", view)
	}
	if !strings.Contains(view, "containers 1") {
		t.Fatalf("view did not include container count:\n%s", view)
	}
	if !strings.Contains(view, "volumes 1") || !strings.Contains(view, "networks 1") || !strings.Contains(view, "machines 1") {
		t.Fatalf("view did not include secondary resource counts:\n%s", view)
	}
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

	view := updated.View()
	for _, want := range []string{"Stats", "CPU time: 1.2s", "Memory:", "[#---------------]", "Network:", "PIDs:     3"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view did not include %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "memoryUsageBytes") {
		t.Fatalf("view rendered raw stats instead of summary:\n%s", view)
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
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})

	view := updated.View()
	for _, want := range []string{"Layer history", "linux/arm64/v8", "375591c23c8d", "metadata", "CMD [\"/bin/sh\"]"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view did not include %q:\n%s", want, view)
		}
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
	if !strings.Contains(view, "containers 1/2") {
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
	if strings.Contains(view, "containers 1/2") {
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

func TestMachinePaneShowsAndActionsUseSelectedMachine(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	msg := model.refreshCmd()().(snapshotMsg)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	updated, _ = updated.Update(msg)
	updated = switchToMachines(t, updated)

	view := updated.View()
	if !strings.Contains(view, "machines 1") {
		t.Fatalf("view did not include machine count:\n%s", view)
	}
	if !strings.Contains(view, "dev-machine") || !strings.Contains(view, "running") {
		t.Fatalf("view did not include selected machine:\n%s", view)
	}

	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if cmd == nil {
		t.Fatalf("expected machine logs command")
	}
	logs := cmd().(outputMsg)
	updated, _ = updated.Update(logs)
	if client.machineLogsID != "dev-machine" {
		t.Fatalf("expected machine log target dev-machine, got %q", client.machineLogsID)
	}
	if !strings.Contains(updated.View(), "machine ready") {
		t.Fatalf("view did not show machine logs:\n%s", updated.View())
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
	done := cmd().(actionDoneMsg)
	updated, refresh := updated.Update(done)
	if refresh == nil {
		t.Fatalf("expected refresh after pull")
	}

	if client.pulled != "docker.io/library/alpine:latest" {
		t.Fatalf("expected pulled image reference, got %q", client.pulled)
	}
}

func TestRunSelectedImagePromptsForOptionalName(t *testing.T) {
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
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	for _, r := range "scratch" {
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
	if client.runName != "scratch" {
		t.Fatalf("expected container name scratch, got %q", client.runName)
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
	done := cmd().(actionDoneMsg)
	updated, refresh := updated.Update(done)
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
	done := cmd().(actionDoneMsg)
	updated, refresh := updated.Update(done)
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
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
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
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}})
	if cmd == nil {
		t.Fatalf("expected push image command")
	}
	done := cmd().(actionDoneMsg)
	updated, refresh := updated.Update(done)
	if refresh == nil {
		t.Fatalf("expected refresh after push")
	}

	if client.pushed != "registry.example.com/alpine:dev" {
		t.Fatalf("expected pushed image reference, got %q", client.pushed)
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

func switchToMachines(t *testing.T, model tea.Model) tea.Model {
	t.Helper()
	updated := model
	for range 4 {
		var cmd tea.Cmd
		updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
		if cmd != nil {
			t.Fatalf("expected no command while tabbing to machines")
		}
	}
	return updated
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
