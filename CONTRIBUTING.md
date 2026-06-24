# Contributing to lazycontainer

Thanks for your interest in improving lazycontainer! This is a small project, so the
bar for contributions is: does it work, is it tested, and does it match the existing
style?

## Dev setup

lazycontainer is a Go TUI for Apple's [`container`](https://github.com/apple/container)
CLI. You'll need Go 1.26+ (see `go.mod`).

```sh
go run ./cmd/lazycontainer                      # run the TUI
go build -o bin/lazycontainer ./cmd/lazycontainer    # build a binary
go test ./...                                   # run tests
```

## Pull requests

- Keep changes focused — one logical change per PR when possible.
- Run `gofmt`, `go vet`, and `go test ./...` before opening a PR.
- Add or update tests when changing behavior.

## Homebrew / releases

See [docs/homebrew.md](docs/homebrew.md) for tap maintenance and release steps.
