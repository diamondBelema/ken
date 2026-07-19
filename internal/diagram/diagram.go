package diagram

import (
	"fmt"
	"os"
	"path/filepath"

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

// ResolveDiagramSource resolves the diagram source from inline Source or external File.
// If Source is provided, it's returned directly (inline mermaid).
// If File is provided and Source is empty, the file is loaded from disk.
// subjectDir is the root subject directory (e.g., ~/Documents/learn/subjects/nucleic-acid)
func ResolveDiagramSource(source, file, subjectDir string) (string, error) {
	if source != "" {
		return source, nil
	}
	if file == "" {
		return "", nil
	}
	// Resolve relative path from subject directory
	absPath := filepath.Join(subjectDir, file)
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read diagram file %s: %w", absPath, err)
	}
	return string(data), nil
}

// IsSVGFile checks if a file path points to an SVG file
func IsSVGFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".svg" || ext == ".SVG"
}
