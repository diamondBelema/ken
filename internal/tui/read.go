package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/diamondBelema/ken/internal/parser"
	"github.com/diamondBelema/ken/internal/progress"
	"github.com/diamondBelema/ken/internal/render"
)

type readState int

const (
	readFileList readState = iota
	readTreeView
	readContentView
)

type ReadModel struct {
	files          []parser.NoteFile
	prog           *progress.Progress
	subject        string
	documents      []parser.Document
	state          readState
	selected       int
	scrollTop      int
	expanded       map[string]bool
	viewWidth      int
	viewHeight     int
	totalLines     int
	renderedLines  []string
	hopTargets     []hopTarget
	hopIdx         int
}

type hopTarget struct {
	line     int
	conceptID string
}

func NewReadModel(files []parser.NoteFile, prog *progress.Progress, subject string) ReadModel {
	m := ReadModel{
		files:    files,
		prog:     prog,
		subject:  subject,
		state:    readFileList,
		expanded: make(map[string]bool),
	}

	for _, f := range files {
		doc := parser.ParseTaggedNote(f.Content)
		m.documents = append(m.documents, doc)
	}

	return m
}

func (m ReadModel) Init() tea.Cmd {
	return nil
}

func (m ReadModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewWidth = msg.Width
		m.viewHeight = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case readFileList:
			return m.updateFileList(msg)
		case readTreeView:
			return m.updateTreeView(msg)
		case readContentView:
			return m.updateContentView(msg)
		}
	}

	return m, nil
}

func (m ReadModel) updateFileList(msg tea.KeyMsg) (ReadModel, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.selected < len(m.files)-1 {
			m.selected++
			m.clampFileListScroll()
		}
	case "k", "up":
		if m.selected > 0 {
			m.selected--
			m.clampFileListScroll()
		}
	case "g":
		m.selected = 0
		m.clampFileListScroll()
	case "G":
		m.selected = len(m.files) - 1
		m.clampFileListScroll()
	case "enter":
		if len(m.files) > 0 {
			m.state = readTreeView
			m.scrollTop = 0
		}
	case "q", "esc":
		return m, tea.Quit
	}
	return m, nil
}

func (m ReadModel) updateTreeView(msg tea.KeyMsg) (ReadModel, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.state = readFileList
		m.scrollTop = 0
		return m, nil
	case "j", "down":
		m.scrollTop++
		m.clampTreeScroll()
	case "k", "up":
		if m.scrollTop > 0 {
			m.scrollTop--
		}
	case "g":
		m.scrollTop = 0
	case "G":
		m.scrollTop = 9999
		m.clampTreeScroll()
	case "enter", " ":
		m.state = readContentView
		m.scrollTop = 0
		m.buildHopTargets()
		m.hopIdx = 0
		return m, nil
	}
	return m, nil
}

func (m ReadModel) updateContentView(msg tea.KeyMsg) (ReadModel, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.state = readTreeView
		m.scrollTop = 0
		return m, nil

	case "j", "down", "ctrl+e":
		m.scrollTop++
		m.clampContentScroll()
	case "k", "up", "ctrl+y":
		if m.scrollTop > 0 {
			m.scrollTop--
		}
	case "g":
		m.scrollTop = 0
	case "G":
		m.scrollTop = m.totalLines - (m.viewHeight - 5)
		m.clampContentScroll()

	case " ", "pgdown", "ctrl+f":
		m.scrollTop += m.viewHeight - 5
		m.clampContentScroll()
	case "pgup", "ctrl+b":
		m.scrollTop -= m.viewHeight - 5
		if m.scrollTop < 0 {
			m.scrollTop = 0
		}

	case "n", "]", "tab":
		m.hopForward()
	case "N", "[", "shift+tab":
		m.hopBackward()

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		m.hopToNth(msg.String())
	}
	return m, nil
}

func (m *ReadModel) hopForward() {
	if len(m.hopTargets) == 0 {
		return
	}
	m.hopIdx++
	if m.hopIdx >= len(m.hopTargets) {
		m.hopIdx = 0
	}
	m.scrollToLine(m.hopTargets[m.hopIdx].line)
}

func (m *ReadModel) hopBackward() {
	if len(m.hopTargets) == 0 {
		return
	}
	m.hopIdx--
	if m.hopIdx < 0 {
		m.hopIdx = len(m.hopTargets) - 1
	}
	m.scrollToLine(m.hopTargets[m.hopIdx].line)
}

func (m *ReadModel) hopToNth(key string) {
	if len(m.hopTargets) == 0 {
		return
	}
	n := int(key[0] - '0')
	if n > 0 && n <= len(m.hopTargets) {
		m.hopIdx = n - 1
		m.scrollToLine(m.hopTargets[m.hopIdx].line)
	}
}

