package render

import (
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
)

func RenderMarkdown(content string, width int) string {
	if content == "" {
		return ""
	}

	styleConfig := styles.DarkStyleConfig

	bright := "252"
	heading := "39"
	muted := "247"
	code := "112"

	styleConfig.Document = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: &bright,
		},
	}
	styleConfig.Heading = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: &heading,
			Bold:  boolPtr(true),
		},
	}
	styleConfig.H1 = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: &heading,
			Bold:  boolPtr(true),
		},
	}
	styleConfig.H2 = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: &heading,
			Bold:  boolPtr(true),
		},
	}
	styleConfig.H3 = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: &heading,
			Bold:  boolPtr(true),
		},
	}
	styleConfig.BlockQuote = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: &muted,
		},
	}
	styleConfig.CodeBlock = ansi.StyleCodeBlock{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: &code,
			},
		},
	}
	styleConfig.Emph = ansi.StylePrimitive{
		Italic: boolPtr(true),
		Color:  &bright,
	}
	styleConfig.Strong = ansi.StylePrimitive{
		Bold:  boolPtr(true),
		Color: &bright,
	}
	styleConfig.Link = ansi.StylePrimitive{
		Color:     &heading,
		Underline: boolPtr(true),
	}
	styleConfig.Item = ansi.StylePrimitive{
		Color: &bright,
	}
	styleConfig.Paragraph = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: &bright,
		},
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(width),
		glamour.WithStyles(styleConfig),
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

func boolPtr(b bool) *bool {
	return &b
}
