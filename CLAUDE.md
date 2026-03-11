# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

**smon** — a terminal process monitor TUI that groups processes by application rather than showing individual PIDs. Built with Go + Bubbletea.

## Commands

```bash
# Build
go build -o smon

# Run
./smon
./smon -i 1s          # 1s refresh interval
./smon -n             # disable Docker detection
./smon --no-ports     # disable Node port scanning

# Test
go test ./...
go test -run TestGrouper ./...

# Lint (if golangci-lint installed)
golangci-lint run
```

## Architecture

The pipeline is the heart of the system: `gopsutil.Processes()` → grouper → TUI.

**Data flow:**
1. Every 2s tick → async `Cmd` collects all PIDs via gopsutil
2. Each PID runs through the matcher pipeline in `grouper.go` (first match wins):
   - Docker (PID→container map, refreshed every 10s)
   - Known apps by exe path (browsers, Electron apps, etc.)
   - Node.js (parse cmdline + detect listening port from `/proc/<pid>/net/tcp`)
   - Java / Python (parse cmdline for main class/script)
   - Generic fallback (group by exe basename)
3. Results aggregated into `[]AppGroup`, sorted, returned as `processDataMsg`
4. Bubbletea model replaces `groups`, re-renders

**Key files:**
- `main.go` — CLI flags (pflag), `tea.NewProgram`
- `model.go` — Bubbletea `Model`, `Init`, `Update`, `View`; holds `groups`, `cursor`, `sortBy`, `filterText`, `dockerMap`
- `grouper.go` — orchestrates the matcher pipeline, aggregates `AppGroup`s
- `matchers.go` — individual matcher functions (Docker, browser, Node, Java, Python, generic)
- `process.go` — gopsutil wrapper; collects exe, cmdline, CPU%, RSS, PPID per PID
- `docker.go` — Docker Engine API client; builds `map[int32]string` (PID→container name) by walking PPID chains
- `ports.go` — parses `/proc/<pid>/net/tcp` and `/proc/<pid>/net/tcp6` for listening sockets (state `0A`), hex-decodes port
- `styles.go` — Lipgloss styles using ANSI terminal palette (adapts to user's theme)

## Critical Implementation Details

**Docker**: Always guard with availability check. If Docker socket is missing, set a flag and skip Docker matching entirely — never crash or block.

**CPU%**: `gopsutil process.Percent(0)` uses cached previous sample. First tick will show 0% for all — expected.

**Kill**: Send SIGTERM (`d`/`Delete`) or SIGKILL (`D`) to all PIDs in group. Ignore `ESRCH` (process already gone). Transient footer message for 2s.

**Node port detection**: Parse hex local address from `/proc/<pid>/net/tcp{,6}`, filter state `0A` (listening), cache per-PID per tick.

**App name matcher**: Use `map[string]string` of exe substring → display name. Easy to extend without modifying core logic.

**Bubbletea messages**: `tickMsg` → `processDataMsg`, `dockerRefreshMsg` (10s cadence), `killResultMsg`, `tea.WindowSizeMsg`.

**Styles**: Use ANSI color indices only (no hardcoded hex) so colors adapt to the terminal theme.

## Dependencies

```
github.com/charmbracelet/bubbletea v1.x
github.com/charmbracelet/lipgloss  v1.x
github.com/shirou/gopsutil/v3      v3.x
github.com/docker/docker           v27.x
github.com/spf13/pflag             v1.x
```
