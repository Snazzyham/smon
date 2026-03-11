package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	tea "github.com/charmbracelet/bubbletea"
)

// refreshDocker returns a tea.Cmd that rebuilds the Docker PID→container map.
func refreshDocker() tea.Cmd {
	return func() tea.Msg {
		m, _ := buildDockerMap()
		if m == nil {
			m = make(map[int32]string)
		}
		return dockerRefreshMsg(m)
	}
}

// buildDockerMap queries the Docker API for running containers and maps
// each container's root PID (and direct children) to its container name.
func buildDockerMap() (map[int32]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make(map[int32]string)
	for _, c := range containers {
		if len(c.Names) == 0 {
			continue
		}
		name := strings.TrimPrefix(c.Names[0], "/")

		inspect, err := cli.ContainerInspect(ctx, c.ID)
		if err != nil {
			continue
		}
		pid := int32(inspect.State.Pid)
		if pid > 0 {
			result[pid] = name
			mapChildPIDs(pid, name, result)
		}
	}

	return result, nil
}

// mapChildPIDs walks /proc to find direct children of parentPID
// and adds them to the map under the same container name.
func mapChildPIDs(parentPID int32, name string, m map[int32]string) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.ParseInt(e.Name(), 10, 32)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if err != nil {
			continue
		}
		// Format: pid (comm) state ppid ...
		// Skip past comm (may contain parens/spaces)
		s := string(data)
		idx := strings.LastIndex(s, ") ")
		if idx < 0 {
			continue
		}
		fields := strings.Fields(s[idx+2:])
		if len(fields) < 2 {
			continue
		}
		ppid, err := strconv.ParseInt(fields[1], 10, 32)
		if err != nil {
			continue
		}
		if int32(ppid) == parentPID {
			m[int32(pid)] = name
		}
	}
}
