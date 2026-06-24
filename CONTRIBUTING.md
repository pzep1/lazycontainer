# Contributing to lazycont

Thanks for your interest in improving lazycont! This is a small project, so the
process is light — but a few conventions keep the codebase consistent.

## Getting started

lazycont is a Go TUI for Apple's [`container`](https://github.com/apple/container)
CLI. You need:

- macOS with the `container` CLI installed (for running the app), and
- Go matching `go.mod` (1.26+) to build and test.

```sh
go run ./cmd/lazycontainer                      # run the TUI
go build -o bin/lazycontainer ./cmd/lazycontainer    # build a binary
```

## Before you open a pull request

Run the same checks CI runs, and make sure they all pass:

```sh
gofmt -l .          # should print nothing
go vet ./...
go build ./...
go test -race ./...
```

- **Formatting:** all Go is `gofmt`-clean. Match the surrounding style.
- **Tests:** add or update tests for behavior changes. The TUI is tested by
  driving the `Model` with messages and asserting on state and rendered output
  (see `internal/tui/*_test.go`); the CLI client is tested against a fake runner.
- **Keep it focused:** one logical change per PR.

## Pull requests

1. Fork and branch from `main`.
2. Make your change with tests and gofmt-clean code.
3. Open a PR against `main`. CI must pass and the PR needs one approval before
   merging. Resolve review conversations before merging.

## Commit messages

Write a concise imperative subject line (e.g. "Add Top tab for containers")
and explain the why in the body when it isn't obvious.

## License

By contributing, you agree that your contributions are licensed under the
project's [GNU General Public License v3.0 or later](LICENSE).
