package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/diamondBelema/ken/internal/progress"
	"github.com/diamondBelema/ken/internal/study"
	"github.com/diamondBelema/ken/internal/tui"
	"github.com/spf13/cobra"
)

var flashcardsCmd = &cobra.Command{
	Use:   "flashcards <subject>",
	Short: "Study flashcards for a subject",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		subject := args[0]

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

		sess, err := study.LoadFlashcards(subjectDir, subject)
		if err != nil {
			return err
		}

		m := tui.NewFlashcardModel(sess, prog)
		p := tea.NewProgram(m)
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}

		if err := progress.Save(progPath, prog); err != nil {
			return fmt.Errorf("failed to save progress: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(flashcardsCmd)
}
