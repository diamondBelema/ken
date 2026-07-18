package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/diamondBelema/ken/internal/progress"
	"github.com/diamondBelema/ken/internal/render"
)

type summariesState int

const (
	summariesList summariesState = iota
	summariesDetail
	summariesNew
)

type SummariesModel struct {
	progress    *progress.Progress
	subject     string
	state       summariesState
	summaries   []progress.Summary
	summaryIDs  []string
	selected    int
	viewWidth   int
	titleInput  textinput.Model
	contentInput textinput.Model
}

func NewSummariesModel(prog *progress.Progress, subject string) SummariesModel {
	ti := textinput.New()
	ti.Placeholder = "Summary title..."
	ti.Focus()
	ti.CharLimit = 200

	ci := textinput.New()
	ci.Placeholder = "Summary content..."
	ci.CharLimit = 5000

	return SummariesModel{
		progress:     prog,
		subject:      subject,
		state:        summariesList,
		titleInput:   ti,
		contentInput: ci,
	}
}

func (m SummariesModel) Init() tea.Cmd {
	return nil
}

func (m SummariesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case summariesList:
		return m.updateList(msg)
	case summariesDetail:
		return m.updateDetail(msg)
	case summariesNew:
		return m.updateNew(msg)
	}
	return m, nil
}

func (m SummariesModel) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.selected < len(m.summaries)-1 {
				m.selected++
			}
		case "k", "up":
			if m.selected > 0 {
				m.selected--
			}
		case "g":
			m.selected = 0
		case "G":
			m.selected = len(m.summaries) - 1
		case "enter":
			if len(m.summaries) > 0 {
				m.state = summariesDetail
			}
		case "s":
			return m.startNew(), nil
		case "q", "esc":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.viewWidth = msg.Width
	}
	return m, nil
}

func (m SummariesModel) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc", "q":
			m.state = summariesList
		}
	}
	return m, nil
}

func (m SummariesModel) updateNew(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.titleInput.Focused() {
				m.titleInput.Blur()
				m.contentInput.Focus()
				return m, nil
			}
			title := m.titleInput.Value()
			content := m.contentInput.Value()
			if strings.TrimSpace(title) != "" && strings.TrimSpace(content) != "" {
				m.progress.AddSummary(title, content, &progress.EntityRef{
					Type: "subject",
					ID:   m.subject,
				})
				m.refreshSummaries()
			}
			m.state = summariesList
			m.titleInput.SetValue("")
			m.contentInput.SetValue("")
			return m, nil
		case "esc":
			m.state = summariesList
			m.titleInput.SetValue("")
			m.contentInput.SetValue("")
			return m, nil
		case "tab":
			if m.titleInput.Focused() {
				m.titleInput.Blur()
				m.contentInput.Focus()
			} else {
				m.contentInput.Blur()
				m.titleInput.Focus()
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	if m.titleInput.Focused() {
		m.titleInput, cmd = m.titleInput.Update(msg)
	} else {
		m.contentInput, cmd = m.contentInput.Update(msg)
	}
	return m, cmd
}

func (m *SummariesModel) startNew() SummariesModel {
	m.state = summariesNew
	m.titleInput.SetValue("")
	m.contentInput.SetValue("")
	m.titleInput.Focus()
	return *m
}

func (m *SummariesModel) refreshSummaries() {
	m.summaries = nil
	m.summaryIDs = nil

	for id, summary := range m.progress.Summaries {
		if summary.LinkedTo != nil && summary.LinkedTo.Type == "subject" && summary.LinkedTo.ID == m.subject {
			m.summaries = append(m.summaries, summary)
			m.summaryIDs = append(m.summaryIDs, id)
		}
	}

	sort.Slice(m.summaries, func(i, j int) bool {
		return m.summaries[i].CreatedAt > m.summaries[j].CreatedAt
	})
}

func (m SummariesModel) View() string {
	m.refreshSummaries()

	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf("Summaries — %s", m.subject)))
	b.WriteString("\n")

	switch m.state {
	case summariesList:
		if len(m.summaries) == 0 {
			b.WriteString(subtitleStyle.Render("No summaries found. Press 's' to create one."))
			b.WriteString("\n")
		} else {
			for i, summary := range m.summaries {
				prefix := "  "
				if i == m.selected {
					prefix = "→ "
				}
				b.WriteString(fmt.Sprintf("%s%s\n", prefix, summary.Title))
			}
		}
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("j/k navigate • enter view • s new • q quit"))

	case summariesDetail:
		if len(m.summaries) > 0 {
			summary := m.summaries[m.selected]
			b.WriteString(subtitleStyle.Render(summary.Title))
			b.WriteString("\n")
			b.WriteString(render.RenderMarkdown(summary.Content, m.viewWidth-4))
			b.WriteString("\n")
			b.WriteString(helpStyle.Render("esc back"))
		}

	case summariesNew:
		b.WriteString(noteInputHeaderStyle.Render("New Summary"))
		b.WriteString("\n")
		b.WriteString(subtitleStyle.Render("Title:"))
		b.WriteString("\n")
		b.WriteString(m.titleInput.View())
		b.WriteString("\n")
		b.WriteString(subtitleStyle.Render("Content:"))
		b.WriteString("\n")
		b.WriteString(m.contentInput.View())
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("tab switch field • enter save • esc cancel"))
	}

	return b.String()
}
