package study

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/diamondBelema/ken/internal/parser"
)

func LoadConcepts(subjectDir, subject string) ([]parser.Concept, error) {
	conceptsDir := filepath.Join(subjectDir, subject, "concepts")
	entries, err := os.ReadDir(conceptsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot read concepts directory: %w", err)
	}

	seen := make(map[string]string)
	var allConcepts []parser.Concept

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		path := filepath.Join(conceptsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}

		set, err := parser.ParseConceptSet(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", path, err)
		}

		for _, c := range set.Concepts {
			if prev, exists := seen[c.ID]; exists {
				return nil, fmt.Errorf("duplicate concept ID '%s' found in %s and %s", c.ID, prev, entry.Name())
			}
			seen[c.ID] = entry.Name()
			allConcepts = append(allConcepts, c)
		}
	}

	return allConcepts, nil
}
