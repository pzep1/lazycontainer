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
	calls  [][]string
}

func (f *fakeRunner) Run(_ context.Context, _ string, args ...string) ([]byte, error) {
	f.args = append([]string(nil), args...)
	f.calls = append(f.calls, append([]string(nil), args...))
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

func TestSystemDiskUsageParsesAppleJSONShape(t *testing.T) {
	runner := &fakeRunner{output: []byte(`{
		"containers": {"active": 1, "reclaimable": 5833977856, "sizeInBytes": 6589943808, "total": 9},
		"images": {"active": 4, "reclaimable": 2625372160, "sizeInBytes": 14597648384, "total": 6},
		"volumes": {"active": 8, "reclaimable": 0, "sizeInBytes": 16260087808, "total": 8}
	}`)}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	usage, err := client.SystemDiskUsage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if usage.Containers.Total != 9 || usage.Images.Active != 4 || usage.Volumes.SizeInBytes != 16260087808 {
		t.Fatalf("unexpected system disk usage: %#v", usage)
	}
	if usage.TotalSize() != "34.9 GB" {
		t.Fatalf("unexpected total size %q", usage.TotalSize())
	}
	if usage.TotalReclaimable() != "7.9 GB" {
		t.Fatalf("unexpected reclaimable size %q", usage.TotalReclaimable())
	}

	wantArgs := []string{"system", "df", "--format", "json"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestSystemVersionParsesAppleJSONShape(t *testing.T) {
	runner := &fakeRunner{output: []byte(`[
		{"appName":"container","buildType":"release","commit":"ee848e3ebfd7c73b04dd419683be54fb450b8779","version":"1.0.0"},
		{"appName":"container-apiserver","buildType":"release","commit":"ee848e3ebfd7c73b04dd419683be54fb450b8779","version":"container-apiserver version 1.0.0 (build: release, commit: ee848e3)"}
	]`)}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	versions, err := client.SystemVersion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}
	if versions[0].AppName != "container" || versions[0].Version != "1.0.0" {
		t.Fatalf("unexpected first version: %#v", versions[0])
	}

	wantArgs := []string{"system", "version", "--format", "json"}
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
				{
					"digest": "sha256:def",
					"platform": {"os": "linux", "architecture": "arm64", "variant": "v8"},
					"size": 4203982,
					"config": {
						"architecture": "arm64",
						"os": "linux",
						"variant": "v8",
						"history": [
							{"created_by": "ADD alpine-minirootfs-3.24.0-aarch64.tar.gz / # buildkit"},
							{"created_by": "CMD [\"/bin/sh\"]", "empty_layer": true}
						],
						"rootfs": {
							"type": "layers",
							"diff_ids": ["sha256:375591c23c8de111a75382d674cf6688f56adecb5e3018d29ada57c10135db5e"]
						}
					}
				}
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
	if len(images[0].Variants[0].Config.History) != 2 {
		t.Fatalf("expected image history entries, got %d", len(images[0].Variants[0].Config.History))
	}
	if len(images[0].Variants[0].Config.RootFS.DiffIDs) != 1 {
		t.Fatalf("expected rootfs layer entries, got %d", len(images[0].Variants[0].Config.RootFS.DiffIDs))
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

func TestMachinesParsesFlexibleAppleJSONShape(t *testing.T) {
	runner := &fakeRunner{output: []byte(`[
		{
			"id": "dev-machine",
			"default": true,
			"configuration": {
				"name": "dev-machine",
				"image": {"reference": "docker.io/library/alpine:3.22"},
				"resources": {"cpus": 2, "memoryInBytes": 2147483648},
				"creationDate": "2026-06-20T12:00:00Z"
			},
			"status": {"state": "running"}
		}
	]`)}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	machines, err := client.Machines(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(machines) != 1 {
		t.Fatalf("expected 1 machine, got %d", len(machines))
	}
	if machines[0].Name() != "dev-machine" {
		t.Fatalf("unexpected machine name %q", machines[0].Name())
	}
	if machines[0].State() != "running" {
		t.Fatalf("unexpected machine state %q", machines[0].State())
	}
	if machines[0].Image() != "docker.io/library/alpine:3.22" {
		t.Fatalf("unexpected machine image %q", machines[0].Image())
	}
	if machines[0].Memory() != "2.0 GB" {
		t.Fatalf("unexpected machine memory %q", machines[0].Memory())
	}

	wantArgs := []string{"machine", "list", "--format", "json"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestRegistriesParsesAppleJSONShape(t *testing.T) {
	runner := &fakeRunner{output: []byte(`[
		{"server":"ghcr.io","username":"alice","scheme":"https"},
		"docker.io"
	]`)}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	registries, err := client.Registries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(registries) != 2 {
		t.Fatalf("expected 2 registry logins, got %d", len(registries))
	}
	if registries[0].Name() != "ghcr.io" {
		t.Fatalf("unexpected registry name %q", registries[0].Name())
	}
	if registries[0].User() != "alice" {
		t.Fatalf("unexpected registry user %q", registries[0].User())
	}
	if registries[0].RegistryScheme() != "https" {
		t.Fatalf("unexpected registry scheme %q", registries[0].RegistryScheme())
	}
	if registries[1].Name() != "docker.io" {
		t.Fatalf("unexpected string registry name %q", registries[1].Name())
	}

	wantArgs := []string{"registry", "list", "--format", "json"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestBuilderStatusParsesEmptyAppleJSONShape(t *testing.T) {
	runner := &fakeRunner{output: []byte(`[]`)}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	builder, err := client.BuilderStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if builder.Present {
		t.Fatalf("expected no builder to be present")
	}
	if builder.State() != "not created" {
		t.Fatalf("unexpected builder state %q", builder.State())
	}

	wantArgs := []string{"builder", "status", "--format", "json"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestBuilderStatusParsesObjectShape(t *testing.T) {
	runner := &fakeRunner{output: []byte(`{
		"id":"buildkit",
		"state":"running",
		"configuration":{"resources":{"cpus":4,"memoryInBytes":4294967296}}
	}`)}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	builder, err := client.BuilderStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !builder.Present {
		t.Fatalf("expected builder to be present")
	}
	if builder.Name() != "buildkit" {
		t.Fatalf("unexpected builder name %q", builder.Name())
	}
	if builder.State() != "running" {
		t.Fatalf("unexpected builder state %q", builder.State())
	}
	if builder.CPUs() != "4" {
		t.Fatalf("unexpected builder cpus %q", builder.CPUs())
	}
	if builder.Memory() != "4.0 GB" {
		t.Fatalf("unexpected builder memory %q", builder.Memory())
	}
}

func TestShellCommandUsesInteractiveTTYExec(t *testing.T) {
	client := &Client{Binary: "container"}

	cmd, err := client.ShellCommand("db", "/bin/sh")
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"exec", "--interactive", "--tty", "db", "/bin/sh"}
	if cmd.Args[0] != "container" {
		t.Fatalf("expected container binary, got %q", cmd.Args[0])
	}
	if !reflect.DeepEqual(cmd.Args[1:], wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, cmd.Args[1:])
	}
}

func TestExecRunsShellCommandInContainer(t *testing.T) {
	runner := &fakeRunner{output: []byte("ok\n")}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	output, err := client.Exec(context.Background(), "db", "cat /etc/os-release")
	if err != nil {
		t.Fatal(err)
	}
	if output != "ok\n" {
		t.Fatalf("unexpected output %q", output)
	}

	wantArgs := []string{"exec", "db", "/bin/sh", "-lc", "cat /etc/os-release"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestCommandRunsArbitraryContainerArgs(t *testing.T) {
	runner := &fakeRunner{output: []byte("image list\n")}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	output, err := client.Command(context.Background(), []string{"image", "list", "--format", "json"})
	if err != nil {
		t.Fatal(err)
	}
	if output != "image list\n" {
		t.Fatalf("unexpected output %q", output)
	}

	wantArgs := []string{"image", "list", "--format", "json"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestCommandRequiresArgs(t *testing.T) {
	runner := &fakeRunner{}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	if _, err := client.Command(context.Background(), []string{"", "  "}); err == nil {
		t.Fatalf("expected empty command to fail")
	}
}

func TestCommandPreservesEmptyArguments(t *testing.T) {
	runner := &fakeRunner{}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	_, err := client.Command(context.Background(), []string{"run", "--env", ""})
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"run", "--env", ""}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestRegistryLoginCommandUsesServerAndOptionalUsername(t *testing.T) {
	client := &Client{Binary: "container"}

	cmd, err := client.RegistryLoginCommand("ghcr.io", "alice")
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"registry", "login", "--username", "alice", "ghcr.io"}
	if !reflect.DeepEqual(cmd.Args[1:], wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, cmd.Args[1:])
	}
}

func TestLogoutRegistryUsesSelectedRegistry(t *testing.T) {
	runner := &fakeRunner{}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	if err := client.LogoutRegistry(context.Background(), "ghcr.io"); err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"registry", "logout", "ghcr.io"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestBuilderLifecycleCommandsUseAppleSubcommands(t *testing.T) {
	runner := &fakeRunner{}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second, LongTimeout: time.Second}

	if err := client.StartBuilder(context.Background()); err != nil {
		t.Fatal(err)
	}
	wantStart := []string{"builder", "start"}
	if !reflect.DeepEqual(runner.args, wantStart) {
		t.Fatalf("start args mismatch\nwant: %#v\n got: %#v", wantStart, runner.args)
	}

	if err := client.StopBuilder(context.Background()); err != nil {
		t.Fatal(err)
	}
	wantStop := []string{"builder", "stop"}
	if !reflect.DeepEqual(runner.args, wantStop) {
		t.Fatalf("stop args mismatch\nwant: %#v\n got: %#v", wantStop, runner.args)
	}

	if err := client.DeleteBuilder(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	wantDelete := []string{"builder", "delete", "--force"}
	if !reflect.DeepEqual(runner.args, wantDelete) {
		t.Fatalf("delete args mismatch\nwant: %#v\n got: %#v", wantDelete, runner.args)
	}
}

func TestMachineCommandsUseSelectedMachineID(t *testing.T) {
	client := &Client{Binary: "container"}

	logs, err := client.FollowMachineLogsCommand("dev-machine", 50)
	if err != nil {
		t.Fatal(err)
	}
	wantLogs := []string{"machine", "logs", "--follow", "-n", "50", "dev-machine"}
	if !reflect.DeepEqual(logs.Args[1:], wantLogs) {
		t.Fatalf("machine logs args mismatch\nwant: %#v\n got: %#v", wantLogs, logs.Args[1:])
	}

	shell, err := client.MachineShellCommand("dev-machine")
	if err != nil {
		t.Fatal(err)
	}
	wantShell := []string{"machine", "run", "--interactive", "--tty", "--name", "dev-machine"}
	if !reflect.DeepEqual(shell.Args[1:], wantShell) {
		t.Fatalf("machine shell args mismatch\nwant: %#v\n got: %#v", wantShell, shell.Args[1:])
	}

	runner := &fakeRunner{output: []byte("booted\n")}
	client = &Client{Binary: "container", Runner: runner, Timeout: time.Second, LongTimeout: time.Second}
	body, err := client.MachineLogs(context.Background(), "dev-machine", 50)
	if err != nil {
		t.Fatal(err)
	}
	if body != "booted\n" {
		t.Fatalf("unexpected machine logs body %q", body)
	}
	wantTail := []string{"machine", "logs", "-n", "50", "dev-machine"}
	if !reflect.DeepEqual(runner.args, wantTail) {
		t.Fatalf("machine logs args mismatch\nwant: %#v\n got: %#v", wantTail, runner.args)
	}

	if _, err := client.InspectMachine(context.Background(), "dev-machine"); err != nil {
		t.Fatal(err)
	}
	wantInspect := []string{"machine", "inspect", "dev-machine"}
	if !reflect.DeepEqual(runner.args, wantInspect) {
		t.Fatalf("machine inspect args mismatch\nwant: %#v\n got: %#v", wantInspect, runner.args)
	}

	if err := client.StopMachine(context.Background(), "dev-machine"); err != nil {
		t.Fatal(err)
	}
	wantStop := []string{"machine", "stop", "dev-machine"}
	if !reflect.DeepEqual(runner.args, wantStop) {
		t.Fatalf("machine stop args mismatch\nwant: %#v\n got: %#v", wantStop, runner.args)
	}

	if err := client.DeleteMachine(context.Background(), "dev-machine"); err != nil {
		t.Fatal(err)
	}
	wantDelete := []string{"machine", "delete", "dev-machine"}
	if !reflect.DeepEqual(runner.args, wantDelete) {
		t.Fatalf("machine delete args mismatch\nwant: %#v\n got: %#v", wantDelete, runner.args)
	}
}

func TestSystemCommandsUseAppleSubcommands(t *testing.T) {
	client := &Client{Binary: "container"}

	follow, err := client.FollowSystemLogsCommand("10m")
	if err != nil {
		t.Fatal(err)
	}
	wantFollow := []string{"system", "logs", "--follow", "--last", "10m"}
	if !reflect.DeepEqual(follow.Args[1:], wantFollow) {
		t.Fatalf("system follow logs args mismatch\nwant: %#v\n got: %#v", wantFollow, follow.Args[1:])
	}

	runner := &fakeRunner{output: []byte("system ready\n")}
	client = &Client{Binary: "container", Runner: runner, Timeout: time.Second, LongTimeout: time.Second}
	body, err := client.SystemLogs(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if body != "system ready\n" {
		t.Fatalf("unexpected system logs body %q", body)
	}
	wantLogs := []string{"system", "logs", "--last", "5m"}
	if !reflect.DeepEqual(runner.args, wantLogs) {
		t.Fatalf("system logs args mismatch\nwant: %#v\n got: %#v", wantLogs, runner.args)
	}

	if err := client.StartSystem(context.Background()); err != nil {
		t.Fatal(err)
	}
	wantStart := []string{"system", "start"}
	if !reflect.DeepEqual(runner.args, wantStart) {
		t.Fatalf("system start args mismatch\nwant: %#v\n got: %#v", wantStart, runner.args)
	}

	if err := client.StopSystem(context.Background()); err != nil {
		t.Fatal(err)
	}
	wantStop := []string{"system", "stop"}
	if !reflect.DeepEqual(runner.args, wantStop) {
		t.Fatalf("system stop args mismatch\nwant: %#v\n got: %#v", wantStop, runner.args)
	}
}

func TestCreateMachineUsesPlainProgressAndOptionalName(t *testing.T) {
	runner := &fakeRunner{}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	if err := client.CreateMachine(context.Background(), "alpine:3.22", "dev-machine"); err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"machine", "create", "--progress", "plain", "--name", "dev-machine", "alpine:3.22"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestSetDefaultMachineUsesSelectedMachineID(t *testing.T) {
	runner := &fakeRunner{}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	if err := client.SetDefaultMachine(context.Background(), "dev-machine"); err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"machine", "set-default", "dev-machine"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestSetMachineUsesSelectedMachineAndSettings(t *testing.T) {
	runner := &fakeRunner{}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second, LongTimeout: time.Second}

	settings := []string{"cpus=4", "memory=8G", "home-mount=ro"}
	if err := client.SetMachine(context.Background(), "dev-machine", settings); err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"machine", "set", "--name", "dev-machine", "cpus=4", "memory=8G", "home-mount=ro"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestFollowLogsCommandUsesFollowWithTail(t *testing.T) {
	client := &Client{Binary: "container"}

	cmd, err := client.FollowLogsCommand("db", 200)
	if err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"logs", "--follow", "-n", "200", "db"}
	if cmd.Args[0] != "container" {
		t.Fatalf("expected container binary, got %q", cmd.Args[0])
	}
	if !reflect.DeepEqual(cmd.Args[1:], wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, cmd.Args[1:])
	}
}

func TestPullImageUsesPlainProgress(t *testing.T) {
	runner := &fakeRunner{}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	if err := client.PullImage(context.Background(), "docker.io/library/alpine:latest"); err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"image", "pull", "--progress", "plain", "docker.io/library/alpine:latest"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestRunImageUsesDetachedContainerWithOptionalName(t *testing.T) {
	runner := &fakeRunner{}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	if err := client.RunImage(context.Background(), "docker.io/library/alpine:latest", ContainerLaunchOptions{Name: "scratch"}); err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"run", "--detach", "--name", "scratch", "docker.io/library/alpine:latest"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestCreateContainerUsesSelectedImageWithOptionalName(t *testing.T) {
	runner := &fakeRunner{}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	if err := client.CreateContainer(context.Background(), "docker.io/library/alpine:latest", ContainerLaunchOptions{Name: "scratch"}); err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"create", "--name", "scratch", "docker.io/library/alpine:latest"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestRunImageUsesLaunchOptions(t *testing.T) {
	runner := &fakeRunner{}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	options := ContainerLaunchOptions{
		Name: "web",
		Flags: []string{
			"--publish", "127.0.0.1:8080:80/tcp",
			"--env", "GREETING=hello world",
			"--volume", "cache:/cache",
			"--network", "frontend",
			"--workdir", "/app",
		},
		Arguments: []string{"npm", "start"},
	}
	if err := client.RunImage(context.Background(), "ghcr.io/example/web:latest", options); err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{
		"run", "--detach",
		"--name", "web",
		"--publish", "127.0.0.1:8080:80/tcp",
		"--env", "GREETING=hello world",
		"--volume", "cache:/cache",
		"--network", "frontend",
		"--workdir", "/app",
		"ghcr.io/example/web:latest",
		"npm", "start",
	}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestRunImageDoesNotDuplicateExplicitDetachOrName(t *testing.T) {
	runner := &fakeRunner{}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	options := ContainerLaunchOptions{
		Name:  "ignored",
		Flags: []string{"--detach", "--name", "web"},
	}
	if err := client.RunImage(context.Background(), "ghcr.io/example/web:latest", options); err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"run", "--detach", "--name", "web", "ghcr.io/example/web:latest"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestBuildImageUsesPlainProgressAndDefaultContext(t *testing.T) {
	runner := &fakeRunner{}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	if err := client.BuildImage(context.Background(), "registry.example.com/app:dev", ""); err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"build", "--progress", "plain", "--tag", "registry.example.com/app:dev", "."}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestBuildImageUsesProvidedContext(t *testing.T) {
	runner := &fakeRunner{}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	if err := client.BuildImage(context.Background(), "registry.example.com/app:dev", "./services/api"); err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"build", "--progress", "plain", "--tag", "registry.example.com/app:dev", "./services/api"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestTagImageUsesSelectedSourceAndTargetReference(t *testing.T) {
	runner := &fakeRunner{}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	if err := client.TagImage(context.Background(), "docker.io/library/alpine:latest", "registry.example.com/alpine:dev"); err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"image", "tag", "docker.io/library/alpine:latest", "registry.example.com/alpine:dev"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestPushImageUsesPlainProgress(t *testing.T) {
	runner := &fakeRunner{}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	if err := client.PushImage(context.Background(), "registry.example.com/alpine:dev"); err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"image", "push", "--progress", "plain", "registry.example.com/alpine:dev"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestSaveImageUsesOutputPath(t *testing.T) {
	runner := &fakeRunner{}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	if err := client.SaveImage(context.Background(), "docker.io/library/alpine:latest", "alpine.tar"); err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"image", "save", "--output", "alpine.tar", "docker.io/library/alpine:latest"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestLoadImageUsesInputPath(t *testing.T) {
	runner := &fakeRunner{}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	if err := client.LoadImage(context.Background(), "alpine.tar"); err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"image", "load", "--input", "alpine.tar"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestCopyUsesSourceAndDestination(t *testing.T) {
	runner := &fakeRunner{}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	if err := client.Copy(context.Background(), "db:/etc/hosts", "./hosts"); err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"copy", "db:/etc/hosts", "./hosts"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestExportContainerUsesOutputPath(t *testing.T) {
	runner := &fakeRunner{}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	if err := client.ExportContainer(context.Background(), "db", "db.tar"); err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"export", "--output", "db.tar", "db"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestCreateVolumeUsesNameAndOptionalSize(t *testing.T) {
	runner := &fakeRunner{}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	if err := client.CreateVolume(context.Background(), "cache", "10G"); err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"volume", "create", "-s", "10G", "cache"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestCreateNetworkUsesNameAndOptionalSubnet(t *testing.T) {
	runner := &fakeRunner{}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	if err := client.CreateNetwork(context.Background(), "frontend", "192.168.90.0/24"); err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"network", "create", "--subnet", "192.168.90.0/24", "frontend"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}

func TestRestartStopsThenStartsContainer(t *testing.T) {
	runner := &fakeRunner{}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	if err := client.Restart(context.Background(), "db"); err != nil {
		t.Fatal(err)
	}

	wantCalls := [][]string{
		{"stop", "db"},
		{"start", "db"},
	}
	if !reflect.DeepEqual(runner.calls, wantCalls) {
		t.Fatalf("calls mismatch\nwant: %#v\n got: %#v", wantCalls, runner.calls)
	}
}

func TestPruneContainersUsesApplePruneCommand(t *testing.T) {
	runner := &fakeRunner{}
	client := &Client{Binary: "container", Runner: runner, Timeout: time.Second}

	if err := client.PruneContainers(context.Background()); err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"prune"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("args mismatch\nwant: %#v\n got: %#v", wantArgs, runner.args)
	}
}
