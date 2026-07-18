package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/diamondBelema/ken/internal/tui"
	"github.com/spf13/cobra"
)

var progressCmd = &cobra.Command{
	Use:   "progress [subject]",
	Short: "View concept-level confidence breakdown",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		subject := ""
		if len(args) > 0 {
			subject = args[0]
		}

		m := tui.NewProgressModel(subject)
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(progressCmd)
}
