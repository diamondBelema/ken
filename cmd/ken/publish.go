package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/diamondBelema/ken/internal/parser"
	"github.com/diamondBelema/ken/internal/registry"
	"github.com/spf13/cobra"
)

var publishAll bool

var publishCmd = &cobra.Command{
	Use:   "publish [subject]",
	Short: "Publish subject(s) to the Ken registry",
	Long: `Publish a subject or all subjects to GitHub and register in the Ken registry.

Examples:
  ken publish nucleic-acid      # publish one subject
  ken publish --all             # publish all subjects with ken.yaml

Steps:
  1. Validates ken.yaml exists (run 'ken package' first if needed)
  2. Creates GitHub repo if it doesn't exist
  3. Pushes content to GitHub
  4. Creates or updates registry index entries

Requires: gh CLI authenticated with push access.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		subjectsDir := filepath.Join(home, "Documents", "learn", "subjects")

		if err := checkGH(); err != nil {
			return err
		}

		if publishAll {
			return publishAllSubjects(subjectsDir)
		}

		if len(args) == 0 {
			return fmt.Errorf("specify a subject or use --all")
		}

		return publishSubject(subjectsDir, args[0])
	},
}

func publishAllSubjects(subjectsDir string) error {
	entries, err := os.ReadDir(subjectsDir)
	if err != nil {
		return fmt.Errorf("failed to read subjects directory: %w", err)
	}

	var subjects []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		kenYaml := filepath.Join(subjectsDir, e.Name(), "ken.yaml")
		if _, err := os.Stat(kenYaml); err == nil {
			subjects = append(subjects, e.Name())
		}
	}

	if len(subjects) == 0 {
		return fmt.Errorf("no subjects with ken.yaml found — run 'ken package' on each subject first")
	}

	fmt.Printf("Found %d subject%s to publish:\n", len(subjects), plural(len(subjects)))
	for _, s := range subjects {
		fmt.Printf("  - %s\n", s)
	}
	fmt.Println()

	// All subjects share one repo
	reader := bufio.NewReader(os.Stdin)
	defaultRepo := "https://github.com/" + detectGitHubUser() + "/ken-subjects"
	fmt.Printf("GitHub repo URL for all subjects [%s]: ", defaultRepo)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	repoURL := defaultRepo
	if input != "" {
		repoURL = input
	}

	owner, repo := parseGitHubRepo(repoURL)
	if owner == "" || repo == "" {
		return fmt.Errorf("invalid repository URL: %s", repoURL)
	}

	fmt.Printf("Repository: %s/%s\n\n", owner, repo)

	if err := ensureGitHubRepo(owner, repo); err != nil {
		return err
	}

	// Build a working directory — clone remote if it exists, otherwise init fresh
	tmpDir, err := os.MkdirTemp("", "ken-publish-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	run := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmpDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// Try cloning existing repo
	cloneErr := run("clone", repoURL, tmpDir)
	if cloneErr != nil {
		// Repo doesn't exist yet, init fresh
		if err := gitInit(tmpDir); err != nil {
			return err
		}
		if err := run("remote", "add", "origin", repoURL); err != nil {
			return fmt.Errorf("failed to add remote: %w", err)
		}
	} else {
		fmt.Println("Cloned existing repository")
	}

	// Copy all subjects into the repo
	for _, s := range subjects {
		src := filepath.Join(subjectsDir, s)
		dst := filepath.Join(tmpDir, s)
		if err := copyDir(src, dst); err != nil {
			return fmt.Errorf("failed to copy %s: %w", s, err)
		}
	}

	// Update ken.yaml repo URLs
	for _, s := range subjects {
		manifest, err := parser.LoadManifest(filepath.Join(subjectsDir, s))
		if err != nil {
			continue
		}
		manifest.Repository = repoURL
		parser.SaveManifest(filepath.Join(tmpDir, s), manifest)
	}

	msg := fmt.Sprintf("Update %d subject%s", len(subjects), plural(len(subjects)))
	if err := gitCommitMsg(tmpDir, msg); err != nil {
		return err
	}

	// Detect default branch
	defaultBranch := "main"
	cmd := exec.Command("git", " symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = tmpDir
	out, err := cmd.Output()
	if err == nil {
		defaultBranch = strings.TrimSpace(strings.TrimPrefix(string(out), "refs/remotes/origin/"))
	} else {
		// Fallback: check what branch exists on remote
		cmd = exec.Command("git", "branch", "-r")
		cmd.Dir = tmpDir
		out, _ = cmd.Output()
		if strings.Contains(string(out), "origin/master") {
			defaultBranch = "master"
		}
	}

	if err := run("push", "-u", "origin", defaultBranch, "--tags"); err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	fmt.Printf("Pushed to GitHub (%s branch)\n", defaultBranch)

	// Collect all manifests and update registry in one PR
	var manifests []parser.Manifest
	for _, s := range subjects {
		manifest, err := parser.LoadManifest(filepath.Join(subjectsDir, s))
		if err != nil {
			continue
		}
		manifest.Repository = repoURL
		manifests = append(manifests, manifest)
	}

	if err := registry.UpdateIndex(manifests...); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update registry: %v\n", err)
	} else {
		fmt.Printf("Registered %d package%s in registry\n", len(manifests), plural(len(manifests)))
	}

	fmt.Printf("\nPublished %d subject%s\n", len(subjects), plural(len(subjects)))
	return nil
}

func publishSubject(subjectsDir, subject string) error {
	subjectDir := filepath.Join(subjectsDir, subject)

	if _, err := os.Stat(subjectDir); os.IsNotExist(err) {
		return fmt.Errorf("subject not found: %s", subjectDir)
	}

	manifest, err := parser.LoadManifest(subjectDir)
	if err != nil {
		return fmt.Errorf("no ken.yaml found — run 'ken package %s' first", subject)
	}

	if manifest.Repository == "" {
		manifest.Repository = promptGitHubRepo(manifest)
		parser.SaveManifest(subjectDir, manifest)
	}

	owner, repo := parseGitHubRepo(manifest.Repository)
	if owner == "" || repo == "" {
		return fmt.Errorf("invalid repository URL in ken.yaml: %s", manifest.Repository)
	}

	fmt.Printf("Publishing %s v%s\n", manifest.Name, manifest.Version)
	fmt.Printf("Repository: %s/%s\n\n", owner, repo)

	if err := ensureGitHubRepo(owner, repo); err != nil {
		return err
	}

	if err := gitPush(subjectDir); err != nil {
		return err
	}

	if err := registry.UpdateIndex(manifest); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update registry index: %v\n", err)
		fmt.Println("You may need to manually update the registry.")
	}

	fmt.Printf("\nPublished %s v%s\n", manifest.Name, manifest.Version)
	fmt.Printf("Install with: ken add %s/%s\n", owner, repo)
	return nil
}

func checkGH() error {
	cmd := exec.Command("gh", "auth", "status")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh CLI not authenticated — run 'gh auth login' first")
	}
	return nil
}

func promptGitHubRepo(m parser.Manifest) string {
	fmt.Println("No repository URL in ken.yaml.")
	reader := bufio.NewReader(os.Stdin)

	defaultRepo := fmt.Sprintf("https://github.com/%s/%s", detectGitHubUser(), m.Subjects[0])
	fmt.Printf("GitHub repo URL [%s]: ", defaultRepo)

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultRepo
	}
	return input
}

func parseGitHubRepo(url string) (owner, repo string) {
	url = strings.TrimPrefix(url, "https://github.com/")
	url = strings.TrimPrefix(url, "http://github.com/")
	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimRight(url, "/")

	parts := strings.SplitN(url, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

func ensureGitHubRepo(owner, repo string) error {
	cmd := exec.Command("gh", "repo", "view", owner+"/"+repo, "--json", "name")
	cmd.Stdout = os.Stdout
	cmd.Stderr = nil
	if err := cmd.Run(); err == nil {
		fmt.Println("Repository exists on GitHub")
		return nil
	}

	fmt.Printf("Creating repository %s/%s...\n", owner, repo)
	createCmd := exec.Command("gh", "repo", "create", owner+"/"+repo,
		"--public",
		"--description", "Ken packages",
		"--clone=false",
	)
	createCmd.Stdout = os.Stdout
	createCmd.Stderr = os.Stderr
	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("failed to create GitHub repo: %w", err)
	}

	return nil
}

func gitPush(dir string) error {
	run := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	cmd := exec.Command("git", "remote")
	cmd.Dir = dir
	out, _ := cmd.Output()
	if !strings.Contains(string(out), "origin") {
		manifest, _ := parser.LoadManifest(dir)
		if manifest.Repository != "" {
			if err := run("remote", "add", "origin", manifest.Repository); err != nil {
				return fmt.Errorf("failed to add remote: %w", err)
			}
		}
	}

	if err := run("push", "-u", "origin", "main", "--tags"); err != nil {
		if err := run("push", "-u", "origin", "master", "--tags"); err != nil {
			return fmt.Errorf("failed to push: %w", err)
		}
	}

	fmt.Println("Pushed to GitHub")
	return nil
}

func gitCommitMsg(dir, msg string) error {
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

	if err := run("commit", "-m", msg); err != nil {
		fmt.Println("Nothing to commit")
		return nil
	}

	fmt.Println("Committed all files")
	return nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directories
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

func init() {
	publishCmd.Flags().BoolVar(&publishAll, "all", false, "publish all subjects with ken.yaml")
	rootCmd.AddCommand(publishCmd)
}
