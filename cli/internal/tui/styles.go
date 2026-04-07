package tui

import "github.com/charmbracelet/lipgloss"

var (
	Red    = lipgloss.Color("#eb3449")
	Green  = lipgloss.Color("#4ade80")
	Yellow = lipgloss.Color("#facc15")
	Blue   = lipgloss.Color("#60a5fa")
	Gray   = lipgloss.Color("#a3a3a3")
	DarkBg = lipgloss.Color("#171717")
	White  = lipgloss.Color("#f5f5f5")
	Amber  = lipgloss.Color("#f59e0b")

	BannerStyle = lipgloss.NewStyle().
			Foreground(Red).
			Bold(true).
			PaddingLeft(1)

	TabActiveStyle = lipgloss.NewStyle().
			Foreground(White).
			Background(Red).
			Padding(0, 2).
			Bold(true)

	TabInactiveStyle = lipgloss.NewStyle().
				Foreground(Gray).
				Padding(0, 2)

	TabBarStyle = lipgloss.NewStyle().
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(Gray).
			MarginBottom(1)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(Gray).
			Align(lipgloss.Right).
			PaddingRight(3)

	StatusRunning  = lipgloss.NewStyle().Foreground(Green).Bold(true)
	StatusStopped  = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Bold(true)
	StatusSleeping = lipgloss.NewStyle().Foreground(Yellow).Bold(true)
	StatusWaking   = lipgloss.NewStyle().Foreground(Blue).Bold(true)
	StatusUnknown  = lipgloss.NewStyle().Foreground(Gray)

	TitleStyle = lipgloss.NewStyle().
			Foreground(Red).
			Bold(true).
			MarginBottom(1)

	HelpStyle = lipgloss.NewStyle().
			Foreground(Gray)
)

func StatusStyle(status string) lipgloss.Style {
	switch status {
	case "running":
		return StatusRunning
	case "exited", "stopped", "dead":
		return StatusStopped
	case "sleeping":
		return StatusSleeping
	case "waking":
		return StatusWaking
	default:
		return StatusUnknown
	}
}
