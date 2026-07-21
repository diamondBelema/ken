package parser

import (
	"regexp"
	"strings"
)

type Document struct {
	Title    string
	Sections []Section
}

type Section struct {
	Heading   string
	Level     int
	ConceptID string
	Blocks    []Block
	Children  []Section
}

type Block struct {
	Type        string
	Content     string
	ConceptRefs []string
}

var conceptTagRe = regexp.MustCompile(`\[c-([a-zA-Z0-9_-]+)\]`)

func ParseTaggedNote(content string) Document {
	lines := strings.Split(content, "\n")
	var sections []Section
	var currentSection *Section
	var currentBlocks []Block

	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			heading, level, conceptID := parseHeading(line)
			if level == 1 {
				if currentSection != nil {
					currentSection.Blocks = currentBlocks
					sections = append(sections, *currentSection)
					currentBlocks = nil
				}
				currentSection = &Section{
					Heading:   heading,
					Level:     level,
					ConceptID: conceptID,
				}
			} else if currentSection != nil {
				if currentSection.Children == nil {
					currentSection.Blocks = currentBlocks
					sections = append(sections, *currentSection)
					currentBlocks = nil
					currentSection = &Section{
						Heading:   heading,
						Level:     level,
						ConceptID: conceptID,
					}
				} else {
					child := Section{
						Heading:   heading,
						Level:     level,
						ConceptID: conceptID,
					}
					currentSection.Children = append(currentSection.Children, child)
				}
			}
		} else if strings.TrimSpace(line) != "" {
			conceptRefs := extractConceptRefs(line)
			blockType := classifyBlock(line)
			currentBlocks = append(currentBlocks, Block{
				Type:        blockType,
				Content:     line,
				ConceptRefs: conceptRefs,
			})
		}
	}

	if currentSection != nil {
		currentSection.Blocks = currentBlocks
		sections = append(sections, *currentSection)
	}

	title := ""
	if len(sections) > 0 {
		title = sections[0].Heading
	}

	return Document{
		Title:    title,
		Sections: sections,
	}
}

func parseHeading(line string) (string, int, string) {
	level := 0
	for i := 0; i < len(line) && line[i] == '#'; i++ {
		level++
	}

	heading := strings.TrimSpace(line[level:])
	conceptID := ""

	if matches := conceptTagRe.FindStringSubmatch(heading); matches != nil {
		conceptID = matches[1]
		heading = strings.TrimSpace(conceptTagRe.ReplaceAllString(heading, ""))
	}

	return heading, level, conceptID
}

func extractConceptRefs(line string) []string {
	var refs []string
	matches := conceptTagRe.FindAllStringSubmatch(line, -1)
	for _, m := range matches {
		refs = append(refs, m[1])
	}
	return refs
}

func classifyBlock(line string) string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
		return "bullet"
	}
	if strings.HasPrefix(trimmed, "```") {
		return "code"
	}
	if strings.HasPrefix(trimmed, "|") {
		return "table"
	}
	if strings.HasPrefix(trimmed, "$$") || strings.HasPrefix(trimmed, "\\(") {
		return "formula"
	}
	return "paragraph"
}
