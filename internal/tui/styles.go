package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorPrimary    = lipgloss.Color("39")
	colorSecondary  = lipgloss.Color("241")
	colorAccent     = lipgloss.Color("141")
	colorSuccess    = lipgloss.Color("112")
	colorWarning    = lipgloss.Color("220")
	colorDanger     = lipgloss.Color("196")
	colorMuted      = lipgloss.Color("243")
	colorText       = lipgloss.Color("252")
	colorTextBright = lipgloss.Color("230")
	colorBorder     = lipgloss.Color("62")
	colorBg         = lipgloss.Color("236")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorTextBright).
			Background(colorPrimary).
			Padding(0, 2).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginTop(1).
			MarginBottom(0)

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			MarginTop(1)

	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1)

	frontStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorTextBright)

	backStyle = lipgloss.NewStyle().
			Foreground(colorText)

	notesStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true).
			MarginTop(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			MarginTop(1)

	gradeButtonStyle = lipgloss.NewStyle().
				Bold(true).
				Padding(0, 1).
				MarginRight(1)

	gradeUnknownStyle    = gradeButtonStyle.Foreground(colorDanger)
	gradeKnownLittleStyle = gradeButtonStyle.Foreground(colorWarning)
	gradeKnownFairlyStyle = gradeButtonStyle.Foreground(lipgloss.Color("220"))
	gradeKnownWellStyle  = gradeButtonStyle.Foreground(colorSuccess)
	gradeMasteredStyle   = gradeButtonStyle.Foreground(colorAccent)

	finishedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSuccess).
			MarginTop(2)

	noteInputHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPrimary).
				MarginTop(1)

 listItemStyle = lipgloss.NewStyle().
			Foreground(colorText).
			PaddingLeft(1)

	listItemSelectedStyle = lipgloss.NewStyle().
				Foreground(colorTextBright).
				Background(colorPrimary).
				PaddingLeft(1).
				Bold(true)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Background(colorBg).
			Padding(0, 1)

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder)
)
