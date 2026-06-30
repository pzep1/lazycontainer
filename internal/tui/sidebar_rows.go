package tui

import (
	"fmt"
	"strconv"
	"strings"
)

func (m Model) countLabel(filtered int, total int) string {
	if activeFilter(m.filter) == "" || filtered == total {
		return strconv.Itoa(total)
	}
	return fmt.Sprintf("%d/%d", filtered, total)
}

// imageInUseCount is how many loaded containers run from the named image.
func (m Model) imageInUseCount(name string) int {
	n := 0
	for _, c := range m.containers {
		if c.ImageName() == name {
			n++
		}
	}
	return n
}

// volumeInUseCount is how many loaded containers mount the named volume.
func (m Model) volumeInUseCount(name string) int {
	n := 0
	for _, c := range m.containers {
		for _, mount := range c.Configuration.Mounts {
			if mount.Source == name {
				n++
				break
			}
		}
	}
	return n
}

// networkInUseCount is how many loaded containers attach to the named network.
func (m Model) networkInUseCount(name string) int {
	n := 0
	for _, c := range m.containers {
		for _, net := range c.Configuration.Networks {
			if net.Network == name {
				n++
				break
			}
		}
	}
	return n
}

// trimImageRef shortens an image reference for dense rows by dropping the
// registry host / namespace, keeping the final "name:tag" segment.
func trimImageRef(ref string) string {
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		return ref[i+1:]
	}
	return ref
}

// inUseBadge is a compact, fixed-width in-use marker for sidebar rows: "●N" when
// something references the resource, "·" when nothing does. It stays plain text
// so it survives fitColumns' width math (the row is colored as a whole).
func inUseBadge(count int) string {
	if count > 0 {
		return fmt.Sprintf("●%d", count)
	}
	return "·"
}

// styleSidebarRow highlights a list row. The selected row is rendered with the
// bright background when its section is focused, or bold-only when it is not, so
// every panel shows its current item without the focused panel's emphasis.
func styleSidebarRow(line string, width int, selected bool, focused bool) string {
	if !selected {
		return line
	}
	if focused {
		return selectedStyle.Width(width).Render(truncate(line, width))
	}
	return cursorRestStyle.Render(truncate(line, width))
}

