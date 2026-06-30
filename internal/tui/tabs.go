package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pzep1/lazycont/internal/compose"
)

// tabFetchedMsg carries the result of an asynchronous fetch for a fetched tab
// (logs, inspect, top). It is matched against the model's current tab and
// selection key on arrival so stale results from a previous selection are
// dropped without needing a generation counter.
type tabFetchedMsg struct {
	tab   mainTab
	key   string
	title string
	body  string
	err   error
}

// tabsFor returns the ordered main-panel tabs for a resource kind.
func tabsFor(kind resourceKind) []mainTab {
	switch kind {
	case resourceContainers:
		return []mainTab{tabConfig, tabLogs, tabStats, tabEnv, tabPorts, tabMounts, tabHealth, tabTop, tabInspect}
	case resourceServices:
		return []mainTab{tabConfig, tabLogs, tabInspect}
	case resourceImages, resourceVolumes, resourceNetworks:
		return []mainTab{tabConfig, tabInspect}
	case resourceMachines:
		return []mainTab{tabConfig, tabLogs, tabInspect}
	case resourceSystem:
		return []mainTab{tabConfig, tabLogs}
	default: // builder, registries
		return []mainTab{tabConfig}
	}
}

func (t mainTab) label() string {
	switch t {
	case tabConfig:
		return "Config"
	case tabLogs:
		return "Logs"
	case tabStats:
		return "Stats"
	case tabEnv:
		return "Env"
	case tabPorts:
		return "Ports"
	case tabMounts:
		return "Mounts"
	case tabHealth:
		return "Health"
	case tabTop:
		return "Top"
	case tabInspect:
		return "Inspect"
	default:
		return "?"
	}
}

func hasTab(kind resourceKind, tab mainTab) bool {
	for _, t := range tabsFor(kind) {
		if t == tab {
			return true
		}
	}
	return false
}

func (m Model) activeTabs() []mainTab {
	return tabsFor(m.active)
}

func (m Model) activeMainTab() mainTab {
	tabs := m.activeTabs()
	idx := m.tabIndex[m.active]
	if idx < 0 || idx >= len(tabs) {
		return tabs[0]
	}
	return tabs[idx]
}

func (m *Model) clampActiveTab() {
	n := len(tabsFor(m.active))
	if n == 0 {
		return
	}
	if m.tabIndex[m.active] >= n {
		m.tabIndex[m.active] = n - 1
	}
	if m.tabIndex[m.active] < 0 {
		m.tabIndex[m.active] = 0
	}
}

// cycleTab moves the active resource's tab selection by delta and returns a
// command to fetch the new tab's content if needed.
func (m *Model) cycleTab(delta int) tea.Cmd {
	tabs := tabsFor(m.active)
	if len(tabs) <= 1 {
		return nil
	}
	idx := m.tabIndex[m.active] + delta
	idx %= len(tabs)
	if idx < 0 {
		idx += len(tabs)
	}
	m.tabIndex[m.active] = idx
	m.bufferKind = bufNone
	m.panelOffset = 0
	m.statusLine = "tab: " + tabs[idx].label()
	return m.ensureMainPanelCmd()
}

// activateTab selects a specific tab for the active resource (falling back to
// the first tab when unavailable) and returns a fetch command if needed.
func (m *Model) activateTab(tab mainTab) tea.Cmd {
	tabs := tabsFor(m.active)
	target := 0
	for i, t := range tabs {
		if t == tab {
			target = i
			break
		}
	}
	m.tabIndex[m.active] = target
	m.bufferKind = bufNone
	m.panelOffset = 0
	m.statusLine = strings.ToLower(tabs[target].label())
	return m.ensureMainPanelCmd()
}

