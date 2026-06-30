// Package compose gives lazycontainer a lazydocker-style "services" experience
// on top of Apple's `container` CLI, which has no native compose support. It
// parses a Compose file and translates each service into the individual
// `container` commands needed to bring it up, take it down, and inspect it —
// the orchestration lives here, not in the CLI.
package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Filenames are the Compose file names looked for in a directory, in priority
// order (matching docker compose's resolution).
var Filenames = []string{
	"compose.yaml",
	"compose.yml",
	"docker-compose.yaml",
	"docker-compose.yml",
}

// Project is a parsed Compose file: a named, ordered set of services.
type Project struct {
	Name     string
	File     string
	Services []Service
}

// Service is one Compose service reduced to the fields lazycontainer can drive
// through `container run`.
type Service struct {
	Name          string
	Image         string
	Build         string // build context directory, when the image must be built
	Dockerfile    string
	ContainerName string
	Ports         []string // "published:target[/proto]"
	Environment   []string // "KEY=VALUE"
	EnvFiles      []string
	Volumes       []string // "source:target[:mode]"
	Networks      []string
	Command       []string
	DependsOn     []string
	Restart       string
}

// ContainerNameFor returns the container name a service's container is created
// with: its explicit container_name, else "<project>-<service>" (mirroring
// docker compose's default naming so lists line up).
func (p Project) ContainerNameFor(service Service) string {
	if name := strings.TrimSpace(service.ContainerName); name != "" {
		return name
	}
	project := strings.TrimSpace(p.Name)
	if project == "" {
		return service.Name
	}
	return project + "-" + service.Name
}

// Service looks up a service by name.
func (p Project) Service(name string) (Service, bool) {
	for _, service := range p.Services {
		if service.Name == name {
			return service, true
		}
	}
	return Service{}, false
}

// Discover finds a Compose file in dir, returning its path and whether one
// exists.
func Discover(dir string) (string, bool) {
	for _, name := range Filenames {
		path := filepath.Join(dir, name)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, true
		}
	}
	return "", false
}

// Load reads and parses a Compose file. The project name defaults to the file's
// parent directory (matching docker compose) when the file omits `name`.
func Load(path string) (Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Project{}, err
	}
	project, err := Parse(data)
	if err != nil {
		return Project{}, fmt.Errorf("parse %s: %w", path, err)
	}
	project.File = path
	if strings.TrimSpace(project.Name) == "" {
		project.Name = defaultProjectName(path)
	}
	return project, nil
}

func defaultProjectName(path string) string {
	dir := filepath.Base(filepath.Dir(path))
	dir = strings.TrimSpace(dir)
	if dir == "" || dir == "." || dir == string(filepath.Separator) {
		return "compose"
	}
	// Compose normalizes the project name to lowercase with limited punctuation.
	return strings.ToLower(dir)
}

type rawFile struct {
	Name     string    `yaml:"name"`
	Services yaml.Node `yaml:"services"`
}

// Parse decodes Compose YAML into a Project, preserving service order.
func Parse(data []byte) (Project, error) {
	var file rawFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return Project{}, err
	}
	project := Project{Name: strings.TrimSpace(file.Name)}
	if file.Services.Kind == 0 {
		return project, nil
	}
	if file.Services.Kind != yaml.MappingNode {
		return Project{}, fmt.Errorf("services must be a mapping")
	}
	// A mapping node stores [key, value, key, value, ...]; iterate pairs so
	// services keep their file order.
	for i := 0; i+1 < len(file.Services.Content); i += 2 {
		name := file.Services.Content[i].Value
		service, err := decodeService(name, file.Services.Content[i+1])
		if err != nil {
			return Project{}, fmt.Errorf("service %q: %w", name, err)
		}
		project.Services = append(project.Services, service)
	}
	return project, nil
}

type rawService struct {
	Image         string    `yaml:"image"`
	Build         yaml.Node `yaml:"build"`
	ContainerName string    `yaml:"container_name"`
	Ports         yaml.Node `yaml:"ports"`
	Environment   yaml.Node `yaml:"environment"`
	EnvFile       yaml.Node `yaml:"env_file"`
	Volumes       yaml.Node `yaml:"volumes"`
	Networks      yaml.Node `yaml:"networks"`
	Command       yaml.Node `yaml:"command"`
	DependsOn     yaml.Node `yaml:"depends_on"`
	Restart       string    `yaml:"restart"`
}

func decodeService(name string, node *yaml.Node) (Service, error) {
	var raw rawService
	if err := node.Decode(&raw); err != nil {
		return Service{}, err
	}
	service := Service{
		Name:          name,
		Image:         strings.TrimSpace(raw.Image),
		ContainerName: strings.TrimSpace(raw.ContainerName),
		Restart:       strings.TrimSpace(raw.Restart),
		Ports:         decodePorts(raw.Ports),
		Environment:   decodeEnvironment(raw.Environment),
		EnvFiles:      decodeStringList(raw.EnvFile),
		Volumes:       decodeVolumes(raw.Volumes),
		Networks:      decodeKeysOrList(raw.Networks),
		Command:       decodeCommand(raw.Command),
		DependsOn:     decodeKeysOrList(raw.DependsOn),
	}
	service.Build, service.Dockerfile = decodeBuild(raw.Build)
	return service, nil
}

