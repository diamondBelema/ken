package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginBottom(1)

	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1)

	frontStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230"))

	backStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	notesStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true).
			MarginTop(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	gradeButtonStyle = lipgloss.NewStyle().
				Bold(true).
				Padding(0, 1).
				MarginRight(1)

	gradeUnknownStyle = gradeButtonStyle.Foreground(lipgloss.Color("196"))
	gradeKnownLittleStyle = gradeButtonStyle.Foreground(lipgloss.Color("208"))
	gradeKnownFairlyStyle = gradeButtonStyle.Foreground(lipgloss.Color("220"))
	gradeKnownWellStyle = gradeButtonStyle.Foreground(lipgloss.Color("112"))
	gradeMasteredStyle = gradeButtonStyle.Foreground(lipgloss.Color("141"))

	finishedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("112")).
			MarginTop(2)

	noteInputHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("39")).
				MarginTop(1)
)
