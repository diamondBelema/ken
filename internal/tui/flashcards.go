package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/diamondBelema/ken/internal/mastery"
	"github.com/diamondBelema/ken/internal/progress"
	"github.com/diamondBelema/ken/internal/study"
)

type flashcardState int

const (
	fcShowingFront flashcardState = iota
	fcShowingBack
	fcFinished
)

type FlashcardModel struct {
	session  *study.FlashcardSession
	progress *progress.Progress
	state    flashcardState
	score    int
	total    int
	err      error
}

type flashcardQuitMsg struct{}

func NewFlashcardModel(sess *study.FlashcardSession, prog *progress.Progress) FlashcardModel {
	return FlashcardModel{
		session:  sess,
		progress: prog,
		state:    fcShowingFront,
		total:    len(sess.Cards),
	}
}

func (m FlashcardModel) Init() tea.Cmd {
	return nil
}

func (m FlashcardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case fcShowingFront:
			switch msg.String() {
			case " ", "enter":
				m.state = fcShowingBack
				return m, nil
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

func (m *FlashcardModel) gradeCard(level mastery.ConfidenceLevel) FlashcardModel {
	card := m.session.Current()
	now := unixNow()

	// Update concept confidence if card has a concept_id
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

	// Always update card history
	cs, exists := m.progress.Cards[card.ID]
	if !exists {
		cs = progress.CardState{}
	}
	cs.Reviews++
	cs.LastGrade = confidenceLevelString(level)
	m.progress.Cards[card.ID] = cs

	// Track score
	if level >= mastery.KnownFairly {
		m.score++
	}

	// Advance
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
		b.WriteString(helpStyle.Render("space/enter to flip • q to quit"))

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
		b.WriteString(helpStyle.Render("q to quit"))

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
