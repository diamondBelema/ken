package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/diamondBelema/ken/internal/tui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ken",
	Short: "Terminal-based spaced-repetition study harness",
	RunE: func(cmd *cobra.Command, args []string) error {
		for {
			m := tui.NewDashboardModel()
			p := tea.NewProgram(m, tea.WithAltScreen())
			finalModel, err := p.Run()
			if err != nil {
				return fmt.Errorf("TUI error: %w", err)
			}

			dm, ok := finalModel.(tui.DashboardModel)
			if !ok || dm.Result().Action == "" {
				return nil
			}

			result := dm.Result()
			switch result.Action {
			case "flashcards":
				flashcardsCmd.SetArgs([]string{result.Subject})
				if err := flashcardsCmd.Execute(); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				}
			case "quiz":
				quizCmd.SetArgs([]string{result.Subject})
				if err := quizCmd.Execute(); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				}
			case "notes":
				notesCmd.SetArgs([]string{result.Subject})
				if err := notesCmd.Execute(); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				}
			case "summaries":
				summariesCmd.SetArgs([]string{result.Subject})
				if err := summariesCmd.Execute(); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				}
			case "read":
				readCmd.SetArgs([]string{result.Subject})
				if err := readCmd.Execute(); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				}
			case "progress":
				progressCmd.SetArgs([]string{result.Subject})
				if err := progressCmd.Execute(); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				}
			default:
				return nil
			}
		}
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
