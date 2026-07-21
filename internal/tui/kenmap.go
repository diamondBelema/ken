package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/diamondBelema/ken/internal/diagram"
	"github.com/diamondBelema/ken/internal/groups"
	"github.com/diamondBelema/ken/internal/parser"
	"github.com/diamondBelema/ken/internal/progress"
	"github.com/diamondBelema/ken/internal/render"
)

type mapViewState int

const (
	mapTreeState mapViewState = iota
	mapSummaryView
	mapDiagramView
)

type mapNode struct {
	concept  parser.Concept
	depth    int
	children []*mapNode
}

type KenMapModel struct {
	subject        string
	concepts       []parser.Concept
	prog           *progress.Progress
	groups         []groups.CourseGroup
	groupFilter    string
	err            error
	viewWidth      int
	viewHeight     int
	viewState      mapViewState
	selected       int
	scrollTop      int
	flatNodes      []*mapNode
	expanded       map[string]bool
	diagramContent string
	diagramConcept string
}

func NewKenMapModel(subject string, concepts []parser.Concept, prog *progress.Progress, courseGroups []groups.CourseGroup, groupFilter string) KenMapModel {
	m := KenMapModel{
		subject:     subject,
		concepts:    concepts,
		prog:        prog,
		groups:      courseGroups,
		groupFilter: groupFilter,
		expanded:    make(map[string]bool),
	}

	tree := m.buildTree()
	m.flatNodes = m.flattenTree(tree)

	return m
}

func (m *KenMapModel) buildTree() []*mapNode {
	nodeMap := make(map[string]*mapNode)
	var roots []*mapNode

	for _, c := range m.concepts {
		nodeMap[c.ID] = &mapNode{concept: c}
	}

	for _, c := range m.concepts {
		node := nodeMap[c.ID]
		if c.ParentID == "" || c.ParentID == "none" {
			roots = append(roots, node)
		} else if parent, ok := nodeMap[c.ParentID]; ok {
			parent.children = append(parent.children, node)
		} else {
			roots = append(roots, node)
		}
	}

	if m.groupFilter != "" {
		return m.filterByGroup(roots, nodeMap)
	}

	return roots
}

func (m *KenMapModel) filterByGroup(roots []*mapNode, nodeMap map[string]*mapNode) []*mapNode {
	groupConcepts := make(map[string]bool)
	for _, g := range m.groups {
		if g.ID == m.groupFilter || g.Name == m.groupFilter {
			for _, id := range g.Concepts {
				groupConcepts[id] = true
			}
		}
	}
	if len(groupConcepts) == 0 {
		return roots
	}

	rootsInGroup := make(map[string]bool)
	for _, c := range m.concepts {
		if groupConcepts[c.ID] {
			ancestors := m.getAncestors(c, nodeMap)
			for _, id := range ancestors {
				groupConcepts[id] = true
			}
			rootsInGroup[c.ID] = true
		}
	}

	var filtered []*mapNode
	for _, r := range roots {
		if node := m.filterSubtree(r, groupConcepts); node != nil {
			filtered = append(filtered, node)
		}
	}
	return filtered
}

func (m *KenMapModel) getAncestors(c parser.Concept, nodeMap map[string]*mapNode) []string {
	var ids []string
	visited := make(map[string]bool)
	id := c.ParentID
	for id != "" && id != "none" && !visited[id] {
		ids = append(ids, id)
		visited[id] = true
		if parent, ok := nodeMap[id]; ok {
			id = parent.concept.ParentID
		} else {
			break
		}
	}
	return ids
}

func (m *KenMapModel) filterSubtree(node *mapNode, keep map[string]bool) *mapNode {
	if !keep[node.concept.ID] {
		var kept []*mapNode
		for _, child := range node.children {
			if filtered := m.filterSubtree(child, keep); filtered != nil {
				kept = append(kept, filtered)
			}
		}
		if len(kept) == 0 {
			return nil
		}
		node.children = kept
		return node
	}

	for i, child := range node.children {
		node.children[i] = m.filterSubtree(child, keep)
	}
	return node
}

func (m *KenMapModel) flattenTree(nodes []*mapNode) []*mapNode {
	var flat []*mapNode
	var walk func([]*mapNode, int)
	walk = func(nodes []*mapNode, depth int) {
		for _, n := range nodes {
			n.depth = depth
			flat = append(flat, n)
			if m.expanded[n.concept.ID] {
				walk(n.children, depth+1)
			}
		}
	}
	walk(nodes, 0)
	return flat
}

func (m KenMapModel) Init() tea.Cmd {
	return nil
}

