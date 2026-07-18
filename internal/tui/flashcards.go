package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/diamondBelema/ken/internal/mastery"
	"github.com/diamondBelema/ken/internal/progress"
	"github.com/diamondBelema/ken/internal/study"
)

type flashcardState int

const (
	fcShowingFront flashcardState = iota
	fcShowingBack
	fcNoteInput
	fcFinished
)

type FlashcardModel struct {
	session       *study.FlashcardSession
	progress      *progress.Progress
	state         flashcardState
	score         int
	total         int
	err           error
	noteInput     textinput.Model
	noteLinkedTo  *progress.EntityRef
	noteCycleIdx  int
}

type flashcardQuitMsg struct{}

func NewFlashcardModel(sess *study.FlashcardSession, prog *progress.Progress) FlashcardModel {
	ti := textinput.New()
	ti.Placeholder = "Type a note... (Enter to save, Esc to cancel)"
	ti.Focus()
	ti.CharLimit = 1000

	return FlashcardModel{
		session:  sess,
		progress: prog,
		state:    fcShowingFront,
		total:    len(sess.Cards),
		noteInput: ti,
	}
}

func (m FlashcardModel) Init() tea.Cmd {
	return nil
}

func (m FlashcardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.state == fcNoteInput {
		return m.updateNoteInput(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case fcShowingFront:
			switch msg.String() {
			case " ", "enter":
				m.state = fcShowingBack
				return m, nil
			case "n":
				return m.startNoteInput(), nil
			case "q", "esc", "ctrl+c":
				return m, tea.Quit
			}

		case fcShowingBack:
			switch msg.String() {
			case "1":
				m = m.gradeCard(mastery.Unknown)
				return m, nil
			case "2":
				m = m.gradeCard(mastery.KnownLittle)
				return m, nil
			case "3":
				m = m.gradeCard(mastery.KnownFairly)
				return m, nil
			case "4":
				m = m.gradeCard(mastery.KnownWell)
				return m, nil
			case "5":
				m = m.gradeCard(mastery.Mastered)
				return m, nil
			case "n":
				return m.startNoteInput(), nil
			case "q", "esc", "ctrl+c":
				return m, tea.Quit
			}

		case fcFinished:
			switch msg.String() {
			case "q", "esc", "ctrl+c", "enter":
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m FlashcardModel) startNoteInput() FlashcardModel {
	m.state = fcNoteInput
	m.noteInput.SetValue("")
	m.noteInput.Focus()
	m.noteCycleIdx = 0
	m.noteLinkedTo = m.getCurrentCardLink()
	return m
}

func (m FlashcardModel) getCurrentCardLink() *progress.EntityRef {
	card := m.session.Current()
	if card.ConceptID != "" {
		return &progress.EntityRef{Type: "concept", ID: card.ConceptID}
	}
	return &progress.EntityRef{Type: "card", ID: card.ID}
}

func (m FlashcardModel) cycleLinkTarget() {
	card := m.session.Current()
	targets := []*progress.EntityRef{}

	if card.ConceptID != "" {
		targets = append(targets, &progress.EntityRef{Type: "concept", ID: card.ConceptID})
	}
	targets = append(targets, &progress.EntityRef{Type: "card", ID: card.ID})
	targets = append(targets, nil)

	m.noteCycleIdx = (m.noteCycleIdx + 1) % len(targets)
	m.noteLinkedTo = targets[m.noteCycleIdx]
}

func (m FlashcardModel) updateNoteInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			content := m.noteInput.Value()
			if strings.TrimSpace(content) != "" {
				m.progress.AddNote(content, m.noteLinkedTo)
			}
			m.state = fcShowingBack
			m.noteInput.SetValue("")
			return m, nil
		case "esc":
			m.state = fcShowingBack
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

func (m *FlashcardModel) gradeCard(level mastery.ConfidenceLevel) FlashcardModel {
	card := m.session.Current()
	now := unixNow()

	if card.ConceptID != "" {
		cs, exists := m.progress.Concepts[card.ConceptID]
		if !exists {
			cs = progress.ConceptState{Confidence: 0.5}
		}
		masteryState := mastery.ConceptState{
			Confidence:     cs.Confidence,
			LastReviewedAt: cs.LastReviewedAt,
		}
		updated := mastery.UpdateFromFlashcard(masteryState, level, now)
		m.progress.Concepts[card.ConceptID] = progress.ConceptState{
			Confidence:     updated.Confidence,
			LastReviewedAt: updated.LastReviewedAt,
		}
	}

	cs, exists := m.progress.Cards[card.ID]
	if !exists {
		cs = progress.CardState{}
	}
	cs.Reviews++
	cs.LastGrade = confidenceLevelString(level)
	m.progress.Cards[card.ID] = cs

	if level >= mastery.KnownFairly {
		m.score++
	}

	if m.session.Advance() {
		m.state = fcShowingFront
	} else {
		m.state = fcFinished
	}

	return *m
}

func (m FlashcardModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var b strings.Builder

	switch m.state {
	case fcShowingFront:
		card := m.session.Current()
		cur, total := m.session.Progress()

		b.WriteString(titleStyle.Render(fmt.Sprintf("Flashcards — %s", m.session.Subject)))
		b.WriteString("\n")
		b.WriteString(subtitleStyle.Render(fmt.Sprintf("Card %d of %d", cur, total)))
		b.WriteString("\n")
		b.WriteString(cardStyle.Render(frontStyle.Render(card.Front)))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("space/enter to flip • n to add note • q to quit"))

	case fcShowingBack:
		card := m.session.Current()
		cur, total := m.session.Progress()

		b.WriteString(titleStyle.Render(fmt.Sprintf("Flashcards — %s", m.session.Subject)))
		b.WriteString("\n")
		b.WriteString(subtitleStyle.Render(fmt.Sprintf("Card %d of %d", cur, total)))
		b.WriteString("\n")
		b.WriteString(cardStyle.Render(
			frontStyle.Render(card.Front) + "\n\n" + backStyle.Render(card.Back),
		))
		if card.Notes != "" {
			b.WriteString("\n")
			b.WriteString(notesStyle.Render(card.Notes))
		}
		b.WriteString("\n\n")
		b.WriteString(gradeUnknownStyle.Render("1:Unknown"))
		b.WriteString(gradeKnownLittleStyle.Render("2:KnownLittle"))
		b.WriteString(gradeKnownFairlyStyle.Render("3:KnownFairly"))
		b.WriteString(gradeKnownWellStyle.Render("4:KnownWell"))
		b.WriteString(gradeMasteredStyle.Render("5:Mastered"))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("n to add note • q to quit"))

	case fcNoteInput:
		card := m.session.Current()
		cur, total := m.session.Progress()

		b.WriteString(titleStyle.Render(fmt.Sprintf("Flashcards — %s", m.session.Subject)))
		b.WriteString("\n")
		b.WriteString(subtitleStyle.Render(fmt.Sprintf("Card %d of %d", cur, total)))
		b.WriteString("\n")
		b.WriteString(cardStyle.Render(
			frontStyle.Render(card.Front) + "\n\n" + backStyle.Render(card.Back),
		))
		b.WriteString("\n")

		linkLabel := "unlinked"
		if m.noteLinkedTo != nil {
			switch m.noteLinkedTo.Type {
			case "concept":
				linkLabel = fmt.Sprintf("concept: %s", m.noteLinkedTo.ID)
			case "card":
				linkLabel = fmt.Sprintf("card: %s", m.noteLinkedTo.ID)
			case "quiz":
				linkLabel = fmt.Sprintf("quiz: %s", m.noteLinkedTo.ID)
			case "note":
				linkLabel = fmt.Sprintf("note: %s", m.noteLinkedTo.ID)
			}
		}
		b.WriteString(noteInputHeaderStyle.Render(fmt.Sprintf("New Note → %s", linkLabel)))
		b.WriteString("\n")
		b.WriteString(m.noteInput.View())
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("tab to cycle link • enter to save • esc to cancel"))

	case fcFinished:
		b.WriteString(titleStyle.Render("Study Complete!"))
		b.WriteString("\n")
		b.WriteString(subtitleStyle.Render(fmt.Sprintf("Score: %d/%d (%.0f%%)", m.score, m.total, float64(m.score)/float64(m.total)*100)))
		b.WriteString("\n")
		b.WriteString(finishedStyle.Render("Progress saved. Press enter to exit."))
	}

	return b.String()
}

func confidenceLevelString(l mastery.ConfidenceLevel) string {
	switch l {
	case mastery.Unknown:
		return "unknown"
	case mastery.KnownLittle:
		return "known_little"
	case mastery.KnownFairly:
		return "known_fairly"
	case mastery.KnownWell:
		return "known_well"
	case mastery.Mastered:
		return "mastered"
	default:
		return "unknown"
	}
}

func unixNow() int64 {
	return time.Now().Unix()
}
