package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/diamondBelema/ken/internal/discovery"
	"github.com/spf13/cobra"
)

var subjectsCmd = &cobra.Command{
	Use:   "subjects",
	Short: "List subjects with concept/card/quiz counts",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}

		subjectsDir := filepath.Join(home, "Documents", "learn", "subjects")
		subjects, err := discovery.Discover(subjectsDir)
		if err != nil {
			return err
		}

		if len(subjects) == 0 {
			fmt.Println("No subjects found.")
			return nil
		}

		for _, s := range subjects {
			fmt.Printf("%s: %d concepts, %d flashcards, %d quizzes\n",
				s.Name, s.ConceptFiles, s.FlashcardFiles, s.QuizFiles)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(subjectsCmd)
}
