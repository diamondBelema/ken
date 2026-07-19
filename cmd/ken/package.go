package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/diamondBelema/ken/internal/parser"
	"github.com/spf13/cobra"
)

var packageCmd = &cobra.Command{
	Use:   "package <subject>",
	Short: "Prepare a subject for publishing",
	Long: `Package a subject for publishing to the Ken registry.

If no ken.yaml exists, one will be generated from your content.
Then initializes a git repo, commits all files, and creates a version tag.

Examples:
  ken package nucleic-acid
  ken package anatomy`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		subject := args[0]

		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		subjectDir := filepath.Join(home, "Documents", "learn", "subjects", subject)

		if _, err := os.Stat(subjectDir); os.IsNotExist(err) {
			return fmt.Errorf("subject not found: %s", subjectDir)
		}

		manifest, err := ensureManifest(subjectDir, subject)
		if err != nil {
			return err
		}

		fmt.Printf("Packaging %s v%s\n", manifest.Name, manifest.Version)

		if err := gitInit(subjectDir); err != nil {
			return err
		}

		if err := gitCommit(subjectDir, manifest); err != nil {
			return err
		}

		if err := gitTag(subjectDir, manifest.Version); err != nil {
			return err
		}

		fmt.Printf("\nReady to publish: ken publish %s\n", manifest.ID)
		return nil
	},
}

func ensureManifest(dir, subject string) (parser.Manifest, error) {
	manifest, err := parser.LoadManifest(dir)
	if err == nil {
		if manifest.ID == "" {
			manifest.ID = detectGitHubUser() + "/" + subject
		}
		if manifest.Repository == "" {
			manifest.Repository = "https://github.com/" + detectGitHubUser() + "/" + subject
		}
		return manifest, nil
	}

	fmt.Println("No ken.yaml found. Generating one from your content...")

	user := detectGitHubUser()

	manifest = parser.Manifest{
		ID:         user + "/" + subject,
		Name:       humanize(subject),
		Version:    "1.0.0",
		Author:     user,
		Subjects:   []string{subject},
		Repository: "https://github.com/" + user + "/" + subject,
	}

	manifest.Concepts = countItems(filepath.Join(dir, "concepts"), "concepts")
	manifest.Flashcards = countItems(filepath.Join(dir, "flashcards"), "flashcards")

	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("Author name [%s]: ", manifest.Author)
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name != "" {
		manifest.Author = name
	}

	fmt.Print("Description: ")
	desc, _ := reader.ReadString('\n')
	manifest.Description = strings.TrimSpace(desc)

	fmt.Print("Tags (comma-separated): ")
	tags, _ := reader.ReadString('\n')
	for _, t := range strings.Split(tags, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			manifest.Tags = append(manifest.Tags, t)
		}
	}

	if err := parser.SaveManifest(dir, manifest); err != nil {
		return manifest, fmt.Errorf("failed to save ken.yaml: %w", err)
	}

	fmt.Println("Created ken.yaml")
	return manifest, nil
}

func detectGitHubUser() string {
	out, err := exec.Command("git", "config", "user.name").Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return "unknown"
}

func countItems(dir, itemType string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	total := 0
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			switch itemType {
			case "concepts":
				if strings.HasPrefix(trimmed, "- id: ") {
					total++
				}
			case "flashcards":
				if trimmed == "front:" {
					total++
				}
			}
		}
	}
	return total
}

func countFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".md" {
			count++
		}
	}
	return count
}

func humanize(s string) string {
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

func gitInit(dir string) error {
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		fmt.Println("Git repo already initialized")
		return nil
	}

	fmt.Println("Initializing git repo...")
	run := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	if err := run("init"); err != nil {
		return fmt.Errorf("failed to init git repo: %w", err)
	}

	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.tmp\n.DS_Store\n"), 0644); err != nil {
		return err
	}

	return nil
}

func gitCommit(dir string, m parser.Manifest) error {
	run := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	if err := run("add", "-A"); err != nil {
		return fmt.Errorf("failed to stage files: %w", err)
	}

	msg := fmt.Sprintf(" ken %s v%s", m.Name, m.Version)
	if err := run("commit", "-m", msg); err != nil {
		fmt.Println("Nothing to commit (already up to date)")
		return nil
	}

	fmt.Println("Committed all files")
	return nil
}

func gitTag(dir, version string) error {
	cmd := exec.Command("git", "tag", "-l", version)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(out)) == version {
		fmt.Printf("Tag %s already exists\n", version)
		return nil
	}

	cmd = exec.Command("git", "tag", version)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tag: %w", err)
	}

	fmt.Printf("Tagged %s\n", version)
	return nil
}

func init() {
	rootCmd.AddCommand(packageCmd)
}
