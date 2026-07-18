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

type ProgressModel struct {
	subject  string
	subjects []discovery.SubjectInfo
	progData map[string]*progress.Progress
	err      error
}

type progressLoadedMsg struct {
	subject  string
	subjects []discovery.SubjectInfo
	progData map[string]*progress.Progress
}

type progressErrMsg struct {
	err error
}

func NewProgressModel(subject string) ProgressModel {
	return ProgressModel{subject: subject}
}

func (m ProgressModel) Init() tea.Cmd {
	return func() tea.Msg {
		home, err := os.UserHomeDir()
		if err != nil {
			return progressErrMsg{err}
		}

		subjectsDir := filepath.Join(home, "Documents", "learn", "subjects")
		subjects, err := discovery.Discover(subjectsDir)
		if err != nil {
			return progressErrMsg{err}
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

		return progressLoadedMsg{subject: m.subject, subjects: subjects, progData: progData}
	}
}

func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case progressLoadedMsg:
		m.subject = msg.subject
		m.subjects = msg.subjects
		m.progData = msg.progData
	case progressErrMsg:
		m.err = msg.err
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m ProgressModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to exit.\n", m.err)
	}

	if len(m.subjects) == 0 {
		return titleStyle.Render("Progress") + "\n\n" +
			subtitleStyle.Render("No subjects found.") + "\n\n" +
			helpStyle.Render("Press q to exit.")
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Progress"))
	b.WriteString("\n\n")

	subjectsToShow := m.subjects
	if m.subject != "" {
		for _, s := range m.subjects {
			if s.Name == m.subject {
				subjectsToShow = []discovery.SubjectInfo{s}
				break
			}
		}
	}

	threshold := 0.7
	for _, s := range subjectsToShow {
		prog := m.progData[s.Name]
		if prog == nil {
			continue
		}

		b.WriteString(subtitleStyle.Render(s.Name))
		b.WriteString("\n")

		for _, id := range progress.ConceptIDs(prog) {
			cs := prog.Concepts[id]
			status := "unknown"
			if cs.LastReviewedAt != nil {
				if cs.Confidence >= threshold {
					status = fmt.Sprintf("%.0f%% confident", cs.Confidence*100)
				} else {
					status = fmt.Sprintf("%.0f%% (needs review)", cs.Confidence*100)
				}
			}
			b.WriteString(fmt.Sprintf("  %s: %s\n", id, status))
		}
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("Press q to exit."))
	return b.String()
}
