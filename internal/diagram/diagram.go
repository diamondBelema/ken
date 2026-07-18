package diagram

import (
	"fmt"
	"os"

	mermaigo "github.com/yashikota/mermaigo/pkg/mermaid"
	gomermaid "github.com/zkrebbekx/go-mermaid"
)

func RenderASCII(source string) (string, error) {
	out, err := mermaigo.RenderText(source, &mermaigo.TextRenderOptions{UseAscii: false})
	if err != nil {
		return "", fmt.Errorf("failed to render mermaid to ASCII: %w", err)
	}
	return out, nil
}

func RenderSVG(source string) ([]byte, error) {
	out, err := gomermaid.Render(source, gomermaid.WithTheme(gomermaid.Dark))
	if err != nil {
		return nil, fmt.Errorf("failed to render mermaid to SVG: %w", err)
	}
	return out, nil
}

func RenderSVGToFile(source, path string) error {
	svg, err := RenderSVG(source)
	if err != nil {
		return err
	}
	return os.WriteFile(path, svg, 0644)
}

func LoadSourceFromFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read diagram file %s: %w", path, err)
	}
	return string(data), nil
}
