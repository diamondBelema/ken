package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/diamondBelema/ken/internal/parser"
	"github.com/diamondBelema/ken/internal/render"
)

type ReadModel struct {
	files     []parser.NoteFile
	selected  int
	viewWidth int
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
	case tea.WindowSizeMsg:
		m.viewWidth = msg.Width
	}
	return m, nil
}

func (m ReadModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Read Notes"))
	b.WriteString("\n\n")

	if len(m.files) == 0 {
		b.WriteString(subtitleStyle.Render("No notes found."))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("Add .md files to notes/ directory"))
		return b.String()
	}

	for i, f := range m.files {
		prefix := "  "
		if i == m.selected {
			prefix = "→ "
		}
		b.WriteString(fmt.Sprintf("%s%s\n", prefix, f.Name))
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k navigate • enter read • q quit"))

	if len(m.files) > 0 {
		b.WriteString("\n\n")
		b.WriteString(subtitleStyle.Render(m.files[m.selected].Name))
		b.WriteString("\n")
		b.WriteString(render.RenderMarkdown(m.files[m.selected].Content, m.viewWidth-4))
	}

	return b.String()
}
