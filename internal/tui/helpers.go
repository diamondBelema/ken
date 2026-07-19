package tui

import (
	"fmt"
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

func buildConceptMarkdown(concept *parser.Concept) string {
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

func renderConceptDetail(concept *parser.Concept, prog *progress.Progress, conceptID string, width int) []string {
	if concept == nil {
		return []string{lipgloss.NewStyle().Foreground(colorMuted).Render("  No concept data available.")}
	}

	md := buildConceptMarkdown(concept)
	rendered := render.RenderMarkdown(md, width-4)
	return strings.Split(rendered, "\n")
}