func (m *ReadModel) scrollToLine(line int) {
	visible := m.viewHeight - 5
	if visible < 1 {
		visible = 1
	}

	if line < m.scrollTop {
		m.scrollTop = line
	} else if line >= m.scrollTop+visible {
		m.scrollTop = line - visible + 1
	}
}

func (m *ReadModel) buildHopTargets() {
	m.hopTargets = nil
	if m.selected >= len(m.documents) {
		return
	}

	doc := m.documents[m.selected]
	f := m.files[m.selected]

	rendered := render.RenderMarkdown(f.Content, m.viewWidth-4)
	m.renderedLines = strings.Split(rendered, "\n")
	m.totalLines = len(m.renderedLines)

	headingLine := 0
	var walk func([]parser.Section)
	walk = func(sections []parser.Section) {
		for _, s := range sections {
			if s.ConceptID != "" {
				m.hopTargets = append(m.hopTargets, hopTarget{
					line:      headingLine,
					conceptID: s.ConceptID,
				})
			}
			headingLine++
			for _, child := range s.Children {
				_ = child
			}
		}
	}
	_ = doc
	_ = walk

	m.rebuildHopTargetsFromContent(f.Content)
}

func (m *ReadModel) rebuildHopTargetsFromContent(content string) {
	m.hopTargets = nil
	lines := strings.Split(content, "\n")
	conceptTagRe := strings.NewReplacer("[c-", "", "]", "")

	for i, line := range lines {
		if strings.Contains(line, "[c-") {
			idx := strings.Index(line, "[c-")
			end := strings.Index(line[idx:], "]")
			if end > 0 {
				tag := line[idx+1 : idx+end]
				conceptID := conceptTagRe.Replace(tag)
				m.hopTargets = append(m.hopTargets, hopTarget{
					line:      i,
					conceptID: conceptID,
				})
			}
		}
	}
}

func (m ReadModel) View() string {
	switch m.state {
	case readFileList:
		return m.renderFileList()
	case readTreeView:
		return m.renderTreeView()
	case readContentView:
		return m.renderContentView()
	}
	return ""
}

