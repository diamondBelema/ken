package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/diamondBelema/ken/internal/parser"
	"github.com/diamondBelema/ken/internal/progress"
	"github.com/diamondBelema/ken/internal/render"
)

type reflectState int

const (
	reflectTyping reflectState = iota
	reflectShowAnswer
	reflectSaved
)

type ReflectModel struct {
	subject      string
	concepts     []parser.Concept
	prog         *progress.Progress
	state        reflectState
	currentIdx   int
	input        textarea.Model
	showAnswer   bool
	message      string
	viewWidth    int
	viewHeight   int
}

func NewReflectModel(subject string, concepts []parser.Concept, prog *progress.Progress, startConceptID string) ReflectModel {
	ta := textarea.New()
	ta.Placeholder = "Type your explanation here..."
	ta.Focus()
	ta.CharLimit = 5000
	ta.SetWidth(70)
	ta.SetHeight(6)
	ta.ShowLineNumbers = false

	startIdx := 0
	if startConceptID != "" {
		for i, c := range concepts {
			if c.ID == startConceptID {
				startIdx = i
				break
			}
		}
	}

	return ReflectModel{
		subject:    subject,
		concepts:   concepts,
		prog:       prog,
		state:      reflectTyping,
		currentIdx: startIdx,
		input:      ta,
	}
}

func (m ReflectModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m ReflectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewWidth = msg.Width
		m.viewHeight = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case reflectTyping:
			return m.updateTyping(msg)
		case reflectShowAnswer:
			return m.updateShowAnswer(msg)
		case reflectSaved:
			return m.updateSaved(msg)
		}
	}

	return m, nil
}

func (m ReflectModel) updateTyping(msg tea.KeyMsg) (ReflectModel, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc":
		return m, tea.Quit

	case "enter":
		m.state = reflectShowAnswer
		return m, nil

	case "tab":
		m.showAnswer = !m.showAnswer
		if m.showAnswer {
			m.state = reflectShowAnswer
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m ReflectModel) updateShowAnswer(msg tea.KeyMsg) (ReflectModel, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.state = reflectTyping
		m.showAnswer = false
		return m, nil

	case "enter":
		userExplanation := strings.TrimSpace(m.input.Value())
		if userExplanation != "" {
			concept := m.concepts[m.currentIdx]
			m.prog.AddNote(
				"",
				userExplanation,
				&progress.EntityRef{Type: "concept", ID: concept.ID},
				"reflection",
			)

			cs := m.prog.Concepts[concept.ID]
			cs.Reflection.Count++
			now := time.Now().Unix()
			cs.Reflection.LastAt = &now
			m.prog.Concepts[concept.ID] = cs
		}

		m.input.Reset()
		m.showAnswer = false
		m.state = reflectSaved
		m.message = "Note saved."

		return m, tea.Tick(time.Millisecond*800, func(t time.Time) tea.Msg {
			return reflectNextMsg{}
		})

	case "n":
		m.input.Reset()
		m.showAnswer = false
		if m.currentIdx < len(m.concepts)-1 {
			m.currentIdx++
			m.state = reflectTyping
		} else {
			m.message = "All concepts reflected."
			m.state = reflectSaved
			return m, tea.Quit
		}
		return m, nil
	}

	return m, nil
}

func (m ReflectModel) updateSaved(msg tea.KeyMsg) (ReflectModel, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "enter", " ":
		if m.currentIdx < len(m.concepts)-1 {
			m.currentIdx++
			m.state = reflectTyping
			return m, nil
		}
		return m, tea.Quit
	}
	return m, nil
}

type reflectNextMsg struct{}

func (m ReflectModel) View() string {
	if m.currentIdx >= len(m.concepts) {
		return fmt.Sprintf("\n  %s\n\n  %s\n", finishedStyle.Render("All concepts reflected."), helpStyle.Render("Press q to quit."))
	}

	concept := m.concepts[m.currentIdx]
	b := strings.Builder{}

	title := titleStyle.Render(fmt.Sprintf("KEN REFLECT — %s", m.subject))
	b.WriteString(title)
	b.WriteString("\n\n")

	progress := lipgloss.NewStyle().Foreground(colorMuted).Render(
		fmt.Sprintf("Concept %d / %d", m.currentIdx+1, len(m.concepts)),
	)
	b.WriteString(progress)
	b.WriteString("\n\n")

	conceptName := subtitleStyle.Render(concept.Name)
	b.WriteString(conceptName)
	b.WriteString("\n\n")

	if concept.Summary != "" {
		rendered := render.RenderMarkdown(concept.Summary, m.viewWidth-4)
		b.WriteString(rendered)
		b.WriteString("\n")
	}

	switch m.state {
	case reflectTyping:
		if m.showAnswer && concept.Description != "" {
			b.WriteString(sectionStyle.Render("Canonical Answer:"))
			b.WriteString("\n")
			rendered := render.RenderMarkdown(concept.Description, m.viewWidth-4)
			b.WriteString(rendered)
			b.WriteString("\n")
		}

		b.WriteString(sectionStyle.Render("Your Explanation:"))
		b.WriteString("\n")
		b.WriteString(m.input.View())
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("enter: see answer / save  tab: toggle answer  esc: quit"))

	case reflectShowAnswer:
		b.WriteString(sectionStyle.Render("Canonical Answer:"))
		b.WriteString("\n")
		if concept.Description != "" {
			rendered := render.RenderMarkdown(concept.Description, m.viewWidth-4)
			b.WriteString(rendered)
		} else {
			b.WriteString("  " + lipgloss.NewStyle().Foreground(colorMuted).Render("(no description available)"))
		}
		b.WriteString("\n")

		if m.input.Value() != "" {
			b.WriteString(sectionStyle.Render("Your Explanation:"))
			b.WriteString("\n")
			rendered := lipgloss.NewStyle().PaddingLeft(2).Foreground(colorMuted).Render(m.input.Value())
			b.WriteString(rendered)
			b.WriteString("\n\n")
		}

		b.WriteString(helpStyle.Render("enter: save & next  n: skip  esc: back to typing"))

	case reflectSaved:
		b.WriteString(finishedStyle.Render(m.message))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("press any key to continue..."))
	}

	return b.String()
}
