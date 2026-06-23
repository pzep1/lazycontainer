# lazycont

A lazydocker-style terminal UI for Apple's [`container`](https://github.com/apple/container) CLI.

This is an early usable slice focused on day-to-day container work:

- browse containers and images
- browse volumes and networks
- browse image builder status
- browse container machines
- browse registry logins
- browse Apple container system diagnostics
- filter resource lists across names and metadata
- inspect selected resources
- view image variant and layer history
- tail container, machine, or system logs
- follow container, machine, or system logs in the terminal
- run one-off commands in selected running containers
- view container CPU, memory, network, disk, and process metrics
- export a selected container filesystem as a tar archive
- copy files or folders between a selected container and the local filesystem
- build local images from a Dockerfile or Containerfile
- start, stop, and delete the image builder
- retag selected images
- save and load image archives
- push selected images to a registry
- log in to and out of registries
- pull images, create stopped containers, and run selected images with ports, env vars, mounts, networks, and command args
- start, restart, stop, kill, and delete containers
- create, configure, stop, delete, and set default machines
- create volumes and networks
- start and stop Apple container services
- prune stopped containers and unused images, volumes, or networks
- delete images, volumes, networks, or machines
- auto-refresh Apple container system status, lists, and one-shot stats

## Requirements

- macOS with Apple's `container` CLI installed and initialized
- Go matching `go.mod` when building from source

## Install

Homebrew tap formula, once the tap is published:

```sh
brew install --HEAD pz/lazycont/lazycont
```

The formula is HEAD-only until the first tagged release. See [docs/homebrew.md](docs/homebrew.md) for the tap publishing and stable release steps.

## Run From Source

```sh
go run ./cmd/lazycont
```

Or build a local binary:

```sh
go build -o bin/lazycont ./cmd/lazycont
```

## Keys

| Key | Action |
| --- | --- |
| `tab` | Switch between containers, images, builder, volumes, networks, machines, registries, and system |
| `/` | Filter resource lists |
| `esc` | Clear the active filter |
| `up` / `k` | Move selection up |
| `down` / `j` | Move selection down |
| `r` | Refresh lists and status |
| `u` | Toggle periodic auto-refresh |
| `a` | Pull an image by reference |
| `b` | Build an image from a local context as `<tag> [context-dir]` |
| `t` | Tag selected image with a new reference |
| `P` | Push selected image to its registry |
| `O` | Save selected image to an OCI tar archive |
| `L` | Load an image from an OCI tar archive |
| `R` | Run the selected image detached, with options like `name=web p=8080:80 env=K=V -- cmd` |
| `N` | Create a stopped container from the selected image, with options like `name=web p=8080:80 env=K=V -- cmd` |
| `g` | Log in to a registry from the registries pane as `<server> [username]` |
| `M` | Create a machine from the machines pane as `<image> [name]` |
| `m` | Configure selected machine as `cpus=4 memory=8G home-mount=ro` |
| `S` | Set selected machine as the default machine |
| `C` | Create a volume as `<name> [size]` or network as `<name> [subnet]` from its pane |
| `i` / `enter` | Inspect selected resource |
| `c` | Copy files for the selected container as `<src> <dest>`; use `:/path` for the selected container |
| `E` | Export selected container filesystem to a tar archive |
| `l` | Tail selected container, machine, or system logs |
| `f` | Follow selected container, machine, or system logs until the command exits |
| `e` | Open `/bin/sh` in the selected running container, or a shell in the selected machine |
| `X` | Run a one-off command in the selected running container and show its output |
| `s` | Start selected container, builder, or Apple container services |
| `ctrl+r` | Restart selected running container |
| `x` | Stop selected container, machine, builder, or Apple container services |
| `K` | Kill selected container |
| `d` | Delete selected container, image, builder, volume, network, or machine, or log out of selected registry, with confirmation |
| `p` | Prune stopped containers or unused images, volumes, or networks, with confirmation |
| `?` | Toggle help |
| `q` / `ctrl+c` | Quit |

Destructive actions require a second confirmation key before the command is executed.
