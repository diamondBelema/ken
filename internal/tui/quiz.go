package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
}

func NewQuizModel(sess *study.QuizSession, prog *progress.Progress) QuizModel {
	ti := textinput.New()
	ti.Placeholder = "Type a note... (Enter to save, Esc to cancel)"
	ti.Focus()
	ti.CharLimit = 1000

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
	var b strings.Builder

	switch m.state {
	case quizAnswering:
		q := m.session.Current()
		cur, total := m.session.Progress()

		b.WriteString(titleStyle.Render(fmt.Sprintf("Quiz — %s", m.session.Subject)))
		b.WriteString("\n")
		b.WriteString(subtitleStyle.Render(fmt.Sprintf("Question %d of %d", cur, total)))
		b.WriteString("\n")
		b.WriteString(cardStyle.Render(q.Question))
		b.WriteString("\n")

		switch q.Type {
		case "mcq":
			for i, opt := range q.Options {
				b.WriteString(fmt.Sprintf("  %d: %s\n", i+1, opt))
			}
		case "true_false":
			b.WriteString("  t: true\n  f: false\n")
		case "fill_blank":
			b.WriteString(fmt.Sprintf("  Your answer: %s_\n", m.message))
			b.WriteString("  (press enter to submit)\n")
		}

		b.WriteString("\n")
		b.WriteString(helpStyle.Render("q to quit"))

	case quizFeedback:
		q := m.session.Current()
		if m.correct {
			b.WriteString(finishedStyle.Render("Correct!"))
		} else {
			b.WriteString(gradeUnknownStyle.Render("Incorrect"))
		}
		if q.Explanation != "" {
			b.WriteString("\n")
			b.WriteString(notesStyle.Render(q.Explanation))
		}
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("n to add note • enter to continue • q to quit"))

	case quizNoteInput:
		q := m.session.Current()
		cur, total := m.session.Progress()

		b.WriteString(titleStyle.Render(fmt.Sprintf("Quiz — %s", m.session.Subject)))
		b.WriteString("\n")
		b.WriteString(subtitleStyle.Render(fmt.Sprintf("Question %d of %d", cur, total)))
		b.WriteString("\n")
		b.WriteString(cardStyle.Render(q.Question))
		b.WriteString("\n")

		linkLabel := "unlinked"
		if m.noteLinkedTo != nil {
			switch m.noteLinkedTo.Type {
			case "concept":
				linkLabel = fmt.Sprintf("concept: %s", m.noteLinkedTo.ID)
			case "quiz":
				linkLabel = fmt.Sprintf("quiz: %s", m.noteLinkedTo.ID)
			}
		}
		b.WriteString(noteInputHeaderStyle.Render(fmt.Sprintf("New Note → %s", linkLabel)))
		b.WriteString("\n")
		b.WriteString(m.noteInput.View())
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("tab to cycle link • enter to save • esc to cancel"))

	case quizFinished:
		total := len(m.session.Questions)
		b.WriteString(titleStyle.Render("Quiz Complete!"))
		b.WriteString("\n")
		pct := float64(m.session.Score) / float64(total) * 100
		b.WriteString(subtitleStyle.Render(fmt.Sprintf("Score: %d/%d (%.0f%%)", m.session.Score, total, pct)))
		b.WriteString("\n")
		b.WriteString(finishedStyle.Render("Progress saved. Press enter to exit."))
	}

	return b.String()
}
