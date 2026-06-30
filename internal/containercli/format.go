package containercli

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (s SystemStatus) DetailLines(usage SystemDiskUsage, versions []SystemVersion) []string {
	lines := []string{
		"System",
		"  Status:      " + emptyDash(s.Status),
		"  API server:  " + emptyDash(s.APIServerAppName),
		"  API build:   " + emptyDash(s.APIServerBuild),
		"  API commit:  " + emptyDash(shortDigest(s.APIServerCommit)),
		"  API version: " + emptyDash(s.APIServerVersion),
		"  App root:    " + emptyDash(s.AppRoot),
		"  Install:     " + emptyDash(s.InstallRoot),
		"",
		"Disk usage",
	}
	lines = append(lines, usage.DetailLines()...)
	if len(versions) > 0 {
		lines = append(lines, "", "Versions")
		for _, version := range versions {
			lines = append(lines, version.DetailLine())
		}
	}
	return lines
}

func (d SystemDNSDomain) Display() string {
	if name := strings.TrimSpace(d.Name); name != "" {
		return name
	}
	if len(d.Raw) > 0 {
		return strings.Join(sortedMapLines(d.Raw), ", ")
	}
	return "-"
}

func (p SystemProperty) Display() string {
	id := strings.TrimSpace(p.ID)
	value := strings.TrimSpace(p.Value)
	switch {
	case id != "" && value != "":
		return id + ": " + value
	case id != "":
		return id
	case len(p.Raw) > 0:
		return strings.Join(sortedMapLines(p.Raw), ", ")
	default:
		return "-"
	}
}

func (u SystemDiskUsage) DetailLines() []string {
	return []string{
		diskUsageLine("  Containers", u.Containers),
		diskUsageLine("  Images", u.Images),
		diskUsageLine("  Volumes", u.Volumes),
	}
}

func (u SystemDiskUsage) TotalSize() string {
	return FormatBytes(u.Containers.SizeInBytes + u.Images.SizeInBytes + u.Volumes.SizeInBytes)
}

func (u SystemDiskUsage) TotalReclaimable() string {
	return FormatBytes(u.Containers.Reclaimable + u.Images.Reclaimable + u.Volumes.Reclaimable)
}

func diskUsageLine(name string, category DiskUsageCategory) string {
	return fmt.Sprintf("%s: %d total, %d active, %s used, %s reclaimable", name, category.Total, category.Active, FormatBytes(category.SizeInBytes), FormatBytes(category.Reclaimable))
}

func (v SystemVersion) DetailLine() string {
	name := emptyDash(v.AppName)
	version := emptyDash(v.Version)
	build := emptyDash(v.BuildType)
	commit := emptyDash(shortDigest(v.Commit))
	return fmt.Sprintf("  %s: %s (%s, %s)", name, version, build, commit)
}

func (c Container) Name() string {
	if c.ID != "" {
		return c.ID
	}
	return c.Configuration.ID
}

func (c Container) ImageName() string {
	return c.Configuration.Image.Reference
}

func (c Container) State() string {
	if c.Status.State == "" {
		return "unknown"
	}
	return c.Status.State
}

func (c Container) Ports() string {
	if len(c.Configuration.PublishedPorts) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(c.Configuration.PublishedPorts))
	for _, port := range c.Configuration.PublishedPorts {
		host := port.HostAddress
		if host == "" {
			host = "0.0.0.0"
		}
		proto := port.Proto
		if proto == "" {
			proto = "tcp"
		}
		parts = append(parts, fmt.Sprintf("%s:%d->%d/%s", host, port.HostPort, port.ContainerPort, proto))
	}
	return strings.Join(parts, ", ")
}

func (c Container) Platform() string {
	return formatPlatform(c.Configuration.Platform)
}

// FirstPublishedURL returns an http URL for the container's first published
// port, assuming an HTTP service (matching lazydocker's open-in-browser).
func (c Container) FirstPublishedURL() (string, bool) {
	for _, port := range c.Configuration.PublishedPorts {
		if port.HostPort <= 0 {
			continue
		}
		host := port.HostAddress
		if host == "" || host == "0.0.0.0" || host == "::" {
			host = "localhost"
		}
		return fmt.Sprintf("http://%s:%d", host, port.HostPort), true
	}
	return "", false
}

