package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/diamondBelema/ken/internal/tui"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "View detailed stats and confidence trends",
	RunE: func(cmd *cobra.Command, args []string) error {
		m := tui.NewStatsModel()
		p := tea.NewProgram(m)
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
}
