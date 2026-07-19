package parser

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Manifest struct {
	ID           string   `yaml:"id"`
	Name         string   `yaml:"name"`
	Version      string   `yaml:"version"`
	Author       string   `yaml:"author"`
	Description  string   `yaml:"description"`
	License      string   `yaml:"license"`
	Subjects     []string `yaml:"subjects"`
	Concepts     int      `yaml:"concepts"`
	Flashcards   int      `yaml:"flashcards"`
	Tags         []string `yaml:"tags"`
	Dependencies []string `yaml:"dependencies"`
	Repository   string   `yaml:"repository"`
}

func ParseManifest(data []byte) (Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("failed to parse ken.yaml: %w", err)
	}
	if err := ValidateManifest(m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

func ValidateManifest(m Manifest) error {
	if m.ID == "" {
		return fmt.Errorf("ken.yaml: missing required 'id' field")
	}
	if m.Name == "" {
		return fmt.Errorf("ken.yaml: missing required 'name' field")
	}
	if m.Version == "" {
		return fmt.Errorf("ken.yaml: missing required 'version' field")
	}
	if m.Author == "" {
		return fmt.Errorf("ken.yaml: missing required 'author' field")
	}
	if len(m.Subjects) == 0 {
		return fmt.Errorf("ken.yaml: missing required 'subjects' field (at least one subject)")
	}
	return nil
}

func LoadManifest(dir string) (Manifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, "ken.yaml"))
	if err != nil {
		return Manifest{}, fmt.Errorf("no ken.yaml found in %s", dir)
	}
	return ParseManifest(data)
}

func SaveManifest(dir string, m Manifest) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	header := []byte("# Ken package manifest\n")
	return os.WriteFile(filepath.Join(dir, "ken.yaml"), append(header, data...), 0644)
}
