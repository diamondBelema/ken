package parser

import (
	"fmt"
	"strconv"
	"strings"
)

type QuizSet struct {
	Set       string
	Questions []Question
}

type Question struct {
	ID          string
	ConceptID   string
	Type        string
	Question    string
	Options     []string
	Answer      interface{}
	Explanation string
}

func ParseQuizSet(data []byte) (QuizSet, error) {
	raw, body, err := SplitFrontmatter(data)
	if err != nil {
		return QuizSet{}, err
	}

	typeStr, _ := raw["type"].(string)
	if typeStr != "quiz_set" {
		return QuizSet{}, fmt.Errorf("expected type 'quiz_set', got '%s'", typeStr)
	}

	setName, _ := raw["set"].(string)

	questionsRaw, ok := raw["questions"].([]interface{})
	if !ok {
		return QuizSet{}, fmt.Errorf("missing or invalid 'questions' field")
	}

	explanations := parseExplanationSections(body)

	var questions []Question
	for _, q := range questionsRaw {
		qm, ok := q.(map[string]interface{})
		if !ok {
			continue
		}

		id, _ := qm["id"].(string)
		if id == "" {
			continue
		}

		qType, _ := qm["type"].(string)
		switch qType {
		case "mcq", "true_false", "fill_blank":
			// valid
		default:
			fmt.Printf("warning: skipping question %s with unknown type %q\n", id, qType)
			continue
		}

		questionText, _ := qm["question"].(string)
		conceptID, _ := qm["concept_id"].(string)

		var options []string
		if optsRaw, ok := qm["options"].([]interface{}); ok {
			for _, o := range optsRaw {
				switch v := o.(type) {
				case string:
					options = append(options, v)
				case int:
					options = append(options, strconv.Itoa(v))
				case float64:
					options = append(options, strconv.FormatFloat(v, 'f', -1, 64))
				case bool:
					if v {
						options = append(options, "true")
					} else {
						options = append(options, "false")
					}
				}
			}
		}

		var answer interface{}
		if a, exists := qm["answer"]; exists {
			answer = a
		} else {
			fmt.Printf("warning: skipping question %s with no answer field\n", id)
			continue
		}

		questions = append(questions, Question{
			ID:          id,
			ConceptID:   conceptID,
			Type:        qType,
			Question:    questionText,
			Options:     options,
			Answer:      answer,
			Explanation: explanations[id],
		})
	}

	return QuizSet{Set: setName, Questions: questions}, nil
}

func parseExplanationSections(body string) map[string]string {
	explanations := make(map[string]string)
	lines := strings.Split(body, "\n")

	var currentID string
	var currentExpl strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			if currentID != "" {
				explanations[currentID] = strings.TrimSpace(currentExpl.String())
				currentExpl.Reset()
			}
			currentID = strings.TrimPrefix(line, "## ")
			currentID = strings.TrimSpace(currentID)
		} else if currentID != "" {
			currentExpl.WriteString(line)
			currentExpl.WriteString("\n")
		}
	}

	if currentID != "" {
		explanations[currentID] = strings.TrimSpace(currentExpl.String())
	}

	return explanations
}
