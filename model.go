package main

import (
	"fmt"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Messages
type tickMsg time.Time
type processDataMsg []AppGroup
type killResultMsg struct {
	name  string
	count int
	err   error
}
type dockerRefreshMsg map[int32]string

type model struct {
	groups     []AppGroup
	cursor     int
	sortBy     SortMode
	filterText string
	filtering  bool
	detail     bool
	killMsg    string
	killMsgTTL int
	width      int
	height     int
	dockerMap  map[int32]string
	interval   time.Duration
	noDocker   bool
	noPorts    bool
}

func newModel(interval time.Duration, noDocker, noPorts bool) model {
	return model{
		interval:  interval,
		noDocker:  noDocker,
		noPorts:   noPorts,
		sortBy:    SortByMem,
		dockerMap: make(map[int32]string),
	}
}

func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		tickCmd(m.interval),
		collectProcessesCmd(m.dockerMap, m.sortBy, m.noPorts),
	}
	if !m.noDocker {
		cmds = append(cmds, refreshDocker())
	}
	return tea.Batch(cmds...)
}

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func collectProcessesCmd(dockerMap map[int32]string, sortBy SortMode, noPorts bool) tea.Cmd {
	return func() tea.Msg {
		procs, err := collectAllProcesses()
		if err != nil {
			return processDataMsg(nil)
		}
		groups := groupProcesses(procs, dockerMap, noPorts, sortBy)
		return processDataMsg(groups)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		cmds := []tea.Cmd{
			tickCmd(m.interval),
			collectProcessesCmd(m.dockerMap, m.sortBy, m.noPorts),
		}
		if m.killMsgTTL > 0 {
			m.killMsgTTL--
			if m.killMsgTTL == 0 {
				m.killMsg = ""
			}
		}
		return m, tea.Batch(cmds...)

	case processDataMsg:
		m.groups = filterGroups([]AppGroup(msg), m.filterText)
		if m.cursor >= len(m.groups) {
			m.cursor = max(0, len(m.groups)-1)
		}
		return m, nil

	case killResultMsg:
		if msg.err != nil {
			m.killMsg = fmt.Sprintf("Error killing %s: %v", msg.name, msg.err)
		} else {
			m.killMsg = fmt.Sprintf("Killed %s (%d processes)", msg.name, msg.count)
		}
		m.killMsgTTL = 1
		return m, nil

	case dockerRefreshMsg:
		m.dockerMap = map[int32]string(msg)
		if !m.noDocker {
			return m, tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
				dm, _ := buildDockerMap()
				if dm == nil {
					dm = make(map[int32]string)
				}
				return dockerRefreshMsg(dm)
			})
		}
		return m, nil
	}

	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.filtering {
		switch msg.String() {
		case "esc":
			m.filtering = false
			m.filterText = ""
			return m, nil
		case "enter":
			m.filtering = false
			return m, nil
		case "backspace":
			if len(m.filterText) > 0 {
				m.filterText = m.filterText[:len(m.filterText)-1]
			}
			return m, nil
		default:
			if len(msg.String()) == 1 {
				m.filterText += msg.String()
			}
			return m, nil
		}
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.groups)-1 {
			m.cursor++
		}
	case "s":
		if m.sortBy == SortByCPU {
			m.sortBy = SortByMem
		} else {
			m.sortBy = SortByCPU
		}
		sortGroups(m.groups, m.sortBy)
	case " ":
		m.detail = !m.detail
	case "esc":
		m.detail = false
	case "d", "delete":
		return m, m.killSelected(syscall.SIGTERM)
	case "D":
		return m, m.killSelected(syscall.SIGKILL)
	case "/":
		m.filtering = true
		m.filterText = ""
	}

	return m, nil
}

