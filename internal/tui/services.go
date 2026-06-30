package tui

import (
	"context"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pzep1/lazycont/internal/compose"
	"github.com/pzep1/lazycont/internal/containercli"
)

// handleServiceKey owns the keys specific to the Services pane so the global
// key handler can delegate Services behavior here rather than carrying
// resourceServices special cases. It returns handled=false for keys it does not
// own, letting the caller fall through to the global handler.
func (m Model) handleServiceKey(key string) (tea.Model, tea.Cmd, bool) {
	switch key {
	case "u": // bring the selected service up
		model, cmd := m.upServiceSelected()
		return model, cmd, true
	case "U": // bring the whole project up
		model, cmd := m.upProjectSelected()
		return model, cmd, true
	case "D": // take the whole project down
		model, cmd := m.downProjectSelected()
		return model, cmd, true
	case "d": // take the selected service down
		model, cmd := m.downServiceSelected()
		return model, cmd, true
	case "R": // recreate the selected service
		model, cmd := m.recreateServiceSelected()
		return model, cmd, true
	case "e": // shell into the service's container
		model, cmd := m.serviceShellSelected()
		return model, cmd, true
	case "s":
		model, cmd := m.serviceLifecycle("starting", "started", func(ctx context.Context, r compose.Runner, s compose.Service) error {
			return r.StartService(ctx, s)
		})
		return model, cmd, true
	case "x":
		model, cmd := m.serviceLifecycle("stopping", "stopped", func(ctx context.Context, r compose.Runner, s compose.Service) error {
			return r.StopService(ctx, s)
		})
		return model, cmd, true
	case "ctrl+r":
		model, cmd := m.serviceLifecycle("restarting", "restarted", func(ctx context.Context, r compose.Runner, s compose.Service) error {
			return r.RestartService(ctx, s)
		})
		return model, cmd, true
	}
	return m, nil, false
}

// composeRunner builds a typed runner that owns compose execution semantics
// (idempotent teardown, surfaced bring-up errors) against the live client.
func (m Model) composeRunner() compose.Runner {
	return compose.Runner{Project: m.project, Exec: m.client.Command}
}

// runComposeOp runs a typed compose operation in the background, reporting the
// done message on success or surfacing the error.
func (m Model) runComposeOp(done string, op func(context.Context, compose.Runner) error) tea.Cmd {
	runner := m.composeRunner()
	return func() tea.Msg {
		if err := op(context.Background(), runner); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{message: done}
	}
}
func (m Model) upServiceSelected() (tea.Model, tea.Cmd) {
	service, ok := m.selectedService()
	if !ok {
		m.statusLine = m.emptyServiceMessage()
		return m, nil
	}
	m.busy = "bringing up " + service.Name
	m.statusLine = m.busy
	return m, m.runComposeOp("brought up "+service.Name, func(ctx context.Context, r compose.Runner) error {
		return r.UpService(ctx, service)
	})
}
func (m Model) upProjectSelected() (tea.Model, tea.Cmd) {
	if m.active != resourceServices || len(m.project.Services) == 0 {
		return m, nil
	}
	m.busy = "bringing up project " + m.project.Name
	m.statusLine = m.busy
	return m, m.runComposeOp("brought up project "+m.project.Name, func(ctx context.Context, r compose.Runner) error {
		return r.UpAll(ctx)
	})
}
func (m Model) recreateServiceSelected() (tea.Model, tea.Cmd) {
	service, ok := m.selectedService()
	if !ok {
		return m, nil
	}
	m.busy = "recreating " + service.Name
	m.statusLine = m.busy
	return m, m.runComposeOp("recreated "+service.Name, func(ctx context.Context, r compose.Runner) error {
		return r.RecreateService(ctx, service)
	})
}
func (m Model) downServiceSelected() (tea.Model, tea.Cmd) {
	service, ok := m.selectedService()
	if !ok {
		return m, nil
	}
	m.confirm = &pendingConfirm{
		label: "Take service " + service.Name + " down (stop & remove)?",
		run: func(ctx context.Context, m Model) (string, error) {
			return "took down service " + service.Name, m.composeRunner().DownService(ctx, service)
		},
	}
	return m, nil
}

