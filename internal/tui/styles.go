package tui

import "github.com/charmbracelet/lipgloss"

var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	special   = lipgloss.AdaptiveColor{Light: "#FF6B6B", Dark: "#FF6B6B"}
	highlight = lipgloss.AdaptiveColor{Light: "#4A90D9", Dark: "#7EB8FF"}
	success   = lipgloss.AdaptiveColor{Light: "#2ECC71", Dark: "#2ECC71"}
	warning   = lipgloss.AdaptiveColor{Light: "#F39C12", Dark: "#F39C12"}
	danger    = lipgloss.AdaptiveColor{Light: "#E74C3C", Dark: "#E74C3C"}
	muted     = lipgloss.AdaptiveColor{Light: "#95A5A6", Dark: "#7F8C8D"}
)

var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(highlight).
			MarginLeft(1).
			MarginBottom(1)

	StatusStyle = lipgloss.NewStyle().
			Foreground(muted).
			MarginLeft(1)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(danger).
			Bold(true).
			MarginLeft(1).
			MarginTop(1)

	InfoStyle = lipgloss.NewStyle().
			Foreground(success).
			MarginLeft(1)

	SubtleStyle = lipgloss.NewStyle().
			Foreground(subtle)

	TabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(muted)

	ActiveTabStyle = TabStyle.Copy().
			Foreground(lipgloss.Color("#FFF")).
			Background(highlight).
			Bold(true)

	HelpStyle = lipgloss.NewStyle().
			Foreground(muted).
			PaddingLeft(1).
			MarginTop(1)

	DividerStyle = lipgloss.NewStyle().
			Foreground(subtle)

	TableStyle = lipgloss.NewStyle().
			MarginLeft(1).
			MarginRight(1)

	DetailStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtle)

	PreviewStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Height(10).
			Width(60).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtle)
)
