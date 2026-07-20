package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/diamondBelema/ken/internal/diagram"
	"github.com/diamondBelema/ken/internal/mastery"
	"github.com/diamondBelema/ken/internal/parser"
	"github.com/diamondBelema/ken/internal/progress"
	"github.com/diamondBelema/ken/internal/study"
	"github.com/diamondBelema/ken/internal/system"
)

type flashcardState int

const (
	fcShowingFront flashcardState = iota
	fcShowingBack
	fcNoteInput
	fcFinished
	fcSummaryView
)

// Message types for diagram/link operations
type diagramViewMsg struct {
	source string
	label  string
}

type diagramOpenMsg struct {
	file string
}

type diagramRenderMsg struct {
	source string
	id     string
}

type linkOpenMsg struct {
	url string
}

type FlashcardModel struct {
	session            *study.FlashcardSession
	progress           *progress.Progress
	concepts           []parser.Concept
	conceptMap         map[string]parser.Concept
	state              flashcardState
	prevState          flashcardState
	score              int
	total              int
	noteInput          textinput.Model
	noteLinkedTo       *progress.EntityRef
	noteCycleIdx       int
	width              int
	height             int
	showConceptDetail  bool
	conceptDetailScroll int
	summaryContent     string
	summaryScroll      int
}

func NewFlashcardModel(sess *study.FlashcardSession, prog *progress.Progress, concepts []parser.Concept) FlashcardModel {
	ti := textinput.New()
	ti.Placeholder = "Type a note..."
	ti.Focus()
	ti.CharLimit = 1000
	ti.Width = 60

	return FlashcardModel{
		session:   sess,
		progress:  prog,
		concepts:  concepts,
		conceptMap: buildConceptMap(concepts),
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

	if m.state == fcSummaryView {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "q", "esc", "s":
				m.state = m.prevState
				return m, nil
			case "j", "down":
				m.summaryScroll++
				m.clampSummaryScroll()
			case "k", "up":
				if m.summaryScroll > 0 {
					m.summaryScroll--
				}
			case "g":
				m.summaryScroll = 0
			case "G":
				m.summaryScroll = 999999
				m.clampSummaryScroll()
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch m.state {
		case fcShowingFront:
			if m.showConceptDetail {
				switch msg.String() {
				case "c":
					m.showConceptDetail = false
					m.conceptDetailScroll = 0
					return m, nil
				case "j", "down":
					m.conceptDetailScroll++
					return m, nil
				case "k", "up":
					if m.conceptDetailScroll > 0 {
						m.conceptDetailScroll--
					}
					return m, nil
				case "g":
					m.conceptDetailScroll = 0
					return m, nil
				case "G":
					m.conceptDetailScroll = 999999
					return m, nil
				}
			}
			switch msg.String() {
			case " ", "enter":
				m.state = fcShowingBack
				return m, nil
			case "c":
				m.showConceptDetail = true
				m.conceptDetailScroll = 0
				return m, nil
			case "n":
				return m.startNoteInput(), nil
			case "s":
				// Show full summary view
				card := m.session.Current()
				if card.ConceptID != "" {
					if concept, ok := m.conceptMap[card.ConceptID]; ok {
						content := renderFullSummary(&concept, m.progress, card.ConceptID, m.width)
						if content != "" {
							m.prevState = m.state
							m.summaryContent = content
							m.summaryScroll = 0
							m.state = fcSummaryView
						}
					}
				}
			case "q", "esc", "ctrl+c":
				return m, tea.Quit
			}

		case fcShowingBack:
			if m.showConceptDetail {
				switch msg.String() {
				case "c":
					m.showConceptDetail = false
					m.conceptDetailScroll = 0
					return m, nil
				case "j", "down":
					m.conceptDetailScroll++
					return m, nil
				case "k", "up":
					if m.conceptDetailScroll > 0 {
						m.conceptDetailScroll--
					}
					return m, nil
				case "g":
					m.conceptDetailScroll = 0
					return m, nil
				case "G":
					m.conceptDetailScroll = 999999
					return m, nil
				}
			}
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
			case "c":
				m.showConceptDetail = true
				m.conceptDetailScroll = 0
				return m, nil
			case "n":
				return m.startNoteInput(), nil
			case "v":
				// ASCII diagram for current concept
				card := m.session.Current()
				if card.ConceptID != "" {
					if concept, ok := m.conceptMap[card.ConceptID]; ok && len(concept.Diagrams) > 0 {
						diag := concept.Diagrams[0]
						source := diag.Source
						if source != "" {
							return m, tea.Cmd(func() tea.Msg {
								return diagramViewMsg{source: source, label: diag.Label}
							})
						}
					}
				}
			case "d":
				// Open SVG diagram for current concept
				card := m.session.Current()
				if card.ConceptID != "" {
					if concept, ok := m.conceptMap[card.ConceptID]; ok && len(concept.Diagrams) > 0 {
						diag := concept.Diagrams[0]
						if diag.File != "" {
							return m, tea.Cmd(func() tea.Msg {
								return diagramOpenMsg{file: diag.File}
							})
						} else if diag.Source != "" {
							return m, tea.Cmd(func() tea.Msg {
								return diagramRenderMsg{source: diag.Source, id: diag.ID}
							})
						}
					}
				}
			case "l":
				// Open first link for current concept
				card := m.session.Current()
				if card.ConceptID != "" {
					if concept, ok := m.conceptMap[card.ConceptID]; ok && len(concept.Links) > 0 {
						return m, tea.Cmd(func() tea.Msg {
							return linkOpenMsg{url: concept.Links[0].URL}
						})
					}
				}
			case "s":
				// Show full summary view
				card := m.session.Current()
				if card.ConceptID != "" {
					if concept, ok := m.conceptMap[card.ConceptID]; ok {
						content := renderFullSummary(&concept, m.progress, card.ConceptID, m.width)
						if content != "" {
							m.prevState = m.state
							m.summaryContent = content
							m.summaryScroll = 0
							m.state = fcSummaryView
						}
					}
				}
			case "q", "esc", "ctrl+c":
				return m, tea.Quit
			}

		case fcFinished:
			switch msg.String() {
			case "q", "esc", "ctrl+c", "enter":
				return m, tea.Quit
			}
		}

	case diagramViewMsg:
		// Show ASCII diagram in a popup or inline
		_, err := diagram.RenderASCII(msg.source)
		if err != nil {
			// Just show error, concept detail will be updated
		}
		// For now, just show in concept detail view
		m.showConceptDetail = true
		m.conceptDetailScroll = 0
		return m, nil

	case diagramOpenMsg:
		// Open external SVG file
		card := m.session.Current()
		if card.ConceptID != "" {
			home, err := os.UserHomeDir()
			if err == nil {
				subjDir := filepath.Join(home, "Documents", "learn", "subjects", m.session.Subject)
				svgPath := filepath.Join(subjDir, msg.file)
				system.OpenFile(svgPath)
			}
		}

	case diagramRenderMsg:
		// Render mermaid to SVG and open
		tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("ken-diagram-%s.svg", msg.id))
		if err := diagram.RenderSVGToFile(msg.source, tmpPath); err == nil {
			system.OpenFile(tmpPath)
		}

	case linkOpenMsg:
		// Open link in browser
		system.OpenURL(msg.url)
	}
	return m, nil
}

