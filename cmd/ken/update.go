package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/diamondBelema/ken/internal/registry"
	"github.com/spf13/cobra"
)

var updateAll bool

var updateCmd = &cobra.Command{
	Use:   "update [package]",
	Short: "Update installed package(s) to latest version",
	Long: `Update one or all installed packages to the latest version from the registry.

Downloads new content and replaces existing files. Your notes and progress
are stored separately and are never touched.

Examples:
  ken update diamond/nucleic-acid    # update one package
  ken update --all                   # update all installed packages`,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		subjectsDir := filepath.Join(home, "Documents", "learn", "subjects")

		if updateAll {
			return runUpdateAll(subjectsDir)
		}

		if len(args) == 0 {
			return fmt.Errorf("specify a package or use --all")
		}

		owner, repo, _ := parsePackageRef(args[0])
		pkgID := owner + "/" + repo

		manifest, msg, err := registry.UpdatePackage(pkgID, subjectsDir)
		if err != nil {
			return err
		}

		if msg != "" {
			fmt.Println(msg)
		}

		fmt.Printf("\nUpdated %s v%s\n", manifest.Name, manifest.Version)
		fmt.Printf("Subjects: %s\n", strings.Join(manifest.Subjects, ", "))
		return nil
	},
}

func runUpdateAll(subjectsDir string) error {
	results, err := registry.UpdateAll(subjectsDir)
	if err != nil {
		return err
	}

	fmt.Println()
	updated := 0
	upToDate := 0
	errors := 0

	for _, r := range results {
		switch {
		case r.Status == "up to date":
			fmt.Printf("  ✓ %s v%s — up to date\n", r.PkgID, r.OldVersion)
			upToDate++
		case r.Status == "updated":
			fmt.Printf("  ↑ %s v%s → v%s\n", r.PkgID, r.OldVersion, r.NewVersion)
			updated++
		default:
			fmt.Printf("  ! %s — %s\n", r.PkgID, r.Status)
			errors++
		}
	}

	fmt.Printf("\n%d updated, %d up to date", updated, upToDate)
	if errors > 0 {
		fmt.Printf(", %d errors", errors)
	}
	fmt.Println()
	return nil
}

func init() {
	updateCmd.Flags().BoolVar(&updateAll, "all", false, "update all installed packages")
	rootCmd.AddCommand(updateCmd)
}
