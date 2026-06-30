package tui

import (
	"fmt"
	"github.com/pzep1/lazycont/internal/containercli"
	"sort"
	"strings"
	"time"
)

func (m Model) statListSummary(containerID string) string {
	derived, hasDerived := m.derivedCPUPercent(containerID)
	for _, stat := range m.stats {
		if !statMatches(stat, containerID) {
			continue
		}
		if hasDerived {
			stat = withDerivedCPU(stat, derived)
		}
		return stat.ListSummary()
	}
	return ""
}
func (m *Model) recordStatHistory(stats []containercli.Stat, now time.Time) {
	if len(stats) == 0 {
		return
	}
	if now.IsZero() {
		now = time.Now()
	}
	if m.statHistory == nil {
		m.statHistory = map[string][]statHistorySample{}
	}
	for _, stat := range stats {
		id := statIdentifier(stat)
		if id == "" {
			continue
		}
		sample, ok := statHistorySampleFromStat(stat, now)
		if !ok {
			continue
		}
		existing := m.statHistory[id]
		if n := len(existing); n > 0 {
			deriveStatRates(&sample, existing[n-1])
		}
		samples := append(existing, sample)
		if len(samples) > maxStatHistorySamples {
			samples = samples[len(samples)-maxStatHistorySamples:]
		}
		m.statHistory[id] = samples
	}
}

// deriveStatRates fills a sample's instantaneous CPU% and network/block
// throughput by differencing its cumulative counters against the previous
// sample. Counters that reset (e.g. a recreated container) yield a negative
// delta and are skipped rather than spiking the graph; intervals shorter than
// minDerivationInterval are skipped for the same reason.
func deriveStatRates(sample *statHistorySample, prev statHistorySample) {
	elapsed := sample.at.Sub(prev.at).Seconds()
	if elapsed < minDerivationInterval.Seconds() {
		return
	}
	if sample.hasCPUTime && prev.hasCPUTime {
		if delta := sample.cpuTimeUsec - prev.cpuTimeUsec; delta >= 0 {
			// cpuTimeUsec is microseconds of CPU time; elapsed*1e6 is the wall-clock
			// microseconds in the interval, so the ratio is core-seconds per second.
			sample.cpuPercent = delta / (elapsed * 1e6) * 100
			sample.hasCPU = true
		}
	}
	if sample.hasNetTotal && prev.hasNetTotal {
		if delta := sample.networkTotal - prev.networkTotal; delta >= 0 {
			sample.networkRate = delta / elapsed
			sample.hasNetwork = true
		}
	}
	if sample.hasBlkTotal && prev.hasBlkTotal {
		if delta := sample.blockTotal - prev.blockTotal; delta >= 0 {
			sample.blockRate = delta / elapsed
			sample.hasBlock = true
		}
	}
}
func (m *Model) pruneStatHistory(containers []containercli.Container) {
	if len(m.statHistory) == 0 || len(containers) == 0 {
		return
	}
	known := make(map[string]struct{}, len(containers))
	for _, container := range containers {
		if name := strings.TrimSpace(container.Name()); name != "" {
			known[name] = struct{}{}
		}
		if id := strings.TrimSpace(container.ID); id != "" {
			known[id] = struct{}{}
		}
	}
	for id := range m.statHistory {
		if _, ok := known[id]; ok {
			continue
		}
		matchesContainer := false
		for knownID := range known {
			if strings.Contains(id, knownID) || strings.Contains(knownID, id) {
				matchesContainer = true
				break
			}
		}
		if !matchesContainer {
			delete(m.statHistory, id)
		}
	}
}
func statIdentifier(stat containercli.Stat) string {
	for _, key := range []string{"id", "ID", "container", "Container", "containerID", "containerId", "name", "Name"} {
		value, ok := stat[key]
		if !ok {
			continue
		}
		id := strings.TrimSpace(fmt.Sprint(value))
		if id != "" {
			return id
		}
	}
	return ""
}
func statHistorySampleFromStat(stat containercli.Stat, now time.Time) (statHistorySample, bool) {
	sample := statHistorySample{at: now}
	if cpuUsec, ok := statNumber(stat, "cpuUsageUsec"); ok {
		// Apple reports a cumulative CPU-time counter; deriveStatRates turns it
		// into a live percentage. Any cpuPercent field present alongside it is
		// ignored, since some runtimes zero-fill it (a literal 0 would otherwise
		// masquerade as a real reading and pin the graph to 0%).
		sample.cpuTimeUsec = cpuUsec
		sample.hasCPUTime = true
	} else if cpuPercent, ok := firstStatNumber(stat, "cpuPercent", "cpuPercentage", "cpuPercentUsage"); ok {
		// No cumulative counter: trust a ready-made percentage (non-Apple runtimes).
		sample.cpuPercent = cpuPercent
		sample.hasCPU = true
	}
	if memory, ok := statNumber(stat, "memoryUsageBytes"); ok {
		sample.memoryBytes = memory
		sample.hasMemory = true
	}
	rx, hasRx := statNumber(stat, "networkRxBytes")
	tx, hasTx := statNumber(stat, "networkTxBytes")
	if hasRx || hasTx {
		sample.networkTotal = rx + tx
		sample.hasNetTotal = true
	}
	read, hasRead := statNumber(stat, "blockReadBytes")
	write, hasWrite := statNumber(stat, "blockWriteBytes")
	if hasRead || hasWrite {
		sample.blockTotal = read + write
		sample.hasBlkTotal = true
	}
	return sample, sample.hasCPU || sample.hasCPUTime || sample.hasMemory || sample.hasNetTotal || sample.hasBlkTotal
}

