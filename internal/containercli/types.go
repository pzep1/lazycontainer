package containercli

type SystemStatus struct {
	APIServerAppName string `json:"apiServerAppName"`
	APIServerBuild   string `json:"apiServerBuild"`
	APIServerCommit  string `json:"apiServerCommit"`
	APIServerVersion string `json:"apiServerVersion"`
	AppRoot          string `json:"appRoot"`
	InstallRoot      string `json:"installRoot"`
	Status           string `json:"status"`
}

type Container struct {
	ID            string                 `json:"id"`
	Configuration ContainerConfiguration `json:"configuration"`
	Status        ContainerStatus        `json:"status"`
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
	Digest   string   `json:"digest"`
	Platform Platform `json:"platform"`
	Size     int64    `json:"size"`
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

type Stat map[string]any
