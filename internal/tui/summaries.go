package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/diamondBelema/ken/internal/parser"
	"github.com/diamondBelema/ken/internal/progress"
	"github.com/diamondBelema/ken/internal/render"
)

type summariesState int

const (
	summariesList summariesState = iota
	summariesDetail
	summariesNew
	summariesEdit
	summariesDeleteConfirm
)

type SummariesModel struct {
	progress     *progress.Progress
	concepts     []parser.Concept
	subject      string
	state        summariesState
	summaries    []progress.Summary
	summaryIDs   []string
	isContent    []bool
	selected     int
	scrollTop    int
	detailScroll int
	viewWidth    int
	viewHeight   int
	titleInput   textinput.Model
	contentInput textinput.Model
	editID       string
}

func NewSummariesModel(prog *progress.Progress, concepts []parser.Concept, subject string) SummariesModel {
	ti := textinput.New()
	ti.Placeholder = "Summary title..."
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 50

	ci := textinput.New()
	ci.Placeholder = "Summary content..."
	ci.CharLimit = 5000
	ci.Width = 60

	return SummariesModel{
		progress:     prog,
		concepts:     concepts,
		subject:      subject,
		state:        summariesList,
		titleInput:   ti,
		contentInput: ci,
	}
}

func (m SummariesModel) Init() tea.Cmd {
	return func() tea.Msg {
		return summariesInitMsg{}
	}
}

type summariesInitMsg struct{}

func (m SummariesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewWidth = msg.Width
		m.viewHeight = msg.Height
	case summariesInitMsg:
		m.refreshSummaries()
		return m, nil
	}

	switch m.state {
	case summariesList:
		return m.updateList(msg)
	case summariesDetail:
		return m.updateDetail(msg)
	case summariesNew:
		return m.updateNew(msg)
	case summariesEdit:
		return m.updateEdit(msg)
	case summariesDeleteConfirm:
		return m.updateDeleteConfirm(msg)
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
			m.selected = len(m.summaries) - 1
			m.clampScroll()
		case "enter":
			if len(m.summaries) > 0 {
				m.state = summariesDetail
			}
		case "s":
			return m.startNew(), nil
		case "e":
			if len(m.summaries) > 0 && !m.isContent[m.selected] {
				return m.startEdit(), nil
			}
		case "x":
			if len(m.summaries) > 0 && !m.isContent[m.selected] {
				m.state = summariesDeleteConfirm
				return m, nil
			}
		case "q", "esc":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m SummariesModel) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc", "q":
			m.state = summariesList
			m.detailScroll = 0
		case "j", "down":
			m.detailScroll++
		case "k", "up":
			if m.detailScroll > 0 {
				m.detailScroll--
			}
		case "g":
			m.detailScroll = 0
		case "G":
			m.detailScroll = 999999
		case "e":
			return m.startEdit(), nil
		case "x":
			m.state = summariesDeleteConfirm
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

func (m SummariesModel) updateEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				m.progress.EditSummary(m.editID, title, content)
				m.refreshSummaries()
			}
			m.state = summariesDetail
			m.titleInput.SetValue("")
			m.contentInput.SetValue("")
			return m, nil
		case "esc":
			m.state = summariesDetail
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

func (m SummariesModel) updateDeleteConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "y", "Y":
			if len(m.summaries) > 0 {
				m.progress.DeleteSummary(m.summaryIDs[m.selected])
				m.refreshSummaries()
				if m.selected >= len(m.summaries) {
					m.selected = len(m.summaries) - 1
				}
			}
			m.state = summariesList
		case "n", "N", "esc":
			m.state = summariesList
		}
	}
	return m, nil
}

func (m *SummariesModel) startNew() SummariesModel {
	m.state = summariesNew
	m.titleInput.SetValue("")
	m.contentInput.SetValue("")
	m.titleInput.Focus()
	return *m
}

func (m *SummariesModel) startEdit() SummariesModel {
	if len(m.summaries) == 0 {
		return *m
	}
	summary := m.summaries[m.selected]
	m.state = summariesEdit
	m.editID = m.summaryIDs[m.selected]
	m.titleInput.SetValue(summary.Title)
	m.contentInput.SetValue(summary.Content)
	m.titleInput.Focus()
	return *m
}