// fetchTargetName is the selected resource id relevant to fetched tabs.
func (m Model) fetchTargetName() string {
	switch m.active {
	case resourceContainers:
		if c, ok := m.selectedContainer(); ok {
			return c.Name()
		}
	case resourceServices:
		// Logs/Inspect for a service operate on its backing container.
		if s, ok := m.selectedService(); ok {
			if c, ok := m.serviceContainer(s); ok {
				return c.Name()
			}
		}
	case resourceImages:
		if i, ok := m.selectedImage(); ok {
			return i.Name()
		}
	case resourceVolumes:
		if v, ok := m.selectedVolume(); ok {
			return v.Name()
		}
	case resourceNetworks:
		if n, ok := m.selectedNetwork(); ok {
			return n.Name()
		}
	case resourceMachines:
		if mm, ok := m.selectedMachine(); ok {
			return mm.Name()
		}
	case resourceSystem:
		if m.systemMatchesFilter() {
			return "system"
		}
	}
	return ""
}

// currentBufferKey identifies the data a fetched tab needs, or "" if the
// selection cannot supply it.
func (m Model) currentBufferKey(tab mainTab) string {
	name := m.fetchTargetName()
	if name == "" {
		return ""
	}
	return fmt.Sprintf("%d|%s|%d", m.active, name, int(tab))
}

// ensureBufferCmd refetches the active fetched tab's content if it is stale.
func (m Model) ensureBufferCmd() tea.Cmd {
	if m.bufferKind == bufOutput {
		return nil
	}
	tab := m.activeMainTab()
	if !tab.fetched() {
		return nil
	}
	key := m.currentBufferKey(tab)
	if key == "" {
		return nil
	}
	if m.bufferKind == bufTab && m.bufferTab == tab && m.bufferKey == key {
		return nil
	}
	return m.fetchTabCmd(tab, key)
}

// forceFetchActiveTabCmd refetches the active fetched tab regardless of
// freshness; used by manual and automatic refresh.
func (m Model) forceFetchActiveTabCmd() tea.Cmd {
	if m.bufferKind == bufOutput {
		return nil
	}
	tab := m.activeMainTab()
	if !tab.fetched() {
		return nil
	}
	key := m.currentBufferKey(tab)
	if key == "" {
		return nil
	}
	return m.fetchTabCmd(tab, key)
}

func (m Model) fetchTabCmd(tab mainTab, key string) tea.Cmd {
	active := m.active
	name := m.fetchTargetName()
	switch tab {
	case tabInspect:
		return func() tea.Msg {
			title, body, err := m.inspectFetch(active, name)
			return tabFetchedMsg{tab: tab, key: key, title: title, body: body, err: err}
		}
	case tabTop:
		return func() tea.Msg {
			title, body, err := m.topFetch(name)
			return tabFetchedMsg{tab: tab, key: key, title: title, body: body, err: err}
		}
	}
	return nil
}

func (m Model) inspectFetch(kind resourceKind, name string) (string, string, error) {
	ctx := context.Background()
	switch kind {
	case resourceContainers, resourceServices:
		body, err := m.client.InspectContainer(ctx, name)
		return "Inspect " + name, body, err
	case resourceImages:
		body, err := m.client.InspectImage(ctx, name)
		return "Inspect " + name, body, err
	case resourceVolumes:
		body, err := m.client.InspectVolume(ctx, name)
		return "Inspect " + name, body, err
	case resourceNetworks:
		body, err := m.client.InspectNetwork(ctx, name)
		return "Inspect " + name, body, err
	case resourceMachines:
		body, err := m.client.InspectMachine(ctx, name)
		return "Inspect " + name, body, err
	}
	return "Inspect", "", nil
}

func (m Model) topFetch(name string) (string, string, error) {
	body, err := m.client.Top(context.Background(), name)
	if strings.TrimSpace(body) == "" && err == nil {
		body = "No process information returned."
	}
	return "Top " + name, body, err
}

// logTail returns the number of log lines to request when starting a follow
// stream.
func (m Model) logTail() int {
	if m.logsTail > 0 {
		return m.logsTail
	}
	return 200
}

// logSince returns the window passed to system logs.
func (m Model) logSince() string {
	if strings.TrimSpace(m.logsSince) != "" {
		return m.logsSince
	}
	return "5m"
}

