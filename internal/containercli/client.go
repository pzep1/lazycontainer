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

func (c *Client) MachineShellCommand(id string) (*exec.Cmd, error) {
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("machine id is required")
	}
	return exec.Command(c.binaryName(), "machine", "run", "--interactive", "--tty", "--name", id), nil
}

func (c *Client) PullImage(ctx context.Context, reference string) error {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return errors.New("image reference is required")
	}
	_, err := c.runLong(ctx, "image", "pull", "--progress", "plain", reference)
	return err
}

func (c *Client) RunImage(ctx context.Context, image string, name string) error {
	image = strings.TrimSpace(image)
	if image == "" {
		return errors.New("image is required")
	}
	args := []string{"run", "--detach"}
	if strings.TrimSpace(name) != "" {
		args = append(args, "--name", strings.TrimSpace(name))
	}
	args = append(args, image)
	_, err := c.runLong(ctx, args...)
	return err
}

func (c *Client) BuildImage(ctx context.Context, tag string, contextDir string) error {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return errors.New("image tag is required")
	}
	contextDir = strings.TrimSpace(contextDir)
	if contextDir == "" {
		contextDir = "."
	}
	_, err := c.runLong(ctx, "build", "--progress", "plain", "--tag", tag, contextDir)
	return err
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

func (c *Client) PushImage(ctx context.Context, reference string) error {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return errors.New("image reference is required")
	}
	_, err := c.runLong(ctx, "image", "push", "--progress", "plain", reference)
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
