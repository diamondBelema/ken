package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/diamondBelema/ken/internal/diagram"
	"github.com/diamondBelema/ken/internal/discovery"
	"github.com/diamondBelema/ken/internal/parser"
	"github.com/diamondBelema/ken/internal/progress"
	"github.com/diamondBelema/ken/internal/render"
	"github.com/diamondBelema/ken/internal/study"
)

type progressViewState int

const (
	progressList progressViewState = iota
	progressDiagramView
	progressDetail
	progressLinkOpen
)

type ProgressModel struct {
	subject        string
	subjects       []discovery.SubjectInfo
	progData       map[string]*progress.Progress
	conceptData    map[string][]parser.Concept
	err            error
	viewWidth      int
	viewHeight     int
	viewState      progressViewState
	selected       int
	scrollTop      int
	cachedItems    []progressItem
	diagramContent string
	diagramConcept string
	detailName     string
	detailContent  string
	detailScroll   int
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
				progress.InitConcepts(prog, concepts)
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
		m.cachedItems = m.collectItems()
	case progressErrMsg:
		m.err = msg.err
	case tea.WindowSizeMsg:
		m.viewWidth = msg.Width
		m.viewHeight = msg.Height
	case tea.KeyMsg:
		if m.viewState == progressDiagramView {
			if msg.String() == "q" || msg.String() == "esc" {
				m.viewState = progressList
				return m, nil
			}
			return m, nil
		}

		if m.viewState == progressDetail {
			switch msg.String() {
			case "q", "esc":
				m.viewState = progressList
				return m, nil
			case "j", "down":
				m.detailScroll++
				m.clampDetailScroll()
			case "k", "up":
				if m.detailScroll > 0 {
					m.detailScroll--
				}
			case "g":
				m.detailScroll = 0
			case "G":
				m.detailScroll = 999999
				m.clampDetailScroll()
			}
			return m, nil
		}

		items := m.collectItems()
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.selected < len(items)-1 {
				m.selected++
				m.clampScroll()
			}
		case "k", "up":
			if m.selected > 0 {
				m.selected--
				m.clampScroll()
			}
		case "g":
			m.selected = 0
			m.clampScroll()
		case "G":
			m.selected = len(items) - 1
			m.clampScroll()
		case "v":
			if item, ok := m.selectedItem(items); ok && item.diagramSource != "" {
				ascii, err := diagram.RenderASCII(item.diagramSource)
				if err != nil {
					ascii = fmt.Sprintf("render error: %v", err)
				}
				m.diagramContent = ascii
				m.diagramConcept = item.id
				m.viewState = progressDiagramView
			}
		case "d":
			if item, ok := m.selectedItem(items); ok && item.diagramSource != "" {
				tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("ken-diagram-%s.svg", item.id))
				if err := diagram.RenderSVGToFile(item.diagramSource, tmpPath); err == nil {
					exec.Command("xdg-open", tmpPath).Start()
				}
			}
		case "l":
			if item, ok := m.selectedItem(items); ok && item.linkURL != "" {
				exec.Command("xdg-open", item.linkURL).Start()
			}
		case "enter", " ":
			if item, ok := m.selectedItem(items); ok && item.fullDesc != "" {
				m.detailName = item.id
				m.detailContent = item.fullDesc
				if item.summary != "" {
					m.detailContent += "\n\n---\n\n" + item.summary
				}
				m.detailScroll = 0
				m.viewState = progressDetail
			}
		}
	}
	return m, nil
}

type progressItem struct {
	id            string
	name          string
	status        string
	statusColor   lipgloss.Style
	desc          string
	fullDesc      string
	summary       string
	userSummaries int
	noteCount     int
	diagramCount  int
	diagramSource string
	linkCount     int
	linkURL       string
	linkTitle     string
}

func (m ProgressModel) collectItems() []progressItem {
	var items []progressItem
	threshold := 0.7

	subjectsToShow := m.subjects
	if m.subject != "" {
		for _, s := range m.subjects {
			if s.Name == m.subject {
				subjectsToShow = []discovery.SubjectInfo{s}
				break
			}
		}
	}

	for _, s := range subjectsToShow {
		prog := m.progData[s.Name]
		if prog == nil {
			continue
		}
		concepts := m.conceptData[s.Name]

		for _, id := range progress.ConceptIDs(prog) {
			cs := prog.Concepts[id]
			item := progressItem{id: id}
			item.status = "unknown"
			item.statusColor = lipgloss.NewStyle().Foreground(colorMuted)
			if cs.LastReviewedAt != nil {
				if cs.Confidence >= threshold {
					item.status = fmt.Sprintf("%.0f%% confident", cs.Confidence*100)
					item.statusColor = lipgloss.NewStyle().Foreground(colorSuccess)
				} else {
					item.status = fmt.Sprintf("%.0f%% (needs review)", cs.Confidence*100)
					item.statusColor = lipgloss.NewStyle().Foreground(colorWarning)
				}
			}

		for _, c := range concepts {
			if c.ID == id {
				if c.Name != "" {
					item.name = c.Name
				}
				if c.Description != "" {
					item.desc = runeTruncate(c.Description, 80)
					item.fullDesc = c.Description
				}
				if c.Summary != "" {
					item.summary = runeTruncate(c.Summary, 80)
				}
					item.userSummaries = len(prog.SummariesForConcept(id))
					item.noteCount = len(prog.NotesForConcept(id))
					item.diagramCount = len(c.Diagrams)
					if len(c.Diagrams) > 0 {
						item.diagramSource = c.Diagrams[0].Source
					}
					item.linkCount = len(c.Links)
					if len(c.Links) > 0 {
						item.linkURL = c.Links[0].URL
						item.linkTitle = c.Links[0].Title
					}
					break
				}
			}
			items = append(items, item)
		}
	}
	return items
}

