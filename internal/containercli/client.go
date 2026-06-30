package containercli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Runner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

type Client struct {
	Binary      string
	Runner      Runner
	Timeout     time.Duration
	LongTimeout time.Duration
}

func New(binary string) *Client {
	return &Client{
		Binary:      binary,
		Runner:      ExecRunner{},
		Timeout:     15 * time.Second,
		LongTimeout: 30 * time.Minute,
	}
}

func (c *Client) SystemStatus(ctx context.Context) (SystemStatus, error) {
	var status SystemStatus
	err := c.runJSON(ctx, &status, "system", "status", "--format", "json")
	return status, err
}

func (c *Client) SystemDiskUsage(ctx context.Context) (SystemDiskUsage, error) {
	var usage SystemDiskUsage
	err := c.runJSON(ctx, &usage, "system", "df", "--format", "json")
	return usage, err
}

func (c *Client) SystemVersion(ctx context.Context) ([]SystemVersion, error) {
	var versions []SystemVersion
	err := c.runJSON(ctx, &versions, "system", "version", "--format", "json")
	return versions, err
}

// SystemDNS lists the local DNS domains registered with the container subsystem.
func (c *Client) SystemDNS(ctx context.Context) ([]SystemDNSDomain, error) {
	var domains []SystemDNSDomain
	err := c.runJSON(ctx, &domains, "system", "dns", "list", "--format", "json")
	return domains, err
}

// SystemProperties lists host-level container subsystem properties.
func (c *Client) SystemProperties(ctx context.Context) ([]SystemProperty, error) {
	var properties []SystemProperty
	err := c.runJSON(ctx, &properties, "system", "property", "list", "--format", "json")
	return properties, err
}

func (c *Client) Containers(ctx context.Context) ([]Container, error) {
	var containers []Container
	err := c.runJSON(ctx, &containers, "list", "--all", "--format", "json")
	return containers, err
}

func (c *Client) Images(ctx context.Context) ([]Image, error) {
	var images []Image
	err := c.runJSON(ctx, &images, "image", "list", "--format", "json", "--verbose")
	return images, err
}

func (c *Client) Volumes(ctx context.Context) ([]Volume, error) {
	var volumes []Volume
	err := c.runJSON(ctx, &volumes, "volume", "list", "--format", "json")
	return volumes, err
}

func (c *Client) Networks(ctx context.Context) ([]NetworkResource, error) {
	var networks []NetworkResource
	err := c.runJSON(ctx, &networks, "network", "list", "--format", "json")
	return networks, err
}

func (c *Client) Machines(ctx context.Context) ([]Machine, error) {
	var machines []Machine
	err := c.runJSON(ctx, &machines, "machine", "list", "--format", "json")
	return machines, err
}

func (c *Client) Registries(ctx context.Context) ([]RegistryLogin, error) {
	var registries []RegistryLogin
	err := c.runJSON(ctx, &registries, "registry", "list", "--format", "json")
	return registries, err
}

func (c *Client) BuilderStatus(ctx context.Context) (BuilderStatus, error) {
	var status BuilderStatus
	err := c.runJSON(ctx, &status, "builder", "status", "--format", "json")
	return status, err
}

func (c *Client) Stats(ctx context.Context, containerIDs ...string) ([]Stat, error) {
	args := append([]string{"stats"}, containerIDs...)
	args = append(args, "--format", "json", "--no-stream")
	var stats []Stat
	err := c.runJSON(ctx, &stats, args...)
	return stats, err
}

func (c *Client) Logs(ctx context.Context, id string, lines int) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", errors.New("container id is required")
	}
	if lines <= 0 {
		lines = 200
	}
	output, err := c.run(ctx, "logs", "-n", fmt.Sprintf("%d", lines), id)
	return string(output), err
}

func (c *Client) FollowLogsCommand(id string, lines int) (*exec.Cmd, error) {
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("container id is required")
	}
	if lines <= 0 {
		lines = 200
	}
	return exec.Command(c.binaryName(), "logs", "--follow", "-n", fmt.Sprintf("%d", lines), id), nil
}

