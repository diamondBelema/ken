package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	width         int
	height        int
}

type flashcardQuitMsg struct{}

func NewFlashcardModel(sess *study.FlashcardSession, prog *progress.Progress) FlashcardModel {
	ti := textinput.New()
	ti.Placeholder = "Type a note..."
	ti.Focus()
	ti.CharLimit = 1000
	ti.Width = 60

	return FlashcardModel{
		session:   sess,
		progress:  prog,
		state:     fcShowingFront,
		total:     len(sess.Cards),
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
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
		return fmt.Sprintf("\n  Error: %v\n\n  Press q to exit.\n", m.err)
	}

	if m.width == 0 {
		m.width = 80
	}

	var b strings.Builder

	header := titleStyle.Render(fmt.Sprintf("  flashcards · %s  ", m.session.Subject))
	b.WriteString(header)
	b.WriteString("\n\n")

	switch m.state {
	case fcShowingFront:
		card := m.session.Current()
		cur, total := m.session.Progress()

		progressBar := m.renderProgressBar(cur, total)
		b.WriteString("  ")
		b.WriteString(progressBar)
		b.WriteString("\n\n")

		cardContent := lipgloss.NewStyle().
			Width(m.width - 8).
			Render(frontStyle.Render(card.Front))

		b.WriteString(cardStyle.Render(cardContent))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  space/enter flip  ·  n note  ·  q quit"))

	case fcShowingBack:
		card := m.session.Current()
		cur, total := m.session.Progress()

		progressBar := m.renderProgressBar(cur, total)
		b.WriteString("  ")
		b.WriteString(progressBar)
		b.WriteString("\n\n")

		cardContent := lipgloss.NewStyle().
			Width(m.width - 8).
			Render(frontStyle.Render(card.Front) + "\n\n" + backStyle.Render(card.Back))

		b.WriteString(cardStyle.Render(cardContent))
		if card.Notes != "" {
			b.WriteString("\n  ")
			b.WriteString(notesStyle.Render(card.Notes))
		}
		b.WriteString("\n\n")

		grades := lipgloss.JoinHorizontal(lipgloss.Center,
			gradeUnknownStyle.Render("1:Unknown"),
			gradeKnownLittleStyle.Render("2:KnownLittle"),
			gradeKnownFairlyStyle.Render("3:KnownFairly"),
			gradeKnownWellStyle.Render("4:KnownWell"),
			gradeMasteredStyle.Render("5:Mastered"),
		)
		b.WriteString("  ")
		b.WriteString(grades)
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  n note  ·  q quit"))

	case fcNoteInput:
		card := m.session.Current()

		linkLabel := "unlinked"
		if m.noteLinkedTo != nil {
			switch m.noteLinkedTo.Type {
			case "concept":
				linkLabel = fmt.Sprintf("→ %s", m.noteLinkedTo.ID)
			case "card":
				linkLabel = fmt.Sprintf("→ %s", m.noteLinkedTo.ID)
			}
		}

		cardContent := lipgloss.NewStyle().
			Width(m.width - 8).
			Render(frontStyle.Render(card.Front) + "\n\n" + backStyle.Render(card.Back))

		b.WriteString(cardStyle.Render(cardContent))
		b.WriteString("\n")
		b.WriteString(noteInputHeaderStyle.Render(fmt.Sprintf("  note %s", linkLabel)))
		b.WriteString("\n  ")
		b.WriteString(m.noteInput.View())
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  tab cycle  ·  enter save  ·  esc cancel"))

	case fcFinished:
		pct := 0.0
		if m.total > 0 {
			pct = float64(m.score) / float64(m.total) * 100
		}

		summary := lipgloss.NewStyle().
			Width(m.width - 8).
			Align(lipgloss.Center).
			Render(
				finishedStyle.Render("Session Complete"),
				"\n\n",
				lipgloss.NewStyle().Foreground(colorTextBright).Render(fmt.Sprintf("Score: %d / %d", m.score, m.total)),
				"\n",
				lipgloss.NewStyle().Foreground(colorSuccess).Render(fmt.Sprintf("%.0f%%", pct)),
			)

		b.WriteString(cardStyle.Render(summary))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("  enter/quit"))
	}

	return b.String()
}

func (m FlashcardModel) renderProgressBar(current, total int) string {
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
