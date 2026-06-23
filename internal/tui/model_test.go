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