// BootLogs returns a container's VM boot output via `container logs --boot`.
// Apple runs each container in its own lightweight VM, so boot logs are a
// per-container diagnostic with no Docker equivalent.
func (c *Client) BootLogs(ctx context.Context, id string, lines int) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", errors.New("container id is required")
	}
	if lines <= 0 {
		lines = 200
	}
	output, err := c.run(ctx, "logs", "--boot", "-n", fmt.Sprintf("%d", lines), id)
	return string(output), err
}

// MachineBootLogs returns a machine VM's boot output via
// `container machine logs --boot`.
func (c *Client) MachineBootLogs(ctx context.Context, id string, lines int) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", errors.New("machine id is required")
	}
	if lines <= 0 {
		lines = 200
	}
	output, err := c.run(ctx, "machine", "logs", "--boot", "-n", fmt.Sprintf("%d", lines), id)
	return string(output), err
}

func (c *Client) MachineLogs(ctx context.Context, id string, lines int) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", errors.New("machine id is required")
	}
	if lines <= 0 {
		lines = 200
	}
	output, err := c.run(ctx, "machine", "logs", "-n", fmt.Sprintf("%d", lines), id)
	return string(output), err
}

func (c *Client) FollowMachineLogsCommand(id string, lines int) (*exec.Cmd, error) {
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("machine id is required")
	}
	if lines <= 0 {
		lines = 200
	}
	return exec.Command(c.binaryName(), "machine", "logs", "--follow", "-n", fmt.Sprintf("%d", lines), id), nil
}

func (c *Client) SystemLogs(ctx context.Context, last string) (string, error) {
	last = strings.TrimSpace(last)
	if last == "" {
		last = "5m"
	}
	output, err := c.run(ctx, "system", "logs", "--last", last)
	return string(output), err
}

func (c *Client) FollowSystemLogsCommand(last string) (*exec.Cmd, error) {
	last = strings.TrimSpace(last)
	if last == "" {
		last = "5m"
	}
	return exec.Command(c.binaryName(), "system", "logs", "--follow", "--last", last), nil
}

func (c *Client) InspectContainer(ctx context.Context, id string) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", errors.New("container id is required")
	}
	output, err := c.run(ctx, "inspect", id)
	return string(output), err
}

func (c *Client) InspectImage(ctx context.Context, image string) (string, error) {
	if strings.TrimSpace(image) == "" {
		return "", errors.New("image is required")
	}
	output, err := c.run(ctx, "image", "inspect", image)
	return string(output), err
}

func (c *Client) InspectVolume(ctx context.Context, volume string) (string, error) {
	if strings.TrimSpace(volume) == "" {
		return "", errors.New("volume is required")
	}
	output, err := c.run(ctx, "volume", "inspect", volume)
	return string(output), err
}

func (c *Client) InspectNetwork(ctx context.Context, network string) (string, error) {
	if strings.TrimSpace(network) == "" {
		return "", errors.New("network is required")
	}
	output, err := c.run(ctx, "network", "inspect", network)
	return string(output), err
}

func (c *Client) InspectMachine(ctx context.Context, id string) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", errors.New("machine id is required")
	}
	output, err := c.run(ctx, "machine", "inspect", id)
	return string(output), err
}

func (c *Client) ShellCommand(id string, shell string) (*exec.Cmd, error) {
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("container id is required")
	}
	if strings.TrimSpace(shell) == "" {
		shell = "/bin/sh"
	}
	return exec.Command(c.binaryName(), "exec", "--interactive", "--tty", id, shell), nil
}

func (c *Client) Exec(ctx context.Context, id string, command string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", errors.New("container id is required")
	}
	command = strings.TrimSpace(command)
	if command == "" {
		return "", errors.New("command is required")
	}
	output, err := c.run(ctx, "exec", id, "/bin/sh", "-lc", command)
	return string(output), err
}

// Top lists the processes running inside a container. Apple's container CLI has
// no dedicated top subcommand, so this runs `ps` inside the container.
func (c *Client) Top(ctx context.Context, id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", errors.New("container id is required")
	}
	output, err := c.run(ctx, "exec", id, "ps", "-ef")
	return string(output), err
}

