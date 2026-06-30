package compose

import (
	"context"
	"strings"
)

// Exec runs a single `container` invocation and returns its combined output and
// error. It matches containercli.Client.Command, so the TUI injects the real
// client without the compose package depending on it.
type Exec func(ctx context.Context, args []string) (string, error)

// Runner executes compose operations for a project through an injected
// `container` executor. It owns the execution semantics so the UI never has to
// loop over raw command steps: bring-up failures are surfaced immediately, and
// teardown is idempotent — stopping or deleting an already-absent container is
// treated as success, but any other failure (permissions, daemon down,
// unexpected state) is returned so the UI cannot report a teardown that did not
// actually happen.
type Runner struct {
	Project Project
	Exec    Exec
}

// run executes one step and surfaces any error.
func (r Runner) run(ctx context.Context, args []string) error {
	_, err := r.Exec(ctx, args)
	return err
}

// runIdempotent executes a teardown step, swallowing only "already absent"
// failures so a down/recreate is repeatable.
func (r Runner) runIdempotent(ctx context.Context, args []string) error {
	out, err := r.Exec(ctx, args)
	if err == nil || isNotFound(err, out) {
		return nil
	}
	return err
}

func (r Runner) runAll(ctx context.Context, steps [][]string) error {
	for _, step := range steps {
		if err := r.run(ctx, step); err != nil {
			return err
		}
	}
	return nil
}

func (r Runner) runAllIdempotent(ctx context.Context, steps [][]string) error {
	for _, step := range steps {
		if err := r.runIdempotent(ctx, step); err != nil {
			return err
		}
	}
	return nil
}

// UpService builds (when needed) and starts a single service's container.
func (r Runner) UpService(ctx context.Context, service Service) error {
	return r.runAll(ctx, r.Project.UpArgs(service))
}

// UpAll brings the whole project up in dependency order.
func (r Runner) UpAll(ctx context.Context) error {
	return r.runAll(ctx, r.Project.UpAll())
}

// DownService stops and removes a single service's container (idempotent).
func (r Runner) DownService(ctx context.Context, service Service) error {
	return r.runAllIdempotent(ctx, r.Project.DownArgs(service))
}

// DownAll takes the whole project down in reverse dependency order (idempotent).
func (r Runner) DownAll(ctx context.Context) error {
	return r.runAllIdempotent(ctx, r.Project.DownAll())
}

// StartService / StopService start and stop an existing service container.
func (r Runner) StartService(ctx context.Context, service Service) error {
	return r.run(ctx, r.Project.StartArgs(service))
}

func (r Runner) StopService(ctx context.Context, service Service) error {
	return r.run(ctx, r.Project.StopArgs(service))
}

// RestartService stops then starts a service container.
func (r Runner) RestartService(ctx context.Context, service Service) error {
	return r.runAll(ctx, r.Project.RestartArgs(service))
}

// RecreateService tears a service down (idempotent, since it may not exist yet)
// and brings it back up, surfacing any bring-up error.
func (r Runner) RecreateService(ctx context.Context, service Service) error {
	if err := r.DownService(ctx, service); err != nil {
		return err
	}
	return r.UpService(ctx, service)
}

// notFoundMarkers are the substrings that indicate a stop/delete targeted an
// already-absent container — the only teardown failure treated as success.
var notFoundMarkers = []string{
	"not found",
	"no such",
	"does not exist",
	"doesn't exist",
	"not running",
	"no container",
	"cannot find",
}

// isNotFound reports whether a teardown error (or its command output) indicates
// the target container was already absent.
func isNotFound(err error, output string) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error() + "\n" + output)
	for _, marker := range notFoundMarkers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}
