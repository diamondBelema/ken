package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/diamondBelema/ken/internal/discovery"
	"github.com/diamondBelema/ken/internal/lint"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var lintJSON bool

var lintCmd = &cobra.Command{
	Use:   "lint [subject]",
	Short: "Validate content files and report issues",
	Long: `Lint walks every content file in a subject (or all subjects), checks for
parse errors, duplicate IDs, broken references, and content mistakes, then
prints a grouped report. Exits non-zero if any error-level issue was found.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		subjectsDir := filepath.Join(home, "Documents", "learn", "subjects")

		if len(args) > 0 {
			return lintSubject(subjectsDir, args[0])
		}

		subjects, err := discovery.Discover(subjectsDir)
		if err != nil {
			return err
		}
		if len(subjects) == 0 {
			fmt.Println("No subjects found.")
			return nil
		}

		var reports []lint.Report
		for _, s := range subjects {
			report, err := lint.LintSubject(subjectsDir, s.Name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", s.Name, err)
				continue
			}
			reports = append(reports, report)
		}

		if lintJSON {
			return printJSON(reports)
		}

		hasErrors := false
		for _, r := range reports {
			printReport(r)
			if r.HasErrors() {
				hasErrors = true
			}
		}

		printSummary(reports)
		if hasErrors {
			os.Exit(1)
		}
		return nil
	},
}

func lintSubject(subjectsDir, subject string) error {
	report, err := lint.LintSubject(subjectsDir, subject)
	if err != nil {
		return err
	}

	if lintJSON {
		return printJSON([]lint.Report{report})
	}

	printReport(report)
	printSummary([]lint.Report{report})
	if report.HasErrors() {
		os.Exit(1)
	}
	return nil
}

func printReport(report lint.Report) {
	if len(report.Issues) == 0 {
		fmt.Printf("%s: no issues found\n", report.Subject)
		return
	}

	// Group issues by file
	byFile := map[string][]lint.Issue{}
	for _, issue := range report.Issues {
		key := issue.File
		if key == "" {
			key = "(subject)"
		}
		byFile[key] = append(byFile[key], issue)
	}

	// Sort files for stable output
	files := make([]string, 0, len(byFile))
	for f := range byFile {
		files = append(files, f)
	}
	sort.Strings(files)

	fmt.Println()
	// Use section-style for subject name (bold, accent color)
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("141"))
	fmt.Println(sectionStyle.Render(report.Subject))

	for _, file := range files {
		issues := byFile[file]
		fmt.Printf("  %s\n", file)

		for _, issue := range issues {
			var prefix string
			switch issue.Severity {
			case lint.SeverityError:
				prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("  error")
			case lint.SeverityWarning:
				prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render(" warning")
			case lint.SeverityInfo:
				prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Render("    info")
			}

			idSuffix := ""
			if issue.ID != "" {
				idSuffix = lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Render(fmt.Sprintf(" [%s]", issue.ID))
			}

			fmt.Printf("    %s%s %s\n", prefix, idSuffix, issue.Message)
		}
	}
}

func printSummary(reports []lint.Report) {
	totalErrors, totalWarnings, totalInfos := 0, 0, 0
	for _, r := range reports {
		e, w, i := r.CountBySeverity()
		totalErrors += e
		totalWarnings += w
		totalInfos += i
	}

	parts := []string{}
	if totalErrors > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("%d error%s", totalErrors, plural(totalErrors))))
	}
	if totalWarnings > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render(fmt.Sprintf("%d warning%s", totalWarnings, plural(totalWarnings))))
	}
	if totalInfos > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Render(fmt.Sprintf("%d info", totalInfos)))
	}

	subjectWord := "subject"
	if len(reports) != 1 {
		subjectWord = "subjects"
	}

	if len(parts) == 0 {
		fmt.Printf("\nAll good across %d %s.\n", len(reports), subjectWord)
	} else {
		summary := lipgloss.NewStyle().Bold(true).Render(strings.Join(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Render(" · ")))
		fmt.Printf("\n%s across %d %s\n", summary, len(reports), subjectWord)
	}
}

func printJSON(reports []lint.Report) error {
	var output interface{}
	if len(reports) == 1 {
		output = reports[0]
	} else {
		output = reports
	}
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func init() {
	lintCmd.Flags().BoolVar(&lintJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(lintCmd)
}
