package tui

import "github.com/charmbracelet/lipgloss"

type styles struct {
	header   lipgloss.Style
	subtle   lipgloss.Style
	meta     lipgloss.Style
	read     lipgloss.Style
	readMeta lipgloss.Style
	unread   lipgloss.Style
	selected lipgloss.Style
	footer   lipgloss.Style
}

func defaultStyles() styles {
	return styles{
		header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("25")).
			Padding(0, 1),
		subtle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("246")).
			PaddingLeft(1),
		meta: lipgloss.NewStyle().
			Foreground(lipgloss.Color("109")),
		read: lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")),
		readMeta: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		unread: lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")),
		selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("59")),
		footer: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("237")).
			Padding(0, 1),
	}
}
