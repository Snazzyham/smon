package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	flag "github.com/spf13/pflag"
)

func main() {
	interval := flag.DurationP("interval", "i", 2*time.Second, "Refresh interval")
	noDocker := flag.BoolP("no-docker", "n", false, "Disable Docker container detection")
	noPorts := flag.Bool("no-ports", false, "Disable port scanning for Node processes")
	flag.Parse()

	m := newModel(*interval, *noDocker, *noPorts)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