// tabContent renders the active tab's (title, body) when no output buffer
// overrides the panel.
func (m Model) tabContent(now time.Time) (string, string) {
	tab := m.activeMainTab()
	switch tab {
	case tabConfig:
		return m.configTabContent(now)
	case tabEnv:
		return m.envTabContent()
	case tabStats:
		return m.statsTabContent()
	case tabPorts:
		return m.portsTabContent()
	case tabMounts:
		return m.mountsTabContent()
	case tabHealth:
		return m.healthTabContent(now)
	case tabLogs:
		return m.logsTabContent()
	case tabInspect, tabTop:
		if m.bufferKind == bufTab && m.bufferTab == tab {
			title := m.panelTitle
			if title == "" {
				title = tab.label()
			}
			return title, m.panelBody
		}
		// No fetch target (e.g. a service that has no container yet) would
		// otherwise sit on "Loading…" forever, so explain the empty state.
		if m.fetchTargetName() == "" {
			if m.active == resourceServices {
				return tab.label(), "No container yet — press u to bring the service up."
			}
			return tab.label(), "Nothing to " + strings.ToLower(tab.label()) + "."
		}
		return tab.label(), "Loading " + strings.ToLower(tab.label()) + "…"
	}
	return tab.label(), ""
}

// configTabContent renders the per-resource detail view (the former
// panelDetails view).
func (m Model) configTabContent(now time.Time) (string, string) {
	switch m.active {
	case resourceContainers:
		container, ok := m.selectedContainer()
		if !ok {
			return "Config", "No container selected."
		}
		return "Config " + container.Name(), strings.Join(container.DetailLines(now), "\n")
	case resourceServices:
		service, ok := m.selectedService()
		if !ok {
			return "Config", m.emptyServiceMessage()
		}
		return "Config " + service.Name, strings.Join(m.serviceDetailLines(service), "\n")
	case resourceImages:
		image, ok := m.selectedImage()
		if !ok {
			return "Config", "No image selected."
		}
		return "Config " + image.Name(), strings.Join(image.DetailLines(now), "\n")
	case resourceBuilder:
		if !m.builderMatchesFilter() {
			return "Config", "No builder selected."
		}
		return "Config builder", strings.Join(m.builder.DetailLines(), "\n")
	case resourceVolumes:
		volume, ok := m.selectedVolume()
		if !ok {
			return "Config", "No volume selected."
		}
		return "Config " + volume.Name(), strings.Join(volume.DetailLines(now), "\n")
	case resourceNetworks:
		network, ok := m.selectedNetwork()
		if !ok {
			return "Config", "No network selected."
		}
		return "Config " + network.Name(), strings.Join(network.DetailLines(now), "\n")
	case resourceMachines:
		machine, ok := m.selectedMachine()
		if !ok {
			return "Config", "No machine selected."
		}
		return "Config " + machine.Name(), strings.Join(machine.DetailLines(now), "\n")
	case resourceRegistries:
		registry, ok := m.selectedRegistry()
		if !ok {
			return "Config", "No registry selected."
		}
		return "Config " + registry.Name(), strings.Join(registry.DetailLines(), "\n")
	case resourceSystem:
		if !m.systemMatchesFilter() {
			return "Config", "No system selected."
		}
		return "Config system", strings.Join(m.systemDetailLines(), "\n")
	default:
		return "Config", ""
	}
}

// systemDetailLines renders the System pane detail, appending Apple-native DNS
// domains and subsystem properties when the CLI reports them.
func (m Model) systemDetailLines() []string {
	lines := m.system.DetailLines(m.systemUsage, m.systemVersions)
	if len(m.systemDNS) > 0 {
		lines = append(lines, "", "DNS domains")
		for _, domain := range m.systemDNS {
			lines = append(lines, "  "+domain.Display())
		}
	}
	if len(m.systemProperties) > 0 {
		lines = append(lines, "", "Properties")
		for _, property := range m.systemProperties {
			lines = append(lines, "  "+property.Display())
		}
	}
	return lines
}

