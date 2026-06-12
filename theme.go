package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type systemTheme struct {
	name   string
	colors map[string]string
}

func loadCurrentSystemTheme() (systemTheme, bool) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return systemTheme{}, false
	}

	themesDir := filepath.Join(configDir, "themes")
	currentThemePath := filepath.Join(themesDir, ".current-theme")
	data, err := os.ReadFile(currentThemePath)
	if err != nil {
		return systemTheme{}, false
	}

	themeName := strings.TrimSpace(string(data))
	if themeName == "" {
		return systemTheme{}, false
	}

	// The theme apply script stores a bare theme name, but keep this defensive so
	// a malformed .current-theme cannot escape ~/.config/themes.
	themeName = strings.TrimSuffix(filepath.Base(themeName), ".conf")
	if themeName == "" || strings.HasPrefix(themeName, ".") {
		return systemTheme{}, false
	}

	theme, err := loadSystemThemeFile(filepath.Join(themesDir, themeName+".conf"), themeName)
	if err != nil {
		return systemTheme{}, false
	}
	return theme, true
}

func loadSystemThemeFile(path, name string) (systemTheme, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return systemTheme{}, err
	}

	colors, err := parseSystemTheme(data)
	if err != nil {
		return systemTheme{}, err
	}

	return systemTheme{name: name, colors: colors}, nil
}

func parseSystemTheme(data []byte) (map[string]string, error) {
	raw := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = cleanThemeValue(value)
		if key != "" && value != "" {
			raw[key] = value
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	resolved := make(map[string]string)
	var resolve func(key string, seen map[string]bool) string
	resolve = func(key string, seen map[string]bool) string {
		if color, ok := resolved[key]; ok {
			return color
		}

		value, ok := raw[key]
		if !ok {
			return ""
		}

		if strings.HasPrefix(value, "$") {
			ref := strings.TrimSpace(strings.TrimPrefix(value, "$"))
			if ref == "" || seen[key] {
				return ""
			}
			seen[key] = true
			value = resolve(ref, seen)
		}

		color, ok := normalizeThemeColor(value)
		if !ok {
			return ""
		}

		resolved[key] = color
		return color
	}

	for key := range raw {
		resolve(key, make(map[string]bool))
	}

	for _, key := range []string{"bg", "fg", "dark", "medium", "light"} {
		if resolved[key] == "" {
			return nil, fmt.Errorf("theme missing required color %q", key)
		}
	}

	setDefault := func(key, fallback string) {
		if resolved[key] == "" {
			resolved[key] = resolved[fallback]
		}
	}
	setDefault("accent", "light")
	setDefault("surface", "dark")
	setDefault("text", "fg")
	setDefault("text_dim", "medium")

	setSemanticDefault := func(key, terminalColor string) {
		if resolved[key] != "" {
			return
		}
		if resolved[terminalColor] != "" {
			resolved[key] = resolved[terminalColor]
			return
		}
		resolved[key] = resolved["light"]
	}
	setSemanticDefault("success", "green")
	setSemanticDefault("warning", "yellow")
	setSemanticDefault("error", "red")

	return resolved, nil
}

func cleanThemeValue(value string) string {
	value = strings.TrimSpace(value)
	// Theme files normally use whole-line comments. Also allow inline comments
	// after whitespace without treating a leading #RRGGBB as a comment.
	if i := strings.Index(value, " #"); i >= 0 {
		value = strings.TrimSpace(value[:i])
	}
	return strings.Trim(value, "\"'")
}

func normalizeThemeColor(value string) (string, bool) {
	value = strings.TrimSpace(strings.TrimPrefix(value, "#"))
	if len(value) == 3 {
		value = string([]byte{value[0], value[0], value[1], value[1], value[2], value[2]})
	}
	if len(value) != 6 {
		return "", false
	}
	for _, r := range value {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return "", false
		}
	}
	return "#" + strings.ToUpper(value), true
}
