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

var reflectCmd = &cobra.Command{
	Use:   "reflect <subject> [concept-id]",
	Short: "Reflection layer — type your explanation, then see the canonical answer",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		subject := args[0]
		var conceptID string
		if len(args) > 1 {
			conceptID = args[1]
		}

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

		m := tui.NewReflectModel(subject, concepts, prog, conceptID)
		p := tea.NewProgram(m, tea.WithAltScreen())
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
	rootCmd.AddCommand(reflectCmd)
}
