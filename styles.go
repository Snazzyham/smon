package main

import "github.com/charmbracelet/lipgloss"

// Colors pull from terminal palette — mapped by the user's theme system:
//   0=bg  1=red  2=green  3=yellow  4=blue(medium)  5=magenta  6=cyan(light)  7=white(fg)
//   8=bright_black(dark)  9-15=bright variants

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))  // light accent
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))  // medium accent
	selectedStyle = lipgloss.NewStyle().Reverse(true).Bold(true)                    // terminal-native reverse
	normalStyle   = lipgloss.NewStyle()
	footerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))             // medium (secondary text)
	killMsgStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))             // green (success)
)
