package main

import (
	"path/filepath"
	"sort"
	"strings"
)

// AppGroup represents one logical application row in the TUI.
type AppGroup struct {
	Name         string
	PIDs         []int32
	Procs        []ProcessInfo // individual process details for the detail view
	CPUPercent   float64
	MemoryMB     float64
	ProcessCount int
}

// SortMode determines the sort column.
type SortMode int

const (
	SortByCPU SortMode = iota
	SortByMem
)

// groupProcesses runs each PID through the matcher pipeline,
// aggregates into AppGroups, and sorts.
func groupProcesses(procs []ProcessInfo, dockerMap map[int32]string, noPorts bool, sortBy SortMode) []AppGroup {
	groups := make(map[string]*AppGroup)

	for _, p := range procs {
		name := classifyProcess(p, dockerMap, noPorts)
		if name == "" {
			continue
		}

		g, ok := groups[name]
		if !ok {
			g = &AppGroup{Name: name}
			groups[name] = g
		}
		g.PIDs = append(g.PIDs, p.PID)
		g.Procs = append(g.Procs, p)
		g.CPUPercent += p.CPUPercent
		g.MemoryMB += p.MemoryMB
		g.ProcessCount++
	}

	result := make([]AppGroup, 0, len(groups))
	for _, g := range groups {
		result = append(result, *g)
	}

	sortGroups(result, sortBy)
	return result
}

// classifyProcess runs a PID through matchers in priority order, returning
// the display name for its group. Returns "" to skip the process.
func classifyProcess(p ProcessInfo, dockerMap map[int32]string, noPorts bool) string {
	// Skip kernel threads with no identifiable info
	if p.Exe == "" && p.Cmdline == "" {
		return ""
	}

	// 1. Docker containers
	if name, ok := dockerMap[p.PID]; ok {
		return "Docker: " + name
	}

	// 2. Known applications
	if name := matchKnownApp(p); name != "" {
		return name
	}

	// 3. Node.js
	if name := matchNode(p, noPorts); name != "" {
		return name
	}

	// 4. Java
	if name := matchJava(p); name != "" {
		return name
	}

	// 5. Python
	if name := matchPython(p); name != "" {
		return name
	}

	// 6. Generic: group by exe basename
	base := filepath.Base(p.Exe)
	if base == "" || base == "." {
		parts := strings.Fields(p.Cmdline)
		if len(parts) > 0 {
			return filepath.Base(parts[0])
		}
		return ""
	}
	return strings.ToLower(base)
}

func sortGroups(groups []AppGroup, sortBy SortMode) {
	sort.Slice(groups, func(i, j int) bool {
		switch sortBy {
		case SortByMem:
			return groups[i].MemoryMB > groups[j].MemoryMB
		default:
			return groups[i].CPUPercent > groups[j].CPUPercent
		}
	})
}
