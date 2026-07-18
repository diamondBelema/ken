package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/diamondBelema/ken/internal/parser"
	"github.com/diamondBelema/ken/internal/tui"
	"github.com/spf13/cobra"
)

var readCmd = &cobra.Command{
	Use:   "read <subject>",
	Short: "Read lecture notes and content for a subject",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		subject := args[0]

		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}

		subjectDir := filepath.Join(home, "Documents", "learn", "subjects")

		files, err := parser.LoadNoteFiles(subjectDir, subject)
		if err != nil {
			return fmt.Errorf("failed to load notes: %w", err)
		}

		m := tui.NewReadModel(files)
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(readCmd)
}
