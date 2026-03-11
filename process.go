package main

import (
	"path/filepath"

	"github.com/shirou/gopsutil/v3/process"
)

// ProcessInfo holds collected data for a single PID.
type ProcessInfo struct {
	PID        int32
	PPID       int32
	Exe        string
	Cmdline    string
	CPUPercent float64
	MemoryMB   float64
}

// ExeBase returns the basename of the executable path.
func (p ProcessInfo) ExeBase() string {
	if p.Exe == "" {
		return ""
	}
	return filepath.Base(p.Exe)
}

// collectAllProcesses gathers info for every running PID via gopsutil.
func collectAllProcesses() ([]ProcessInfo, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, err
	}

	result := make([]ProcessInfo, 0, len(procs))
	for _, p := range procs {
		info := ProcessInfo{PID: p.Pid}

		if ppid, err := p.Ppid(); err == nil {
			info.PPID = ppid
		}
		if exe, err := p.Exe(); err == nil {
			info.Exe = exe
		}
		if cmdline, err := p.Cmdline(); err == nil {
			info.Cmdline = cmdline
		}
		if cpu, err := p.CPUPercent(); err == nil {
			info.CPUPercent = cpu
		}
		if mem, err := p.MemoryInfo(); err == nil && mem != nil {
			info.MemoryMB = float64(mem.RSS) / 1024 / 1024
		}

		result = append(result, info)
	}

	return result, nil
}
