package groups

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type CourseGroup struct {
	ID       string   `yaml:"id"`
	Name     string   `yaml:"name"`
	Concepts []string `yaml:"concepts"`
}

type GroupsFile struct {
	Groups []CourseGroup `yaml:"groups"`
}

// Load reads groups.yaml from the subject directory.
// Returns nil (no error) if the file doesn't exist — subjects without groups are valid.
func Load(subjectDir string) ([]CourseGroup, error) {
	path := filepath.Join(subjectDir, "groups.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read groups.yaml: %w", err)
	}

	var gf GroupsFile
	if err := yaml.Unmarshal(data, &gf); err != nil {
		return nil, fmt.Errorf("failed to parse groups.yaml: %w", err)
	}

	return gf.Groups, nil
}

// ConceptInGroup checks if a concept ID belongs to a specific group.
func ConceptInGroup(group CourseGroup, conceptID string) bool {
	for _, id := range group.Concepts {
		if id == conceptID {
			return true
		}
	}
	return false
}

// ConceptsInAnyGroup returns the set of concept IDs that belong to at least one group.
func ConceptsInAnyGroup(groups []CourseGroup) map[string]bool {
	m := make(map[string]bool)
	for _, g := range groups {
		for _, id := range g.Concepts {
			m[id] = true
		}
	}
	return m
}
