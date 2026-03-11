package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// getListeningPort returns the first listening TCP port for a PID,
// or 0 if none found. Reads /proc/<pid>/net/tcp and tcp6.
func getListeningPort(pid int32) int {
	for _, proto := range []string{"tcp6", "tcp"} {
		path := fmt.Sprintf("/proc/%d/net/%s", pid, proto)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if port := parseListeningPort(string(data)); port > 0 {
			return port
		}
	}
	return 0
}

// parseListeningPort parses /proc/net/tcp format and returns the first
// listening port (state 0A).
func parseListeningPort(data string) int {
	lines := strings.Split(data, "\n")
	for _, line := range lines[1:] { // skip header
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		// field[3] is the connection state; 0A = LISTEN
		if fields[3] != "0A" {
			continue
		}
		// field[1] is local_address as hex_ip:hex_port
		parts := strings.Split(fields[1], ":")
		if len(parts) != 2 {
			continue
		}
		port, err := strconv.ParseInt(parts[1], 16, 32)
		if err != nil {
			continue
		}
		if port > 0 {
			return int(port)
		}
	}
	return 0
}