func (c Container) Memory() string {
	return FormatBytes(c.Configuration.Resources.MemoryInBytes)
}

func (c Container) CreatedAgo(now time.Time) string {
	return relativeTime(c.Configuration.CreationDate, now)
}

func (c Container) StartedAgo(now time.Time) string {
	return relativeTime(c.Status.StartedDate, now)
}

func (c Container) DetailLines(now time.Time) []string {
	lines := []string{
		"Container",
		"  ID:       " + c.Name(),
		"  State:    " + c.State(),
		"  Image:    " + emptyDash(c.ImageName()),
		"  Platform: " + emptyDash(c.Platform()),
		"  Created:  " + c.CreatedAgo(now),
		"  Started:  " + c.StartedAgo(now),
		"  Ports:    " + c.Ports(),
		"  Memory:   " + c.Memory(),
		"  Runtime:  " + emptyDash(c.Configuration.RuntimeHandler),
		"",
		"Process",
		"  Exec:     " + emptyDash(c.Configuration.InitProcess.Executable),
		"  Args:     " + emptyDash(strings.Join(c.Configuration.InitProcess.Arguments, " ")),
		"  Workdir:  " + emptyDash(c.Configuration.InitProcess.WorkingDirectory),
	}
	if len(c.Configuration.Mounts) > 0 {
		lines = append(lines, "", "Mounts")
		for _, mount := range c.Configuration.Mounts {
			source := mount.Source
			if source != "" {
				source = filepath.Base(source)
			}
			lines = append(lines, fmt.Sprintf("  %s -> %s", emptyDash(source), emptyDash(mount.Destination)))
		}
	}
	return lines
}

func (i Image) Name() string {
	if i.Configuration.Name != "" {
		return i.Configuration.Name
	}
	return i.ID
}

func (i Image) Digest() string {
	if i.Configuration.Descriptor.Digest != "" {
		return i.Configuration.Descriptor.Digest
	}
	if i.ID != "" {
		return "sha256:" + i.ID
	}
	return ""
}

func (i Image) Size() string {
	var total int64
	for _, variant := range i.Variants {
		total += variant.Size
	}
	if total == 0 {
		total = i.Configuration.Descriptor.Size
	}
	return FormatBytes(total)
}

func (i Image) Platforms() string {
	if len(i.Variants) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(i.Variants))
	seen := map[string]struct{}{}
	for _, variant := range i.Variants {
		platform := formatPlatform(variant.Platform)
		if platform == "" || platform == "-" {
			continue
		}
		if _, ok := seen[platform]; ok {
			continue
		}
		seen[platform] = struct{}{}
		parts = append(parts, platform)
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ", ")
}

func (i Image) CreatedAgo(now time.Time) string {
	return relativeTime(i.Configuration.CreationDate, now)
}

func (i Image) DetailLines(now time.Time) []string {
	lines := []string{
		"Image",
		"  Name:      " + i.Name(),
		"  Digest:    " + emptyDash(i.Digest()),
		"  Created:   " + i.CreatedAgo(now),
		"  Size:      " + i.Size(),
		"  Platforms: " + i.Platforms(),
		"",
		"Variants",
	}
	if len(i.Variants) == 0 {
		return append(lines, "  -")
	}
	for _, variant := range i.Variants {
		lines = append(lines, fmt.Sprintf("  %s  %s  %s", formatPlatform(variant.Platform), FormatBytes(variant.Size), shortDigest(variant.Digest)))
	}
	if historyLines := i.LayerHistoryLines(); len(historyLines) > 0 {
		lines = append(lines, "", "Layer history")
		lines = append(lines, historyLines...)
	}
	return lines
}

