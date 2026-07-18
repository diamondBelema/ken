package study

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/diamondBelema/ken/internal/parser"
)

type QuizSession struct {
	Subject   string
	Questions []parser.Question
	Index     int
	Score     int
	Answered  int
}

func LoadQuizzes(subjectDir, subject string) (*QuizSession, error) {
	quizzesDir := filepath.Join(subjectDir, subject, "quizzes")
	entries, err := os.ReadDir(quizzesDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read quizzes directory: %w", err)
	}

	seen := make(map[string]string)
	var allQuestions []parser.Question

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		path := filepath.Join(quizzesDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}

		set, err := parser.ParseQuizSet(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", path, err)
		}

		for _, q := range set.Questions {
			if prev, exists := seen[q.ID]; exists {
				return nil, fmt.Errorf("duplicate question ID '%s' found in %s and %s", q.ID, prev, entry.Name())
			}
			seen[q.ID] = entry.Name()
			allQuestions = append(allQuestions, q)
		}
	}

	if len(allQuestions) == 0 {
		return nil, fmt.Errorf("no questions found in %s", quizzesDir)
	}

	return &QuizSession{
		Subject:   subject,
		Questions: allQuestions,
		Index:     0,
	}, nil
}

func (s *QuizSession) Current() parser.Question {
	return s.Questions[s.Index]
}

func (s *QuizSession) Advance() bool {
	s.Index++
	return s.Index < len(s.Questions)
}

func (s *QuizSession) IsFinished() bool {
	return s.Index >= len(s.Questions)
}

func (s *QuizSession) RecordAnswer(correct bool) {
	s.Answered++
	if correct {
		s.Score++
	}
}

func (s *QuizSession) Progress() (int, int) {
	return s.Index + 1, len(s.Questions)
}