func (m model) killSelected(sig syscall.Signal) tea.Cmd {
	if len(m.groups) == 0 || m.cursor >= len(m.groups) {
		return nil
	}
	group := m.groups[m.cursor]
	return func() tea.Msg {
		var lastErr error
		for _, pid := range group.PIDs {
			if err := syscall.Kill(int(pid), sig); err != nil && err != syscall.ESRCH {
				lastErr = err
			}
		}
		return killResultMsg{name: group.Name, count: len(group.PIDs), err: lastErr}
	}
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Header
	sortLabel := "CPU"
	if m.sortBy == SortByMem {
		sortLabel = "MEM"
	}
	header := titleStyle.Render(fmt.Sprintf(" smon — sorted by: %s ▼", sortLabel))
	controls := footerStyle.Render("  [s]ort  [d]elete  [q]uit")
	b.WriteString(header + controls + "\n\n")

	// Column headers
	nameW := m.nameWidth()
	colHeader := fmt.Sprintf(" %-*s  %10s  %10s  %6s", nameW, "APP", "CPU %", "MEM (MB)", "PROCS")
	b.WriteString(headerStyle.Render(colHeader) + "\n")
	b.WriteString(headerStyle.Render(strings.Repeat("─", min(m.width, len(colHeader)+2))) + "\n")

	// Table rows
	visibleRows := m.height - 7
	if visibleRows < 1 {
		visibleRows = 1
	}

	start := 0
	if m.cursor >= start+visibleRows {
		start = m.cursor - visibleRows + 1
	}
	end := min(start+visibleRows, len(m.groups))

	for i := start; i < end; i++ {
		g := m.groups[i]
		marker := "  "
		if i == m.cursor {
			marker = "▶ "
		}

		name := g.Name
		if len(name) > nameW {
			name = name[:nameW-1] + "…"
		}

		row := fmt.Sprintf("%s%-*s  %9.1f%%  %8.0f MB  %5d",
			marker, nameW, name, g.CPUPercent, g.MemoryMB, g.ProcessCount)

		if i == m.cursor {
			b.WriteString(selectedStyle.Render(row))
		} else {
			b.WriteString(normalStyle.Render(row))
		}
		b.WriteString("\n")
	}

	// Detail panel
	if m.detail && m.cursor < len(m.groups) {
		b.WriteString(m.renderDetail())
	}

	// Footer
	b.WriteString("\n")
	if m.killMsg != "" {
		b.WriteString(killMsgStyle.Render(" " + m.killMsg))
	} else if m.filtering {
		b.WriteString(footerStyle.Render(fmt.Sprintf(" /%s█", m.filterText)))
	} else if m.detail {
		b.WriteString(footerStyle.Render(" [space] close detail │ d = kill │ q = quit"))
	} else {
		b.WriteString(footerStyle.Render(" ▶ = selected │ ↑↓/jk navigate │ s = sort │ [space] detail │ d = kill │ D = force kill │ / = filter │ q = quit"))
	}

	return b.String()
}

func (m model) nameWidth() int {
	w := m.width - 35
	if w < 15 {
		w = 15
	}
	if w > 40 {
		w = 40
	}
	return w
}

func (m model) renderDetail() string {
	g := m.groups[m.cursor]
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(headerStyle.Render(fmt.Sprintf(" %s", g.Name)) + "\n")
	b.WriteString(headerStyle.Render(strings.Repeat("─", min(m.width, 80))) + "\n")

	// Column header
	b.WriteString(footerStyle.Render(fmt.Sprintf("  %-8s  %8s  %9s  %s", "PID", "CPU %", "MEM (MB)", "COMMAND")) + "\n")

	// Cap how many rows we show so it doesn't overflow the terminal
	maxRows := m.height/3
	if maxRows < 3 {
		maxRows = 3
	}
	procs := g.Procs
	if len(procs) > maxRows {
		procs = procs[:maxRows]
	}

	for _, p := range procs {
		cmd := p.Cmdline
		if cmd == "" {
			cmd = p.Exe
		}
		// Truncate command to fit terminal width
		maxCmdW := m.width - 33
		if maxCmdW < 10 {
			maxCmdW = 10
		}
		if len(cmd) > maxCmdW {
			cmd = "…" + cmd[len(cmd)-maxCmdW+1:]
		}
		b.WriteString(fmt.Sprintf("  %-8d  %7.1f%%  %8.0f MB  %s\n",
			p.PID, p.CPUPercent, p.MemoryMB, cmd))
	}

	if len(g.Procs) > maxRows {
		b.WriteString(footerStyle.Render(fmt.Sprintf("  … and %d more", len(g.Procs)-maxRows)) + "\n")
	}

	return b.String()
}

func filterGroups(groups []AppGroup, filter string) []AppGroup {
	if filter == "" {
		return groups
	}
	f := strings.ToLower(filter)
	var result []AppGroup
	for _, g := range groups {
		if strings.Contains(strings.ToLower(g.Name), f) {
			result = append(result, g)
		}
	}
	return result
}
