<div align="center">

# lazycontainer

**A lazydocker-style terminal UI for Apple's [`container`](https://github.com/apple/container) CLI.**

Browse, inspect, and drive your containers, images, volumes, networks, machines, registries, and the builder — all from one fast, keyboard-driven TUI.

### `brew install pzep1/lazycont/lazycontainer`

![Release](https://img.shields.io/github/v/release/pzep1/lazycontainer)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)
![Platform: macOS](https://img.shields.io/badge/platform-macOS%20(Apple%20silicon)-lightgrey)
![Go](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go&logoColor=white)
![Status](https://img.shields.io/badge/status-early%20but%20usable-success)

</div>

## Install

```sh
brew install pzep1/lazycont/lazycontainer
```

Requires macOS with Apple's [`container`](https://github.com/apple/container) CLI installed and its system service started. [Other ways to run ↓](#more-ways-to-run)

```text
 lazycontainer  ● running                                        updated 16:57:03
╭──────────────────────────────╮╭─────────────────────────────────────────────╮
│ ▌ Containers (3)  state/cpu/mem ││ Config Logs Stats Env Ports Mounts Health  │
│ web       running  2.1%   45MB ││ 12:07:01 server listening on :8080          │
│ db        running  0.4%   60MB ││ 12:07:04 GET /   200  1ms                    │
│ cache     stopped         -    ││ 12:07:06 GET /api 200 12ms                  │
│   Services (2)   state/cpu/mem ││ ▏following live — End re-attaches            │
│ api       running  0.6%   38MB ││                                             │
│ worker    stopped         -    ││                                             │
│   Images (6)       size  used ││                                              │
│ nginx              1.8.2  142M ││                                             │
│   Builder (running)     state ││                                              │
│   Volumes (2)      size  used ││                                              │
│   Networks (1)     mode  used ││                                              │
│   Machines (1)          state ││                                              │
│   System (running)    status ││                                               │
╰──────────────────────────────╯╰─────────────────────────────────────────────╯
 refreshed · u auto:on            space menu · ? help · q quit · s start · l logs
```

Every resource stacks as a panel down the left, lazydocker-style, and they are
**all visible at once** — no accordion. The focused panel gets the accent bar,
the largest share of vertical space, and the action keys; `tab` / `shift+tab` (or
`1`–`9`) move focus between panels, while the main panel on the right tracks the
selected item.

## Highlights

- ⚡ **Live everything** — stream container, machine, and system logs in-pane with autoscroll, watch **CPU%, memory, network, and disk I/O as live ASCII graphs**, and auto-refresh lists, stats, and status. CPU% is derived from Apple's cumulative counters so you get a real live percentage.
- 🗂️ **Tabbed main panel** — flip a selected container between **Config · Logs · Stats · Env · Ports · Mounts · Health · Top · Inspect** with `[` / `]`; other resources get the tabs that fit them.
- 📊 **Fleet overview strip** — a pinned summary line shows container counts, mean CPU%, memory in use, disk used/reclaimable, and builder state at a glance.
- 🔗 **In-use badges** — images, volumes, and networks show a **●N** count of how many containers reference them, so you can see what's safe to prune.
- ⌨️ **Drive it from the keyboard** — start/stop/restart/kill containers, exec shells, copy & export filesystems, pull/build/tag/push/save/load images, and manage volumes, networks, machines, registries, and the builder. Jump straight to any pane with `1`–`9`.
- 🧩 **Compose without compose** — Apple's `container` CLI has no `compose`, so lazycontainer brings the **Services** panel anyway: drop a `compose.yaml` beside your project and bring services **up/down**, **start/stop/restart**, and **recreate** them — individually or the whole stack — straight from the TUI.
- 📦 **Bulk actions** — a `B` menu to stop/kill/remove every container or prune images, volumes, and networks in one keystroke (with confirmation).
- 🍎 **Apple-native extras** — view a container or machine's **VM boot logs** (`ctrl+b`), and see registered **local DNS domains** and **system properties** right in the System pane.
- 🧭 **Discoverable** — a context-aware **actions menu** (`space`), a **bulk actions menu** (`B`), a scrollable **keybinding reference** (`?`), and **screen modes** (`+` / `_`: normal → half → fullscreen).
- 🎨 **Yours to shape** — custom commands (flat or per-context, with interactive `attach`), theme/border/layout, log window, and refresh interval — all reloaded live when you edit the config.
- 🖱️ **Mouse-friendly** — click panes and rows, scroll with the wheel, filter lists with `/`.

<details>
<summary><b>Full feature list</b></summary>

- browse containers, **compose services**, images, volumes, networks, image builder status, machines, registry logins, and Apple container system diagnostics
- orchestrate a `compose.yaml` stack: bring services up/down, start/stop/restart, and recreate them — per service or whole-project — in dependency order, with no `docker compose` required
- switch the main panel between per-resource tabs (`[` / `]`), or jump straight to a pane with `1`–`8`
- stream container, machine, and system logs live with autoscroll, or follow them full screen
- view a container's or machine's VM boot logs (`ctrl+b`)
- watch CPU%, memory, network, and disk I/O as live ASCII graphs, plus a current CPU/memory/network/disk/PID summary — CPU% is derived live from Apple's cumulative `cpuUsageUsec` counter
- view container environment variables and running processes (Env and Top tabs)
- expand the main panel with screen modes: normal, half, fullscreen
- context-aware actions menu, a bulk-actions menu (`B`), and a scrollable keybinding reference
- run bulk actions: stop/kill/remove all containers, or prune unused images, volumes, and networks
- see registered local DNS domains and system properties in the System pane
- open a container's first published port in the browser
- filter resource lists across names and metadata, and hide noisy resources with a config `ignore` list
- inspect selected resources (raw JSON)
- scan container CPU and memory directly in the container list
- run ad-hoc or named custom Apple `container` commands without leaving the TUI
- open the lazycontainer config file from the TUI (changes reload live)
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

The Homebrew formula taps [`pzep1/homebrew-lazycont`](https://github.com/pzep1/homebrew-lazycont) and builds **lazycontainer v0.3.0** from the latest tagged release.

Or run from source (needs Go 1.26+ to match `go.mod`):

```sh
go run ./cmd/lazycontainer                      # run directly
go build -o bin/lazycontainer ./cmd/lazycontainer    # or build a local binary
```

The formula depends on Homebrew's Apple `container` package. See [docs/homebrew.md](docs/homebrew.md) for tap maintenance and release steps.

## Keybindings

Press `?` in the app for the same reference, scrollable. Press `space` for a menu of every action available on the selected resource — no memorization required.

#### Global

| Key | Action |
| --- | --- |
| `tab` / `shift+tab` | Switch resource pane (containers, services, images, builder, volumes, networks, machines, registries, system) |
| `←` / `→` or `h` | Previous / next resource pane |
| `1`–`9` | Jump to resource pane (1=containers … 9=system) |
| `[` / `]` | Previous / next main-panel tab |
| `+` / `_` | Cycle screen mode: normal, half, fullscreen |
| `space` · `B` | Open the context-aware actions menu · bulk actions menu |
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
| `ctrl+b` | View the container's VM boot logs |
| `c` · `E` | Copy files `<src> <dest>` (`:/path` = selected container) · export filesystem to a tar |
| `w` | Open the first published port in the browser |
| `B` | Bulk actions: stop / kill / remove all containers |

#### Services (compose)

Drop a `compose.yaml` (or `docker-compose.yml`) beside your project — the **Services** pane lists its services with the state of the container backing each.

| Key | Action |
| --- | --- |
| `u` · `U` | Up the selected service · up the whole project (dependency order) |
| `d` · `D` | Down the service · down the whole project (with confirmation) |
| `R` | Recreate the service (down, then up) |
| `s` · `x` · `ctrl+r` | Start · stop · restart the service's container |
| `l` · `e` · `i` | Stream logs · open a shell · inspect the service's container |

> Services are translated into `container run` (with the service's ports, env, volumes, networks, and command), `stop`, and `delete` calls — Apple's CLI has no native compose, so lazycontainer does the orchestration. Build-only services are `container build`-tagged then run.

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
| `e` · `ctrl+b` | Open a shell in the selected machine · view its VM boot logs |
| `g` | Log in to a registry `<server> [username]` |
| `B` | Bulk actions: prune unused volumes or networks (volumes/networks pane) |

#### Builder, system & shared

| Key | Action |
| --- | --- |
| `s` · `x` | Start / stop the builder or Apple container services |
| `d` | Delete the selected resource, or log out of a registry — with confirmation |
| `p` · `B` | Prune unused resources · bulk actions menu — with confirmation |

> The System pane also surfaces Apple-native local DNS domains and `system property` settings alongside status, disk usage, and versions.

> Destructive actions require a second confirmation key. In the Logs tab, output follows live and sticks to the bottom; scroll up (`pgup`, wheel) to detach autoscroll and press `end` to re-attach.

## Config

lazycontainer reads optional settings from `~/Library/Application Support/lazycontainer/config.json` on macOS. An existing config under `~/Library/Application Support/lazycont/` is still picked up until you create one in the new directory. Everything is optional — the simplest config just adds custom commands:

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
  "ignore": ["buildkit", "infra-"],
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
| `ignore` | Substrings; any container, image, volume, network, machine, or registry whose name (or, for containers, image) contains one is hidden from every list. |
| `gui.sidePanelWidth` | Sidebar width as a fraction of the screen. |
| `gui.screenMode` | Startup mode: `normal`, `half`, `fullscreen`. |
| `gui.border` | `rounded`, `single`, `double`, or `hidden`. |
| `gui.theme` | Colors accept 256-color codes or names. |

</details>

Press `o` in the TUI to create this file if needed and open it in your editor — edits are reloaded live.

## License

Released under the [GNU General Public License v3.0 or later](LICENSE).
