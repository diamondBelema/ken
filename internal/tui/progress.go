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

type ProgressModel struct {
	subject     string
	subjects    []discovery.SubjectInfo
	progData    map[string]*progress.Progress
	conceptData map[string][]parser.Concept
	err         error
	viewWidth   int
	viewHeight  int
}

type progressLoadedMsg struct {
	subject     string
	subjects    []discovery.SubjectInfo
	progData    map[string]*progress.Progress
	conceptData map[string][]parser.Concept
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
			}
		}

		return progressLoadedMsg{subject: m.subject, subjects: subjects, progData: progData, conceptData: conceptData}
	}
}

func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case progressLoadedMsg:
		m.subject = msg.subject
		m.subjects = msg.subjects
		m.progData = msg.progData
		m.conceptData = msg.conceptData
	case progressErrMsg:
		m.err = msg.err
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.viewWidth = msg.Width
		m.viewHeight = msg.Height
	}
	return m, nil
}

func (m ProgressModel) View() string {
	if m.viewWidth == 0 {
		m.viewWidth = 80
	}

	var b strings.Builder

	header := titleStyle.Render("  progress  ")
	b.WriteString(header)
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(fmt.Sprintf("  Error: %v\n", m.err))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  q quit"))
		return b.String()
	}

	if len(m.subjects) == 0 {
		empty := lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(4, 2).
			Render("No subjects found.")
		b.WriteString(empty)
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("  q quit"))
		return b.String()
	}

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

		concepts := m.conceptData[s.Name]

		b.WriteString(subtitleStyle.Render(s.Name))
		b.WriteString("\n")

		for _, id := range progress.ConceptIDs(prog) {
			cs := prog.Concepts[id]
			status := "unknown"
			statusColor := colorMuted
			if cs.LastReviewedAt != nil {
				if cs.Confidence >= threshold {
					status = fmt.Sprintf("%.0f%% confident", cs.Confidence*100)
					statusColor = colorSuccess
				} else {
					status = fmt.Sprintf("%.0f%% (needs review)", cs.Confidence*100)
					statusColor = colorWarning
				}
			}

			conceptStyle := lipgloss.NewStyle().Foreground(colorTextBright).Bold(true)
			statusStyle := lipgloss.NewStyle().Foreground(statusColor)

			b.WriteString(fmt.Sprintf("  %s  %s\n", conceptStyle.Render(id), statusStyle.Render(status)))

			for _, c := range concepts {
				if c.ID == id {
					if c.Description != "" {
						desc := truncate(c.Description, 80)
						b.WriteString(fmt.Sprintf("    %s\n", lipgloss.NewStyle().Foreground(colorMuted).Render(desc)))
					}

					if c.Summary != "" {
						b.WriteString(fmt.Sprintf("    %s\n", lipgloss.NewStyle().Foreground(colorMuted).Render(truncate(c.Summary, 80))))
					}

					userSummaries := prog.SummariesForConcept(id)
					if len(userSummaries) > 0 {
						b.WriteString(fmt.Sprintf("    %s\n", lipgloss.NewStyle().Foreground(colorAccent).Render(fmt.Sprintf("%d user summaries", len(userSummaries)))))
					}

					notes := prog.NotesForConcept(id)
					if len(notes) > 0 {
						b.WriteString(fmt.Sprintf("    %s\n", lipgloss.NewStyle().Foreground(colorAccent).Render(fmt.Sprintf("%d notes", len(notes)))))
					}

					if len(c.Diagrams) > 0 {
						b.WriteString(fmt.Sprintf("    %s\n", lipgloss.NewStyle().Foreground(colorAccent).Render(fmt.Sprintf("%d diagrams", len(c.Diagrams)))))
					}

					if len(c.Links) > 0 {
						b.WriteString(fmt.Sprintf("    %s\n", lipgloss.NewStyle().Foreground(colorAccent).Render(fmt.Sprintf("%d links", len(c.Links)))))
					}

					break
				}
			}
		}
		b.WriteString("\n")
	}

	b.WriteString(strings.Repeat("─", m.viewWidth))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  q quit"))
	return b.String()
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
