package containercli

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestSystemDNSDomainParsesStringAndObjectShapes(t *testing.T) {
	var domains []SystemDNSDomain
	if err := json.Unmarshal([]byte(`["test", {"domain": "myapp.test"}]`), &domains); err != nil {
		t.Fatal(err)
	}
	if len(domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(domains))
	}
	if got := domains[0].Display(); got != "test" {
		t.Fatalf("string domain Display = %q, want test", got)
	}
	if got := domains[1].Display(); got != "myapp.test" {
		t.Fatalf("object domain Display = %q, want myapp.test", got)
	}
}

func TestSystemPropertyParsesObjectShape(t *testing.T) {
	var properties []SystemProperty
	if err := json.Unmarshal([]byte(`[{"id": "build.rosetta", "value": "true"}]`), &properties); err != nil {
		t.Fatal(err)
	}
	if len(properties) != 1 {
		t.Fatalf("expected 1 property, got %d", len(properties))
	}
	if got := properties[0].Display(); got != "build.rosetta: true" {
		t.Fatalf("property Display = %q, want build.rosetta: true", got)
	}
}

func TestFirstPublishedURL(t *testing.T) {
	c := Container{Configuration: ContainerConfiguration{PublishedPorts: []Port{
		{HostPort: 0, ContainerPort: 5432},
		{HostAddress: "0.0.0.0", HostPort: 8080, ContainerPort: 80},
	}}}
	url, ok := c.FirstPublishedURL()
	if !ok || url != "http://localhost:8080" {
		t.Fatalf("FirstPublishedURL = %q, %v", url, ok)
	}

	none := Container{}
	if _, ok := none.FirstPublishedURL(); ok {
		t.Fatalf("expected no URL when no ports are published")
	}
}

func TestStatSummaryLinesFormatsAppleStatsShape(t *testing.T) {
	stat := Stat{
		"id":               "web",
		"memoryUsageBytes": float64(47431680),
		"memoryLimitBytes": float64(1073741824),
		"cpuUsageUsec":     float64(1234567),
		"networkRxBytes":   float64(1289011),
		"networkTxBytes":   float64(876544),
		"blockReadBytes":   float64(4718592),
		"blockWriteBytes":  float64(2202009),
		"numProcesses":     float64(3),
	}

	got := stat.SummaryLines()
	want := []string{
		"  CPU time: 1.2s",
		"  Memory:   45.2 MB / 1.0 GB  [#---------------]   4.4%",
		"  Network:  1.2 MB rx / 856.0 KB tx",
		"  Block IO: 4.5 MB read / 2.1 MB write",
		"  PIDs:     3",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("summary mismatch\nwant: %#v\n got: %#v", want, got)
	}
}

func TestStatSummaryLinesShowsCPUPercentWhenAvailable(t *testing.T) {
	stat := Stat{
		"id":             "web",
		"cpuPercent":     float64(125.12),
		"numProcesses":   float64(12),
		"networkRxBytes": float64(0),
	}

	got := stat.SummaryLines()
	want := []string{
		"  CPU:      125.1%  [################]",
		"  Network:  - rx / - tx",
		"  PIDs:     12",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("summary mismatch\nwant: %#v\n got: %#v", want, got)
	}
}

func TestStatListSummaryFormatsCompactCPUAndMemory(t *testing.T) {
	stat := Stat{
		"id":               "web",
		"cpuPercent":       float64(12.34),
		"memoryUsageBytes": float64(47431680),
		"memoryLimitBytes": float64(1073741824),
	}

	if got, want := stat.ListSummary(), " 12.3% cpu    4.4% mem"; got != want {
		t.Fatalf("list summary = %q, want %q", got, want)
	}
}

func TestStatListSummaryShowsMemoryBytesWithoutLimit(t *testing.T) {
	stat := Stat{
		"id":               "web",
		"memoryUsageBytes": float64(47431680),
	}

	if got, want := stat.ListSummary(), "  45.2 MB mem"; got != want {
		t.Fatalf("list summary = %q, want %q", got, want)
	}
}

func TestStatListSummaryShowsCPUTimeWhenPercentUnavailable(t *testing.T) {
	stat := Stat{
		"id":           "web",
		"cpuUsageUsec": float64(1234567),
	}

	if got, want := stat.ListSummary(), "1.2s cpu"; got != want {
		t.Fatalf("list summary = %q, want %q", got, want)
	}
}

func TestImageLayerHistoryLinesFormatsVerboseImageShape(t *testing.T) {
	image := Image{
		Configuration: ImageConfiguration{Name: "docker.io/library/alpine:latest"},
		Variants: []ImageVariant{{
			Platform: Platform{OS: "linux", Architecture: "arm64", Variant: "v8"},
			Config: ImageVariantConfig{
				History: []ImageHistory{
					{CreatedBy: "ADD alpine-minirootfs-3.24.0-aarch64.tar.gz / # buildkit"},
					{CreatedBy: "CMD [\"/bin/sh\"]", EmptyLayer: true},
				},
				RootFS: ImageRootFS{
					DiffIDs: []string{"sha256:375591c23c8de111a75382d674cf6688f56adecb5e3018d29ada57c10135db5e"},
					Type:    "layers",
				},
			},
		}},
	}

	got := image.LayerHistoryLines()
	want := []string{
		"  linux/arm64/v8  2 history entries  1 filesystem layers",
		"    375591c23c8d  ADD alpine-minirootfs-3.24.0-aarch64.tar.gz / # buildkit",
		"    metadata  CMD [\"/bin/sh\"]",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("history mismatch\nwant: %#v\n got: %#v", want, got)
	}
}
