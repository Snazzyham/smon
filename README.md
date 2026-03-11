# smon

A process monitor TUI that groups processes by application. Instead of showing 40 rows for a browser's content/GPU/utility processes, smon collapses them into one row with aggregated CPU and memory usage.

```
 smon — sorted by: CPU ▼                              [s]ort  [d]elete  [q]uit

 APP                              CPU %      MEM (MB)    PROCS
 ─────────────────────────────────────────────────────────────
 ▶ Zen Browser                    10.3%      1,842 MB      38
   Node (astro dev :4321)          4.7%        312 MB       3
   Docker: postgres-dev            2.1%        256 MB       8
   Discord                         1.8%        487 MB      12
   VS Code                         1.2%        623 MB      15
   Node (vite :5173)               0.9%        189 MB       2
   fish                            0.1%         18 MB       4
   pipewire                        0.0%         12 MB       3
```

## Install

```bash
go build -o smon
sudo cp smon /usr/local/bin/
```

Or using `go install` (requires the repo to be in your `GOPATH` or have a proper module path):

```bash
go install .
```

`go install` places the binary at `$GOPATH/bin/smon` (usually `~/go/bin/smon`). Make sure that's on your `PATH`. The `cp` approach is simpler and works anywhere.

## Usage

```bash
smon                  # default 2s refresh
smon -i 1s            # 1s refresh interval
smon -n               # disable Docker container detection
smon --no-ports       # disable Node.js port scanning
```

## Keybindings

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `s` | Toggle sort: CPU% ↔ MEM |
| `Space` | Show/hide per-process detail for selected row |
| `d` / `Delete` | Kill selected group (SIGTERM) |
| `D` | Force kill (SIGKILL) |
| `/` | Filter by app name (`Esc` to clear) |
| `q` / `Ctrl+C` | Quit |

Press `Space` on any row to expand a detail panel showing each individual PID, its CPU%, memory, and full command line — useful for identifying mystery processes.

## Process Grouping

Processes are classified through a matcher pipeline (first match wins):

1. **Docker containers** — maps PIDs to container names via Docker API
2. **Known apps** — Zen Browser, Discord, Slack, Spotify, VS Code, Cursor, Chrome, Firefox, Telegram, Steam
3. **Node.js** — identifies the framework (Astro, Vite, Next.js, etc.) and detects listening ports
4. **Java** — shows main class or jar name
5. **Python** — shows script or `-m` module name
6. **Generic** — groups by executable basename

To add a new known app, extend the `knownApps` slice in `matchers.go`.

## Dependencies

- [Bubbletea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) — terminal styling
- [gopsutil](https://github.com/shirou/gopsutil) — cross-platform process info
- [Docker Engine API](https://github.com/docker/docker) — container detection (optional, degrades gracefully)
