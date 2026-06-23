# lazycont

A lazydocker-style terminal UI for Apple's [`container`](https://github.com/apple/container) CLI.

This is an early usable slice focused on day-to-day container work:

- switch the main panel between per-resource tabs (`[` / `]`): containers show Logs, Stats, Env, Config, Top, and Inspect
- stream container, machine, and system logs live in the Logs tab with autoscroll
- watch CPU% and memory usage as live ASCII graphs in the Stats tab
- view container environment variables and running processes in the Env and Top tabs
- expand the main panel with screen modes (`+` / `_`): normal, half, and fullscreen
- open a context-aware actions menu (`space`) and a scrollable keybinding reference (`?`)
- open a container's first published port in the browser (`w`)
- customize appearance and behavior from config (theme, border, side-panel width, log tail/window, refresh interval, per-context custom commands)
- browse containers and images
- browse volumes and networks
- browse image builder status
- browse container machines
- browse registry logins
- browse Apple container system diagnostics
- filter resource lists across names and metadata
- click and scroll resource panes with mouse support
- inspect selected resources
- scan container CPU and memory directly in the container list
- run ad-hoc Apple `container` commands without leaving the TUI
- run named custom Apple `container` commands from config
- open the lazycont config file from the TUI
- view image variant and layer history
- tail container, machine, or system logs
- follow container, machine, or system logs in the terminal
- run one-off commands in selected running containers
- view container CPU, memory, network, disk, process metrics, and recent metric trends
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
- auto-refresh Apple container system status, lists, one-shot stats, and the active logs pane

## Requirements

- macOS with Apple's `container` CLI installed and its system service started
- Go matching `go.mod` when building from source

## Install

```sh
brew install pzep1/lazycont/lazycont
```

This taps `pzep1/homebrew-lazycont` and builds lazycont from the latest tagged release. To track the development branch instead, use:

```sh
brew install --HEAD pzep1/lazycont/lazycont
```

The formula depends on Homebrew's Apple `container` package. See [docs/homebrew.md](docs/homebrew.md) for tap maintenance and release steps.

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
| `tab` / `shift+tab` | Switch between containers, images, builder, volumes, networks, machines, registries, and system |
| `[` / `]` | Switch the main panel to the previous / next tab |
| `+` / `_` | Cycle screen mode: normal, half, fullscreen |
| `space` | Open the context-aware actions menu |
| `/` | Filter resource lists |
| `esc` | Clear the active filter, or close command output |
| `up` / `k` | Move selection up |
| `down` / `j` | Move selection down |
| mouse click | Select a resource tab or row |
| mouse wheel | Scroll details/log output or move the resource selection |
| `:` | Run an ad-hoc Apple `container` command, such as `image list --format json` |
| `;` | Run a named custom Apple `container` command from config |
| `o` | Open the lazycont config file in `$VISUAL`, `$EDITOR`, or `vi` |
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
| `i` / `enter` | Open the Inspect tab for the selected resource |
| `c` | Copy files for the selected container as `<src> <dest>`; use `:/path` for the selected container |
| `E` | Export selected container filesystem to a tar archive |
| `l` | Open the Logs tab and stream selected container, machine, or system logs |
| `f` | Follow selected container, machine, or system logs full screen until the command exits |
| `e` | Open `/bin/sh` in the selected running container, or a shell in the selected machine |
| `w` | Open the selected container's first published port in the browser |
| `X` | Run a one-off command in the selected running container and show its output |
| `s` | Start selected container, builder, or Apple container services |
| `ctrl+r` | Restart selected running container |
| `x` | Stop selected container, machine, builder, or Apple container services |
| `K` | Kill selected container |
| `d` | Delete selected container, image, builder, volume, network, or machine, or log out of selected registry, with confirmation |
| `p` | Prune stopped containers or unused images, volumes, or networks, with confirmation |
| `?` | Toggle the scrollable keybinding reference |
| `q` / `ctrl+c` | Quit |

In the Logs tab, logs follow live and stick to the bottom; scroll up (`pgup`, wheel) to detach autoscroll and press `end` to re-attach. Open the actions menu with `space` to see every action available for the selected resource without memorizing keys.

Destructive actions require a second confirmation key before the command is executed.

## Config

lazycont reads optional custom commands from `~/Library/Application Support/lazycont/config.json` on macOS:

```json
{
  "commands": [
    {
      "name": "Images as JSON",
      "args": ["image", "list", "--format", "json"]
    },
    {
      "name": "Selected container logs",
      "args": ["logs", "--tail", "200", "{container}"]
    }
  ]
}
```

Press `;` in the TUI and enter a command number, exact name, or unique part of a name. Each entry runs as `container <args>`.

Custom command args can use `{container}`, `{image}`, `{volume}`, `{network}`, `{machine}`, `{registry}`, or `{resource}`. `{resource}` expands to the selected item in the active pane.

### Per-context commands, attach, and appearance

Beyond the flat `commands` list, the config accepts per-context custom commands, an interactive `attach` flag, and appearance/behavior settings:

```json
{
  "commands": [],
  "customCommands": {
    "containers": [
      { "name": "Shell", "args": ["exec", "-it", "{container}", "/bin/sh"], "attach": true }
    ],
    "images": [
      { "name": "Image as JSON", "args": ["image", "inspect", "{image}"] }
    ]
  },
  "logs": { "tail": 200, "since": "5m" },
  "refreshIntervalMs": 5000,
  "gui": {
    "sidePanelWidth": 0.3333,
    "screenMode": "normal",
    "border": "rounded",
    "theme": { "activeBorderColor": "39", "selectedLineBgColor": "57" }
  }
}
```

- `customCommands` groups commands by pane (`containers`, `images`, `volumes`, `networks`, `machines`, `registries`, `builder`, `system`). All commands — flat and per-context — appear in the `;` picker; placeholders scope them to the relevant resource.
- `attach: true` hands the terminal to the command (use for interactive shells) instead of capturing its output.
- `logs.tail` sets how many lines to request when a Logs tab opens; `logs.since` is the system-log window.
- `refreshIntervalMs` overrides the auto-refresh interval.
- `gui.sidePanelWidth` is the sidebar width as a fraction of the screen; `gui.screenMode` is the startup mode (`normal`, `half`, `fullscreen`); `gui.border` is one of `rounded`, `single`, `double`, `hidden`; `gui.theme` colors accept 256-color codes or names.

Press `o` in the TUI to create this file if needed and open it in your editor.

## License

Released under the [GNU General Public License v3.0 or later](LICENSE).