// serviceDetailLines renders a Compose service: its definition plus the state
// of the container backing it.
func (m Model) serviceDetailLines(service compose.Service) []string {
	image := service.Image
	if image == "" && service.Build != "" {
		image = "build " + service.Build
	}
	if image == "" {
		image = "—"
	}
	lines := []string{
		"Service",
		"  Name:      " + service.Name,
		"  Image:     " + image,
		"  Container: " + m.project.ContainerNameFor(service),
		"  State:     " + m.serviceState(service),
	}
	if len(service.Ports) > 0 {
		lines = append(lines, "  Ports:     "+strings.Join(service.Ports, ", "))
	}
	if len(service.Networks) > 0 {
		lines = append(lines, "  Networks:  "+strings.Join(service.Networks, ", "))
	}
	if len(service.DependsOn) > 0 {
		lines = append(lines, "  Depends:   "+strings.Join(service.DependsOn, ", "))
	}
	if len(service.Environment) > 0 {
		lines = append(lines, "", "Environment")
		for _, env := range service.Environment {
			lines = append(lines, "  "+env)
		}
	}
	if len(service.Volumes) > 0 {
		lines = append(lines, "", "Volumes")
		for _, volume := range service.Volumes {
			lines = append(lines, "  "+volume)
		}
	}
	if len(service.Command) > 0 {
		lines = append(lines, "", "Command", "  "+strings.Join(service.Command, " "))
	}
	lines = append(lines, "", "Project: "+m.project.Name)
	if m.project.File != "" {
		lines = append(lines, "File:    "+m.project.File)
	}
	return lines
}

func (m Model) envTabContent() (string, string) {
	container, ok := m.selectedContainer()
	if !ok {
		return "Env", "No container selected."
	}
	env := container.Configuration.InitProcess.Environment
	if len(env) == 0 {
		return "Env " + container.Name(), "No environment variables."
	}
	lines := make([]string, 0, len(env))
	for _, entry := range env {
		lines = append(lines, "  "+entry)
	}
	sort.Strings(lines)
	return "Env " + container.Name(), strings.Join(lines, "\n")
}

// handleTabFetched stores a fetched tab result, dropping it if the user has
// since navigated away or changed selection.
func (m Model) handleTabFetched(msg tabFetchedMsg) (tea.Model, tea.Cmd) {
	if m.bufferKind == bufOutput {
		return m, nil
	}
	tab := m.activeMainTab()
	if msg.tab != tab || msg.key != m.currentBufferKey(tab) {
		// Stale result: leave any in-flight busy indicator untouched.
		return m, nil
	}
	m.busy = ""
	if msg.err != nil {
		m.err = msg.err
		m.statusLine = msg.err.Error()
		return m, nil
	}
	m.bufferKind = bufTab
	m.bufferTab = msg.tab
	m.bufferKey = msg.key
	m.panelTitle = msg.title
	m.panelBody = msg.body
	m.clampPanelOffset()
	m.statusLine = "loaded " + strings.ToLower(msg.title)
	return m, nil
}

func (m Model) statsTabContent() (string, string) {
	container, ok := m.selectedContainer()
	if !ok {
		return "Stats", "No container selected."
	}
	name := container.Name()
	width := m.panelContentWidth()
	const graphHeight = 6
	samples := m.statHistoryForContainer(name)

	var lines []string
	addGraph := func(caption string, scaleMax float64, format func(float64) string, valueFor func(statHistorySample) (float64, bool)) {
		values, ok := historyValues(samples, valueFor)
		if !ok {
			return
		}
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, graphSection(caption, values, width, graphHeight, scaleMax, format)...)
	}
	addGraph("CPU %", 100, formatPercentValue, func(s statHistorySample) (float64, bool) {
		if s.hasCPU {
			return s.cpuPercent, true
		}
		return 0, false
	})
	addGraph("Memory", 0, formatBytesValue, func(s statHistorySample) (float64, bool) {
		if s.hasMemory {
			return s.memoryBytes, true
		}
		return 0, false
	})
	addGraph("Network", 0, formatRateValue, func(s statHistorySample) (float64, bool) {
		if s.hasNetwork {
			return s.networkRate, true
		}
		return 0, false
	})
	addGraph("Block IO", 0, formatRateValue, func(s statHistorySample) (float64, bool) {
		if s.hasBlock {
			return s.blockRate, true
		}
		return 0, false
	})

	if summary := m.currentStatSummary(name); len(summary) > 0 {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "Current")
		lines = append(lines, summary...)
	}

	if len(lines) == 0 {
		return "Stats " + name, "No stats available (is the container running?)."
	}
	return "Stats " + name, strings.Join(lines, "\n")
}