func (i Image) LayerHistoryLines() []string {
	var lines []string
	for _, variant := range i.Variants {
		if len(variant.Config.History) == 0 && len(variant.Config.RootFS.DiffIDs) == 0 {
			continue
		}
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		platform := formatPlatform(variant.Platform)
		if platform == "-" {
			platform = formatPlatform(Platform{OS: variant.Config.OS, Architecture: variant.Config.Architecture, Variant: variant.Config.Variant})
		}
		lines = append(lines, fmt.Sprintf("  %s  %d history entries  %d filesystem layers", emptyDash(platform), len(variant.Config.History), len(variant.Config.RootFS.DiffIDs)))
		layerIndex := 0
		for _, entry := range variant.Config.History {
			layerID := "metadata"
			if !entry.EmptyLayer && layerIndex < len(variant.Config.RootFS.DiffIDs) {
				layerID = shortDigest(variant.Config.RootFS.DiffIDs[layerIndex])
				layerIndex++
			}
			command := strings.TrimSpace(entry.CreatedBy)
			if command == "" {
				command = entry.Comment
			}
			lines = append(lines, fmt.Sprintf("    %s  %s", layerID, emptyDash(command)))
		}
		for ; layerIndex < len(variant.Config.RootFS.DiffIDs); layerIndex++ {
			lines = append(lines, fmt.Sprintf("    %s  filesystem layer", shortDigest(variant.Config.RootFS.DiffIDs[layerIndex])))
		}
	}
	return lines
}

func (v Volume) Name() string {
	if v.Configuration.Name != "" {
		return v.Configuration.Name
	}
	return v.ID
}

func (v Volume) Size() string {
	return FormatBytes(v.Configuration.SizeInBytes)
}

func (v Volume) CreatedAgo(now time.Time) string {
	return relativeTime(v.Configuration.CreationDate, now)
}

func (v Volume) DetailLines(now time.Time) []string {
	lines := []string{
		"Volume",
		"  Name:    " + v.Name(),
		"  Driver:  " + emptyDash(v.Configuration.Driver),
		"  Format:  " + emptyDash(v.Configuration.Format),
		"  Size:    " + v.Size(),
		"  Created: " + v.CreatedAgo(now),
		"  Source:  " + emptyDash(v.Configuration.Source),
	}
	if len(v.Configuration.Options) > 0 {
		lines = append(lines, "", "Options")
		for _, entry := range sortedMapLines(v.Configuration.Options) {
			lines = append(lines, "  "+entry)
		}
	}
	if len(v.Configuration.Labels) > 0 {
		lines = append(lines, "", "Labels")
		for _, entry := range sortedMapLines(v.Configuration.Labels) {
			lines = append(lines, "  "+entry)
		}
	}
	return lines
}

func (n NetworkResource) Name() string {
	if n.Configuration.Name != "" {
		return n.Configuration.Name
	}
	return n.ID
}

func (n NetworkResource) CreatedAgo(now time.Time) string {
	return relativeTime(n.Configuration.CreationDate, now)
}

func (n NetworkResource) DetailLines(now time.Time) []string {
	lines := []string{
		"Network",
		"  Name:       " + n.Name(),
		"  Mode:       " + emptyDash(n.Configuration.Mode),
		"  Plugin:     " + emptyDash(n.Configuration.Plugin),
		"  Created:    " + n.CreatedAgo(now),
		"  IPv4 GW:    " + emptyDash(n.Status.IPv4Gateway),
		"  IPv4 CIDR:  " + emptyDash(n.Status.IPv4Subnet),
		"  IPv6 CIDR:  " + emptyDash(n.Status.IPv6Subnet),
	}
	if len(n.Configuration.Options) > 0 {
		lines = append(lines, "", "Options")
		for _, entry := range sortedMapLines(n.Configuration.Options) {
			lines = append(lines, "  "+entry)
		}
	}
	if len(n.Configuration.Labels) > 0 {
		lines = append(lines, "", "Labels")
		for _, entry := range sortedMapLines(n.Configuration.Labels) {
			lines = append(lines, "  "+entry)
		}
	}
	return lines
}

func (m Machine) Name() string {
	return firstNonEmpty(
		m.ID,
		stringFromMap(m.Configuration, "name"),
		stringFromMap(m.Raw, "name"),
		stringFromMap(m.Raw, "id"),
	)
}

func (m Machine) State() string {
	statusMap, _ := m.Status.(map[string]any)
	statusString, _ := m.Status.(string)
	return firstNonEmpty(
		statusString,
		stringFromMap(statusMap, "state"),
		stringFromMap(statusMap, "status"),
		stringFromMap(m.Raw, "state"),
		stringFromMap(m.Raw, "status"),
		"unknown",
	)
}