func (m KenMapModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewWidth = msg.Width
		m.viewHeight = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.viewState == mapDiagramView {
			m.viewState = mapTreeState
			m.diagramContent = ""
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit

		case "j", "down":
			if m.selected < len(m.flatNodes)-1 {
				m.selected++
				m.adjustScroll()
			}

		case "k", "up":
			if m.selected > 0 {
				m.selected--
				m.adjustScroll()
			}

		case "enter", " ":
			if m.selected < len(m.flatNodes) {
				node := m.flatNodes[m.selected]
				m.expanded[node.concept.ID] = !m.expanded[node.concept.ID]
				m.rebuildFlatNodes()
				m.adjustScroll()
			}

		case "J":
			if m.selected < len(m.flatNodes)-1 {
				m.selected++
				m.adjustScroll()
				if m.selected < len(m.flatNodes) {
					node := m.flatNodes[m.selected]
					m.expanded[node.concept.ID] = true
					m.rebuildFlatNodes()
					m.adjustScroll()
				}
			}

		case "K":
			if m.selected > 0 {
				m.selected--
				m.adjustScroll()
			}

		case "g":
			if m.selected < len(m.flatNodes) {
				node := m.flatNodes[m.selected]
				cs := m.prog.Concepts[node.concept.ID]
				cs.Familiarity.Seen = true
				m.prog.Concepts[node.concept.ID] = cs
			}

		case "d":
			if m.selected < len(m.flatNodes) {
				node := m.flatNodes[m.selected]
				if len(node.concept.Diagrams) > 0 {
					d := node.concept.Diagrams[0]
					source, err := diagram.ResolveDiagramSource(d.Source, d.File, "")
					if err == nil && source != "" {
						ascii, err := diagram.RenderASCII(source)
						if err == nil {
							m.diagramContent = ascii
							m.diagramConcept = node.concept.ID
							m.viewState = mapDiagramView
						}
					}
				}
			}

		case "/":
			m.viewState = mapSummaryView

		case "n":
			m.expanded = make(map[string]bool)
			m.selected = 0
			m.rebuildFlatNodes()
			m.adjustScroll()
		}
	}

	return m, nil
}

func (m *KenMapModel) adjustScroll() {
	visible := m.viewHeight - 4
	if visible < 1 {
		visible = 1
	}

	if m.selected < m.scrollTop {
		m.scrollTop = m.selected
	}

	if m.selected >= m.scrollTop+visible {
		m.scrollTop = m.selected - visible + 1
	}
}

func (m *KenMapModel) rebuildFlatNodes() {
	tree := m.buildTree()
	m.flatNodes = m.flattenTree(tree)
}

func (m KenMapModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
	}

	if m.viewState == mapDiagramView {
		return m.renderDiagram()
	}

	if m.viewState == mapSummaryView {
		return m.renderSummaryView()
	}

	return m.renderTree()
}

func (m KenMapModel) renderTree() string {
	b := strings.Builder{}

	title := titleStyle.Render(fmt.Sprintf("KEN MAP — %s", m.subject))
	b.WriteString(title)
	b.WriteString("\n\n")

	if len(m.flatNodes) == 0 {
		b.WriteString("  No concepts found.\n\n")
		b.WriteString(helpStyle.Render("q: quit"))
		return b.String()
	}

	visible := m.viewHeight - 6
	if visible < 1 {
		visible = 1
	}

	end := m.scrollTop + visible
	if end > len(m.flatNodes) {
		end = len(m.flatNodes)
	}

	for i := m.scrollTop; end > m.scrollTop; i++ {
		node := m.flatNodes[i]
		prefix := strings.Repeat("  ", node.depth)

		arrow := " "
		if len(node.children) > 0 {
			if m.expanded[node.concept.ID] {
				arrow = "▼"
			} else {
				arrow = "▶"
			}
		}

		seen := ""
		cs := m.prog.Concepts[node.concept.ID]
		if cs.Familiarity.Seen {
			seen = lipgloss.NewStyle().Foreground(colorMuted).Render(" [seen]")
		}

		summary := ""
		if node.concept.Summary != "" {
			s := node.concept.Summary
			if len(s) > 60 {
				s = s[:60] + "…"
			}
			summary = " " + lipgloss.NewStyle().Foreground(colorMuted).Render(s)
		}

		line := fmt.Sprintf("%s%s %s%s%s", prefix, arrow, node.concept.Name, seen, summary)

		if i == m.selected {
			line = "→ " + line
		} else {
			line = "  " + line
		}

		b.WriteString(line)
		b.WriteString("\n")

		end--
		if i+1 >= len(m.flatNodes) {
			break
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑↓: navigate  space/enter: expand/collapse  d: diagrams  /: summary  g: mark seen  n: collapse all  q: quit"))
	return b.String()
}

func (m KenMapModel) renderDiagram() string {
	b := strings.Builder{}

	title := titleStyle.Render(fmt.Sprintf("DIAGRAM — %s", m.diagramConcept))
	b.WriteString(title)
	b.WriteString("\n\n")

	lines := strings.Split(m.diagramContent, "\n")
	visible := m.viewHeight - 6
	if visible < 1 {
		visible = 1
	}

	start := 0
	if len(lines) > visible {
		start = (len(lines) - visible) / 2
	}
	end := start + visible
	if end > len(lines) {
		end = len(lines)
	}

	for _, line := range lines[start:end] {
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("press any key to return"))
	return b.String()
}

func (m KenMapModel) renderSummaryView() string {
	b := strings.Builder{}

	title := titleStyle.Render(fmt.Sprintf("SUMMARY — %s", m.subject))
	b.WriteString(title)
	b.WriteString("\n\n")

	for i := m.scrollTop; i < len(m.flatNodes) && i < m.scrollTop+m.viewHeight-6; i++ {
		node := m.flatNodes[i]

		b.WriteString(subtitleStyle.Render(node.concept.Name))
		b.WriteString("\n")

		summary := node.concept.Summary
		if summary == "" {
			summary = lipgloss.NewStyle().Foreground(colorMuted).Render("(no summary)")
		}

		rendered := render.RenderMarkdown(summary, m.viewWidth-4)
		b.WriteString("  ")
		b.WriteString(strings.ReplaceAll(rendered, "\n", "\n  "))
		b.WriteString("\n\n")
	}

	b.WriteString(helpStyle.Render("↑↓: scroll  esc: back to tree  q: quit"))
	return b.String()
}
