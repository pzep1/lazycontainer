<div align="center">

# lazycont

**A lazydocker-style terminal UI for Apple's [`container`](https://github.com/apple/container) CLI.**

Browse, inspect, and drive your containers, images, volumes, networks, machines, registries, and the builder — all from one fast, keyboard-driven TUI.

### `brew install pzep1/lazycont/lazycont`

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)
![Platform: macOS](https://img.shields.io/badge/platform-macOS%20(Apple%20silicon)-lightgrey)
![Go](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go&logoColor=white)
![Status](https://img.shields.io/badge/status-early%20but%20usable-success)

</div>

## Install

```sh
brew install pzep1/lazycont/lazycont
```

Requires macOS with Apple's [`container`](https://github.com/apple/container) CLI installed and its system service started. [Other ways to run ↓](#more-ways-to-run)

```text
┌ lazycont | apple container: running ──────────────────── updated 16:57:03 ┐
│ ┌─────────────────────────────┐┌──────────────────────────────────────┐  │
│ │  containers 3   images 6     ││ Logs  Stats  Env  Config  Top  Inspect│  │
│ │  volumes 2   networks 1  …   ││                                       │  │
│ │                              ││ 12:07:01 server listening on :8080    │  │
│ │ name           state/cpu/mem ││ 12:07:04 GET /   200  1ms             │  │
│ │ web      running  2.1%  45MB ││ 12:07:06 GET /api 200 12ms            │  │
│ │ db       running  0.4%  60MB ││ ▏following live — End re-attaches     │  │
│ │ cache    stopped          -  ││                                       │  │
│ └─────────────────────────────┘└──────────────────────────────────────┘  │
│ refreshed | u auto:on | space menu | ? help                               │
└───────────────────────────────────────────────────────────────────────────┘
```

## Highlights

- ⚡ **Live everything** — stream container, machine, and system logs in-pane with autoscroll, watch **CPU% and memory as live ASCII graphs**, and auto-refresh lists, stats, and status.
- 🗂️ **Tabbed main panel** — flip a selected container between **Logs · Stats · Env · Config · Top · Inspect** with `[` / `]`; other resources get the tabs that fit them.
- ⌨️ **Drive it from the keyboard** — start/stop/restart/kill containers, exec shells, copy & export filesystems, pull/build/tag/push/save/load images, and manage volumes, networks, machines, registries, and the builder.
- 🧭 **Discoverable** — a context-aware **actions menu** (`space`), a scrollable **keybinding reference** (`?`), and **screen modes** (`+` / `_`: normal → half → fullscreen).
- 🎨 **Yours to shape** — custom commands (flat or per-context, with interactive `attach`), theme/border/layout, log window, and refresh interval — all reloaded live when you edit the config.
- 🖱️ **Mouse-friendly** — click panes and rows, scroll with the wheel, filter lists with `/`.

<details>
<summary><b>Full feature list</b></summary>

- browse containers, images, volumes, networks, image builder status, machines, registry logins, and Apple container system diagnostics
- switch the main panel between per-resource tabs (`[` / `]`)
- stream container, machine, and system logs live with autoscroll, or follow them full screen
- watch CPU% and memory as live ASCII graphs, plus a current CPU/memory/network/disk/PID summary
- view container environment variables and running processes (Env and Top tabs)
- expand the main panel with screen modes: normal, half, fullscreen
- context-aware actions menu and scrollable keybinding reference
- open a container's first published port in the browser
- filter resource lists across names and metadata
- inspect selected resources (raw JSON)
- scan container CPU and memory directly in the container list
- run ad-hoc or named custom Apple `container` commands without leaving the TUI
- open the lazycont config file from the TUI (changes reload live)
- view image variant and layer history
- run one-off commands in selected running containers
- export a selected container filesystem as a tar archive
- copy files or folders between a selected container and the local filesystem
- build local images from a Dockerfile or Containerfile
- start, stop, and delete the image builder
- retag images; save and load image archives; push images to a registry
- log in to and out of registries
- pull images, create stopped containers, and run images with ports, env vars, mounts, networks, and command args
- start, restart, stop, kill, and delete containers
- create, configure, stop, delete, and set default machines
- create volumes and networks
- start and stop Apple container services
- prune stopped containers and unused images, volumes, or networks
- delete images, volumes, networks, or machines
- auto-refresh system status, lists, one-shot stats, and the active logs pane

</details>

## More ways to run

The Homebrew formula taps `pzep1/homebrew-lazycont` and builds lazycont from the latest tagged release. To track the development branch instead:

```sh
brew install --HEAD pzep1/lazycont/lazycont
```

Or run from source (needs Go 1.26+ to match `go.mod`):

```sh
go run ./cmd/lazycont                      # run directly
go build -o bin/lazycont ./cmd/lazycont    # or build a local binary
```

The formula depends on Homebrew's Apple `container` package. See [docs/homebrew.md](docs/homebrew.md) for tap maintenance and release steps.

## Keybindings

Press `?` in the app for the same reference, scrollable. Press `space` for a menu of every action available on the selected resource — no memorization required.

#### Global

| Key | Action |
| --- | --- |
| `tab` / `shift+tab` | Switch resource pane (containers, images, builder, volumes, networks, machines, registries, system) |
| `[` / `]` | Previous / next main-panel tab |
| `+` / `_` | Cycle screen mode: normal, half, fullscreen |
| `space` | Open the context-aware actions menu |
| `/` · `esc` | Filter the list · clear filter or close command output |
| `:` · `;` | Run an ad-hoc · named custom `container` command |
| `o` · `r` · `u` | Open config in `$VISUAL`/`$EDITOR`/`vi` · refresh · toggle auto-refresh |
| `?` · `q` / `ctrl+c` | Toggle help · quit |

#### Selection & main panel

| Key | Action |
| --- | --- |
| `up` / `k`, `down` / `j` | Move selection |
| mouse click / wheel | Select a tab or row · scroll the panel or list |
| `i` / `enter` | Open the Inspect tab for the selected resource |
| `l` | Open the Logs tab and stream logs (containers, machines, system) |
| `f` | Follow logs full screen until the command exits |
| `pgup`/`pgdn`, `home`/`end` | Scroll the panel (`end` re-enables log autoscroll) |

#### Containers

| Key | Action |
| --- | --- |
| `s` · `ctrl+r` · `x` · `K` | Start · restart · stop · kill |
| `e` · `X` | Open `/bin/sh` · run a one-off command and show its output |
| `c` · `E` | Copy files `<src> <dest>` (`:/path` = selected container) · export filesystem to a tar |
| `w` | Open the first published port in the browser |

#### Images

| Key | Action |
| --- | --- |
| `a` · `b` | Pull by reference · build as `<tag> [context-dir]` |
| `R` · `N` | Run detached · create stopped, e.g. `name=web p=8080:80 env=K=V -- cmd` |
| `t` · `P` | Tag with a new reference · push to its registry |
| `O` · `L` | Save to · load from an OCI tar archive |

#### Volumes · networks · machines · registries

| Key | Action |
| --- | --- |
| `C` | Create a volume `<name> [size]` or network `<name> [subnet]` from its pane |
| `M` · `m` · `S` | Create a machine `<image> [name]` · configure `cpus=4 memory=8G` · set default |
| `e` | Open a shell in the selected machine |
| `g` | Log in to a registry `<server> [username]` |

#### Builder, system & shared

| Key | Action |
| --- | --- |
| `s` · `x` | Start / stop the builder or Apple container services |
| `d` | Delete the selected resource, or log out of a registry — with confirmation |
| `p` | Prune stopped containers or unused images, volumes, or networks — with confirmation |

> Destructive actions require a second confirmation key. In the Logs tab, output follows live and sticks to the bottom; scroll up (`pgup`, wheel) to detach autoscroll and press `end` to re-attach.

## Config

lazycont reads optional settings from `~/Library/Application Support/lazycont/config.json` on macOS. Everything is optional — the simplest config just adds custom commands:

```json
{
  "commands": [
    { "name": "Images as JSON", "args": ["image", "list", "--format", "json"] },
    { "name": "Selected container logs", "args": ["logs", "--tail", "200", "{container}"] }
  ]
}
```

Press `;` and enter a command number, exact name, or unique part of a name; each entry runs as `container <args>`. Args can use `{container}`, `{image}`, `{volume}`, `{network}`, `{machine}`, `{registry}`, or `{resource}` (the selected item in the active pane).

<details>
<summary><b>Per-context commands, attach, and appearance</b></summary>

Beyond the flat `commands` list, the config accepts per-context custom commands, an interactive `attach` flag, and appearance/behavior settings:

```json
{
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

| Setting | Meaning |
| --- | --- |
| `customCommands` | Per-pane commands (`containers`, `images`, `volumes`, `networks`, `machines`, `registries`, `builder`, `system`). All commands appear in the `;` picker; placeholders scope them to the relevant resource. |
| `attach` | `true` hands the terminal to the command (interactive shells) instead of capturing output. |
| `logs.tail` / `logs.since` | Lines requested when a Logs tab opens · system-log window. |
| `refreshIntervalMs` | Overrides the auto-refresh interval. |
| `gui.sidePanelWidth` | Sidebar width as a fraction of the screen. |
| `gui.screenMode` | Startup mode: `normal`, `half`, `fullscreen`. |
| `gui.border` | `rounded`, `single`, `double`, or `hidden`. |
| `gui.theme` | Colors accept 256-color codes or names. |

</details>

Press `o` in the TUI to create this file if needed and open it in your editor — edits are reloaded live.

## License

Released under the [GNU General Public License v3.0 or later](LICENSE).