func (m Machine) Image() string {
	return firstNonEmpty(
		stringFromNestedMap(m.Configuration, "image", "reference"),
		stringFromNestedMap(m.Configuration, "image", "name"),
		stringFromNestedMap(m.Raw, "image", "reference"),
		stringFromNestedMap(m.Raw, "image", "name"),
		stringFromMap(m.Configuration, "image"),
		stringFromMap(m.Raw, "image"),
	)
}

func (m Machine) CPUs() string {
	if value, ok := numberFromMap(m.Configuration, "cpus"); ok {
		return fmt.Sprintf("%.0f", value)
	}
	if value, ok := numberFromNestedMap(m.Configuration, "resources", "cpus"); ok {
		return fmt.Sprintf("%.0f", value)
	}
	return "-"
}

func (m Machine) Memory() string {
	if value, ok := numberFromMap(m.Configuration, "memoryInBytes"); ok {
		return FormatBytes(int64(value))
	}
	if value, ok := numberFromNestedMap(m.Configuration, "resources", "memoryInBytes"); ok {
		return FormatBytes(int64(value))
	}
	return "-"
}

func (m Machine) CreatedAgo(now time.Time) string {
	return relativeTime(firstNonEmpty(
		stringFromMap(m.Configuration, "creationDate"),
		stringFromMap(m.Raw, "creationDate"),
	), now)
}

func (m Machine) DetailLines(now time.Time) []string {
	lines := []string{
		"Machine",
		"  ID:       " + m.Name(),
		"  State:    " + m.State(),
		"  Default:  " + boolLabel(m.Default),
		"  Image:    " + emptyDash(m.Image()),
		"  CPUs:     " + m.CPUs(),
		"  Memory:   " + m.Memory(),
		"  Created:  " + m.CreatedAgo(now),
	}
	if len(m.Configuration) > 0 {
		lines = append(lines, "", "Configuration")
		for _, entry := range sortedMapLines(m.Configuration) {
			lines = append(lines, "  "+entry)
		}
	}
	return lines
}

func (r RegistryLogin) Name() string {
	return firstNonEmpty(
		r.Server,
		r.Registry,
		r.Hostname,
		stringFromMap(r.Raw, "server"),
		stringFromMap(r.Raw, "registry"),
		stringFromMap(r.Raw, "hostname"),
		stringFromMap(r.Raw, "host"),
		r.Value,
	)
}

func (r RegistryLogin) User() string {
	return firstNonEmpty(
		r.Username,
		stringFromMap(r.Raw, "username"),
		stringFromMap(r.Raw, "user"),
	)
}

func (r RegistryLogin) RegistryScheme() string {
	return firstNonEmpty(
		r.Scheme,
		stringFromMap(r.Raw, "scheme"),
	)
}

func (r RegistryLogin) DetailLines() []string {
	lines := []string{
		"Registry login",
		"  Server:   " + emptyDash(r.Name()),
		"  Username: " + emptyDash(r.User()),
		"  Scheme:   " + emptyDash(r.RegistryScheme()),
	}
	if len(r.Raw) > 0 {
		lines = append(lines, "", "Raw")
		for _, entry := range sortedMapLines(r.Raw) {
			lines = append(lines, "  "+entry)
		}
	}
	return lines
}

func (b BuilderStatus) Name() string {
	if !b.Present {
		return "builder"
	}
	return firstNonEmpty(
		b.ID,
		b.ContainerID,
		b.NameValue,
		stringFromMap(b.Raw, "id"),
		stringFromMap(b.Raw, "containerID"),
		stringFromMap(b.Raw, "containerId"),
		stringFromMap(b.Raw, "name"),
		b.Value,
		"builder",
	)
}

func (b BuilderStatus) State() string {
	if !b.Present {
		return "not created"
	}
	return firstNonEmpty(
		b.StateValue,
		b.StatusValue,
		stringFromMap(b.Raw, "state"),
		stringFromMap(b.Raw, "status"),
		"unknown",
	)
}

func (b BuilderStatus) CPUs() string {
	if value, ok := numberFromMap(b.Configuration, "cpus"); ok {
		return fmt.Sprintf("%.0f", value)
	}
	if value, ok := numberFromNestedMap(b.Configuration, "resources", "cpus"); ok {
		return fmt.Sprintf("%.0f", value)
	}
	if value, ok := numberFromMap(b.Raw, "cpus"); ok {
		return fmt.Sprintf("%.0f", value)
	}
	if value, ok := numberFromNestedMap(b.Raw, "resources", "cpus"); ok {
		return fmt.Sprintf("%.0f", value)
	}
	return "-"
}

