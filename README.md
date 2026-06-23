# lazycont

A lazydocker-style terminal UI for Apple's [`container`](https://github.com/apple/container) CLI.

This is an early usable slice focused on day-to-day container work:

- browse containers and images
- browse volumes and networks
- browse container machines
- filter resource lists across names and metadata
- inspect selected resources
- tail container or machine logs
- follow container or machine logs in the terminal
- pull images and run selected images as detached containers
- start, stop, kill, and delete containers
- stop and delete machines
- delete or prune images
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
| `R` | Run the selected image detached, with an optional name |
| `i` / `enter` | Inspect selected resource |
| `l` | Tail selected container or machine logs |
| `f` | Follow selected container or machine logs until the command exits |
| `e` | Open `/bin/sh` in the selected running container, or a shell in the selected machine |
| `s` | Start selected container |
| `x` | Stop selected container or machine |
| `K` | Kill selected container |
| `d` | Delete selected container, image, volume, network, or machine, with confirmation |
| `p` | Prune unused images, volumes, or networks, with confirmation |
| `?` | Toggle help |
| `q` / `ctrl+c` | Quit |

Destructive actions require a second confirmation key before the command is executed.
