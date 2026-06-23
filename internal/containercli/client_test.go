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

	if err := client.RunImage(context.Background(), "docker.io/library/alpine:latest", "scratch"); err != nil {
		t.Fatal(err)
	}

	wantArgs := []string{"run", "--detach", "--name", "scratch", "docker.io/library/alpine:latest"}
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
