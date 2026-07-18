package discovery

import (
	"fmt"
	"os"
	"path/filepath"
)

type SubjectInfo struct {
	Name           string
	ConceptFiles   int
	FlashcardFiles int
	QuizFiles      int
}

func Discover(subjectsDir string) ([]SubjectInfo, error) {
	entries, err := os.ReadDir(subjectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("learn directory not found at %s — create it and add subject folders", subjectsDir)
		}
		return nil, fmt.Errorf("failed to read subjects directory: %w", err)
	}

	var subjects []SubjectInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		info := SubjectInfo{Name: entry.Name()}
		subjectDir := filepath.Join(subjectsDir, entry.Name())

		info.ConceptFiles = countMDFiles(filepath.Join(subjectDir, "concepts"))
		info.FlashcardFiles = countMDFiles(filepath.Join(subjectDir, "flashcards"))
		info.QuizFiles = countMDFiles(filepath.Join(subjectDir, "quizzes"))

		subjects = append(subjects, info)
	}

	return subjects, nil
}

func countMDFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
			count++
		}
	}
	return count
}
