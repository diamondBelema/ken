package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/diamondBelema/ken/internal/discovery"
	"github.com/diamondBelema/ken/internal/progress"
)

type DashboardModel struct {
	subjects []discovery.SubjectInfo
	progData map[string]*progress.Progress
	err      error
}

type dashboardLoadedMsg struct {
	subjects []discovery.SubjectInfo
	progData map[string]*progress.Progress
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
		}

		return dashboardLoadedMsg{subjects: subjects, progData: progData}
	}
}

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case dashboardLoadedMsg:
		m.subjects = msg.subjects
		m.progData = msg.progData
	case dashboardErrMsg:
		m.err = msg.err
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
		return fmt.Sprintf("Error: %v\n\nPress q to exit.\n", m.err)
	}

	if len(m.subjects) == 0 {
		return titleStyle.Render("ken") + "\n\n" +
			subtitleStyle.Render("No subjects found.") + "\n" +
			helpStyle.Render("Add content to ~/Documents/learn/subjects/") + "\n\n" +
			helpStyle.Render("Press q to exit.")
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("ken — Dashboard"))
	b.WriteString("\n\n")

	totalConcepts := 0
	totalAboveThreshold := 0
	threshold := 0.7

	for _, s := range m.subjects {
		prog := m.progData[s.Name]
		conceptCount := len(prog.Concepts)
		aboveThreshold := 0

		for _, cs := range prog.Concepts {
			if cs.Confidence >= threshold {
				aboveThreshold++
			}
		}

		totalConcepts += conceptCount
		totalAboveThreshold += aboveThreshold

		b.WriteString(subtitleStyle.Render(s.Name))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %d concepts (%d above %.0f%% confidence)\n",
			conceptCount, aboveThreshold, threshold*100))
		b.WriteString(fmt.Sprintf("  %d flashcards, %d quizzes\n",
			s.FlashcardFiles, s.QuizFiles))
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render(fmt.Sprintf("Total: %d concepts across %d subjects", totalConcepts, len(m.subjects))))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Press q to exit."))

	return b.String()
}
