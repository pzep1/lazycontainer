package tui

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pz/lazycont/internal/containercli"
)

type fakeClient struct {
	started        string
	pulled         string
	runImage       string
	runName        string
	followLogsID   string
	deleted        string
	deletedVolume  string
	deletedNetwork string
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

func (f *fakeClient) ShellCommand(string, string) (*exec.Cmd, error) {
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

func (f *fakeClient) Start(_ context.Context, id string) error {
	f.started = id
	return nil
}

func (f *fakeClient) Stop(context.Context, string) error {
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
	msg := model.Init()().(snapshotMsg)
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
	if !strings.Contains(view, "volumes 1") || !strings.Contains(view, "networks 1") {
		t.Fatalf("view did not include secondary resource counts:\n%s", view)
	}
}

func TestDeleteRequiresConfirmation(t *testing.T) {
	client := &fakeClient{}
	model := New(client)
	msg := model.Init()().(snapshotMsg)
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
	msg := model.Init()().(snapshotMsg)
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

func testContainer(id string, image string) containercli.Container {
	return containercli.Container{
		ID: id,
		Configuration: containercli.ContainerConfiguration{
			ID: id,
			Image: containercli.ImageRef{
				Reference: image,
			},
			Platform: containercli.Platform{OS: "linux", Architecture: "arm64"},
		},
		Status: containercli.ContainerStatus{State: "stopped"},
	}
}
