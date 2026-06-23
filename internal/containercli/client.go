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
	Binary  string
	Runner  Runner
	Timeout time.Duration
}

func New(binary string) *Client {
	return &Client{
		Binary:  binary,
		Runner:  ExecRunner{},
		Timeout: 15 * time.Second,
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

func (c *Client) Start(ctx context.Context, id string) error {
	_, err := c.run(ctx, "start", id)
	return err
}

func (c *Client) Stop(ctx context.Context, id string) error {
	_, err := c.run(ctx, "stop", id)
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

func (c *Client) PruneImages(ctx context.Context, all bool) error {
	args := []string{"image", "prune"}
	if all {
		args = append(args, "--all")
	}
	_, err := c.run(ctx, args...)
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
	if c.Runner == nil {
		c.Runner = ExecRunner{}
	}
	binary := c.Binary
	if binary == "" {
		binary = "container"
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return c.Runner.Run(runCtx, binary, args...)
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