func (m Model) downProjectSelected() (tea.Model, tea.Cmd) {
	if m.active != resourceServices || len(m.project.Services) == 0 {
		return m, nil
	}
	name := m.project.Name
	m.confirm = &pendingConfirm{
		label: "Take the whole project down (stop & remove all)?",
		run: func(ctx context.Context, m Model) (string, error) {
			return "took down project " + name, m.composeRunner().DownAll(ctx)
		},
	}
	return m, nil
}

// serviceLifecycle runs a start/stop/restart for the selected service through
// the typed compose runner, guarding the "no container yet" case for a helpful
// message.
func (m Model) serviceLifecycle(busy string, done string, op func(context.Context, compose.Runner, compose.Service) error) (tea.Model, tea.Cmd) {
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
	m.busy = busy + " " + id
	m.statusLine = m.busy
	runner := m.composeRunner()
	return m, func() tea.Msg {
		err := op(context.Background(), runner, service)
		return actionDoneMsg{message: done + " " + id, err: err}
	}
}

// serviceShellSelected opens a shell in the container backing the selected
// service.
func (m Model) serviceShellSelected() (tea.Model, tea.Cmd) {
	service, ok := m.selectedService()
	if !ok {
		return m, nil
	}
	container, ok := m.serviceContainer(service)
	if !ok {
		m.statusLine = service.Name + " has no container yet — press u to bring it up"
		return m, nil
	}
	return m.shellForContainer(container.Name())
}
func (m Model) selectedService() (compose.Service, bool) {
	indexes := m.filteredServiceIndexes()
	if len(indexes) == 0 || m.serviceCursor < 0 || m.serviceCursor >= len(indexes) {
		return compose.Service{}, false
	}
	return m.project.Services[indexes[m.serviceCursor]], true
}

// serviceContainer returns the live container backing a service, matched by the
// service's container name, if one exists.
func (m Model) serviceContainer(service compose.Service) (containercli.Container, bool) {
	name := m.project.ContainerNameFor(service)
	for _, container := range m.containers {
		if container.Name() == name {
			return container, true
		}
	}
	return containercli.Container{}, false
}

// serviceState reports a service's container state ("running", "stopped", …) or
// "—" when no container has been created for it yet.
func (m Model) serviceState(service compose.Service) string {
	if container, ok := m.serviceContainer(service); ok {
		return container.State()
	}
	return "—"
}
func (m Model) renderServiceList(width int, height int) []string {
	if len(m.project.Services) == 0 {
		return []string{mutedStyle.Render(m.emptyServiceMessage())}
	}
	indexes := m.filteredServiceIndexes()
	if len(indexes) == 0 {
		return []string{mutedStyle.Render(m.emptyListMessage("services"))}
	}
	rows := []string{mutedStyle.Render(fitColumns("service", "state", width))}
	start := visibleStart(m.serviceCursor, height-1, len(indexes))
	end := start + height - 1
	if end > len(indexes) {
		end = len(indexes)
	}
	for idx := start; idx < end; idx++ {
		service := m.project.Services[indexes[idx]]
		state := m.serviceState(service)
		name := truncate(service.Name, 22)
		summary := state
		if c, ok := m.serviceContainer(service); ok {
			if s := m.statListSummary(c.Name()); s != "" {
				summary = fmt.Sprintf("%s  %s", state, s)
			}
		}
		line := fitColumns(name, summary, width)
		if idx == m.serviceCursor {
			line = selectedStyle.Width(width).Render(truncate(line, width))
		} else {
			line = colorState(line, state)
		}
		rows = append(rows, line)
	}
	return rows
}

// emptyServiceMessage explains the Services pane when no Compose file is found
// or it failed to parse.
func (m Model) emptyServiceMessage() string {
	if m.projectErr != nil {
		return "compose error: " + m.projectErr.Error()
	}
	return "No compose.yaml found. Add one to manage a service stack here."
}
func (m Model) filteredServiceIndexes() []int {
	filter := activeFilter(m.filter)
	indexes := make([]int, 0, len(m.project.Services))
	for idx, service := range m.project.Services {
		name := m.project.ContainerNameFor(service)
		if m.isIgnored(service.Name, name, service.Image) {
			continue
		}
		if filter == "" || matchFields(filter, service.Name, name, service.Image, m.serviceState(service)) {
			indexes = append(indexes, idx)
		}
	}
	return indexes
}
