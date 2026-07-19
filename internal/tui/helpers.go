package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/diamondBelema/ken/internal/parser"
	"github.com/diamondBelema/ken/internal/progress"
	"github.com/diamondBelema/ken/internal/render"
)

func renderProgressBar(current, total int) string {
	barWidth := 20
	filled := 0
	if total > 0 {
		filled = (current * barWidth) / total
	}

	barFilled := strings.Repeat("━", filled)
	barEmpty := strings.Repeat("─", barWidth-filled)

	filledStyle := lipgloss.NewStyle().Foreground(colorPrimary)
	emptyStyle := lipgloss.NewStyle().Foreground(colorMuted)

	result := filledStyle.Render(barFilled) + emptyStyle.Render(barEmpty)
	result += fmt.Sprintf(" %d/%d", current, total)

	return result
}

func runeTruncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	runes := []rune(s)
	if len(runes) > maxLen {
		if maxLen <= 3 {
			return strings.Repeat(".", maxLen)
		}
		return string(runes[:maxLen-3]) + "..."
	}
	return s
}

func buildConceptMap(concepts []parser.Concept) map[string]parser.Concept {
	m := make(map[string]parser.Concept, len(concepts))
	for _, c := range concepts {
		m[c.ID] = c
	}
	return m
}

func lookupConcept(conceptMap map[string]parser.Concept, conceptID string) *parser.Concept {
	if conceptID == "" {
		return nil
	}
	c, ok := conceptMap[conceptID]
	if !ok {
		return nil
	}
	return &c
}

type conceptTreeNode struct {
	concept parser.Concept
	depth   int
	children []conceptTreeNode
}

func buildHierarchy(concepts []parser.Concept) []parser.Concept {
	conceptMap := buildConceptMap(concepts)
	childrenOf := make(map[string][]parser.Concept)
	var roots []parser.Concept

	for _, c := range concepts {
		if c.ParentID == "" {
			roots = append(roots, c)
		} else {
			childrenOf[c.ParentID] = append(childrenOf[c.ParentID], c)
		}
	}

	var result []parser.Concept
	var walk func(parentID string, depth int)
	walk = func(parentID string, depth int) {
		var kids []parser.Concept
		if parentID == "" {
			kids = roots
		} else {
			kids = childrenOf[parentID]
		}
		for _, c := range kids {
			result = append(result, c)
			_ = conceptMap
			walk(c.ID, depth+1)
		}
	}
	walk("", 0)
	return result
}

func conceptDepth(conceptMap map[string]parser.Concept, id string) int {
	depth := 0
	for {
		c, ok := conceptMap[id]
		if !ok || c.ParentID == "" {
			return depth
		}
		depth++
		id = c.ParentID
	}
}

func conceptChildren(conceptMap map[string]parser.Concept, parentID string) []parser.Concept {
	var kids []parser.Concept
	for _, c := range conceptMap {
		if c.ParentID == parentID {
			kids = append(kids, c)
		}
	}
	sort.Slice(kids, func(i, j int) bool {
		return kids[i].ID < kids[j].ID
	})
	return kids
}

func conceptBreadcrumb(conceptMap map[string]parser.Concept, id string) string {
	var path []string
	for {
		c, ok := conceptMap[id]
		if !ok {
			break
		}
		path = append([]string{c.ID}, path...)
		if c.ParentID == "" {
			break
		}
		id = c.ParentID
	}
	return strings.Join(path, " > ")
}

func sortedConceptsByHierarchy(concepts []parser.Concept) []parser.Concept {
	childrenOf := make(map[string][]parser.Concept)
	var roots []parser.Concept
	for _, c := range concepts {
		if c.ParentID == "" {
			roots = append(roots, c)
		} else {
			childrenOf[c.ParentID] = append(childrenOf[c.ParentID], c)
		}
	}

	sort.SliceStable(roots, func(i, j int) bool {
		return roots[i].ID < roots[j].ID
	})
	for k := range childrenOf {
		sort.SliceStable(childrenOf[k], func(i, j int) bool {
			return childrenOf[k][i].ID < childrenOf[k][j].ID
		})
	}

	var result []parser.Concept
	var walk func(parents []parser.Concept)
	walk = func(parents []parser.Concept) {
		for _, p := range parents {
			result = append(result, p)
			walk(childrenOf[p.ID])
		}
	}
	walk(roots)
	return result
}

func buildConceptMarkdown(concept *parser.Concept, conceptMap map[string]parser.Concept) string {
	if concept == nil {
		return ""
	}

	var parts []string

	parts = append(parts, fmt.Sprintf("# %s\n", concept.Name))

	if concept.Description != "" {
		parts = append(parts, concept.Description+"\n")
	}

	if concept.Summary != "" {
		parts = append(parts, fmt.Sprintf("## Summary\n\n%s\n", concept.Summary))
	}

	if conceptMap != nil {
		if concept.ParentID != "" {
			if parent, ok := conceptMap[concept.ParentID]; ok {
				parts = append(parts, fmt.Sprintf("## Parent\n\n%s (`%s`)\n", parent.Name, parent.ID))
			}
		}
		kids := conceptChildren(conceptMap, concept.ID)
		if len(kids) > 0 {
			var kidLines []string
			for _, k := range kids {
				kidLines = append(kidLines, fmt.Sprintf("- %s (`%s`)", k.Name, k.ID))
			}
			parts = append(parts, fmt.Sprintf("## Children\n\n%s\n", strings.Join(kidLines, "\n")))
		}
	}

	if len(concept.Diagrams) > 0 {
		parts = append(parts, fmt.Sprintf("## Diagrams\n\n%d diagram(s) available (press 'v' in progress view to see ASCII)\n", len(concept.Diagrams)))
	}

	if len(concept.Links) > 0 {
		var linkLines []string
		for _, link := range concept.Links {
			linkLines = append(linkLines, fmt.Sprintf("- [%s](%s)", link.Title, link.URL))
		}
		parts = append(parts, fmt.Sprintf("## Links\n\n%s\n", strings.Join(linkLines, "\n")))
	}

	return strings.Join(parts, "\n")
}

