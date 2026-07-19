package main

import (
	"fmt"

	"github.com/diamondBelema/ken/internal/registry"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed packages",
	Long:  `List all packages installed from the Ken registry.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		state, err := registry.ListInstalled()
		if err != nil {
			return err
		}

		if len(state.Installed) == 0 {
			fmt.Println("No packages installed.")
			fmt.Println("\nUse  ken search <query>  to find packages")
			fmt.Println("Use  ken add <id>        to install a package")
			return nil
		}

		nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("141"))
		idStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))

		fmt.Printf("\nInstalled packages:\n\n")

		for id, pkg := range state.Installed {
			fmt.Printf("  %s  %s  %s\n",
				nameStyle.Render(id),
				idStyle.Render("v"+pkg.Version),
				dimStyle.Render("→ "+pkg.SubjectDir))
		}

		fmt.Printf("\n%d package%s installed\n", len(state.Installed), plural(len(state.Installed)))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