func (c *Client) Command(ctx context.Context, args []string) (string, error) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return "", errors.New("container command is required")
	}
	output, err := c.runLong(ctx, args...)
	return string(output), err
}

// CommandProcess builds an attachable `container <args>` command for custom
// commands that take over the terminal (attach: true).
func (c *Client) CommandProcess(args []string) (*exec.Cmd, error) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return nil, errors.New("container command is required")
	}
	return exec.Command(c.binaryName(), args...), nil
}

func (c *Client) MachineShellCommand(id string) (*exec.Cmd, error) {
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("machine id is required")
	}
	return exec.Command(c.binaryName(), "machine", "run", "--interactive", "--tty", "--name", id), nil
}

func (c *Client) CreateMachine(ctx context.Context, image string, name string) error {
	image = strings.TrimSpace(image)
	if image == "" {
		return errors.New("machine image is required")
	}
	args := []string{"machine", "create", "--progress", "plain"}
	if strings.TrimSpace(name) != "" {
		args = append(args, "--name", strings.TrimSpace(name))
	}
	args = append(args, image)
	_, err := c.runLong(ctx, args...)
	return err
}

func (c *Client) SetDefaultMachine(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("machine id is required")
	}
	_, err := c.run(ctx, "machine", "set-default", id)
	return err
}

func (c *Client) SetMachine(ctx context.Context, id string, settings []string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("machine id is required")
	}
	args := []string{"machine", "set", "--name", id}
	for _, setting := range settings {
		setting = strings.TrimSpace(setting)
		if setting != "" {
			args = append(args, setting)
		}
	}
	if len(args) == 4 {
		return errors.New("machine setting is required")
	}
	_, err := c.runLong(ctx, args...)
	return err
}

func (c *Client) PullImage(ctx context.Context, reference string) (string, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return "", errors.New("image reference is required")
	}
	output, err := c.runLong(ctx, "image", "pull", "--progress", "plain", reference)
	return string(output), err
}

func (c *Client) RunImage(ctx context.Context, image string, options ContainerLaunchOptions) error {
	image = strings.TrimSpace(image)
	if image == "" {
		return errors.New("image is required")
	}
	args := []string{"run"}
	if !hasAnyFlag(options.Flags, "-d", "--detach") {
		args = append(args, "--detach")
	}
	args = appendContainerLaunchArgs(args, image, options)
	_, err := c.runLong(ctx, args...)
	return err
}

func (c *Client) CreateContainer(ctx context.Context, image string, options ContainerLaunchOptions) error {
	image = strings.TrimSpace(image)
	if image == "" {
		return errors.New("image is required")
	}
	args := []string{"create"}
	args = appendContainerLaunchArgs(args, image, options)
	_, err := c.runLong(ctx, args...)
	return err
}

func appendContainerLaunchArgs(args []string, image string, options ContainerLaunchOptions) []string {
	if strings.TrimSpace(options.Name) != "" && !hasAnyFlag(options.Flags, "--name") {
		args = append(args, "--name", strings.TrimSpace(options.Name))
	}
	for _, flag := range options.Flags {
		if strings.TrimSpace(flag) != "" {
			args = append(args, flag)
		}
	}
	args = append(args, image)
	for _, argument := range options.Arguments {
		if strings.TrimSpace(argument) != "" {
			args = append(args, argument)
		}
	}
	return args
}

func hasAnyFlag(args []string, names ...string) bool {
	for _, arg := range args {
		for _, name := range names {
			if arg == name || strings.HasPrefix(arg, name+"=") {
				return true
			}
		}
	}
	return false
}

func (c *Client) BuildImage(ctx context.Context, tag string, contextDir string) (string, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return "", errors.New("image tag is required")
	}
	contextDir = strings.TrimSpace(contextDir)
	if contextDir == "" {
		contextDir = "."
	}
	output, err := c.runLong(ctx, "build", "--progress", "plain", "--tag", tag, contextDir)
	return string(output), err
}

