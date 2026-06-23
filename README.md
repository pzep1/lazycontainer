# lazycont

A lazydocker-style terminal UI for Apple's [`container`](https://github.com/apple/container) CLI.

This is an early usable slice focused on day-to-day container work:

- browse containers and images
- browse volumes and networks
- filter resource lists across names and metadata
- inspect selected resources
- tail container logs
- start, stop, kill, and delete containers
- delete or prune images
- refresh Apple container system status, lists, and one-shot stats

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
| `tab` | Switch between containers, images, volumes, and networks |
| `/` | Filter resource lists |
| `esc` | Clear the active filter |
| `up` / `k` | Move selection up |
| `down` / `j` | Move selection down |
| `r` | Refresh lists and status |
| `i` / `enter` | Inspect selected resource |
| `l` | Tail selected container logs |
| `e` | Open `/bin/sh` in the selected running container |
| `s` | Start selected container |
| `x` | Stop selected container |
| `K` | Kill selected container |
| `d` | Delete selected container, image, volume, or network, with confirmation |
| `p` | Prune unused images, volumes, or networks, with confirmation |
| `?` | Toggle help |
| `q` / `ctrl+c` | Quit |

Destructive actions require a second confirmation key before the command is executed.