func (m ProgressModel) selectedItem(items []progressItem) (progressItem, bool) {
	if m.selected >= 0 && m.selected < len(items) {
		return items[m.selected], true
	}
	return progressItem{}, false
}

func (m *ProgressModel) clampScroll() {
	items := m.cachedItems
	if len(items) == 0 {
		return
	}

	// Header=4 (title+margin+2 newlines), separator=1, footer=1 → 6 lines overhead
	available := m.viewHeight - 6
	if available < 1 {
		available = 10
	}

	// Count how many items actually fit from scrollTop
	linesUsed := 0
	visibleCount := 0
	for i := m.scrollTop; i < len(items) && linesUsed < available; i++ {
		linesUsed++ // item line
		if items[i].desc != "" {
			linesUsed++ // description line
		}
		if items[i].summary != "" {
			linesUsed++ // summary line
		}
		visibleCount++
	}
	if visibleCount < 1 {
		visibleCount = 1
	}

	if m.selected < m.scrollTop {
		m.scrollTop = m.selected
	}
	if m.selected >= m.scrollTop+visibleCount {
		m.scrollTop = m.selected - visibleCount + 1
	}
	if m.scrollTop < 0 {
		m.scrollTop = 0
	}
}

func (m *ProgressModel) clampDetailScroll() {
	rendered := render.RenderMarkdown(m.detailContent, m.viewWidth-4)
	lines := strings.Split(rendered, "\n")

	// Header = title(1) + blank(1) + separator(1) = 3, footer = 1
	visible := m.viewHeight - 4
	if visible < 1 {
		visible = 10
	}

	maxScroll := len(lines) - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.detailScroll > maxScroll {
		m.detailScroll = maxScroll
	}
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

	if m.viewState == progressDiagramView {
		b.WriteString(subtitleStyle.Render(fmt.Sprintf("  diagram · %s", m.diagramConcept)))
		b.WriteString("\n\n")
		b.WriteString(m.diagramContent)
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("  q/esc back"))
		return b.String()
	}

	if m.viewState == progressDetail {
		detailTitle := m.detailName
		if item, ok := m.selectedItem(m.cachedItems); ok && item.name != "" {
			detailTitle = item.name
		}
		b.WriteString(subtitleStyle.Render(fmt.Sprintf("  %s", detailTitle)))
		b.WriteString("\n\n")

		rendered := render.RenderMarkdown(m.detailContent, m.viewWidth-4)
		lines := strings.Split(rendered, "\n")

		visible := m.viewHeight - 4
		if visible < 1 {
			visible = 10
		}

		maxScroll := len(lines) - visible
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.detailScroll > maxScroll {
			m.detailScroll = maxScroll
		}

		start := m.detailScroll
		end := start + visible
		if end > len(lines) {
			end = len(lines)
		}

		for _, line := range lines[start:end] {
			b.WriteString(line)
			b.WriteString("\n")
		}

		b.WriteString(strings.Repeat("─", m.viewWidth))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  j/k scroll  ·  g/G top/bottom  ·  q/esc back"))
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

	items := m.cachedItems
	available := m.viewHeight - 6
	if available < 1 {
		available = 10
	}

	// Count how many items actually fit from scrollTop
	linesUsed := 0
	visibleCount := 0
	for i := m.scrollTop; i < len(items) && linesUsed < available; i++ {
		linesUsed++ // item line
		if items[i].desc != "" {
			linesUsed++ // description line
		}
		if items[i].summary != "" {
			linesUsed++ // summary line
		}
		visibleCount++
	}
	if visibleCount < 1 {
		visibleCount = 1
	}

	if len(items) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Padding(4, 2).Render("No concepts found."))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("  q quit"))
		return b.String()
	}

	end := m.scrollTop + visibleCount
	if end > len(items) {
		end = len(items)
	}

	for i := m.scrollTop; i < end; i++ {
		item := items[i]
		displayName := item.id
		if item.name != "" {
			displayName = item.name
		}
		if i == m.selected {
			b.WriteString(listItemSelectedStyle.Render(fmt.Sprintf("  %s  %s", displayName, item.status)))
			b.WriteString("\n")
		} else {
			b.WriteString(fmt.Sprintf("  %s  %s\n", lipgloss.NewStyle().Foreground(colorTextBright).Bold(true).Render(displayName), item.statusColor.Render(item.status)))
		}
		if item.desc != "" {
			b.WriteString(fmt.Sprintf("    %s\n", lipgloss.NewStyle().Foreground(colorMuted).Render(item.desc)))
		}
		if item.summary != "" {
			b.WriteString(fmt.Sprintf("    %s\n", lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render(item.summary)))
		}
	}

	b.WriteString(strings.Repeat("─", m.viewWidth))
	b.WriteString("\n")

	if len(items) > visibleCount {
		b.WriteString(helpStyle.Render(fmt.Sprintf("  %d-%d of %d  ·  j/k navigate  ·  enter view  ·  v diagram  ·  d svg  ·  l link  ·  q quit", m.scrollTop+1, end, len(items))))
	} else {
		b.WriteString(helpStyle.Render("  j/k navigate  ·  enter view  ·  v diagram  ·  d svg  ·  l link  ·  q quit"))
	}

	return b.String()
}


