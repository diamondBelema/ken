package parser

import (
	"fmt"
	"strings"
)

type FlashcardSet struct {
	Set    string
	Source string
	Cards  []Flashcard
}

type Flashcard struct {
	ID         string
	ConceptID  string
	Front      string
	Back       string
	Tags       []string
	Notes      string
}

func ParseFlashcardSet(data []byte) (FlashcardSet, error) {
	raw, body, err := SplitFrontmatter(data)
	if err != nil {
		return FlashcardSet{}, err
	}

	typeStr, _ := raw["type"].(string)
	if typeStr != "flashcard_set" {
		return FlashcardSet{}, fmt.Errorf("expected type 'flashcard_set', got '%s'", typeStr)
	}

	setName, _ := raw["set"].(string)
	source, _ := raw["source"].(string)

	cardsRaw, ok := raw["cards"].([]interface{})
	if !ok {
		return FlashcardSet{}, fmt.Errorf("missing or invalid 'cards' field")
	}

	notes := parseNotesSections(body)

	var cards []Flashcard
	for i, c := range cardsRaw {
		cm, ok := c.(map[string]interface{})
		if !ok {
			return FlashcardSet{}, fmt.Errorf("card at index %d is not a valid map", i)
		}

		id, _ := cm["id"].(string)
		if id == "" {
			return FlashcardSet{}, fmt.Errorf("card at index %d missing required 'id' field", i)
		}

		conceptID, _ := cm["concept_id"].(string)
		front, _ := cm["front"].(string)
		back, _ := cm["back"].(string)

		var tags []string
		if tagsRaw, ok := cm["tags"].([]interface{}); ok {
			for _, t := range tagsRaw {
				if s, ok := t.(string); ok {
					tags = append(tags, s)
				}
			}
		}

		cards = append(cards, Flashcard{
			ID:        id,
			ConceptID: conceptID,
			Front:     front,
			Back:      back,
			Tags:      tags,
			Notes:     notes[id],
		})
	}

	return FlashcardSet{Set: setName, Source: source, Cards: cards}, nil
}

func parseNotesSections(body string) map[string]string {
	notes := make(map[string]string)
	lines := strings.Split(body, "\n")

	var currentID string
	var currentNote strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "## Notes: ") {
			if currentID != "" {
				notes[currentID] = strings.TrimSpace(currentNote.String())
				currentNote.Reset()
			}
			currentID = strings.TrimPrefix(line, "## Notes: ")
			currentID = strings.TrimSpace(currentID)
		} else if currentID != "" {
			currentNote.WriteString(line)
			currentNote.WriteString("\n")
		}
	}

	if currentID != "" {
		notes[currentID] = strings.TrimSpace(currentNote.String())
	}

	return notes
}
