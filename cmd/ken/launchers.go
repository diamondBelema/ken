package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/diamondBelema/ken/internal/parser"
	"github.com/diamondBelema/ken/internal/progress"
	"github.com/diamondBelema/ken/internal/study"
	"github.com/diamondBelema/ken/internal/tui"
)

func runFlashcards(subject string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}
	subjectDir := filepath.Join(home, "Documents", "learn", "subjects")

	progPath, err := progress.SubjectPath(subject)
	if err != nil {
		return err
	}
	prog, err := progress.Load(progPath)
	if err != nil {
		return fmt.Errorf("failed to load progress: %w", err)
	}

	concepts, err := study.LoadConcepts(subjectDir, subject)
	if err != nil {
		return fmt.Errorf("failed to load concepts: %w", err)
	}
	progress.InitConcepts(prog, concepts)

	sess, err := study.LoadFlashcards(subjectDir, subject, prog)
	if err != nil {
		return err
	}

	m := tui.NewFlashcardModel(sess, prog, concepts)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	if err := progress.Save(progPath, prog); err != nil {
		return fmt.Errorf("failed to save progress: %w", err)
	}
	return nil
}

func runQuiz(subject string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}
	subjectDir := filepath.Join(home, "Documents", "learn", "subjects")

	progPath, err := progress.SubjectPath(subject)
	if err != nil {
		return err
	}
	prog, err := progress.Load(progPath)
	if err != nil {
		return fmt.Errorf("failed to load progress: %w", err)
	}

	concepts, err := study.LoadConcepts(subjectDir, subject)
	if err != nil {
		return fmt.Errorf("failed to load concepts: %w", err)
	}
	progress.InitConcepts(prog, concepts)

	sess, err := study.LoadQuizzes(subjectDir, subject, prog)
	if err != nil {
		return err
	}

	m := tui.NewQuizModel(sess, prog, concepts)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	if err := progress.Save(progPath, prog); err != nil {
		return fmt.Errorf("failed to save progress: %w", err)
	}
	return nil
}

func runNotes(subject string) error {
	progPath, err := progress.SubjectPath(subject)
	if err != nil {
		return err
	}
	prog, err := progress.Load(progPath)
	if err != nil {
		return fmt.Errorf("failed to load progress: %w", err)
	}

	home, _ := os.UserHomeDir()
	subjectsDir := filepath.Join(home, "Documents", "learn", "subjects")
	concepts, _ := study.LoadConcepts(subjectsDir, subject)

	m := tui.NewNotesModel(prog, concepts, subject)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	if err := progress.Save(progPath, prog); err != nil {
		return fmt.Errorf("failed to save progress: %w", err)
	}
	return nil
}

func runSummaries(subject string) error {
	progPath, err := progress.SubjectPath(subject)
	if err != nil {
		return err
	}
	prog, err := progress.Load(progPath)
	if err != nil {
		return fmt.Errorf("failed to load progress: %w", err)
	}

	home, _ := os.UserHomeDir()
	subjectsDir := filepath.Join(home, "Documents", "learn", "subjects")
	concepts, _ := study.LoadConcepts(subjectsDir, subject)

	m := tui.NewSummariesModel(prog, concepts, subject)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	if err := progress.Save(progPath, prog); err != nil {
		return fmt.Errorf("failed to save progress: %w", err)
	}
	return nil
}

func runRead(subject string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}
	subjectDir := filepath.Join(home, "Documents", "learn", "subjects")

	files, err := parser.LoadNoteFiles(subjectDir, subject)
	if err != nil {
		return fmt.Errorf("failed to load notes: %w", err)
	}

	progPath, err := progress.SubjectPath(subject)
	if err != nil {
		return err
	}
	prog, err := progress.Load(progPath)
	if err != nil {
		return fmt.Errorf("failed to load progress: %w", err)
	}

	concepts, err := study.LoadConcepts(subjectDir, subject)
	if err != nil {
		return fmt.Errorf("failed to load concepts: %w", err)
	}
	progress.InitConcepts(prog, concepts)

	m := tui.NewReadModel(files, prog, subject)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}

func runProgress(subject string) error {
	m := tui.NewProgressModel(subject)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}
