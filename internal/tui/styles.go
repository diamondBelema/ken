package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorPrimary       = lipgloss.Color("39")
	colorSecondary     = lipgloss.Color("241")
	colorAccent        = lipgloss.Color("141")
	colorSuccess       = lipgloss.Color("112")
	colorWarning       = lipgloss.Color("220")
	colorDanger        = lipgloss.Color("196")
	colorMuted         = lipgloss.Color("247")
	colorText          = lipgloss.Color("252")
	colorTextBright    = lipgloss.Color("230")
	colorBorder        = lipgloss.Color("62")
	colorBg            = lipgloss.Color("236")
	colorBgHighlight   = lipgloss.Color("237")

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

	gradeUnknownStyle     = gradeButtonStyle.Foreground(colorDanger)
	gradeKnownLittleStyle = gradeButtonStyle.Foreground(lipgloss.Color("214"))
	gradeKnownFairlyStyle = gradeButtonStyle.Foreground(lipgloss.Color("178"))
	gradeKnownWellStyle   = gradeButtonStyle.Foreground(colorSuccess)
	gradeMasteredStyle    = gradeButtonStyle.Foreground(lipgloss.Color("45"))

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

	// Dashboard styles
	dashCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			Padding(0, 1)

	dashCardSelectedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary).
				Padding(0, 1).
				Background(colorBgHighlight)

	dashSubjectStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorTextBright)

	dashSubjectSelectedStyle = lipgloss.NewStyle().
					Bold(true).
					Foreground(colorPrimary)

	dashDetailStyle = lipgloss.NewStyle().
				Foreground(colorMuted)

	dashBadgeDueStyle = lipgloss.NewStyle().
				Foreground(colorWarning)

	dashBadgeNeverStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Italic(true)

	dashHintStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Italic(true)

	dashActionBarStyle = lipgloss.NewStyle().
				Foreground(colorTextBright).
				Bold(true)

	dashActionItemStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				PaddingRight(2)

	dashActionItemSelStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true).
				PaddingRight(2)

	dashFilterStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	dashHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorTextBright).
				MarginBottom(0)

	dashTaglineStyle = lipgloss.NewStyle().
				Foreground(colorText).
				MarginBottom(1)

	dashStatsRowStyle = lipgloss.NewStyle().
				Foreground(colorMuted)

	// Confidence distribution
	dashDistWeakStyle  = lipgloss.NewStyle().Foreground(colorDanger)
	dashDistDevStyle   = lipgloss.NewStyle().Foreground(colorWarning)
	dashDistStrongStyle = lipgloss.NewStyle().Foreground(colorSuccess)

	// Confidence bar
	dashConfBarFilled = lipgloss.NewStyle().Foreground(colorSuccess)
	dashConfBarEmpty  = lipgloss.NewStyle().Foreground(colorMuted)

	// Activity panel
	dashPanelHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPrimary)

	dashPanelSubjectStyle = lipgloss.NewStyle().
				Foreground(colorTextBright).
				Bold(true)

	dashPanelItemStyle = lipgloss.NewStyle().
				Foreground(colorText)

	dashPanelTimeStyle = lipgloss.NewStyle().
				Foreground(colorMuted)

	dashPanelEmptyStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Italic(true)

	dashSeparatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("102"))
)
