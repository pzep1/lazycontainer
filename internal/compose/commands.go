package compose

import (
	"path/filepath"
	"strings"
)

// imageRef is the image a service runs: its explicit image, or — for a
// build-only service — the project-scoped tag BuildArgs builds it as.
func (p Project) imageRef(service Service) string {
	if image := strings.TrimSpace(service.Image); image != "" {
		return image
	}
	return p.ContainerNameFor(service)
}

// needsBuild reports whether a service must be built before it can run (it has a
// build context and no prebuilt image).
func (s Service) needsBuild() bool {
	return strings.TrimSpace(s.Image) == "" && strings.TrimSpace(s.Build) != ""
}

// BuildArgs returns the `container build` invocation for a service with a build
// context, or nil when the service uses a prebuilt image.
func (p Project) BuildArgs(service Service) []string {
	if strings.TrimSpace(service.Build) == "" {
		return nil
	}
	args := []string{"build", "--tag", p.imageRef(service)}
	if dockerfile := strings.TrimSpace(service.Dockerfile); dockerfile != "" {
		// Compose resolves `dockerfile` relative to `context`, but
		// `container build --file` resolves relative to the working directory,
		// so join them to point at the real file.
		args = append(args, "--file", filepath.Join(service.Build, dockerfile))
	}
	args = append(args, service.Build)
	return args
}

// RunArgs returns the `container run --detach` invocation that creates and
// starts a service's container.
func (p Project) RunArgs(service Service) []string {
	args := []string{"run", "--detach", "--name", p.ContainerNameFor(service)}
	for _, port := range service.Ports {
		args = append(args, "--publish", port)
	}
	for _, env := range service.Environment {
		args = append(args, "--env", env)
	}
	for _, file := range service.EnvFiles {
		args = append(args, "--env-file", file)
	}
	for _, volume := range service.Volumes {
		args = append(args, "--volume", volume)
	}
	for _, network := range service.Networks {
		args = append(args, "--network", network)
	}
	args = append(args, p.imageRef(service))
	args = append(args, service.Command...)
	return args
}

// UpArgs returns the invocations to bring a single service up: build first when
// the service has a build context and no image, then run --detach.
func (p Project) UpArgs(service Service) [][]string {
	var steps [][]string
	if service.needsBuild() {
		if build := p.BuildArgs(service); build != nil {
			steps = append(steps, build)
		}
	}
	steps = append(steps, p.RunArgs(service))
	return steps
}

// DownArgs returns the invocations to take a single service down: stop then
// force-remove its container.
func (p Project) DownArgs(service Service) [][]string {
	name := p.ContainerNameFor(service)
	return [][]string{
		{"stop", name},
		{"delete", "--force", name},
	}
}

// StartArgs / StopArgs start and stop an existing service container.
func (p Project) StartArgs(service Service) []string {
	return []string{"start", p.ContainerNameFor(service)}
}

func (p Project) StopArgs(service Service) []string {
	return []string{"stop", p.ContainerNameFor(service)}
}

// RestartArgs stops then starts a service container (Apple has no native
// restart verb).
func (p Project) RestartArgs(service Service) [][]string {
	name := p.ContainerNameFor(service)
	return [][]string{
		{"stop", name},
		{"start", name},
	}
}

// UpAll returns every invocation to bring the whole project up, services in
// dependency order so a service's dependencies start first.
func (p Project) UpAll() [][]string {
	var steps [][]string
	for _, service := range p.orderedServices() {
		steps = append(steps, p.UpArgs(service)...)
	}
	return steps
}

// DownAll returns every invocation to take the whole project down, in reverse
// dependency order so dependents stop before their dependencies.
func (p Project) DownAll() [][]string {
	ordered := p.orderedServices()
	var steps [][]string
	for i := len(ordered) - 1; i >= 0; i-- {
		steps = append(steps, p.DownArgs(ordered[i])...)
	}
	return steps
}

// orderedServices returns services topologically sorted by depends_on so
// dependencies come first. It is stable (preserves file order among
// independent services) and tolerant of cycles or unknown deps (it falls back
// to file order rather than dropping a service).
func (p Project) orderedServices() []Service {
	index := make(map[string]int, len(p.Services))
	for i, service := range p.Services {
		index[service.Name] = i
	}
	visited := make(map[string]bool, len(p.Services))
	onStack := make(map[string]bool, len(p.Services))
	ordered := make([]Service, 0, len(p.Services))

	var visit func(name string)
	visit = func(name string) {
		if visited[name] || onStack[name] {
			return // already placed, or a cycle — break it
		}
		idx, ok := index[name]
		if !ok {
			return // dependency on an unknown service: ignore
		}
		onStack[name] = true
		for _, dep := range p.Services[idx].DependsOn {
			visit(dep)
		}
		onStack[name] = false
		visited[name] = true
		ordered = append(ordered, p.Services[idx])
	}

	for _, service := range p.Services {
		visit(service.Name)
	}
	return ordered
}
