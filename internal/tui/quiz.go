package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/diamondBelema/ken/internal/mastery"
	"github.com/diamondBelema/ken/internal/progress"
	"github.com/diamondBelema/ken/internal/study"
)

type quizState int

const (
	quizAnswering quizState = iota
	quizFeedback
	quizNoteInput
	quizFinished
)

type QuizModel struct {
	session      *study.QuizSession
	progress     *progress.Progress
	state        quizState
	selected     int
	correct      bool
	message      string
	noteInput    textinput.Model
	noteLinkedTo *progress.EntityRef
	noteCycleIdx int
	width        int
	height       int
}

func NewQuizModel(sess *study.QuizSession, prog *progress.Progress) QuizModel {
	ti := textinput.New()
	ti.Placeholder = "Type a note..."
	ti.Focus()
	ti.CharLimit = 1000
	ti.Width = 60

	return QuizModel{
		session:   sess,
		progress:  prog,
		state:     quizAnswering,
		noteInput: ti,
	}
}

func (m QuizModel) Init() tea.Cmd {
	return nil
}

func (m QuizModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.state == quizNoteInput {
		return m.updateNoteInput(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch m.state {
		case quizAnswering:
			q := m.session.Current()
			switch q.Type {
			case "mcq":
				if idx, err := strconv.Atoi(msg.String()); err == nil && idx >= 1 && idx <= len(q.Options) {
					m.selected = idx - 1
					m = m.checkAnswer()
					return m, nil
				}
			case "true_false":
				switch msg.String() {
				case "t", "T":
					m.selected = 1
					m = m.checkAnswer()
					return m, nil
				case "f", "F":
					m.selected = 0
					m = m.checkAnswer()
					return m, nil
				}
			case "fill_blank":
				if msg.String() == "enter" && m.message != "" {
					m = m.checkFillBlank()
					return m, nil
				}
				if len(msg.String()) == 1 {
					m.message += msg.String()
					return m, nil
				}
			}
			if msg.String() == "q" || msg.String() == "esc" || msg.String() == "ctrl+c" {
				return m, tea.Quit
			}

		case quizFeedback:
			switch msg.String() {
			case "enter", "space":
				if m.session.Advance() {
					m.state = quizAnswering
					m.message = ""
				} else {
					m.state = quizFinished
				}
				return m, nil
			case "n":
				return m.startNoteInput(), nil
			case "q", "esc", "ctrl+c":
				return m, tea.Quit
			}

		case quizFinished:
			switch msg.String() {
			case "q", "esc", "ctrl+c", "enter":
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m QuizModel) startNoteInput() QuizModel {
	m.state = quizNoteInput
	m.noteInput.SetValue("")
	m.noteInput.Focus()
	m.noteCycleIdx = 0
	m.noteLinkedTo = m.getCurrentQuestionLink()
	return m
}

func (m QuizModel) getCurrentQuestionLink() *progress.EntityRef {
	q := m.session.Current()
	if q.ConceptID != "" {
		return &progress.EntityRef{Type: "concept", ID: q.ConceptID}
	}
	return &progress.EntityRef{Type: "quiz", ID: q.ID}
}

func (m QuizModel) cycleLinkTarget() {
	q := m.session.Current()
	targets := []*progress.EntityRef{}

	if q.ConceptID != "" {
		targets = append(targets, &progress.EntityRef{Type: "concept", ID: q.ConceptID})
	}
	targets = append(targets, &progress.EntityRef{Type: "quiz", ID: q.ID})
	targets = append(targets, nil)

	m.noteCycleIdx = (m.noteCycleIdx + 1) % len(targets)
	m.noteLinkedTo = targets[m.noteCycleIdx]
}

func (m QuizModel) updateNoteInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			content := m.noteInput.Value()
			if strings.TrimSpace(content) != "" {
				m.progress.AddNote(content, m.noteLinkedTo)
			}
			m.state = quizFeedback
			m.noteInput.SetValue("")
			return m, nil
		case "esc":
			m.state = quizFeedback
			m.noteInput.SetValue("")
			return m, nil
		case "tab":
			m.cycleLinkTarget()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.noteInput, cmd = m.noteInput.Update(msg)
	return m, cmd
}

func (m QuizModel) checkAnswer() QuizModel {
	q := m.session.Current()
	var userAnswer interface{}

	switch q.Type {
	case "mcq":
		userAnswer = m.selected
	case "true_false":
		userAnswer = m.selected == 1
	}

	m.correct = compareAnswer(userAnswer, q.Answer)
	m.recordResult()
	m.state = quizFeedback
	return m
}

func (m QuizModel) checkFillBlank() QuizModel {
	q := m.session.Current()
	m.correct = compareAnswer(strings.TrimSpace(m.message), q.Answer)
	m.recordResult()
	m.state = quizFeedback
	return m
}

func (m *QuizModel) recordResult() {
	q := m.session.Current()
	m.session.RecordAnswer(m.correct)

	qs, exists := m.progress.Quizzes[q.ID]
	if !exists {
		qs = progress.QuizState{}
	}
	qs.Attempts++
	if m.correct {
		qs.Correct++
		qs.Streak++
	} else {
		qs.Streak = 0
	}
	m.progress.Quizzes[q.ID] = qs

	if q.ConceptID != "" {
		cs, exists := m.progress.Concepts[q.ConceptID]
		if !exists {
			cs = progress.ConceptState{Confidence: 0.5}
		}
		masteryState := mastery.ConceptState{
			Confidence:     cs.Confidence,
			LastReviewedAt: cs.LastReviewedAt,
		}
		updated := mastery.UpdateFromQuiz(masteryState, m.correct, unixNow())
		m.progress.Concepts[q.ConceptID] = progress.ConceptState{
			Confidence:     updated.Confidence,
			LastReviewedAt: updated.LastReviewedAt,
		}
	}
}

func compareAnswer(user, spec interface{}) bool {
	switch u := user.(type) {
	case int:
		if s, ok := spec.(int); ok {
			return u == s
		}
	case bool:
		if s, ok := spec.(bool); ok {
			return u == s
		}
	case string:
		if s, ok := spec.(string); ok {
			return strings.EqualFold(u, s)
		}
	}
	return false
}

func (m QuizModel) View() string {
	if m.width == 0 {
		m.width = 80
	}

	var b strings.Builder

	header := titleStyle.Render(fmt.Sprintf("  quiz · %s  ", m.session.Subject))
	b.WriteString(header)
	b.WriteString("\n\n")

	switch m.state {
	case quizAnswering:
		q := m.session.Current()
		cur, total := m.session.Progress()

		progressBar := m.renderProgressBar(cur, total)
		b.WriteString("  ")
		b.WriteString(progressBar)
		b.WriteString("\n\n")

		questionBox := lipgloss.NewStyle().
			Width(m.width - 8).
			Render(frontStyle.Render(q.Question))

		b.WriteString(cardStyle.Render(questionBox))
		b.WriteString("\n")

		switch q.Type {
		case "mcq":
			for i, opt := range q.Options {
				num := lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render(fmt.Sprintf("%d", i+1))
				b.WriteString(fmt.Sprintf("  %s  %s\n", num, opt))
			}
		case "true_false":
			b.WriteString("  t  true\n")
			b.WriteString("  f  false\n")
		case "fill_blank":
			input := lipgloss.NewStyle().
				Foreground(colorTextBright).
				Render(m.message + "█")
			b.WriteString(fmt.Sprintf("  Your answer: %s\n", input))
			b.WriteString(helpStyle.Render("  enter submit"))
		}

		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  q quit"))

	case quizFeedback:
		q := m.session.Current()

		if m.correct {
			b.WriteString("  ")
			b.WriteString(finishedStyle.Render("Correct"))
		} else {
			b.WriteString("  ")
			b.WriteString(lipgloss.NewStyle().Foreground(colorDanger).Bold(true).Render("Incorrect"))
		}
		b.WriteString("\n")

		if q.Explanation != "" {
			explainBox := lipgloss.NewStyle().
				Width(m.width - 8).
				Render(notesStyle.Render(q.Explanation))
			b.WriteString("\n")
			b.WriteString(cardStyle.Render(explainBox))
		}

		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("  n note  ·  enter continue  ·  q quit"))

	case quizNoteInput:
		q := m.session.Current()

		linkLabel := "unlinked"
		if m.noteLinkedTo != nil {
			switch m.noteLinkedTo.Type {
			case "concept":
				linkLabel = fmt.Sprintf("→ %s", m.noteLinkedTo.ID)
			case "quiz":
				linkLabel = fmt.Sprintf("→ %s", m.noteLinkedTo.ID)
			}
		}

		questionBox := lipgloss.NewStyle().
			Width(m.width - 8).
			Render(frontStyle.Render(q.Question))

		b.WriteString(cardStyle.Render(questionBox))
		b.WriteString("\n")
		b.WriteString(noteInputHeaderStyle.Render(fmt.Sprintf("  note %s", linkLabel)))
		b.WriteString("\n  ")
		b.WriteString(m.noteInput.View())
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  tab cycle  ·  enter save  ·  esc cancel"))

	case quizFinished:
		total := len(m.session.Questions)
		pct := 0.0
		if total > 0 {
			pct = float64(m.session.Score) / float64(total) * 100
		}

		summary := lipgloss.NewStyle().
			Width(m.width - 8).
			Align(lipgloss.Center).
			Render(
				finishedStyle.Render("Quiz Complete"),
				"\n\n",
				lipgloss.NewStyle().Foreground(colorTextBright).Render(fmt.Sprintf("Score: %d / %d", m.session.Score, total)),
				"\n",
				lipgloss.NewStyle().Foreground(colorSuccess).Render(fmt.Sprintf("%.0f%%", pct)),
			)

		b.WriteString(cardStyle.Render(summary))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("  enter/quit"))
	}

	return b.String()
}

func (m QuizModel) renderProgressBar(current, total int) string {
	barWidth := 20
	filled := 0
	if total > 0 {
		filled = (current * barWidth) / total
	}

	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "━"
		} else {
			bar += "─"
		}
	}

	filledStyle := lipgloss.NewStyle().Foreground(colorPrimary)
	emptyStyle := lipgloss.NewStyle().Foreground(colorMuted)

	result := filledStyle.Render(bar[:filled]) + emptyStyle.Render(bar[filled:])
	result += fmt.Sprintf(" %d/%d", current, total)

	return result
}
