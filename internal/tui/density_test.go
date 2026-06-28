package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pzep1/lazycont/internal/containercli"
)

func densityModel(t *testing.T) Model {
	t.Helper()
	model := New(&fakeClient{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	c := testContainerWithState("api", "ghcr.io/acme/api:1.0", "running")
	c.Configuration.PublishedPorts = []containercli.Port{{HostAddress: "0.0.0.0", HostPort: 8080, ContainerPort: 80, Proto: "tcp"}}
	c.Configuration.Mounts = []containercli.Mount{{Source: "data", Destination: "/data", Options: []string{"ro"}}}
	c.Configuration.Networks = []containercli.Network{{Network: "bridge"}}
	updated, _ = updated.Update(snapshotMsg{
		system:     containercli.SystemStatus{Status: "running"},
		containers: []containercli.Container{c},
		images:     []containercli.Image{{Configuration: containercli.ImageConfiguration{Name: "ghcr.io/acme/api:1.0"}}},
		volumes:    []containercli.Volume{{Configuration: containercli.VolumeConfiguration{Name: "data"}}},
		networks:   []containercli.NetworkResource{{Configuration: containercli.NetworkConfiguration{Name: "bridge", Mode: "nat"}}},
		stats:      []containercli.Stat{{"id": "api", "cpuPercent": float64(10), "memoryUsageBytes": float64(100 * 1024 * 1024), "memoryLimitBytes": float64(1024 * 1024 * 1024)}},
	})
	return updated.(Model)
}

func TestOverviewStripShowsFleetSummary(t *testing.T) {
	view := densityModel(t).View()
	for _, want := range []string{"FLEET", "1 ctr (1 up)", "cpu"} {
		if !strings.Contains(view, want) {
			t.Fatalf("overview strip missing %q:\n%s", want, view)
		}
	}
}

func TestInUseCountsAndBadge(t *testing.T) {
	m := densityModel(t)
	if got := m.imageInUseCount("ghcr.io/acme/api:1.0"); got != 1 {
		t.Fatalf("image in-use = %d, want 1", got)
	}
	if got := m.volumeInUseCount("data"); got != 1 {
		t.Fatalf("volume in-use = %d, want 1", got)
	}
	if got := m.networkInUseCount("bridge"); got != 1 {
		t.Fatalf("network in-use = %d, want 1", got)
	}
	if !strings.Contains(m.View(), "●1") {
		t.Fatalf("view missing in-use badge:\n%s", m.View())
	}
}

func TestMouseClickSelectsMainTab(t *testing.T) {
	m := densityModel(t)
	layout, ok := m.viewLayout()
	if !ok {
		t.Fatal("expected layout")
	}
	// Find the column where the "Ports" label starts in the tab strip.
	col := 0
	for i, tb := range m.activeTabs() {
		if i > 0 {
			col++ // separator space
		}
		if tb == tabPorts {
			break
		}
		col += len(tb.label())
	}
	updated, _ := m.Update(tea.MouseMsg{
		X:      layout.panelX + 2 + col + 1,
		Y:      layout.bodyTop + 1,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	if got := updated.(Model).activeMainTab(); got != tabPorts {
		t.Fatalf("clicking the Ports tab selected %v, want Ports", got)
	}
}

func TestContainerDetailTabsRender(t *testing.T) {
	m := densityModel(t)
	cases := map[mainTab]string{
		tabPorts:  "8080 -> 80/tcp",
		tabMounts: "data -> /data  (ro)",
		tabHealth: "1 published",
	}
	for tab, want := range cases {
		v := withTab(m, resourceContainers, tab).View()
		if !strings.Contains(v, want) {
			t.Fatalf("%s tab missing %q:\n%s", tab.label(), want, v)
		}
	}
}