// portsTabContent lists a selected container's published port mappings, built
// from already-loaded configuration (no fetch needed).
func (m Model) portsTabContent() (string, string) {
	container, ok := m.selectedContainer()
	if !ok {
		return "Ports", "No container selected."
	}
	ports := container.Configuration.PublishedPorts
	if len(ports) == 0 {
		return "Ports " + container.Name(), "No published ports."
	}
	lines := make([]string, 0, len(ports))
	for _, p := range ports {
		host := p.HostAddress
		if host == "" {
			host = "0.0.0.0"
		}
		proto := p.Proto
		if proto == "" {
			proto = "tcp"
		}
		lines = append(lines, fmt.Sprintf("  %s:%d -> %d/%s", host, p.HostPort, p.ContainerPort, proto))
	}
	return "Ports " + container.Name(), strings.Join(lines, "\n")
}

// mountsTabContent lists a selected container's mounts (source -> destination,
// with rw/ro), from already-loaded configuration.
func (m Model) mountsTabContent() (string, string) {
	container, ok := m.selectedContainer()
	if !ok {
		return "Mounts", "No container selected."
	}
	mounts := container.Configuration.Mounts
	if len(mounts) == 0 {
		return "Mounts " + container.Name(), "No mounts."
	}
	lines := make([]string, 0, len(mounts))
	for _, mt := range mounts {
		mode := "rw"
		for _, opt := range mt.Options {
			if opt == "ro" || opt == "readonly" {
				mode = "ro"
			}
		}
		src := mt.Source
		if src == "" {
			src = "-"
		}
		lines = append(lines, fmt.Sprintf("  %s -> %s  (%s)", src, emptyDash(mt.Destination), mode))
	}
	return "Mounts " + container.Name(), strings.Join(lines, "\n")
}

// healthTabContent shows a selected container's status overview. Apple's
// container runtime has no docker-style health checks, so this reports the real
// state, uptime, and resource attachment rather than inventing a check result.
func (m Model) healthTabContent(now time.Time) (string, string) {
	container, ok := m.selectedContainer()
	if !ok {
		return "Health", "No container selected."
	}
	lines := []string{
		"  State:    " + container.State(),
		"  Started:  " + emptyDash(container.StartedAgo(now)),
		"  Image:    " + emptyDash(container.ImageName()),
		"  Platform: " + container.Platform(),
		"  Runtime:  " + emptyDash(container.Configuration.RuntimeHandler),
		fmt.Sprintf("  Ports:    %d published", len(container.Configuration.PublishedPorts)),
		fmt.Sprintf("  Mounts:   %d", len(container.Configuration.Mounts)),
		fmt.Sprintf("  Networks: %d attached", len(container.Status.Networks)),
	}
	return "Health " + container.Name(), strings.Join(lines, "\n")
}

// currentStatSummary returns the current-value summary lines for a container's
// most recent stats sample, folding in the live (derived) CPU% when available.
func (m Model) currentStatSummary(name string) []string {
	derived, hasDerived := m.derivedCPUPercent(name)
	for _, stat := range m.stats {
		if statMatches(stat, name) {
			if hasDerived {
				stat = withDerivedCPU(stat, derived)
			}
			return stat.SummaryLines()
		}
	}
	return nil
}

// panelContentWidth estimates the usable text width of the main panel.
func (m Model) panelContentWidth() int {
	if m.width <= 0 {
		return 40
	}
	w := m.width - m.sidebarWidth() - 4
	if w < 12 {
		w = 12
	}
	return w
}
