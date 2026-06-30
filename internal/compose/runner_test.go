package compose

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func runnerProject() Project {
	return Project{
		Name:     "shop",
		Services: []Service{{Name: "web", Image: "nginx"}},
	}
}

func TestRunnerDownIsIdempotentOnNotFound(t *testing.T) {
	p := runnerProject()
	var calls [][]string
	r := Runner{Project: p, Exec: func(_ context.Context, args []string) (string, error) {
		calls = append(calls, args)
		return "Error: container not found", errors.New("container not found")
	}}
	if err := r.DownService(context.Background(), p.Services[0]); err != nil {
		t.Fatalf("teardown of an absent container should succeed, got %v", err)
	}
	want := [][]string{{"stop", "shop-web"}, {"delete", "--force", "shop-web"}}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("down calls = %#v, want %#v", calls, want)
	}
}

func TestRunnerDownSurfacesRealError(t *testing.T) {
	p := runnerProject()
	r := Runner{Project: p, Exec: func(_ context.Context, _ []string) (string, error) {
		return "Error: permission denied", errors.New("permission denied")
	}}
	if err := r.DownService(context.Background(), p.Services[0]); err == nil {
		t.Fatalf("a non-not-found teardown failure must be surfaced, not swallowed")
	}
}

func TestRunnerUpSurfacesError(t *testing.T) {
	p := runnerProject()
	r := Runner{Project: p, Exec: func(_ context.Context, _ []string) (string, error) {
		return "boom", errors.New("daemon unavailable")
	}}
	if err := r.UpService(context.Background(), p.Services[0]); err == nil {
		t.Fatalf("bring-up must surface a run failure")
	}
}

func TestRunnerRecreateSurfacesBringUpError(t *testing.T) {
	p := runnerProject()
	// Teardown reports not-found (idempotent); the run leg conflicts → recreate
	// must surface the bring-up error rather than report success.
	r := Runner{Project: p, Exec: func(_ context.Context, args []string) (string, error) {
		if len(args) > 0 && args[0] == "run" {
			return "name already in use", errors.New("conflict")
		}
		return "not found", errors.New("not found")
	}}
	if err := r.RecreateService(context.Background(), p.Services[0]); err == nil {
		t.Fatalf("recreate must surface a bring-up failure")
	}
}

func TestIsNotFoundClassification(t *testing.T) {
	notFound := []string{"Error: container not found", "no such container", "does not exist", "container is not running"}
	for _, msg := range notFound {
		if !isNotFound(errors.New(msg), "") {
			t.Fatalf("isNotFound(%q) should be true", msg)
		}
	}
	if isNotFound(errors.New("permission denied"), "") {
		t.Fatalf("a real error must not be classified as not-found")
	}
	if isNotFound(nil, "not found") {
		t.Fatalf("a nil error is never not-found")
	}
	// Markers may appear only in the command output, not the error string.
	if !isNotFound(errors.New("exit status 1"), "no such container: shop-web") {
		t.Fatalf("not-found marker in output should classify as not-found")
	}
}