func (b BuilderStatus) Memory() string {
	if value, ok := numberFromMap(b.Configuration, "memoryInBytes"); ok {
		return FormatBytes(int64(value))
	}
	if value, ok := numberFromNestedMap(b.Configuration, "resources", "memoryInBytes"); ok {
		return FormatBytes(int64(value))
	}
	if value, ok := numberFromMap(b.Raw, "memoryInBytes"); ok {
		return FormatBytes(int64(value))
	}
	if value, ok := numberFromNestedMap(b.Raw, "resources", "memoryInBytes"); ok {
		return FormatBytes(int64(value))
	}
	return "-"
}

func (b BuilderStatus) DetailLines() []string {
	lines := []string{
		"Builder",
		"  ID:      " + emptyDash(b.Name()),
		"  State:   " + b.State(),
		"  CPUs:    " + b.CPUs(),
		"  Memory:  " + b.Memory(),
	}
	if !b.Present {
		return append(lines, "", "No builder container is present.")
	}
	if len(b.Configuration) > 0 {
		lines = append(lines, "", "Configuration")
		for _, entry := range sortedMapLines(b.Configuration) {
			lines = append(lines, "  "+entry)
		}
	}
	if len(b.Raw) > 0 {
		lines = append(lines, "", "Raw")
		for _, entry := range sortedMapLines(b.Raw) {
			lines = append(lines, "  "+entry)
		}
	}
	return lines
}

func (s Stat) SummaryLines() []string {
	lines := []string{}
	if cpuPercent, ok := firstNumberFromMap(s, "cpuPercent", "cpuPercentage", "cpuPercentUsage"); ok {
		lines = append(lines, fmt.Sprintf("  CPU:      %s  %s", formatPercent(cpuPercent), metricBar(cpuPercent, 100, 16)))
	} else if cpuUsec, ok := numberFromMap(s, "cpuUsageUsec"); ok {
		lines = append(lines, "  CPU time: "+formatUsec(cpuUsec))
	}

	memoryUsage, hasMemoryUsage := numberFromMap(s, "memoryUsageBytes")
	memoryLimit, hasMemoryLimit := numberFromMap(s, "memoryLimitBytes")
	if hasMemoryUsage && hasMemoryLimit && memoryLimit > 0 {
		percent := memoryUsage / memoryLimit * 100
		lines = append(lines, fmt.Sprintf("  Memory:   %s / %s  %s %s", FormatBytes(int64(memoryUsage)), FormatBytes(int64(memoryLimit)), metricBar(memoryUsage, memoryLimit, 16), formatPercent(percent)))
	} else if hasMemoryUsage {
		lines = append(lines, "  Memory:   "+FormatBytes(int64(memoryUsage)))
	}

	networkRx, hasNetworkRx := numberFromMap(s, "networkRxBytes")
	networkTx, hasNetworkTx := numberFromMap(s, "networkTxBytes")
	if hasNetworkRx || hasNetworkTx {
		lines = append(lines, fmt.Sprintf("  Network:  %s rx / %s tx", FormatBytes(int64(networkRx)), FormatBytes(int64(networkTx))))
	}

	blockRead, hasBlockRead := numberFromMap(s, "blockReadBytes")
	blockWrite, hasBlockWrite := numberFromMap(s, "blockWriteBytes")
	if hasBlockRead || hasBlockWrite {
		lines = append(lines, fmt.Sprintf("  Block IO: %s read / %s write", FormatBytes(int64(blockRead)), FormatBytes(int64(blockWrite))))
	}

	if processes, ok := firstNumberFromMap(s, "numProcesses", "pids", "Pids"); ok {
		lines = append(lines, fmt.Sprintf("  PIDs:     %.0f", processes))
	}
	return lines
}

