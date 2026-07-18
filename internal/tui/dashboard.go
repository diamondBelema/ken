package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/diamondBelema/ken/internal/discovery"
	"github.com/diamondBelema/ken/internal/parser"
	"github.com/diamondBelema/ken/internal/progress"
	"github.com/diamondBelema/ken/internal/study"
)

type DashboardModel struct {
	subjects    []discovery.SubjectInfo
	progData    map[string]*progress.Progress
	conceptData map[string][]parser.Concept
	err         error
	width       int
	height      int
}

type dashboardLoadedMsg struct {
	subjects    []discovery.SubjectInfo
	progData    map[string]*progress.Progress
	conceptData map[string][]parser.Concept
}

type dashboardErrMsg struct {
	err error
}

func NewDashboardModel() DashboardModel {
	return DashboardModel{}
}

func (m DashboardModel) Init() tea.Cmd {
	return func() tea.Msg {
		home, err := os.UserHomeDir()
		if err != nil {
			return dashboardErrMsg{err}
		}

		subjectsDir := filepath.Join(home, "Documents", "learn", "subjects")
		subjects, err := discovery.Discover(subjectsDir)
		if err != nil {
			return dashboardErrMsg{err}
		}

		progData := make(map[string]*progress.Progress)
		conceptData := make(map[string][]parser.Concept)
		for _, s := range subjects {
			progPath, err := progress.SubjectPath(s.Name)
			if err != nil {
				continue
			}
			prog, err := progress.Load(progPath)
			if err != nil {
				continue
			}
			progData[s.Name] = prog

			concepts, err := study.LoadConcepts(subjectsDir, s.Name)
			if err == nil {
				conceptData[s.Name] = concepts
				progress.InitConcepts(prog, concepts)
			}
		}

		return dashboardLoadedMsg{subjects: subjects, progData: progData, conceptData: conceptData}
	}
}

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case dashboardLoadedMsg:
		m.subjects = msg.subjects
		m.progData = msg.progData
		m.conceptData = msg.conceptData
	case dashboardErrMsg:
		m.err = msg.err
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m DashboardModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n  Press q to exit.\n", m.err)
	}

	if m.width == 0 {
		m.width = 80
	}
	if m.height == 0 {
		m.height = 24
	}

	var b strings.Builder

	header := titleStyle.Render("  ken  ")
	b.WriteString(header)
	b.WriteString("\n\n")

	if len(m.subjects) == 0 {
		empty := lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(4, 2).
			Render("No subjects found.\n\n  Add content to ~/Documents/learn/subjects/")
		b.WriteString(empty)
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("  q quit"))
		return b.String()
	}

	totalConcepts := 0
	totalMastered := 0
	threshold := 0.7

	for _, s := range m.subjects {
		prog := m.progData[s.Name]
		concepts := m.conceptData[s.Name]
		conceptCount := len(concepts)
		mastered := 0
		noteCount := 0

		if prog != nil {
			noteCount = len(prog.Notes)
			for _, cs := range prog.Concepts {
				if cs.Confidence >= threshold {
					mastered++
				}
			}
		}

		totalConcepts += conceptCount
		totalMastered += mastered

		subjectBox := m.renderSubject(s, prog, conceptCount, mastered, noteCount)
		b.WriteString(subjectBox)
		b.WriteString("\n")
	}

	b.WriteString(strings.Repeat("─", m.width))
	b.WriteString("\n")

	statsLine := fmt.Sprintf("  %d concepts  ·  %d mastered  ·  %d subjects",
		totalConcepts, totalMastered, len(m.subjects))
	b.WriteString(statusBarStyle.Render(statsLine))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  flashcards <subject>  ·  quiz <subject>  ·  notes <subject>  ·  ken --help"))

	return b.String()
}

func (m DashboardModel) renderSubject(s discovery.SubjectInfo, prog *progress.Progress, conceptCount, mastered, noteCount int) string {
	var b strings.Builder

	b.WriteString(subtitleStyle.Render(s.Name))
	b.WriteString("\n")

	confBar := m.renderConfidenceBar(prog, conceptCount)
	b.WriteString("  ")
	b.WriteString(confBar)
	b.WriteString("\n")

	detail := fmt.Sprintf("  %d concepts  ·  %d cards  ·  %d quizzes  ·  %d notes",
		conceptCount, s.FlashcardFiles, s.QuizFiles, noteCount)
	b.WriteString(helpStyle.Render(detail))

	return b.String()
}

func (m DashboardModel) renderConfidenceBar(prog *progress.Progress, total int) string {
	if total == 0 {
		return lipgloss.NewStyle().Foreground(colorMuted).Render("no concepts loaded")
	}

	mastered := 0
	for _, cs := range prog.Concepts {
		if cs.LastReviewedAt != nil && cs.Confidence >= 0.7 {
			mastered++
		}
	}

	barWidth := 30
	masteredWidth := 0
	if total > 0 {
		masteredWidth = (mastered * barWidth) / total
	}

	filled := strings.Repeat("█", masteredWidth)
	empty := strings.Repeat("░", barWidth-masteredWidth)

	masteredStyle := lipgloss.NewStyle().Foreground(colorSuccess)
	remainingStyle := lipgloss.NewStyle().Foreground(colorMuted)

	result := masteredStyle.Render(filled) + remainingStyle.Render(empty)

	if total > 0 {
		pct := (mastered * 100) / total
		result += fmt.Sprintf(" %d%%", pct)
	}

	return result
}
