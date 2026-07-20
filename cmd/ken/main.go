package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/diamondBelema/ken/internal/tui"
	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "ken",
	Short: "Terminal-based spaced-repetition study harness",
	RunE: func(cmd *cobra.Command, args []string) error {
		for {
			m := tui.NewDashboardModel(version)
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
			var launchErr error
			switch result.Action {
			case "flashcards":
				launchErr = runFlashcards(result.Subject)
			case "quiz":
				launchErr = runQuiz(result.Subject)
			case "notes":
				launchErr = runNotes(result.Subject)
			case "summaries":
				launchErr = runSummaries(result.Subject)
			case "read":
				launchErr = runRead(result.Subject)
			case "progress":
				launchErr = runProgress(result.Subject)
			default:
				return nil
			}
			if launchErr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", launchErr)
			}
		}
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
