package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/diamondBelema/ken/internal/parser"
	"github.com/diamondBelema/ken/internal/render"
)

type ReadModel struct {
	files      []parser.NoteFile
	selected   int
	viewWidth  int
	viewHeight int
}

func NewReadModel(files []parser.NoteFile) ReadModel {
	return ReadModel{
		files: files,
	}
}

func (m ReadModel) Init() tea.Cmd {
	return nil
}

func (m ReadModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewWidth = msg.Width
		m.viewHeight = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.selected < len(m.files)-1 {
				m.selected++
			}
		case "k", "up":
			if m.selected > 0 {
				m.selected--
			}
		case "g":
			m.selected = 0
		case "G":
			m.selected = len(m.files) - 1
		case "q", "esc":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m ReadModel) View() string {
	if m.viewWidth == 0 {
		m.viewWidth = 80
	}

	var b strings.Builder

	header := titleStyle.Render("  read  ")
	b.WriteString(header)
	b.WriteString("\n\n")

	if len(m.files) == 0 {
		empty := lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(4, 2).
			Render("No notes found.\n\n  Add .md files to notes/ directory")
		b.WriteString(empty)
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("  q quit"))
		return b.String()
	}

	for i, f := range m.files {
		if i == m.selected {
			b.WriteString(listItemSelectedStyle.Render(fmt.Sprintf("  %s", f.Name)))
		} else {
			b.WriteString(fmt.Sprintf("  %s\n", listItemStyle.Render(f.Name)))
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  j/k navigate  ·  q quit"))

	if len(m.files) > 0 {
		b.WriteString("\n\n")
		b.WriteString(subtitleStyle.Render(m.files[m.selected].Name))
		b.WriteString("\n")
		b.WriteString(render.RenderMarkdown(m.files[m.selected].Content, m.viewWidth-4))
	}

	return b.String()
}