func (s Stat) ListSummary() string {
	parts := []string{}
	if cpuPercent, ok := firstNumberFromMap(s, "cpuPercent", "cpuPercentage", "cpuPercentUsage"); ok {
		parts = append(parts, formatPercent(cpuPercent)+" cpu")
	} else if cpuUsec, ok := numberFromMap(s, "cpuUsageUsec"); ok {
		parts = append(parts, formatUsec(cpuUsec)+" cpu")
	}
	memoryUsage, hasMemoryUsage := numberFromMap(s, "memoryUsageBytes")
	memoryLimit, hasMemoryLimit := numberFromMap(s, "memoryLimitBytes")
	switch {
	case hasMemoryUsage && hasMemoryLimit && memoryLimit > 0:
		parts = append(parts, formatPercent(memoryUsage/memoryLimit*100)+" mem")
	case hasMemoryUsage:
		parts = append(parts, FormatBytesField(int64(memoryUsage))+" mem")
	}
	return strings.Join(parts, "  ")
}

func FormatBytes(bytes int64) string {
	if bytes <= 0 {
		return "-"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(bytes)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d %s", bytes, units[unit])
	}
	return fmt.Sprintf("%.1f %s", value, units[unit])
}

func ShortID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12]
}

func relativeTime(value string, now time.Time) string {
	if value == "" {
		return "-"
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return value
	}
	duration := now.Sub(parsed)
	if duration < 0 {
		duration = -duration
	}
	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	case duration < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(duration.Hours()))
	case duration < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(duration.Hours()/24))
	default:
		return parsed.Format("2006-01-02")
	}
}

func formatPlatform(platform Platform) string {
	if platform.OS == "" && platform.Architecture == "" {
		return "-"
	}
	result := platform.OS + "/" + platform.Architecture
	if platform.Variant != "" {
		result += "/" + platform.Variant
	}
	return strings.Trim(result, "/")
}

func shortDigest(digest string) string {
	digest = strings.TrimPrefix(digest, "sha256:")
	if len(digest) <= 12 {
		return digest
	}
	return digest[:12]
}

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func boolLabel(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

// formatPercent renders a percentage at a fixed six-column width ("  0.0%" …
// "100.0%") so values stay put as they gain or lose digits between refreshes.
func formatPercent(value float64) string {
	return fmt.Sprintf("%5.1f%%", value)
}

// FormatBytesField renders a byte size right-aligned in a fixed-width field so
// columns that show live sizes don't jitter as the value crosses unit
// boundaries (e.g. "1023.9 MB" → "1.0 GB").
func FormatBytesField(bytes int64) string {
	return fmt.Sprintf("%9s", FormatBytes(bytes))
}

func formatUsec(value float64) string {
	duration := time.Duration(value) * time.Microsecond
	switch {
	case duration < time.Second:
		return fmt.Sprintf("%dms", duration.Milliseconds())
	case duration < time.Minute:
		return fmt.Sprintf("%.1fs", duration.Seconds())
	default:
		return fmt.Sprintf("%dm%02ds", int(duration.Minutes()), int(duration.Seconds())%60)
	}
}

func metricBar(value float64, max float64, width int) string {
	if width <= 0 {
		return ""
	}
	if max <= 0 {
		return "[" + strings.Repeat("-", width) + "]"
	}
	ratio := value / max
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio*float64(width) + 0.5)
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("#", filled) + strings.Repeat("-", width-filled) + "]"
}

func stringFromMap(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, ok := values[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}

func stringFromNestedMap(values map[string]any, key string, nestedKey string) string {
	nested, ok := nestedMap(values, key)
	if !ok {
		return ""
	}
	return stringFromMap(nested, nestedKey)
}

func numberFromMap(values map[string]any, key string) (float64, bool) {
	if values == nil {
		return 0, false
	}
	value, ok := values[key]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

func firstNumberFromMap(values map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		if value, ok := numberFromMap(values, key); ok {
			return value, true
		}
	}
	return 0, false
}

func numberFromNestedMap(values map[string]any, key string, nestedKey string) (float64, bool) {
	nested, ok := nestedMap(values, key)
	if !ok {
		return 0, false
	}
	return numberFromMap(nested, nestedKey)
}

func nestedMap(values map[string]any, key string) (map[string]any, bool) {
	if values == nil {
		return nil, false
	}
	value, ok := values[key]
	if !ok {
		return nil, false
	}
	typed, ok := value.(map[string]any)
	return typed, ok
}

func sortedMapLines(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("%s: %v", key, values[key]))
	}
	return lines
}