func renderConceptDetail(concept *parser.Concept, prog *progress.Progress, conceptID string, width int, conceptMap map[string]parser.Concept) []string {
	if concept == nil {
		return []string{lipgloss.NewStyle().Foreground(colorMuted).Render("  No concept data available.")}
	}

	md := buildConceptMarkdown(concept, conceptMap)
	rendered := render.RenderMarkdown(md, width-4)
	return strings.Split(rendered, "\n")
}

func renderUserNotes(prog *progress.Progress, conceptID, itemID, itemType string, width int) string {
	var notes []progress.Note

	if conceptID != "" {
		notes = append(notes, prog.NotesForConcept(conceptID)...)
	}
	if itemID != "" {
		switch itemType {
		case "card":
			notes = append(notes, prog.NotesForCard(itemID)...)
		case "quiz":
			notes = append(notes, prog.NotesForQuiz(itemID)...)
		}
	}

	if len(notes) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(
		fmt.Sprintf("  Notes (%d):", len(notes))))
	b.WriteString("\n")

	for i, note := range notes {
		link := "unlinked"
		if note.LinkedTo != nil {
			switch note.LinkedTo.Type {
			case "concept":
				link = "concept"
			case "card":
				link = "card"
			case "quiz":
				link = "quiz"
			}
		}
		truncated := runeTruncate(note.Content, width-8)
		b.WriteString(fmt.Sprintf("    %s %s\n",
			lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf("[%d·%s]", i+1, link)),
			lipgloss.NewStyle().Foreground(colorText).Render(truncated)))
	}

	return b.String()
}

// renderConceptInfo renders diagrams, links, and summaries for a concept
func renderConceptInfo(concept *parser.Concept, prog *progress.Progress, conceptID string, width int) string {
	if concept == nil {
		return ""
	}

	var parts []string

	// Content summary
	if concept.Summary != "" {
		parts = append(parts, fmt.Sprintf("  %s",
			lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("Content Summary:")))
		parts = append(parts, fmt.Sprintf("    %s", lipgloss.NewStyle().Foreground(colorText).Italic(true).Render(runeTruncate(concept.Summary, width-8))))
	}

	// User summaries
	userSummaries := prog.SummariesForConcept(conceptID)
	if len(userSummaries) > 0 {
		parts = append(parts, "")
		parts = append(parts, fmt.Sprintf("  %s",
			lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(fmt.Sprintf("User Summaries (%d):", len(userSummaries)))))
		for i, s := range userSummaries {
			parts = append(parts, fmt.Sprintf("    %d. %s", i+1, lipgloss.NewStyle().Foreground(colorText).Render(runeTruncate(s.Title, width-12))))
		}
	}

	// Diagrams
	if len(concept.Diagrams) > 0 {
		parts = append(parts, "")
		parts = append(parts, fmt.Sprintf("  %s",
			lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(fmt.Sprintf("Diagrams (%d):", len(concept.Diagrams)))))
		for _, d := range concept.Diagrams {
			label := d.ID
			if d.Label != "" {
				label = d.Label
			}
			sourceType := "mermaid"
			if d.File != "" {
				sourceType = "svg"
			}
			parts = append(parts, fmt.Sprintf("    • %s (%s)  v:ascii  d:open",
				lipgloss.NewStyle().Foreground(colorText).Render(label),
				lipgloss.NewStyle().Foreground(colorMuted).Render(sourceType)))
		}
	}

	// Links
	if len(concept.Links) > 0 {
		parts = append(parts, "")
		parts = append(parts, fmt.Sprintf("  %s",
			lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(fmt.Sprintf("Links (%d):", len(concept.Links)))))
		for _, link := range concept.Links {
			title := link.Title
			if title == "" {
				title = link.URL
			}
			parts = append(parts, fmt.Sprintf("    • %s  l:open",
				lipgloss.NewStyle().Foreground(colorText).Render(runeTruncate(title, width-16))))
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return "\n" + strings.Join(parts, "\n") + "\n"
}

// renderFullSummary renders the full content summary and user summaries (not truncated)
func renderFullSummary(concept *parser.Concept, prog *progress.Progress, conceptID string, width int) string {
	if concept == nil {
		return ""
	}

	var parts []string

	// Content summary
	if concept.Summary != "" {
		parts = append(parts, fmt.Sprintf("# %s — Summary\n", concept.Name))
		parts = append(parts, concept.Summary)
	}

	// User summaries
	userSummaries := prog.SummariesForConcept(conceptID)
	if len(userSummaries) > 0 {
		parts = append(parts, "")
		parts = append(parts, fmt.Sprintf("## User Summaries (%d)\n", len(userSummaries)))
		for i, s := range userSummaries {
			parts = append(parts, fmt.Sprintf("### %d. %s\n", i+1, s.Title))
			parts = append(parts, s.Content)
			parts = append(parts, "")
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, "\n")
}
