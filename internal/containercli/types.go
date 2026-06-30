package containercli

import (
	"bytes"
	"encoding/json"
)

type SystemStatus struct {
	APIServerAppName string `json:"apiServerAppName"`
	APIServerBuild   string `json:"apiServerBuild"`
	APIServerCommit  string `json:"apiServerCommit"`
	APIServerVersion string `json:"apiServerVersion"`
	AppRoot          string `json:"appRoot"`
	InstallRoot      string `json:"installRoot"`
	Status           string `json:"status"`
}

type SystemDiskUsage struct {
	Containers DiskUsageCategory `json:"containers"`
	Images     DiskUsageCategory `json:"images"`
	Volumes    DiskUsageCategory `json:"volumes"`
}

type DiskUsageCategory struct {
	Active      int   `json:"active"`
	Reclaimable int64 `json:"reclaimable"`
	SizeInBytes int64 `json:"sizeInBytes"`
	Total       int   `json:"total"`
}

type SystemVersion struct {
	AppName   string `json:"appName"`
	BuildType string `json:"buildType"`
	Commit    string `json:"commit"`
	Version   string `json:"version"`
}

type Container struct {
	ID            string                 `json:"id"`
	Configuration ContainerConfiguration `json:"configuration"`
	Status        ContainerStatus        `json:"status"`
}

type ContainerLaunchOptions struct {
	Name      string
	Flags     []string
	Arguments []string
}

type ContainerConfiguration struct {
	ID               string        `json:"id"`
	CreationDate     string        `json:"creationDate"`
	Image            ImageRef      `json:"image"`
	InitProcess      InitProcess   `json:"initProcess"`
	Mounts           []Mount       `json:"mounts"`
	Networks         []Network     `json:"networks"`
	Platform         Platform      `json:"platform"`
	PublishedPorts   []Port        `json:"publishedPorts"`
	PublishedSockets []interface{} `json:"publishedSockets"`
	Resources        Resources     `json:"resources"`
	RuntimeHandler   string        `json:"runtimeHandler"`
	StopSignal       string        `json:"stopSignal"`
}

type ImageRef struct {
	Reference  string     `json:"reference"`
	Descriptor Descriptor `json:"descriptor"`
}

type Descriptor struct {
	Digest    string `json:"digest"`
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
}

type InitProcess struct {
	Executable       string   `json:"executable"`
	Arguments        []string `json:"arguments"`
	Environment      []string `json:"environment"`
	WorkingDirectory string   `json:"workingDirectory"`
	Terminal         bool     `json:"terminal"`
}

type Mount struct {
	Destination string   `json:"destination"`
	Options     []string `json:"options"`
	Source      string   `json:"source"`
	Type        any      `json:"type"`
}

type Network struct {
	Network string         `json:"network"`
	Options map[string]any `json:"options"`
}

type Platform struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
	Variant      string `json:"variant"`
}

type Port struct {
	ContainerPort int    `json:"containerPort"`
	Count         int    `json:"count"`
	HostAddress   string `json:"hostAddress"`
	HostPort      int    `json:"hostPort"`
	Proto         string `json:"proto"`
}

type Resources struct {
	CPUOverhead   int   `json:"cpuOverhead"`
	CPUs          int   `json:"cpus"`
	MemoryInBytes int64 `json:"memoryInBytes"`
}

type ContainerStatus struct {
	Networks    []Network `json:"networks"`
	StartedDate string    `json:"startedDate"`
	State       string    `json:"state"`
}

type Image struct {
	ID            string             `json:"id"`
	Configuration ImageConfiguration `json:"configuration"`
	Variants      []ImageVariant     `json:"variants"`
}

type ImageConfiguration struct {
	CreationDate string     `json:"creationDate"`
	Descriptor   Descriptor `json:"descriptor"`
	Name         string     `json:"name"`
}

type ImageVariant struct {
	Config   ImageVariantConfig `json:"config"`
	Digest   string             `json:"digest"`
	Platform Platform           `json:"platform"`
	Size     int64              `json:"size"`
}

type ImageVariantConfig struct {
	Architecture string         `json:"architecture"`
	Config       ImageConfig    `json:"config"`
	Created      string         `json:"created"`
	History      []ImageHistory `json:"history"`
	OS           string         `json:"os"`
	RootFS       ImageRootFS    `json:"rootfs"`
	Variant      string         `json:"variant"`
}

type ImageConfig struct {
	Cmd        []string       `json:"Cmd"`
	Entrypoint []string       `json:"Entrypoint"`
	Env        []string       `json:"Env"`
	Labels     map[string]any `json:"Labels"`
	StopSignal string         `json:"StopSignal"`
	WorkingDir string         `json:"WorkingDir"`
}

type ImageHistory struct {
	Comment    string `json:"comment"`
	Created    string `json:"created"`
	CreatedBy  string `json:"created_by"`
	EmptyLayer bool   `json:"empty_layer"`
}

type ImageRootFS struct {
	DiffIDs []string `json:"diff_ids"`
	Type    string   `json:"type"`
}

