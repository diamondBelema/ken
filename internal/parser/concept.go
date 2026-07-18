package parser

import (
	"fmt"
	"strings"
)

type Concept struct {
	ID          string
	Name        string
	ParentID    string
	Description string
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
	if typeStr != "concept_set" {
		return ConceptSet{}, fmt.Errorf("expected type 'concept_set', got '%s'", typeStr)
	}

	setName, _ := raw["set"].(string)

	conceptsRaw, ok := raw["concepts"].([]interface{})
	if !ok {
		return ConceptSet{}, fmt.Errorf("missing or invalid 'concepts' field")
	}

	descriptions := parseDescriptionSections(body)

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

		concepts = append(concepts, Concept{
			ID:          id,
			Name:        name,
			ParentID:    parentID,
			Description: descriptions[id],
		})
	}

	return ConceptSet{Set: setName, Concepts: concepts}, nil
}

func parseDescriptionSections(body string) map[string]string {
	descriptions := make(map[string]string)
	lines := strings.Split(body, "\n")

	var currentID string
	var currentDesc strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			if currentID != "" {
				descriptions[currentID] = strings.TrimSpace(currentDesc.String())
				currentDesc.Reset()
			}
			currentID = strings.TrimPrefix(line, "## ")
			currentID = strings.TrimSpace(currentID)
		} else if currentID != "" {
			currentDesc.WriteString(line)
			currentDesc.WriteString("\n")
		}
	}

	if currentID != "" {
		descriptions[currentID] = strings.TrimSpace(currentDesc.String())
	}

	return descriptions
}
