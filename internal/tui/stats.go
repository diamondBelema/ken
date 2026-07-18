package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type StatsModel struct {
	err error
}

func NewStatsModel() StatsModel {
	return StatsModel{}
}

func (m StatsModel) Init() tea.Cmd {
	return nil
}

func (m StatsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m StatsModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Stats"))
	b.WriteString("\n\n")
	b.WriteString(subtitleStyle.Render("No data yet."))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Complete some study sessions to see stats."))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Press q to exit."))
	return b.String()
}