func (m Model) renderContainerList(width int, height int) []string {
	indexes := m.filteredContainerIndexes()
	if len(indexes) == 0 {
		return []string{mutedStyle.Render(m.emptyListMessage("containers"))}
	}
	focused := m.active == resourceContainers
	rows := make([]string, 0, height)
	start := visibleStart(m.containerCursor, height, len(indexes))
	end := start + height
	if end > len(indexes) {
		end = len(indexes)
	}
	now := effectiveNow(m.lastUpdated)
	for idx := start; idx < end; idx++ {
		container := m.containers[indexes[idx]]
		// On a wide enough sidebar, surface the image ref alongside the name;
		// on narrow terminals it degrades to just the name.
		left := truncate(container.Name(), 22)
		if width >= 62 {
			if img := trimImageRef(container.ImageName()); img != "" {
				left = truncate(container.Name(), 22) + "  " + truncate(img, 24)
			}
		}
		meta := padRight(container.State(), 8) + " " + padLeft(container.CreatedAgo(now), 10)
		if summary := m.statListSummary(container.Name()); summary != "" {
			meta = padRight(container.State(), 8) + " " + summary
		}
		line := fitColumns(left, meta, width)
		if idx == m.containerCursor {
			line = styleSidebarRow(line, width, true, focused)
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
	focused := m.active == resourceImages
	rows := make([]string, 0, height)
	start := visibleStart(m.imageCursor, height, len(indexes))
	end := start + height
	if end > len(indexes) {
		end = len(indexes)
	}
	for idx := start; idx < end; idx++ {
		image := m.images[indexes[idx]]
		name := truncate(image.Name(), 34)
		meta := padLeft(image.Size(), 9) + "  " + padLeft(inUseBadge(m.imageInUseCount(image.Name())), 4)
		line := fitColumns(name, meta, width)
		rows = append(rows, styleSidebarRow(line, width, idx == m.imageCursor, focused))
	}
	return rows
}

func (m Model) renderBuilderList(width int, height int) []string {
	if !m.builderMatchesFilter() || height < 1 {
		return nil
	}
	line := fitColumns(m.builder.Name(), m.builder.State(), width)
	return []string{styleSidebarRow(line, width, true, m.active == resourceBuilder)}
}

func (m Model) renderVolumeList(width int, height int) []string {
	indexes := m.filteredVolumeIndexes()
	if len(indexes) == 0 {
		return []string{mutedStyle.Render(m.emptyListMessage("volumes"))}
	}
	focused := m.active == resourceVolumes
	rows := make([]string, 0, height)
	start := visibleStart(m.volumeCursor, height, len(indexes))
	end := start + height
	if end > len(indexes) {
		end = len(indexes)
	}
	for idx := start; idx < end; idx++ {
		volume := m.volumes[indexes[idx]]
		name := truncate(volume.Name(), 34)
		meta := padLeft(volume.Size(), 9) + "  " + padLeft(inUseBadge(m.volumeInUseCount(volume.Name())), 4)
		line := fitColumns(name, meta, width)
		rows = append(rows, styleSidebarRow(line, width, idx == m.volumeCursor, focused))
	}
	return rows
}

func (m Model) renderNetworkList(width int, height int) []string {
	indexes := m.filteredNetworkIndexes()
	if len(indexes) == 0 {
		return []string{mutedStyle.Render(m.emptyListMessage("networks"))}
	}
	focused := m.active == resourceNetworks
	rows := make([]string, 0, height)
	start := visibleStart(m.networkCursor, height, len(indexes))
	end := start + height
	if end > len(indexes) {
		end = len(indexes)
	}
	for idx := start; idx < end; idx++ {
		network := m.networks[indexes[idx]]
		name := truncate(network.Name(), 34)
		meta := padRight(emptyDash(network.Configuration.Mode), 6) + " " + padLeft(inUseBadge(m.networkInUseCount(network.Name())), 4)
		line := fitColumns(name, meta, width)
		rows = append(rows, styleSidebarRow(line, width, idx == m.networkCursor, focused))
	}
	return rows
}

func (m Model) renderMachineList(width int, height int) []string {
	indexes := m.filteredMachineIndexes()
	if len(indexes) == 0 {
		return []string{mutedStyle.Render(m.emptyListMessage("machines"))}
	}
	focused := m.active == resourceMachines
	rows := make([]string, 0, height)
	start := visibleStart(m.machineCursor, height, len(indexes))
	end := start + height
	if end > len(indexes) {
		end = len(indexes)
	}
	now := effectiveNow(m.lastUpdated)
	for idx := start; idx < end; idx++ {
		machine := m.machines[indexes[idx]]
		name := truncate(machine.Name(), 26)
		meta := padRight(machine.State(), 8) + " " + padLeft(machine.CreatedAgo(now), 10)
		if machine.Default {
			name = "* " + name
		}
		line := fitColumns(name, meta, width)
		if idx == m.machineCursor {
			line = styleSidebarRow(line, width, true, focused)
		} else if machine.State() == "running" {
			line = runningStyle.Render(line)
		}
		rows = append(rows, line)
	}
	return rows
}

func (m Model) renderRegistryList(width int, height int) []string {
	indexes := m.filteredRegistryIndexes()
	if len(indexes) == 0 {
		return []string{mutedStyle.Render(m.emptyListMessage("registries"))}
	}
	focused := m.active == resourceRegistries
	rows := make([]string, 0, height)
	start := visibleStart(m.registryCursor, height, len(indexes))
	end := start + height
	if end > len(indexes) {
		end = len(indexes)
	}
	for idx := start; idx < end; idx++ {
		registry := m.registries[indexes[idx]]
		name := truncate(registry.Name(), 34)
		meta := emptyDash(registry.User())
		line := fitColumns(name, meta, width)
		rows = append(rows, styleSidebarRow(line, width, idx == m.registryCursor, focused))
	}
	return rows
}

func (m Model) renderSystemList(width int, height int) []string {
	if !m.systemMatchesFilter() || height < 1 {
		return nil
	}
	meta := fmt.Sprintf("%s  %s used", emptyDash(m.system.Status), padLeft(m.systemUsage.TotalSize(), 9))
	line := fitColumns("system", meta, width)
	return []string{styleSidebarRow(line, width, true, m.active == resourceSystem)}
}

func (m Model) emptyListMessage(kind string) string {
	if activeFilter(m.filter) == "" {
		return "No " + kind + " found."
	}
	return "No " + kind + " match " + fmt.Sprintf("%q.", m.filter)
}