// decodeStringList accepts a scalar string or a sequence of strings.
func decodeStringList(node yaml.Node) []string {
	switch node.Kind {
	case yaml.ScalarNode:
		if v := strings.TrimSpace(node.Value); v != "" {
			return []string{v}
		}
	case yaml.SequenceNode:
		out := make([]string, 0, len(node.Content))
		for _, item := range node.Content {
			if v := strings.TrimSpace(item.Value); v != "" {
				out = append(out, v)
			}
		}
		return out
	}
	return nil
}

// decodeKeysOrList accepts a sequence of names or a mapping whose keys are the
// names (used for networks and depends_on).
func decodeKeysOrList(node yaml.Node) []string {
	switch node.Kind {
	case yaml.SequenceNode:
		return decodeStringList(node)
	case yaml.MappingNode:
		out := make([]string, 0, len(node.Content)/2)
		for i := 0; i+1 < len(node.Content); i += 2 {
			if v := strings.TrimSpace(node.Content[i].Value); v != "" {
				out = append(out, v)
			}
		}
		return out
	}
	return nil
}

// decodeEnvironment accepts a sequence of "KEY=VALUE" or a mapping of KEY:
// VALUE, normalizing both to "KEY=VALUE".
func decodeEnvironment(node yaml.Node) []string {
	switch node.Kind {
	case yaml.SequenceNode:
		return decodeStringList(node)
	case yaml.MappingNode:
		out := make([]string, 0, len(node.Content)/2)
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := strings.TrimSpace(node.Content[i].Value)
			if key == "" {
				continue
			}
			out = append(out, key+"="+node.Content[i+1].Value)
		}
		return out
	}
	return nil
}

// decodePorts accepts short ("8080:80", "8080:80/udp") or long-form
// ({target, published, protocol}) entries, normalizing to "published:target[/proto]".
func decodePorts(node yaml.Node) []string {
	if node.Kind != yaml.SequenceNode {
		return nil
	}
	out := make([]string, 0, len(node.Content))
	for _, item := range node.Content {
		switch item.Kind {
		case yaml.ScalarNode:
			if v := strings.TrimSpace(item.Value); v != "" {
				out = append(out, v)
			}
		case yaml.MappingNode:
			var long struct {
				Target    string `yaml:"target"`
				Published string `yaml:"published"`
				Protocol  string `yaml:"protocol"`
			}
			if err := item.Decode(&long); err != nil {
				continue
			}
			target := strings.TrimSpace(long.Target)
			if target == "" {
				continue
			}
			mapping := target
			if published := strings.TrimSpace(long.Published); published != "" {
				mapping = published + ":" + target
			}
			if proto := strings.TrimSpace(long.Protocol); proto != "" {
				mapping += "/" + proto
			}
			out = append(out, mapping)
		}
	}
	return out
}

// decodeVolumes accepts short ("src:dst[:mode]") or long-form
// ({source, target, read_only}) entries, normalizing to "src:dst[:ro]".
func decodeVolumes(node yaml.Node) []string {
	if node.Kind != yaml.SequenceNode {
		return nil
	}
	out := make([]string, 0, len(node.Content))
	for _, item := range node.Content {
		switch item.Kind {
		case yaml.ScalarNode:
			if v := strings.TrimSpace(item.Value); v != "" {
				out = append(out, v)
			}
		case yaml.MappingNode:
			var long struct {
				Source   string `yaml:"source"`
				Target   string `yaml:"target"`
				ReadOnly bool   `yaml:"read_only"`
			}
			if err := item.Decode(&long); err != nil {
				continue
			}
			target := strings.TrimSpace(long.Target)
			if target == "" {
				continue
			}
			mount := target
			if source := strings.TrimSpace(long.Source); source != "" {
				mount = source + ":" + target
			}
			if long.ReadOnly {
				mount += ":ro"
			}
			out = append(out, mount)
		}
	}
	return out
}

// decodeCommand accepts a sequence (exec form) or a scalar string (split on
// whitespace, a pragmatic approximation of shell form).
func decodeCommand(node yaml.Node) []string {
	switch node.Kind {
	case yaml.SequenceNode:
		return decodeStringList(node)
	case yaml.ScalarNode:
		if v := strings.TrimSpace(node.Value); v != "" {
			return strings.Fields(v)
		}
	}
	return nil
}

// decodeBuild accepts a scalar context path or a long-form
// {context, dockerfile}, returning (context, dockerfile).
func decodeBuild(node yaml.Node) (string, string) {
	switch node.Kind {
	case yaml.ScalarNode:
		return strings.TrimSpace(node.Value), ""
	case yaml.MappingNode:
		var long struct {
			Context    string `yaml:"context"`
			Dockerfile string `yaml:"dockerfile"`
		}
		if err := node.Decode(&long); err != nil {
			return "", ""
		}
		return strings.TrimSpace(long.Context), strings.TrimSpace(long.Dockerfile)
	}
	return "", ""
}
