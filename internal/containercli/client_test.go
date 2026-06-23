package containercli

import (
	"context"
	"reflect"
	"testing"
	"time"
)

type fakeRunner struct {
	output []byte
	args   []string
}

func (f *fakeRunner) Run(_ context.Context, _ string, args ...string) ([]byte, error) {
	f.args = append([]string(nil), args...)
	return f.output, nil
}

func TestContainersParsesAppleJSONShape(t *testing.T) {
	runner := &fakeRunner{output: []byte(`[
		{
			"id": "db",
			"configuration": {
				"id": "db",
				"creationDate": "2026-06-15T08:27:31Z",
				"image": {"reference": "docker.io/library/postgres:15"},
				"platform": {"os": "linux", "architecture": "arm64"},
				"publishedPorts": [{"hostAddress": "127.0.0.1", "hostPort": 5432, "containerPort": 5432, "proto": "tcp"}],
				"resources": {"cpus": 4, "memoryInBytes": 1073741824}
			},
			"status": {"state": "running", "startedDate": "2026-06-22T08:22:47Z"}
		}
	]`)}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	containers, err := client.Containers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(containers))
	}
	got := containers[0]
	if got.Name() != "db" {
		t.Fatalf("expected name db, got %q", got.Name())
	}
	if got.ImageName() != "docker.io/library/postgres:15" {
		t.Fatalf("unexpected image %q", got.ImageName())
	}
	if got.Ports() != "127.0.0.1:5432->5432/tcp" {
		t.Fatalf("unexpected ports %q", got.Ports())
	}

	wantArgs := []string{"list", "--all", "--format", "json"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestImagesParsesVariants(t *testing.T) {
	runner := &fakeRunner{output: []byte(`[
		{
			"id": "abc",
			"configuration": {
				"name": "docker.io/library/alpine:latest",
				"creationDate": "2026-06-09T20:11:09Z",
				"descriptor": {"digest": "sha256:abc", "size": 9218}
			},
			"variants": [
				{"digest": "sha256:def", "platform": {"os": "linux", "architecture": "arm64", "variant": "v8"}, "size": 4203982}
			]
		}
	]`)}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	images, err := client.Images(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(images))
	}
	if images[0].Name() != "docker.io/library/alpine:latest" {
		t.Fatalf("unexpected name %q", images[0].Name())
	}
	if images[0].Platforms() != "linux/arm64/v8" {
		t.Fatalf("unexpected platform %q", images[0].Platforms())
	}
	if images[0].Size() != "4.0 MB" {
		t.Fatalf("unexpected size %q", images[0].Size())
	}
}

func TestVolumesParsesAppleJSONShape(t *testing.T) {
	runner := &fakeRunner{output: []byte(`[
		{
			"id": "data",
			"configuration": {
				"name": "data",
				"creationDate": "2026-06-15T08:27:31Z",
				"driver": "local",
				"format": "ext4",
				"options": {"size": "10G"},
				"sizeInBytes": 10737418240,
				"source": "/Users/example/Library/Application Support/com.apple.container/volumes/data/volume.img"
			}
		}
	]`)}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	volumes, err := client.Volumes(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(volumes) != 1 {
		t.Fatalf("expected 1 volume, got %d", len(volumes))
	}
	if volumes[0].Name() != "data" {
		t.Fatalf("unexpected volume name %q", volumes[0].Name())
	}
	if volumes[0].Size() != "10.0 GB" {
		t.Fatalf("unexpected volume size %q", volumes[0].Size())
	}

	wantArgs := []string{"volume", "list", "--format", "json"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestNetworksParsesAppleJSONShape(t *testing.T) {
	runner := &fakeRunner{output: []byte(`[
		{
			"id": "default",
			"configuration": {
				"name": "default",
				"creationDate": "2026-06-14T20:43:06Z",
				"mode": "nat",
				"plugin": "container-network-vmnet",
				"labels": {"com.apple.container.resource.role": "builtin"}
			},
			"status": {
				"ipv4Gateway": "192.168.64.1",
				"ipv4Subnet": "192.168.64.0/24",
				"ipv6Subnet": "fd00::/64"
			}
		}
	]`)}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	networks, err := client.Networks(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(networks) != 1 {
		t.Fatalf("expected 1 network, got %d", len(networks))
	}
	if networks[0].Name() != "default" {
		t.Fatalf("unexpected network name %q", networks[0].Name())
	}
	if networks[0].Status.IPv4Subnet != "192.168.64.0/24" {
		t.Fatalf("unexpected ipv4 subnet %q", networks[0].Status.IPv4Subnet)
	}

	wantArgs := []string{"network", "list", "--format", "json"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}
