package main

import (
	"fmt"

	"github.com/diamondBelema/ken/internal/registry"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search the registry for packages",
	Long:  `Search the Ken registry for packages matching the query. Searches names, descriptions, IDs, and tags.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]

		fmt.Println("Fetching registry...")
		index, err := registry.FetchIndex()
		if err != nil {
			return err
		}

		results := registry.SearchIndex(index, query)
		if len(results) == 0 {
			fmt.Printf("No packages found matching %q\n", query)
			return nil
		}

		nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("141"))
		idStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
		tagStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
		descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

		fmt.Printf("\nFound %d package%s:\n\n", len(results), plural(len(results)))

		for _, pkg := range results {
			fmt.Printf("  %s  %s\n", nameStyle.Render(pkg.Name), idStyle.Render(pkg.ID))
			if pkg.Description != "" {
				fmt.Printf("    %s\n", descStyle.Render(truncate(pkg.Description, 60)))
			}
			if len(pkg.Tags) > 0 {
				fmt.Printf("    %s\n", tagStyle.Render(joinTags(pkg.Tags)))
			}
			fmt.Println()
		}

		fmt.Println("Use  ken add <id>  to install a package")
		return nil
	},
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func joinTags(tags []string) string {
	result := ""
	for i, tag := range tags {
		if i > 0 {
			result += " · "
		}
		result += tag
	}
	return result
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
