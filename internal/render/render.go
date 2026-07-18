package render

import (
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

func RenderMarkdown(content string, width int) string {
	if content == "" {
		return ""
	}

	style := "dark"
	if !lipgloss.HasDarkBackground() {
		style = "light"
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(style),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return content
	}

	out, err := r.Render(content)
	if err != nil {
		return content
	}

	return out
}
