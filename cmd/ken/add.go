package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/diamondBelema/ken/internal/registry"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <author>/<package>[@version]",
	Short: "Install a package from the registry",
	Long: `Install a Ken package from the registry. Downloads the package and copies
content to your subjects directory.

Examples:
  ken add diamond/anatomy
  ken add diamond/anatomy@1.0.0
  ken add diamond/nucleic-acid@latest`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		owner, repo, version := parsePackageRef(args[0])

		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		subjectsDir := filepath.Join(home, "Documents", "learn", "subjects")

		if err := os.MkdirAll(subjectsDir, 0755); err != nil {
			return fmt.Errorf("failed to create subjects directory: %w", err)
		}

		manifest, err := registry.InstallPackage(owner, repo, version, subjectsDir)
		if err != nil {
			return err
		}

		fmt.Printf("\nInstalled %s v%s\n", manifest.Name, manifest.Version)
		fmt.Printf("Subjects: %s\n", strings.Join(manifest.Subjects, ", "))
		fmt.Printf("Concepts: %d | Flashcards: %d\n", manifest.Concepts, manifest.Flashcards)
		return nil
	},
}

func parsePackageRef(ref string) (owner, repo, version string) {
	version = "latest"

	if idx := strings.Index(ref, "@"); idx != -1 {
		version = ref[idx+1:]
		ref = ref[:idx]
	}

	parts := strings.SplitN(ref, "/", 2)
	if len(parts) == 2 {
		owner = parts[0]
		repo = parts[1]
	} else {
		owner = ""
		repo = parts[0]
	}

	return owner, repo, version
}

func init() {
	rootCmd.AddCommand(addCmd)
}
