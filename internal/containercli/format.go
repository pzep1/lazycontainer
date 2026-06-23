package containercli

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

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
	return lines
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