func (m ReadModel) renderFileList() string {
	b := strings.Builder{}

	title := titleStyle.Render(fmt.Sprintf("KEN READ — %s", m.subject))
	b.WriteString(title)
	b.WriteString("\n\n")

	if len(m.files) == 0 {
		b.WriteString("  No lecture notes found.\n\n")
		b.WriteString(helpStyle.Render("q: quit"))
		return b.String()
	}

	visible := m.viewHeight - 6
	if visible < 1 {
		visible = 1
	}

	start := m.scrollTop
	end := start + visible
	if end > len(m.files) {
		end = len(m.files)
	}

	for i := start; i < end; i++ {
		f := m.files[i]
		doc := m.documents[i]

		line := f.Name
		if doc.Title != "" {
			line = fmt.Sprintf("%s — %s", f.Name, doc.Title)
		}

		if i == m.selected {
			line = "→ " + line
		} else {
			line = "  " + line
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑↓: navigate  enter: tree  q: quit"))
	return b.String()
}

func (m ReadModel) renderTreeView() string {
	b := strings.Builder{}

	if m.selected >= len(m.documents) {
		return "No document selected."
	}

	f := m.files[m.selected]

	title := titleStyle.Render(fmt.Sprintf("CONCEPT TREE — %s", f.Name))
	b.WriteString(title)
	b.WriteString("\n\n")

	doc := m.documents[m.selected]
	visible := m.viewHeight - 6
	if visible < 1 {
		visible = 1
	}

	var allLines []string
	for _, section := range doc.Sections {
		allLines = append(allLines, m.renderSectionTree(section, 0)...)
	}

	start := m.scrollTop
	end := start + visible
	if end > len(allLines) {
		end = len(allLines)
	}
	if start > len(allLines) {
		start = len(allLines)
	}

	for i := start; i < end; i++ {
		b.WriteString(allLines[i])
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑↓: scroll  enter: read  esc: back  q: quit"))
	return b.String()
}

func (m ReadModel) renderSectionTree(section parser.Section, depth int) []string {
	var lines []string

	prefix := strings.Repeat("  ", depth)
	arrow := "▶"
	if len(section.Children) > 0 {
		arrow = "▼"
	}

	status := ""
	if section.ConceptID != "" {
		cs := m.prog.Concepts[section.ConceptID]
		if cs.Familiarity.Seen {
			status += " " + lipgloss.NewStyle().Foreground(colorSuccess).Render("[✓ familiar]")
		} else {
			status += " " + lipgloss.NewStyle().Foreground(colorMuted).Render("[— familiar]")
		}
		if cs.Reflection.Count > 0 {
			status += " " + lipgloss.NewStyle().Foreground(colorSuccess).Render("[✓ reflected]")
		} else {
			status += " " + lipgloss.NewStyle().Foreground(colorMuted).Render("[— reflected]")
		}
		if cs.Mastery.Confidence > 0 {
			confPct := fmt.Sprintf("%.0f%%", cs.Mastery.Confidence*100)
			status += " " + lipgloss.NewStyle().Foreground(colorPrimary).Render("["+confPct+"]")
		}
	}

	heading := section.Heading
	if section.ConceptID != "" {
		heading = fmt.Sprintf("%s %s", lipgloss.NewStyle().Foreground(colorAccent).Render("c-"+section.ConceptID), heading)
	}

	lines = append(lines, fmt.Sprintf("%s%s %s%s\n", prefix, arrow, heading, status))

	for _, child := range section.Children {
		lines = append(lines, m.renderSectionTree(child, depth+1)...)
	}

	return lines
}

func (m ReadModel) renderContentView() string {
	b := strings.Builder{}

	if m.selected >= len(m.files) {
		return "No document selected."
	}

	f := m.files[m.selected]

	docTitle := f.Name
	if m.selected < len(m.documents) && m.documents[m.selected].Title != "" {
		docTitle = m.documents[m.selected].Title
	}

	hopInfo := ""
	if len(m.hopTargets) > 0 {
		target := m.hopTargets[m.hopIdx]
		hopInfo = fmt.Sprintf("  %s %d/%d  %s",
			lipgloss.NewStyle().Foreground(colorAccent).Render("concept"),
			m.hopIdx+1,
			len(m.hopTargets),
			lipgloss.NewStyle().Foreground(colorMuted).Render(target.conceptID))
	}

	title := titleStyle.Render(fmt.Sprintf("READING — %s", docTitle))
	b.WriteString(title)
	b.WriteString("\n")

	scrollPct := 0
	if m.totalLines > 0 {
		scrollPct = m.scrollTop * 100 / m.totalLines
	}
	scrollInfo := lipgloss.NewStyle().Foreground(colorMuted).Render(
		fmt.Sprintf("  line %d-%d/%d (%d%%)", m.scrollTop+1,
			min(m.scrollTop+m.viewHeight-5, m.totalLines),
			m.totalLines, scrollPct))
	b.WriteString(scrollInfo)
	b.WriteString(hopInfo)
	b.WriteString("\n\n")

	if len(m.renderedLines) == 0 {
		m.renderedLines = strings.Split(render.RenderMarkdown(f.Content, m.viewWidth-4), "\n")
		m.totalLines = len(m.renderedLines)
	}

	visible := m.viewHeight - 7
	if visible < 1 {
		visible = 1
	}

	start := m.scrollTop
	end := start + visible
	if end > len(m.renderedLines) {
		end = len(m.renderedLines)
	}
	if start > len(m.renderedLines) {
		start = len(m.renderedLines)
	}

	for i := start; i < end; i++ {
		b.WriteString("  ")
		b.WriteString(m.renderedLines[i])
		b.WriteString("\n")
	}

	if m.totalLines > visible {
		footer := lipgloss.NewStyle().Foreground(colorMuted).Render(
			fmt.Sprintf("  ── %d%% ──", scrollPct))
		b.WriteString(footer)
	} else {
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("↑↓: scroll  pgup/pgdn: page  n/]: next concept  N/[: prev  1-9: jump  esc: back  q: quit"))
	return b.String()
}

func (m *ReadModel) clampFileListScroll() {
	visible := m.viewHeight - 6
	if visible < 1 {
		visible = 1
	}
	maxScroll := len(m.files) - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scrollTop > maxScroll {
		m.scrollTop = maxScroll
	}
	if m.scrollTop < 0 {
		m.scrollTop = 0
	}
}

func (m *ReadModel) clampTreeScroll() {
	visible := m.viewHeight - 6
	if visible < 1 {
		visible = 1
	}
	if m.scrollTop < 0 {
		m.scrollTop = 0
	}
}

func (m *ReadModel) clampContentScroll() {
	if m.totalLines == 0 {
		return
	}
	visible := m.viewHeight - 7
	if visible < 1 {
		visible = 1
	}
	maxScroll := m.totalLines - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scrollTop > maxScroll {
		m.scrollTop = maxScroll
	}
	if m.scrollTop < 0 {
		m.scrollTop = 0
	}
}