type Volume struct {
	ID            string              `json:"id"`
	Configuration VolumeConfiguration `json:"configuration"`
}

type VolumeConfiguration struct {
	CreationDate string         `json:"creationDate"`
	Driver       string         `json:"driver"`
	Format       string         `json:"format"`
	Labels       map[string]any `json:"labels"`
	Name         string         `json:"name"`
	Options      map[string]any `json:"options"`
	SizeInBytes  int64          `json:"sizeInBytes"`
	Source       string         `json:"source"`
}

type NetworkResource struct {
	ID            string               `json:"id"`
	Configuration NetworkConfiguration `json:"configuration"`
	Status        NetworkStatus        `json:"status"`
}

type NetworkConfiguration struct {
	CreationDate string         `json:"creationDate"`
	Labels       map[string]any `json:"labels"`
	Mode         string         `json:"mode"`
	Name         string         `json:"name"`
	Options      map[string]any `json:"options"`
	Plugin       string         `json:"plugin"`
}

type NetworkStatus struct {
	IPv4Gateway string `json:"ipv4Gateway"`
	IPv4Subnet  string `json:"ipv4Subnet"`
	IPv6Subnet  string `json:"ipv6Subnet"`
}

type Machine struct {
	ID            string         `json:"id"`
	Configuration map[string]any `json:"configuration"`
	Status        any            `json:"status"`
	Default       bool           `json:"default"`
	Raw           map[string]any `json:"-"`
}

func (m *Machine) UnmarshalJSON(data []byte) error {
	type machineAlias Machine
	var alias machineAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*m = Machine(alias)
	return json.Unmarshal(data, &m.Raw)
}

type RegistryLogin struct {
	Server   string         `json:"server"`
	Registry string         `json:"registry"`
	Hostname string         `json:"hostname"`
	Username string         `json:"username"`
	Scheme   string         `json:"scheme"`
	Raw      map[string]any `json:"-"`
	Value    string         `json:"-"`
}

func (r *RegistryLogin) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err == nil {
		r.Value = value
		return nil
	}

	type registryLoginAlias RegistryLogin
	var alias registryLoginAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*r = RegistryLogin(alias)
	return json.Unmarshal(data, &r.Raw)
}

// SystemDNSDomain is one local DNS domain from `container system dns list`.
// Apple's container CLI can register resolvable `*.test`-style domains for the
// host — a capability Docker has no equivalent for. The JSON shape is
// version-dependent, so the domain captures a raw object plus a string fallback.
type SystemDNSDomain struct {
	Name string         `json:"-"`
	Raw  map[string]any `json:"-"`
}

func (d *SystemDNSDomain) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err == nil {
		d.Name = value
		return nil
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	d.Raw = raw
	d.Name = firstNonEmpty(
		stringFromMap(raw, "domain"),
		stringFromMap(raw, "name"),
		stringFromMap(raw, "host"),
		stringFromMap(raw, "hostname"),
	)
	return nil
}

// SystemProperty is one entry from `container system property list` — host-level
// configuration for the container subsystem. Shape is version-dependent.
type SystemProperty struct {
	ID    string         `json:"-"`
	Value string         `json:"-"`
	Raw   map[string]any `json:"-"`
}

func (p *SystemProperty) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err == nil {
		p.ID = value
		return nil
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	p.Raw = raw
	p.ID = firstNonEmpty(
		stringFromMap(raw, "id"),
		stringFromMap(raw, "name"),
		stringFromMap(raw, "key"),
	)
	p.Value = firstNonEmpty(
		stringFromMap(raw, "value"),
		stringFromMap(raw, "current"),
		stringFromMap(raw, "default"),
	)
	return nil
}

type BuilderStatus struct {
	ID            string         `json:"id"`
	ContainerID   string         `json:"containerID"`
	NameValue     string         `json:"name"`
	StateValue    string         `json:"state"`
	StatusValue   string         `json:"status"`
	Configuration map[string]any `json:"configuration"`
	Raw           map[string]any `json:"-"`
	Value         string         `json:"-"`
	Present       bool           `json:"-"`
}

func (b *BuilderStatus) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*b = BuilderStatus{}
		return nil
	}
	if trimmed[0] == '[' {
		var entries []json.RawMessage
		if err := json.Unmarshal(trimmed, &entries); err != nil {
			return err
		}
		if len(entries) == 0 {
			*b = BuilderStatus{}
			return nil
		}
		return b.UnmarshalJSON(entries[0])
	}

	var value string
	if err := json.Unmarshal(trimmed, &value); err == nil {
		b.Value = value
		b.Present = value != ""
		return nil
	}

	type builderStatusAlias BuilderStatus
	var alias builderStatusAlias
	if err := json.Unmarshal(trimmed, &alias); err != nil {
		return err
	}
	*b = BuilderStatus(alias)
	b.Present = true
	return json.Unmarshal(trimmed, &b.Raw)
}

type Stat map[string]any