// latestStatSample returns the most recently recorded sample for a container.
func (m Model) latestStatSample(containerID string) (statHistorySample, bool) {
	samples := m.statHistoryForContainer(containerID)
	if len(samples) == 0 {
		return statHistorySample{}, false
	}
	return samples[len(samples)-1], true
}

// derivedCPUPercent returns the latest live CPU% for a container once enough
// samples exist to difference the cumulative CPU-time counter.
func (m Model) derivedCPUPercent(containerID string) (float64, bool) {
	if sample, ok := m.latestStatSample(containerID); ok && sample.hasCPU {
		return sample.cpuPercent, true
	}
	return 0, false
}

// withDerivedCPU returns a shallow copy of a raw stat with the live CPU% folded
// in, so the existing formatter renders the derived percentage instead of the
// meaningless cumulative CPU-time counter.
func withDerivedCPU(stat containercli.Stat, percent float64) containercli.Stat {
	clone := make(containercli.Stat, len(stat)+1)
	for key, value := range stat {
		clone[key] = value
	}
	clone["cpuPercent"] = percent
	return clone
}
func firstStatNumber(stat containercli.Stat, keys ...string) (float64, bool) {
	for _, key := range keys {
		if value, ok := statNumber(stat, key); ok {
			return value, true
		}
	}
	return 0, false
}
func statNumber(stat containercli.Stat, key string) (float64, bool) {
	value, ok := stat[key]
	if !ok {
		return 0, false
	}
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case jsonNumber:
		number, err := v.Float64()
		return number, err == nil
	default:
		return 0, false
	}
}
func (m Model) statHistoryForContainer(containerID string) []statHistorySample {
	if len(m.statHistory) == 0 || strings.TrimSpace(containerID) == "" {
		return nil
	}
	if samples := m.statHistory[containerID]; len(samples) > 0 {
		return samples
	}
	keys := make([]string, 0, len(m.statHistory))
	for key := range m.statHistory {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if strings.Contains(key, containerID) || strings.Contains(containerID, key) {
			return m.statHistory[key]
		}
	}
	return nil
}
func historyValues(samples []statHistorySample, valueFor func(statHistorySample) (float64, bool)) ([]float64, bool) {
	values := make([]float64, 0, len(samples))
	for _, sample := range samples {
		value, ok := valueFor(sample)
		if !ok {
			continue
		}
		values = append(values, value)
	}
	return values, len(values) >= 2
}
func statMatches(stat containercli.Stat, containerID string) bool {
	if containerID == "" {
		return false
	}
	for _, key := range []string{"id", "ID", "container", "Container", "containerID", "containerId", "name", "Name"} {
		value, ok := stat[key]
		if !ok {
			continue
		}
		if strings.Contains(fmt.Sprint(value), containerID) || strings.Contains(containerID, fmt.Sprint(value)) {
			return true
		}
	}
	return false
}
