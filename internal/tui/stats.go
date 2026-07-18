package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type StatsModel struct {
	err         error
	viewWidth   int
	viewHeight  int
}

func NewStatsModel() StatsModel {
	return StatsModel{}
}

func (m StatsModel) Init() tea.Cmd {
	return nil
}

func (m StatsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewWidth = msg.Width
		m.viewHeight = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m StatsModel) View() string {
	if m.viewWidth == 0 {
		m.viewWidth = 80
	}

	var b strings.Builder

	header := titleStyle.Render("  stats  ")
	b.WriteString(header)
	b.WriteString("\n\n")

	empty := lipgloss.NewStyle().
		Foreground(colorMuted).
		Padding(4, 2).
		Render("No data yet.\n\n  Complete some study sessions to see stats.")
	b.WriteString(empty)
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("  q quit"))

	return b.String()
}