func (c *Client) TagImage(ctx context.Context, source string, target string) error {
	source = strings.TrimSpace(source)
	if source == "" {
		return errors.New("source image is required")
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return errors.New("target image is required")
	}
	_, err := c.run(ctx, "image", "tag", source, target)
	return err
}

func (c *Client) PushImage(ctx context.Context, reference string) (string, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return "", errors.New("image reference is required")
	}
	output, err := c.runLong(ctx, "image", "push", "--progress", "plain", reference)
	return string(output), err
}

func (c *Client) SaveImage(ctx context.Context, reference string, outputPath string) (string, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return "", errors.New("image reference is required")
	}
	outputPath = strings.TrimSpace(outputPath)
	if outputPath == "" {
		return "", errors.New("image archive path is required")
	}
	output, err := c.runLong(ctx, "image", "save", "--output", outputPath, reference)
	return string(output), err
}

func (c *Client) LoadImage(ctx context.Context, inputPath string) (string, error) {
	inputPath = strings.TrimSpace(inputPath)
	if inputPath == "" {
		return "", errors.New("image archive path is required")
	}
	output, err := c.runLong(ctx, "image", "load", "--input", inputPath)
	return string(output), err
}

func (c *Client) RegistryLoginCommand(server string, username string) (*exec.Cmd, error) {
	server = strings.TrimSpace(server)
	if server == "" {
		return nil, errors.New("registry server is required")
	}
	args := []string{"registry", "login"}
	if strings.TrimSpace(username) != "" {
		args = append(args, "--username", strings.TrimSpace(username))
	}
	args = append(args, server)
	return exec.Command(c.binaryName(), args...), nil
}

func (c *Client) LogoutRegistry(ctx context.Context, registry string) error {
	registry = strings.TrimSpace(registry)
	if registry == "" {
		return errors.New("registry server is required")
	}
	_, err := c.run(ctx, "registry", "logout", registry)
	return err
}

func (c *Client) StartBuilder(ctx context.Context) error {
	_, err := c.runLong(ctx, "builder", "start")
	return err
}

func (c *Client) StopBuilder(ctx context.Context) error {
	_, err := c.run(ctx, "builder", "stop")
	return err
}

func (c *Client) DeleteBuilder(ctx context.Context, force bool) error {
	args := []string{"builder", "delete"}
	if force {
		args = append(args, "--force")
	}
	_, err := c.runLong(ctx, args...)
	return err
}

func (c *Client) StartSystem(ctx context.Context) error {
	_, err := c.runLong(ctx, "system", "start")
	return err
}

func (c *Client) StopSystem(ctx context.Context) error {
	_, err := c.runLong(ctx, "system", "stop")
	return err
}

func (c *Client) Copy(ctx context.Context, source string, destination string) error {
	source = strings.TrimSpace(source)
	if source == "" {
		return errors.New("copy source is required")
	}
	destination = strings.TrimSpace(destination)
	if destination == "" {
		return errors.New("copy destination is required")
	}
	_, err := c.runLong(ctx, "copy", source, destination)
	return err
}

func (c *Client) ExportContainer(ctx context.Context, id string, outputPath string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("container id is required")
	}
	outputPath = strings.TrimSpace(outputPath)
	if outputPath == "" {
		return errors.New("export output path is required")
	}
	_, err := c.runLong(ctx, "export", "--output", outputPath, id)
	return err
}

func (c *Client) Start(ctx context.Context, id string) error {
	_, err := c.run(ctx, "start", id)
	return err
}

func (c *Client) Stop(ctx context.Context, id string) error {
	_, err := c.run(ctx, "stop", id)
	return err
}

func (c *Client) Restart(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("container id is required")
	}
	if _, err := c.run(ctx, "stop", id); err != nil {
		return err
	}
	_, err := c.run(ctx, "start", id)
	return err
}

func (c *Client) StopMachine(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("machine id is required")
	}
	_, err := c.runLong(ctx, "machine", "stop", id)
	return err
}

func (c *Client) Kill(ctx context.Context, id string) error {
	_, err := c.run(ctx, "kill", id)
	return err
}

