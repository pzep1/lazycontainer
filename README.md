# lazycont

A lazydocker-style terminal UI for Apple's [`container`](https://github.com/apple/container) CLI.

This is an early usable slice focused on day-to-day container work:

- browse containers and images
- browse volumes and networks
- browse container machines
- filter resource lists across names and metadata
- inspect selected resources
- view image variant and layer history
- tail container or machine logs
- follow container or machine logs in the terminal
- view container CPU, memory, network, disk, and process metrics
- export a selected container filesystem as a tar archive
- copy files or folders between a selected container and the local filesystem
- build local images from a Dockerfile or Containerfile
- retag selected images
- push selected images to a registry
- pull images and run selected images as detached containers
- start, restart, stop, kill, and delete containers
- create, stop, delete, and set default machines
- prune stopped containers and unused images, volumes, or networks
- delete images, volumes, networks, or machines
- auto-refresh Apple container system status, lists, and one-shot stats

## Requirements

- macOS with Apple's `container` CLI installed and initialized
- Go 1.24 or newer

## Run

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
| `tab` | Switch between containers, images, volumes, networks, and machines |
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
| `R` | Run the selected image detached, with an optional name |
| `M` | Create a machine from the machines pane as `<image> [name]` |
| `S` | Set selected machine as the default machine |
| `i` / `enter` | Inspect selected resource |
| `c` | Copy files for the selected container as `<src> <dest>`; use `:/path` for the selected container |
| `E` | Export selected container filesystem to a tar archive |
| `l` | Tail selected container or machine logs |
| `f` | Follow selected container or machine logs until the command exits |
| `e` | Open `/bin/sh` in the selected running container, or a shell in the selected machine |
| `s` | Start selected container |
| `ctrl+r` | Restart selected running container |
| `x` | Stop selected container or machine |
| `K` | Kill selected container |
| `d` | Delete selected container, image, volume, network, or machine, with confirmation |
| `p` | Prune stopped containers or unused images, volumes, or networks, with confirmation |
| `?` | Toggle help |
| `q` / `ctrl+c` | Quit |

Destructive actions require a second confirmation key before the command is executed.
