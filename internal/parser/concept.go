package parser

import (
	"fmt"
	"strings"
)

type Diagram struct {
	ID     string
	Label  string
	Source string
	File   string
}

type Concept struct {
	ID          string
	Name        string
	ParentID    string
	Description string
	Summary     string
	Diagrams    []Diagram
}

type ConceptSet struct {
	Set      string
	Concepts []Concept
}

func ParseConceptSet(data []byte) (ConceptSet, error) {
	raw, body, err := SplitFrontmatter(data)
	if err != nil {
		return ConceptSet{}, err
	}

	typeStr, _ := raw["type"].(string)
	setName, _ := raw["set"].(string)

	if typeStr != "concept_set" {
		if title, ok := raw["title"].(string); ok {
			setName = title
		} else if name, ok := raw["name"].(string); ok {
			setName = name
		}
	}

	conceptsRaw, ok := raw["concepts"].([]interface{})
	if !ok {
		return ConceptSet{}, fmt.Errorf("missing or invalid 'concepts' field")
	}

	sections := parseSections(body)

	var concepts []Concept
	for i, c := range conceptsRaw {
		cm, ok := c.(map[string]interface{})
		if !ok {
			return ConceptSet{}, fmt.Errorf("concept at index %d is not a valid map", i)
		}

		id, _ := cm["id"].(string)
		if id == "" {
			return ConceptSet{}, fmt.Errorf("concept at index %d missing required 'id' field", i)
		}

		name, _ := cm["name"].(string)
		parentID, _ := cm["parent_id"].(string)
		if parentID == "" {
			parentID, _ = cm["parent"].(string)
		}

		var diagrams []Diagram
		if diagsRaw, ok := cm["diagrams"].([]interface{}); ok {
			for _, d := range diagsRaw {
				dm, ok := d.(map[string]interface{})
				if !ok {
					continue
				}
				diag := Diagram{
					ID:    getString(dm, "id"),
					Label: getString(dm, "label"),
					Source: getString(dm, "source"),
					File:   getString(dm, "file"),
				}
				diagrams = append(diagrams, diag)
			}
		}

		concepts = append(concepts, Concept{
			ID:          id,
			Name:        name,
			ParentID:    parentID,
			Description: sections[id],
			Summary:     sections[id+":summary"],
			Diagrams:    diagrams,
		})
	}

	return ConceptSet{Set: setName, Concepts: concepts}, nil
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func parseSections(body string) map[string]string {
	sections := make(map[string]string)
	lines := strings.Split(body, "\n")

	var currentID string
	var currentContent strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			if currentID != "" {
				sections[currentID] = strings.TrimSpace(currentContent.String())
				currentContent.Reset()
			}
			currentID = strings.TrimPrefix(line, "## ")
			currentID = strings.TrimSpace(currentID)
		} else if currentID != "" {
			currentContent.WriteString(line)
			currentContent.WriteString("\n")
		}
	}

	if currentID != "" {
		sections[currentID] = strings.TrimSpace(currentContent.String())
	}

	return sections
}