func (m FlashcardModel) startNoteInput() FlashcardModel {
	m.prevState = m.state
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

func (m *FlashcardModel) cycleLinkTarget() {
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

func (m *FlashcardModel) clampSummaryScroll() {
	lines := strings.Split(m.summaryContent, "\n")
	visible := m.height - 4
	if visible < 1 {
		visible = 10
	}
	maxScroll := len(lines) - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.summaryScroll > maxScroll {
		m.summaryScroll = maxScroll
	}
}

func (m FlashcardModel) updateNoteInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			content := m.noteInput.Value()
			if strings.TrimSpace(content) != "" {
				m.progress.AddNote("", content, m.noteLinkedTo)
			}
			m.state = m.prevState
			m.noteInput.SetValue("")
			return m, nil
		case "esc":
			m.state = m.prevState
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
		m.showConceptDetail = false
		m.conceptDetailScroll = 0
	} else {
		m.state = fcFinished
	}

	return *m
}

func (m FlashcardModel) View() string {
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

		progressBar := renderProgressBar(cur, total)
		b.WriteString("  ")
		b.WriteString(progressBar)
		b.WriteString("\n\n")

		if m.showConceptDetail {
			concept := lookupConcept(m.conceptMap, card.ConceptID)
			lines := renderConceptDetail(concept, m.progress, card.ConceptID, m.width, m.conceptMap)
			headerH := 3
			footerH := 1
			visible := m.height - headerH - footerH
			if visible < 1 {
				visible = 10
			}
			maxScroll := len(lines) - visible
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.conceptDetailScroll > maxScroll {
				m.conceptDetailScroll = maxScroll
			}
			start := m.conceptDetailScroll
			end := start + visible
			if end > len(lines) {
				end = len(lines)
			}
			for _, line := range lines[start:end] {
				b.WriteString(line)
				b.WriteString("\n")
			}
			b.WriteString(helpStyle.Render("  j/k scroll  ·  g/G top/bottom  ·  c close  ·  q quit"))
		} else {
			cardContent := lipgloss.NewStyle().
				Width(max(m.width-8, 20)).
				Render(frontStyle.Render(card.Front))

			b.WriteString(cardStyle.Render(cardContent))
			b.WriteString(renderUserNotes(m.progress, card.ConceptID, card.ID, "card", m.width))
			b.WriteString("\n")
			b.WriteString(helpStyle.Render("  space/enter flip  ·  c concept  ·  s summary  ·  n note  ·  q quit"))
		}

	case fcShowingBack:
		card := m.session.Current()
		cur, total := m.session.Progress()

		progressBar := renderProgressBar(cur, total)
		b.WriteString("  ")
		b.WriteString(progressBar)
		b.WriteString("\n\n")

		if m.showConceptDetail {
			concept := lookupConcept(m.conceptMap, card.ConceptID)
			lines := renderConceptDetail(concept, m.progress, card.ConceptID, m.width, m.conceptMap)
			headerH := 3
			footerH := 1
			visible := m.height - headerH - footerH
			if visible < 1 {
				visible = 10
			}
			maxScroll := len(lines) - visible
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.conceptDetailScroll > maxScroll {
				m.conceptDetailScroll = maxScroll
			}
			start := m.conceptDetailScroll
			end := start + visible
			if end > len(lines) {
				end = len(lines)
			}
			for _, line := range lines[start:end] {
				b.WriteString(line)
				b.WriteString("\n")
			}
			b.WriteString(helpStyle.Render("  j/k scroll  ·  g/G top/bottom  ·  c close  ·  q quit"))
		} else {
			cardContent := lipgloss.NewStyle().
				Width(max(m.width-8, 20)).
				Render(frontStyle.Render(card.Front) + "\n\n" + backStyle.Render(card.Back))

			b.WriteString(cardStyle.Render(cardContent))
			if card.Notes != "" {
				b.WriteString("\n  ")
				b.WriteString(notesStyle.Render(card.Notes))
			}
			b.WriteString(renderUserNotes(m.progress, card.ConceptID, card.ID, "card", m.width))

			// Show concept info (diagrams, links, summaries)
			if card.ConceptID != "" {
				if concept, ok := m.conceptMap[card.ConceptID]; ok {
					info := renderConceptInfo(&concept, m.progress, card.ConceptID, m.width)
					if info != "" {
						b.WriteString(info)
					}
				}
			}

			b.WriteString("\n")

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
			b.WriteString(helpStyle.Render("  c concept  ·  v diagram  ·  d svg  ·  l link  ·  s summary  ·  n note  ·  q quit"))
		}

	case fcSummaryView:
		header := titleStyle.Render("  summary  ")
		b.WriteString(header)
		b.WriteString("\n\n")

		lines := strings.Split(m.summaryContent, "\n")
		visible := m.height - 4
		if visible < 1 {
			visible = 10
		}

		m.clampSummaryScroll()
		start := m.summaryScroll
		end := start + visible
		if end > len(lines) {
			end = len(lines)
		}

		for _, line := range lines[start:end] {
			b.WriteString(line)
			b.WriteString("\n")
		}

		b.WriteString(strings.Repeat("─", m.width))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  j/k scroll  ·  g/G top/bottom  ·  s/esc back"))

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
			Width(max(m.width-8, 20)).
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
			Width(max(m.width-8, 20)).
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
