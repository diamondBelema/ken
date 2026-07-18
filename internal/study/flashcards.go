package study

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/diamondBelema/ken/internal/parser"
	"github.com/diamondBelema/ken/internal/progress"
)

type FlashcardSession struct {
	Subject string
	Cards   []parser.Flashcard
	Index   int
}

func LoadFlashcards(subjectDir, subject string, prog *progress.Progress) (*FlashcardSession, error) {
	flashcardsDir := filepath.Join(subjectDir, subject, "flashcards")
	entries, err := os.ReadDir(flashcardsDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read flashcards directory: %w", err)
	}

	seen := make(map[string]string) // id → filename for collision detection
	var allCards []parser.Flashcard

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		path := filepath.Join(flashcardsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}

		set, err := parser.ParseFlashcardSet(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", path, err)
		}

		for _, card := range set.Cards {
			if prev, exists := seen[card.ID]; exists {
				return nil, fmt.Errorf("duplicate card ID '%s' found in %s and %s", card.ID, prev, entry.Name())
			}
			seen[card.ID] = entry.Name()
			allCards = append(allCards, card)
		}
	}

	if len(allCards) == 0 {
		return nil, fmt.Errorf("no flashcards found in %s", flashcardsDir)
	}

	if prog != nil {
		sort.Slice(allCards, func(i, j int) bool {
			ci := conceptConfidence(prog, allCards[i].ConceptID)
			cj := conceptConfidence(prog, allCards[j].ConceptID)
			return ci < cj
		})
	}

	return &FlashcardSession{
		Subject: subject,
		Cards:   allCards,
		Index:   0,
	}, nil
}

func conceptConfidence(prog *progress.Progress, conceptID string) float64 {
	if conceptID == "" {
		return 1.0
	}
	cs, ok := prog.Concepts[conceptID]
	if !ok {
		return 0.5
	}
	return cs.Confidence
}

func (s *FlashcardSession) Current() parser.Flashcard {
	return s.Cards[s.Index]
}

func (s *FlashcardSession) Advance() bool {
	s.Index++
	return s.Index < len(s.Cards)
}

func (s *FlashcardSession) IsFinished() bool {
	return s.Index >= len(s.Cards)
}

func (s *FlashcardSession) Progress() (int, int) {
	return s.Index + 1, len(s.Cards)
}
