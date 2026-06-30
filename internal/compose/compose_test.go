package compose

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseHandlesShortAndLongFormFields(t *testing.T) {
	data := []byte(`
name: shop
services:
  web:
    image: nginx:latest
    container_name: shop-web
    ports:
      - "8080:80"
      - target: 443
        published: "8443"
        protocol: tcp
    environment:
      LOG: debug
      LEVEL: info
    volumes:
      - ./html:/usr/share/nginx/html:ro
      - type: bind
        source: ./conf
        target: /etc/nginx
        read_only: true
    networks:
      - frontend
    command: ["nginx", "-g", "daemon off;"]
    depends_on:
      - api
  api:
    build:
      context: ./api
      dockerfile: Dockerfile.api
    env_file: .env
    networks:
      backend: {}
`)
	project, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if project.Name != "shop" {
		t.Fatalf("project name = %q", project.Name)
	}
	if len(project.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(project.Services))
	}
	// Service order is preserved from the file.
	if project.Services[0].Name != "web" || project.Services[1].Name != "api" {
		t.Fatalf("service order = %q,%q", project.Services[0].Name, project.Services[1].Name)
	}

	web := project.Services[0]
	if web.Image != "nginx:latest" || web.ContainerName != "shop-web" {
		t.Fatalf("web image/name = %q/%q", web.Image, web.ContainerName)
	}
	if !reflect.DeepEqual(web.Ports, []string{"8080:80", "8443:443/tcp"}) {
		t.Fatalf("web ports = %#v", web.Ports)
	}
	if !reflect.DeepEqual(web.Environment, []string{"LOG=debug", "LEVEL=info"}) {
		t.Fatalf("web env = %#v", web.Environment)
	}
	if !reflect.DeepEqual(web.Volumes, []string{"./html:/usr/share/nginx/html:ro", "./conf:/etc/nginx:ro"}) {
		t.Fatalf("web volumes = %#v", web.Volumes)
	}
	if !reflect.DeepEqual(web.Networks, []string{"frontend"}) {
		t.Fatalf("web networks = %#v", web.Networks)
	}
	if !reflect.DeepEqual(web.Command, []string{"nginx", "-g", "daemon off;"}) {
		t.Fatalf("web command = %#v", web.Command)
	}
	if !reflect.DeepEqual(web.DependsOn, []string{"api"}) {
		t.Fatalf("web depends_on = %#v", web.DependsOn)
	}

	api := project.Services[1]
	if api.Build != "./api" || api.Dockerfile != "Dockerfile.api" {
		t.Fatalf("api build = %q dockerfile = %q", api.Build, api.Dockerfile)
	}
	if !reflect.DeepEqual(api.EnvFiles, []string{".env"}) {
		t.Fatalf("api env_file = %#v", api.EnvFiles)
	}
	if !reflect.DeepEqual(api.Networks, []string{"backend"}) {
		t.Fatalf("api networks = %#v", api.Networks)
	}
}

func TestRunArgsTranslatesServiceToContainerRun(t *testing.T) {
	project := Project{Name: "shop", Services: []Service{{
		Name:        "web",
		Image:       "nginx:latest",
		Ports:       []string{"8080:80"},
		Environment: []string{"LOG=debug"},
		Volumes:     []string{"./html:/usr/share/nginx/html"},
		Networks:    []string{"frontend"},
		Command:     []string{"nginx", "-g", "daemon off;"},
	}}}
	got := project.RunArgs(project.Services[0])
	want := []string{
		"run", "--detach", "--name", "shop-web",
		"--publish", "8080:80",
		"--env", "LOG=debug",
		"--volume", "./html:/usr/share/nginx/html",
		"--network", "frontend",
		"nginx:latest",
		"nginx", "-g", "daemon off;",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RunArgs mismatch\nwant: %#v\n got: %#v", want, got)
	}
}

func TestUpArgsBuildsWhenNoImage(t *testing.T) {
	project := Project{Name: "shop", Services: []Service{{
		Name:       "api",
		Build:      "./api",
		Dockerfile: "Dockerfile.api",
	}}}
	steps := project.UpArgs(project.Services[0])
	if len(steps) != 2 {
		t.Fatalf("expected build+run, got %d steps: %#v", len(steps), steps)
	}
	// The Dockerfile is resolved relative to the build context (compose semantics).
	wantBuild := []string{"build", "--tag", "shop-api", "--file", "api/Dockerfile.api", "./api"}
	if !reflect.DeepEqual(steps[0], wantBuild) {
		t.Fatalf("build step = %#v", steps[0])
	}
	// The run step references the freshly-built project-scoped tag.
	wantRun := []string{"run", "--detach", "--name", "shop-api", "shop-api"}
	if !reflect.DeepEqual(steps[1], wantRun) {
		t.Fatalf("run step = %#v", steps[1])
	}
}

func TestDownArgsStopsThenForceRemoves(t *testing.T) {
	project := Project{Name: "shop", Services: []Service{{Name: "web", Image: "nginx"}}}
	got := project.DownArgs(project.Services[0])
	want := [][]string{{"stop", "shop-web"}, {"delete", "--force", "shop-web"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DownArgs = %#v", got)
	}
}

func TestUpAllOrdersDependenciesFirst(t *testing.T) {
	// web depends on api; api depends on db. Up order must be db, api, web.
	project := Project{Name: "shop", Services: []Service{
		{Name: "web", Image: "nginx", DependsOn: []string{"api"}},
		{Name: "api", Image: "api", DependsOn: []string{"db"}},
		{Name: "db", Image: "postgres"},
	}}
	ordered := project.orderedServices()
	var names []string
	for _, s := range ordered {
		names = append(names, s.Name)
	}
	if !reflect.DeepEqual(names, []string{"db", "api", "web"}) {
		t.Fatalf("dependency order = %#v", names)
	}
	// DownAll is the reverse.
	down := project.DownAll()
	if len(down) == 0 || down[0][1] != "shop-web" {
		t.Fatalf("down should start with web (reverse order): %#v", down)
	}
}

func TestOrderedServicesToleratesCycles(t *testing.T) {
	project := Project{Services: []Service{
		{Name: "a", DependsOn: []string{"b"}},
		{Name: "b", DependsOn: []string{"a"}},
	}}
	if got := len(project.orderedServices()); got != 2 {
		t.Fatalf("cycle should not drop services, got %d", got)
	}
}

func TestContainerNameForPrefersExplicitName(t *testing.T) {
	project := Project{Name: "shop"}
	if got := project.ContainerNameFor(Service{Name: "web", ContainerName: "custom"}); got != "custom" {
		t.Fatalf("explicit name = %q", got)
	}
	if got := project.ContainerNameFor(Service{Name: "web"}); got != "shop-web" {
		t.Fatalf("default name = %q", got)
	}
}

func TestDiscoverAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")
	if err := os.WriteFile(path, []byte("services:\n  web:\n    image: nginx\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	found, ok := Discover(dir)
	if !ok || found != path {
		t.Fatalf("Discover = %q, %v", found, ok)
	}
	project, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	// Project name defaults to the parent directory.
	if project.Name != filepath.Base(dir) && project.Name == "" {
		t.Fatalf("expected default project name, got %q", project.Name)
	}
	if len(project.Services) != 1 || project.Services[0].Image != "nginx" {
		t.Fatalf("loaded services = %#v", project.Services)
	}
}
