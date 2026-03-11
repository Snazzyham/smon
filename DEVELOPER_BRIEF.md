# proctop — Developer Brief

A focused, opinionated process monitor TUI built with Go + Bubbletea. Not another htop clone — this groups processes by application, labels them intelligently, and lets you kill things fast.

---

## Why This Exists

Tools like btop/htop show every PID as its own line. A browser with 40 content processes becomes 40 lines of noise. `proctop` groups by application and tells you what actually matters.

---

## Tech Stack

| Component | Choice | Why |
|-----------|--------|-----|
| Language | Go | Single binary, no runtime deps |
| TUI framework | [Bubbletea](https://github.com/charmbracelet/bubbletea) | Elm architecture, clean key handling, tick-based updates |
| Styling | [Lipgloss](https://github.com/charmbracelet/lipgloss) | Pairs with Bubbletea, handles terminal styling |
| Process data | `shirou/gopsutil/v3` | Cross-platform `/proc` wrapper, gives CPU%, RSS, cmdline, PPID |
| Docker labels | Docker Engine API via `docker/docker/client` | Maps container PIDs → container names |

---

## Core Data Model

```go
// AppGroup represents one logical application (all its PIDs collapsed into one row)
type AppGroup struct {
    Name        string   // Display name: "Zen Browser", "Node (astro dev)", "Docker: nginx-proxy"
    PIDs        []int32  // All PIDs belonging to this group
    CPUPercent  float64  // Sum of CPU% across all PIDs
    MemoryMB    float64  // Sum of RSS across all PIDs
    ProcessCount int     // len(PIDs), shown as a subtle badge
}
```

---

## Process Grouping Rules

This is the core logic that makes proctop useful. Implement as a pipeline: for each PID, run through these matchers **in order** and use the first match.

### 1. Docker Containers

- On startup (and every ~10s), query the Docker API for running containers: `client.ContainerList(...)` → build a map of `PID → container name`
- Also map all child PIDs (walk `/proc/<pid>/task` or check PPID ancestry) into the same container group
- Display name format: **`Docker: <container_name>`**
- If Docker socket is unavailable, skip this matcher silently

### 2. Flatpak / Snap / Electron Apps (by exe path)

- Check the exe path or cmdline for known patterns:
  - `/opt/zen-browser*` or cmdline contains `zen-browser` → **"Zen Browser"**
  - Exe contains `discord` or `Discord` → **"Discord"**
  - Exe contains `slack` or `Slack` → **"Slack"**
  - Exe contains `spotify` or `Spotify` → **"Spotify"**
  - Exe contains `code` and cmdline contains `--ms-enable` → **"VS Code"**
  - Exe contains `cursor` → **"Cursor"**
  - `/opt/google/chrome*` or `chromium` → **"Chrome"** / **"Chromium"**
  - Exe contains `firefox` → **"Firefox"**
  - Exe contains `telegram` → **"Telegram"**
  - Exe contains `steam` → **"Steam"**
- The key insight: **group by the resolved app name, not by PID tree.** A browser spawns GPU processes, utility processes, renderers — all should collapse into one row.
- Maintain a lookup table (`map[string]string`) of exe substring → display name. Make it easy to extend.

### 3. Node.js Processes (special handling)

- Match: exe is `node` or `nodejs`, or cmdline starts with `node `
- **Parse the cmdline** to figure out what Node is actually running:
  - Contains `astro dev` or `astro preview` → **"Node (astro dev)"**
  - Contains `vite` or `vite dev` → **"Node (vite)"**
  - Contains `next dev` or `next start` → **"Node (next.js)"**
  - Contains `nuxt` → **"Node (nuxt)"**
  - Contains `remix` → **"Node (remix)"**
  - Contains `webpack` or `webpack-dev-server` → **"Node (webpack)"**
  - Contains `esbuild` → **"Node (esbuild)"**
  - Contains `tsx` or `ts-node` → **"Node (ts: <script_basename>)"**
  - Contains `express` or the main script imports express (just check cmdline) → **"Node (express)"**
  - Contains `nest` → **"Node (nestjs)"**
  - Fallback: **"Node (<basename of main script>)"** — e.g. `node server.js` → "Node (server.js)"
- **Port detection**: After identifying the Node process, check its listening sockets via `/proc/<pid>/net/tcp` (or `gopsutil.Net` connections). If it's listening on a port, append it: **"Node (astro dev :4321)"**
- Group all child processes of a Node parent into the same group (workers, etc.)

### 4. Java Processes

- Match: exe is `java`
- Parse cmdline for the main class or `-jar` name
- Display: **"Java (<main_class_or_jar>)"**

### 5. Python Processes

- Match: exe is `python`, `python3`
- Parse cmdline for script name or module (`-m`)
- Display: **"Python (<script_or_module>)"**

### 6. Everything Else (generic grouping)

- Group by **exe basename** — e.g. all `fish` PIDs become one "fish" row, all `pipewire` PIDs become "pipewire"
- Display: just the basename, lowercase

---

## UI Layout

```
 proctop — sorted by: CPU ▼                              [s]ort  [k]ill  [q]uit

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

 ▶ = selected │ ↑↓ navigate │ s = toggle CPU/MEM sort │ k = kill │ q = quit
```

### Layout Rules

- **Header row** is fixed, shows current sort mode
- **Table body** is scrollable if it exceeds terminal height
- **Selected row** gets a highlight (use Lipgloss — bold + reverse or a subtle background color, take inspiration from the teal/cyan palette the user already has in their terminal based on the screenshot)
- **Columns are right-aligned** for numbers, left-aligned for name
- **PROCS column** shows how many PIDs are in the group (useful context, subtle)
- Refresh every **2 seconds** (configurable with `-i` flag)
- Respect terminal width — truncate app name column if needed, numbers are never truncated

---

## Keybindings

| Key | Action |
|-----|--------|
| `↑` / `k` | Move selection up |
| `↓` / `j` | Move selection down |
| `s` | Toggle sort: CPU% → MEM → CPU% ... Update header indicator |
| `k` or `Delete` | Kill selected group — sends SIGTERM to **all PIDs** in the group. Show a brief inline confirmation: "Killed Zen Browser (38 processes)" for 2 seconds |
| `K` (shift+k) | Force kill — sends SIGKILL instead of SIGTERM |
| `q` or `Ctrl+C` | Quit |
| `/` | Filter/search: type to fuzzy-filter app names. `Esc` to clear |

---

## Bubbletea Architecture

```
Model {
    groups      []AppGroup    // current grouped + sorted snapshot
    cursor      int           // selected row index
    sortBy      SortMode      // CPU or MEM
    filterText  string        // active filter (empty = show all)
    filtering   bool          // whether filter input is active
    killMsg     string        // transient "Killed X" message
    killMsgTTL  int           // ticks remaining to show kill message
    width       int           // terminal width
    height      int           // terminal height
    dockerMap   map[int32]string  // PID → container name cache
}
```

### Messages / Commands

- **`tickMsg`** — fires every 2s, triggers a `Cmd` that reads `/proc`, groups, sorts, and returns a `processDataMsg` with the new `[]AppGroup`
- **`processDataMsg`** — updates `Model.groups`
- **`killResultMsg`** — result of a kill attempt (success/failure + name)
- **`tea.WindowSizeMsg`** — update width/height for responsive layout
- **`dockerRefreshMsg`** — fires every 10s, refreshes the Docker PID→name map

### Update Logic

- On `tickMsg`: spawn async `Cmd` to collect process data (this is I/O, keep it off the main thread)
- On key press: update cursor, sort mode, trigger kill, etc.
- On `processDataMsg`: replace groups, clamp cursor if list shrunk

---

## Implementation Notes

### Process Collection (the `Cmd`)

```
1. gopsutil.Processes() → get all PIDs
2. For each PID, collect: exe, cmdline, CPU%, RSS, PPID
3. Run through grouping pipeline (Docker → App → Node → Generic)
4. Aggregate into AppGroups (sum CPU%, sum RSS, collect PIDs)
5. Sort by current sort mode
6. Return as processDataMsg
```

**Performance**: There can be 500+ PIDs. The grouping loop should be fast since it's just string matching. CPU% from gopsutil requires two samples — use `process.Percent(0)` which uses the cached previous sample. On first tick, CPU% may be 0 for all — that's fine.

### Docker Integration

- Use `docker/docker/client.NewClientWithOpts(client.FromEnv)`
- If Docker socket doesn't exist or connect fails, set a flag and skip Docker matching entirely. **Never crash or block on Docker being unavailable.**
- Refresh the container→PID map every 10 seconds (not every 2s tick — Docker API is slower)
- Get the container's root PID from `ContainerInspect`, then also map child PIDs by walking PPID chains

### Port Detection for Node

- Read `/proc/<pid>/net/tcp6` and `/proc/<pid>/net/tcp`
- Parse the local address column (hex-encoded `IP:PORT`)
- Only report listening sockets (state `0A`)
- Cache per-PID per tick to avoid re-reading

### Kill Behavior

- On `k`: iterate all PIDs in the selected group, send `syscall.SIGTERM`
- On `K`: send `syscall.SIGKILL`
- Some PIDs may already be gone (race condition) — ignore ESRCH errors
- After kill, show transient message in the footer area for 2 seconds
- The next tick will naturally remove dead processes from the list

---

## CLI Flags

```
proctop [flags]

  -i, --interval duration   Refresh interval (default 2s)
  -n, --no-docker           Disable Docker container detection
  --no-ports                Disable port scanning for Node processes
```

---

## Project Structure

```
proctop/
├── main.go              # CLI entry, tea.NewProgram
├── model.go             # Bubbletea Model, Init, Update, View
├── grouper.go           # Process → AppGroup pipeline
├── matchers.go          # Individual matchers (docker, browser, node, generic)
├── process.go           # gopsutil wrapper, process collection
├── docker.go            # Docker client, PID→container map
├── ports.go             # /proc/net/tcp parser for listening ports
├── styles.go            # Lipgloss style definitions
├── go.mod
└── go.sum
```

---

## Dependencies

```
require (
    github.com/charmbracelet/bubbletea v1.x
    github.com/charmbracelet/lipgloss v1.x
    github.com/shirou/gopsutil/v3 v3.x
    github.com/docker/docker v27.x
)
```

---

## What "Done" Looks Like

1. `go build -o proctop && ./proctop` shows a live-updating grouped process table
2. Zen Browser (or any multi-process app) shows as **one row** with aggregated CPU/MEM
3. Node processes show what they're running and which port
4. Docker containers show container names
5. `s` toggles sort, `k` kills, `q` quits
6. Feels instant — no lag on keypress, data refreshes smoothly every 2s
7. Doesn't crash when Docker is unavailable, doesn't crash on permission errors
