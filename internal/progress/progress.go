package progress

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/diamondBelema/ken/internal/parser"
)

type Progress struct {
	FormatVersion int                    `json:"format_version"`
	Concepts      map[string]ConceptState `json:"concepts"`
	Cards         map[string]CardState    `json:"cards"`
	Quizzes       map[string]QuizState    `json:"quizzes"`
}

type ConceptState struct {
	Confidence     float64 `json:"confidence"`
	LastReviewedAt *int64  `json:"last_reviewed_at"`
}

type CardState struct {
	Reviews   int    `json:"reviews"`
	LastGrade string `json:"last_grade"`
}

type QuizState struct {
	Attempts int `json:"attempts"`
	Correct  int `json:"correct"`
	Streak   int `json:"streak"`
}

func Load(path string) (*Progress, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Progress{
				FormatVersion: 1,
				Concepts:      make(map[string]ConceptState),
				Cards:         make(map[string]CardState),
				Quizzes:       make(map[string]QuizState),
			}, nil
		}
		return nil, fmt.Errorf("failed to read progress.json: %w", err)
	}

	var p Progress
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to parse progress.json: %w", err)
	}

	if p.Concepts == nil {
		p.Concepts = make(map[string]ConceptState)
	}
	if p.Cards == nil {
		p.Cards = make(map[string]CardState)
	}
	if p.Quizzes == nil {
		p.Quizzes = make(map[string]QuizState)
	}

	return &p, nil
}

func Save(path string, p *Progress) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal progress: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

func InitConcepts(p *Progress, concepts []parser.Concept) {
	for _, c := range concepts {
		if _, exists := p.Concepts[c.ID]; !exists {
			p.Concepts[c.ID] = ConceptState{
				Confidence:     0.5,
				LastReviewedAt: nil,
			}
		}
	}
}

func ConceptIDs(p *Progress) []string {
	ids := make([]string, 0, len(p.Concepts))
	for id := range p.Concepts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