// StopAll gracefully stops every running container via `container stop --all`.
func (c *Client) StopAll(ctx context.Context) error {
	_, err := c.runLong(ctx, "stop", "--all")
	return err
}

// KillAll force-signals every running container via `container kill --all`.
func (c *Client) KillAll(ctx context.Context) error {
	_, err := c.run(ctx, "kill", "--all")
	return err
}

// DeleteAllContainers removes every container via `container delete --all`,
// adding --force so running containers are removed too.
func (c *Client) DeleteAllContainers(ctx context.Context, force bool) error {
	args := []string{"delete", "--all"}
	if force {
		args = append(args, "--force")
	}
	_, err := c.runLong(ctx, args...)
	return err
}

func (c *Client) DeleteContainer(ctx context.Context, id string, force bool) error {
	args := []string{"delete"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, id)
	_, err := c.run(ctx, args...)
	return err
}

func (c *Client) DeleteImage(ctx context.Context, image string, force bool) error {
	args := []string{"image", "delete"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, image)
	_, err := c.run(ctx, args...)
	return err
}

func (c *Client) CreateVolume(ctx context.Context, name string, size string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("volume name is required")
	}
	args := []string{"volume", "create"}
	if strings.TrimSpace(size) != "" {
		args = append(args, "-s", strings.TrimSpace(size))
	}
	args = append(args, name)
	_, err := c.run(ctx, args...)
	return err
}

func (c *Client) CreateNetwork(ctx context.Context, name string, subnet string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("network name is required")
	}
	args := []string{"network", "create"}
	if strings.TrimSpace(subnet) != "" {
		args = append(args, "--subnet", strings.TrimSpace(subnet))
	}
	args = append(args, name)
	_, err := c.run(ctx, args...)
	return err
}

func (c *Client) DeleteVolume(ctx context.Context, volume string) error {
	_, err := c.run(ctx, "volume", "delete", volume)
	return err
}

func (c *Client) DeleteNetwork(ctx context.Context, network string) error {
	_, err := c.run(ctx, "network", "delete", network)
	return err
}

func (c *Client) DeleteMachine(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("machine id is required")
	}
	_, err := c.runLong(ctx, "machine", "delete", id)
	return err
}

func (c *Client) PruneContainers(ctx context.Context) error {
	_, err := c.run(ctx, "prune")
	return err
}

func (c *Client) PruneImages(ctx context.Context, all bool) error {
	args := []string{"image", "prune"}
	if all {
		args = append(args, "--all")
	}
	_, err := c.run(ctx, args...)
	return err
}

func (c *Client) PruneVolumes(ctx context.Context) error {
	_, err := c.run(ctx, "volume", "prune")
	return err
}

func (c *Client) PruneNetworks(ctx context.Context) error {
	_, err := c.run(ctx, "network", "prune")
	return err
}

func (c *Client) runJSON(ctx context.Context, target any, args ...string) error {
	output, err := c.run(ctx, args...)
	if err != nil {
		return err
	}
	return decodeJSON(output, target)
}

func (c *Client) run(ctx context.Context, args ...string) ([]byte, error) {
	return c.runWithTimeout(ctx, c.Timeout, args...)
}

func (c *Client) runLong(ctx context.Context, args ...string) ([]byte, error) {
	return c.runWithTimeout(ctx, c.LongTimeout, args...)
}

func (c *Client) runWithTimeout(ctx context.Context, timeout time.Duration, args ...string) ([]byte, error) {
	if c.Runner == nil {
		c.Runner = ExecRunner{}
	}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return c.Runner.Run(runCtx, c.binaryName(), args...)
}

func (c *Client) binaryName() string {
	if c.Binary == "" {
		return "container"
	}
	return c.Binary
}

func decodeJSON(output []byte, target any) error {
	trimmed := bytes.TrimSpace(output)
	if len(trimmed) == 0 {
		trimmed = []byte("null")
	}
	if err := json.Unmarshal(trimmed, target); err != nil {
		return fmt.Errorf("decode container JSON: %w", err)
	}
	return nil
}
