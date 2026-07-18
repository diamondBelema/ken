package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/diamondBelema/ken/internal/parser"
	"github.com/diamondBelema/ken/internal/render"
)

type readState int

const (
	readList readState = iota
	readDetail
)

type ReadModel struct {
	files      []parser.NoteFile
	state      readState
	selected   int
	scrollTop  int
	viewWidth  int
	viewHeight int
}

func NewReadModel(files []parser.NoteFile) ReadModel {
	return ReadModel{
		files: files,
		state: readList,
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
		switch m.state {
		case readList:
			switch msg.String() {
			case "j", "down":
				if m.selected < len(m.files)-1 {
					m.selected++
					m.clampScroll()
				}
			case "k", "up":
				if m.selected > 0 {
					m.selected--
					m.clampScroll()
				}
			case "g":
				m.selected = 0
				m.clampScroll()
			case "G":
				m.selected = len(m.files) - 1
				m.clampScroll()
			case "enter":
				if len(m.files) > 0 {
					m.state = readDetail
				}
			case "q", "esc":
				return m, tea.Quit
			}
		case readDetail:
			switch msg.String() {
			case "q", "esc":
				m.state = readList
			case "j", "down":
				m.scrollTop++
			case "k", "up":
				if m.scrollTop > 0 {
					m.scrollTop--
				}
			}
		}
	}
	return m, nil
}

func (m *ReadModel) clampScroll() {
	visible := m.viewHeight - 4
	if visible < 1 {
		visible = 10
	}
	if m.selected < m.scrollTop {
		m.scrollTop = m.selected
	}
	if m.selected >= m.scrollTop+visible {
		m.scrollTop = m.selected - visible + 1
	}
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

	switch m.state {
	case readList:
		visible := m.viewHeight - 4
		if visible < 1 {
			visible = 10
		}
		end := m.scrollTop + visible
		if end > len(m.files) {
			end = len(m.files)
		}

		for i := m.scrollTop; i < end; i++ {
			if i == m.selected {
				b.WriteString(listItemSelectedStyle.Render(fmt.Sprintf("  %s", m.files[i].Name)))
				b.WriteString("\n")
			} else {
				b.WriteString(fmt.Sprintf("  %s\n", listItemStyle.Render(m.files[i].Name)))
			}
		}
		b.WriteString("\n")
		if len(m.files) > visible {
			b.WriteString(helpStyle.Render(fmt.Sprintf("  %d files  ·  j/k navigate  ·  enter read  ·  q quit", len(m.files))))
		} else {
			b.WriteString(helpStyle.Render("  j/k navigate  ·  enter read  ·  q quit"))
		}

	case readDetail:
		if m.selected < len(m.files) {
			b.WriteString(subtitleStyle.Render(m.files[m.selected].Name))
			b.WriteString("\n")
			b.WriteString(render.RenderMarkdown(m.files[m.selected].Content, m.viewWidth-4))
			b.WriteString("\n\n")
			b.WriteString(helpStyle.Render("  j/k scroll  ·  esc back"))
		}
	}

	return b.String()
}
