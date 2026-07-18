package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/diamondBelema/ken/internal/progress"
	"github.com/diamondBelema/ken/internal/tui"
	"github.com/spf13/cobra"
)

var summariesCmd = &cobra.Command{
	Use:   "summaries <subject>",
	Short: "View and manage summaries for a subject",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		subject := args[0]

		progPath, err := progress.SubjectPath(subject)
		if err != nil {
			return err
		}

		prog, err := progress.Load(progPath)
		if err != nil {
			return fmt.Errorf("failed to load progress: %w", err)
		}

		m := tui.NewSummariesModel(prog, subject)
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
	rootCmd.AddCommand(summariesCmd)
}