func (m *SummariesModel) refreshSummaries() {
	m.summaries = nil
	m.summaryIDs = nil
	m.isContent = nil

	// Load content summaries from parsed concepts
	for _, c := range m.concepts {
		if c.Summary != "" {
			m.summaries = append(m.summaries, progress.Summary{
				Title:   c.Name,
				Content: c.Summary,
			})
			m.summaryIDs = append(m.summaryIDs, "content-"+c.ID)
			m.isContent = append(m.isContent, true)
		}
	}

	// Load user-created summaries linked to this subject
	for id, summary := range m.progress.Summaries {
		if summary.LinkedTo != nil && summary.LinkedTo.Type == "subject" && summary.LinkedTo.ID == m.subject {
			m.summaries = append(m.summaries, summary)
			m.summaryIDs = append(m.summaryIDs, id)
			m.isContent = append(m.isContent, false)
		}
	}

	sort.SliceStable(m.summaries, func(i, j int) bool {
		ci := m.isContent[i]
		cj := m.isContent[j]
		if ci != cj {
			return ci // content summaries first
		}
		return m.summaries[i].CreatedAt > m.summaries[j].CreatedAt
	})
}

func (m *SummariesModel) clampScroll() {
	visible := m.viewHeight - 3
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

func (m SummariesModel) View() string {
	if m.viewWidth == 0 {
		m.viewWidth = 80
	}

	var b strings.Builder

	header := titleStyle.Render(fmt.Sprintf("  summaries · %s  ", m.subject))
	b.WriteString(header)
	b.WriteString("\n\n")

	switch m.state {
	case summariesList:
		if len(m.summaries) == 0 {
			empty := lipgloss.NewStyle().
				Foreground(colorMuted).
				Padding(4, 2).
				Render("No summaries found.\n\n  Press 's' to create one.")
			b.WriteString(empty)
		} else {
			visible := m.viewHeight - 3
			if visible < 1 {
				visible = 10
			}
			end := m.scrollTop + visible
			if end > len(m.summaries) {
				end = len(m.summaries)
			}

			for i := m.scrollTop; i < end; i++ {
				label := m.summaries[i].Title
				if m.isContent[i] {
					label = label + "  " + lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render("[content]")
				}
				if i == m.selected {
					b.WriteString(listItemSelectedStyle.Render(fmt.Sprintf("  %s", label)))
					b.WriteString("\n")
				} else {
					b.WriteString(fmt.Sprintf("  %s\n", listItemStyle.Render(label)))
				}
			}
		}
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  j/k navigate  ·  enter view  ·  s new  ·  e edit  ·  x delete  ·  q quit"))

	case summariesDetail:
		if len(m.summaries) > 0 {
			summary := m.summaries[m.selected]
			b.WriteString(subtitleStyle.Render(summary.Title))
			b.WriteString("\n")
			if m.isContent[m.selected] {
				b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render("  Content Summary"))
				b.WriteString("\n")
			}

			rendered := render.RenderMarkdown(summary.Content, m.viewWidth-4)
			lines := strings.Split(rendered, "\n")

			// Header=1(title) + 1(blank) + maybe 1(content label) = 2 or 3, footer=1
			headerH := 3
			if !m.isContent[m.selected] {
				headerH = 2
			}
			visible := m.viewHeight - headerH - 1
			if visible < 1 {
				visible = 10
			}

			// Clamp scroll
			maxScroll := len(lines) - visible
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.detailScroll > maxScroll {
				m.detailScroll = maxScroll
			}

			start := m.detailScroll
			end := start + visible
			if end > len(lines) {
				end = len(lines)
			}

			for _, line := range lines[start:end] {
				b.WriteString(line)
				b.WriteString("\n")
			}

			b.WriteString(helpStyle.Render("  j/k scroll  ·  g/G top/bottom  ·  esc back"))
		}

	case summariesNew:
		b.WriteString(noteInputHeaderStyle.Render("  new summary"))
		b.WriteString("\n")
		b.WriteString(subtitleStyle.Render("  Title:"))
		b.WriteString("\n  ")
		b.WriteString(m.titleInput.View())
		b.WriteString("\n")
		b.WriteString(subtitleStyle.Render("  Content:"))
		b.WriteString("\n  ")
		b.WriteString(m.contentInput.View())
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("  tab switch field  ·  enter save  ·  esc cancel"))

	case summariesEdit:
		b.WriteString(noteInputHeaderStyle.Render("  edit summary"))
		b.WriteString("\n")
		b.WriteString(subtitleStyle.Render("  Title:"))
		b.WriteString("\n  ")
		b.WriteString(m.titleInput.View())
		b.WriteString("\n")
		b.WriteString(subtitleStyle.Render("  Content:"))
		b.WriteString("\n  ")
		b.WriteString(m.contentInput.View())
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("  tab switch field  ·  enter save  ·  esc cancel"))

	case summariesDeleteConfirm:
		b.WriteString(lipgloss.NewStyle().
			Foreground(colorDanger).
			Bold(true).
			Render("  Delete this summary? (y/n)"))
		b.WriteString("\n")
	}

	return b.String()
}
