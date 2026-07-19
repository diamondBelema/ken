package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/diamondBelema/ken/internal/registry"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <author>/<package>",
	Short: "Remove an installed package",
	Long: `Remove a package installed from the registry. Deletes the subject content
directory and removes the package from the installed list.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pkgID := args[0]

		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		subjectsDir := filepath.Join(home, "Documents", "learn", "subjects")

		if err := registry.RemovePackage(pkgID, subjectsDir); err != nil {
			return err
		}

		fmt.Printf("Removed %s\n", pkgID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
