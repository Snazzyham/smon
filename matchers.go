package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// appMatcher pairs a predicate with a display name.
type appMatcher struct {
	match func(ProcessInfo) bool
	name  string
}

// knownApps is the ordered lookup table for well-known applications.
// Extend this slice to add new apps.
var knownApps = []appMatcher{
	{func(p ProcessInfo) bool {
		return strings.Contains(p.Exe, "zen-browser") || strings.Contains(p.Cmdline, "zen-browser")
	}, "Zen Browser"},
	{func(p ProcessInfo) bool { return containsI(p.Exe, "discord") }, "Discord"},
	{func(p ProcessInfo) bool { return containsI(p.Exe, "slack") }, "Slack"},
	{func(p ProcessInfo) bool { return containsI(p.Exe, "spotify") }, "Spotify"},
	{func(p ProcessInfo) bool {
		return strings.Contains(p.Exe, "code") && strings.Contains(p.Cmdline, "--ms-enable")
	}, "VS Code"},
	{func(p ProcessInfo) bool { return strings.Contains(p.Exe, "cursor") }, "Cursor"},
	{func(p ProcessInfo) bool {
		return strings.Contains(p.Exe, "chrome") || strings.Contains(p.Exe, "chromium")
	}, "Chrome"},
	{func(p ProcessInfo) bool { return strings.Contains(p.Exe, "firefox") }, "Firefox"},
	{func(p ProcessInfo) bool { return containsI(p.Exe, "telegram") }, "Telegram"},
	{func(p ProcessInfo) bool { return containsI(p.Exe, "steam") }, "Steam"},
}

func matchKnownApp(p ProcessInfo) string {
	for _, app := range knownApps {
		if app.match(p) {
			return app.name
		}
	}
	return ""
}

func matchNode(p ProcessInfo, noPorts bool) string {
	base := filepath.Base(p.Exe)
	if base != "node" && base != "nodejs" && !strings.HasPrefix(p.Cmdline, "node ") {
		return ""
	}

	cmd := p.Cmdline
	var label string

	switch {
	case strings.Contains(cmd, "astro dev") || strings.Contains(cmd, "astro preview"):
		label = "astro dev"
	case strings.Contains(cmd, "vite"):
		label = "vite"
	case strings.Contains(cmd, "next dev") || strings.Contains(cmd, "next start"):
		label = "next.js"
	case strings.Contains(cmd, "nuxt"):
		label = "nuxt"
	case strings.Contains(cmd, "remix"):
		label = "remix"
	case strings.Contains(cmd, "webpack"):
		label = "webpack"
	case strings.Contains(cmd, "esbuild"):
		label = "esbuild"
	case strings.Contains(cmd, "tsx") || strings.Contains(cmd, "ts-node"):
		label = "ts: " + guessScriptName(cmd)
	case strings.Contains(cmd, "express"):
		label = "express"
	case strings.Contains(cmd, "nest"):
		label = "nestjs"
	default:
		label = guessScriptName(cmd)
	}

	if label == "" {
		label = "node"
	}

	// Port detection
	if !noPorts {
		if port := getListeningPort(p.PID); port > 0 {
			return fmt.Sprintf("Node (%s :%d)", label, port)
		}
	}

	return fmt.Sprintf("Node (%s)", label)
}

func matchJava(p ProcessInfo) string {
	if filepath.Base(p.Exe) != "java" {
		return ""
	}
	parts := strings.Fields(p.Cmdline)
	for i, part := range parts {
		if part == "-jar" && i+1 < len(parts) {
			return fmt.Sprintf("Java (%s)", filepath.Base(parts[i+1]))
		}
	}
	// Last non-flag argument is likely the main class
	for i := len(parts) - 1; i >= 0; i-- {
		if !strings.HasPrefix(parts[i], "-") && parts[i] != "java" {
			name := parts[i]
			if idx := strings.LastIndex(name, "."); idx >= 0 {
				name = name[idx+1:]
			}
			return fmt.Sprintf("Java (%s)", name)
		}
	}
	return "Java"
}

func matchPython(p ProcessInfo) string {
	base := filepath.Base(p.Exe)
	if base != "python" && base != "python3" && !strings.HasPrefix(base, "python3.") {
		return ""
	}
	parts := strings.Fields(p.Cmdline)
	for i, part := range parts {
		if part == "-m" && i+1 < len(parts) {
			return fmt.Sprintf("Python (%s)", parts[i+1])
		}
	}
	for _, part := range parts {
		if !strings.HasPrefix(part, "-") && part != base {
			return fmt.Sprintf("Python (%s)", filepath.Base(part))
		}
	}
	return "Python"
}

// guessScriptName extracts the most likely script filename from a cmdline.
func guessScriptName(cmdline string) string {
	parts := strings.Fields(cmdline)
	for _, part := range parts {
		if strings.HasSuffix(part, ".js") || strings.HasSuffix(part, ".ts") || strings.HasSuffix(part, ".mjs") {
			return filepath.Base(part)
		}
	}
	if len(parts) > 1 {
		last := parts[len(parts)-1]
		if !strings.HasPrefix(last, "-") {
			return filepath.Base(last)
		}
	}
	return ""
}

// containsI performs a case-insensitive substring check.
func containsI(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
