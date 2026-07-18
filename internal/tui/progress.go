package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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

		concepts := m.conceptData[s.Name]

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

			for _, c := range concepts {
				if c.ID == id {
					if c.Description != "" {
						desc := c.Description
						if len(desc) > 80 {
							desc = desc[:80] + "..."
						}
						desc = strings.ReplaceAll(desc, "\n", " ")
						b.WriteString(fmt.Sprintf("    Description: %s\n", desc))
					}

					if c.Summary != "" {
						b.WriteString(fmt.Sprintf("    Content Summary: %s\n", truncate(c.Summary, 80)))
					}

					userSummaries := prog.SummariesForConcept(id)
					for _, s := range userSummaries {
						b.WriteString(fmt.Sprintf("    User Summary: %s\n", s.Title))
					}

					notes := prog.NotesForConcept(id)
					if len(notes) > 0 {
						b.WriteString(fmt.Sprintf("    Notes (%d):\n", len(notes)))
						for _, n := range notes {
							preview := n.Content
							if len(preview) > 60 {
								preview = preview[:60] + "..."
							}
							preview = strings.ReplaceAll(preview, "\n", " ")
							b.WriteString(fmt.Sprintf("      - \"%s\"\n", preview))
						}
					}

					if len(c.Diagrams) > 0 {
						b.WriteString("    Diagrams: ")
						for i, d := range c.Diagrams {
							if i > 0 {
								b.WriteString(", ")
							}
							b.WriteString(d.Label)
						}
						b.WriteString("\n")
					}

					if len(c.Links) > 0 {
						b.WriteString("    Links: ")
						for i, l := range c.Links {
							if i > 0 {
								b.WriteString(", ")
							}
							b.WriteString(fmt.Sprintf("[%s](%s)", l.Title, l.Type))
						}
						b.WriteString("\n")
					}

					break
				}
			}
		}
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("Press q to exit."))
	return b.String()
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
