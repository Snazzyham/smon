package main

import "github.com/charmbracelet/lipgloss"

// Defaults use the terminal palette, so smon still looks right anywhere:
//   2=green(success)  4=blue(secondary)  6=cyan(accent)
// If ~/.config/themes/.current-theme points at a valid theme .conf, init swaps
// these for that palette's truecolor values.

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))
	selectedStyle = lipgloss.NewStyle().Reverse(true).Bold(true)
	normalStyle   = lipgloss.NewStyle()
	footerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	killMsgStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
)

func init() {
	if theme, ok := loadCurrentSystemTheme(); ok {
		applySystemThemeStyles(theme)
	}
}

func applySystemThemeStyles(theme systemTheme) {
	color := func(key string) lipgloss.Color {
		return lipgloss.Color(theme.colors[key])
	}

	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(color("accent"))
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(color("medium"))
	selectedStyle = lipgloss.NewStyle().Bold(true).
		Foreground(color("bg")).
		Background(color("accent"))
	normalStyle = lipgloss.NewStyle().Foreground(color("text"))
	footerStyle = lipgloss.NewStyle().Foreground(color("text_dim"))
	killMsgStyle = lipgloss.NewStyle().Foreground(color("success"))
}
