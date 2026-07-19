package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/diamondBelema/ken/internal/parser"
)

func UpdateIndex(manifests ...parser.Manifest) error {
	if len(manifests) == 0 {
		return nil
	}

	index, err := FetchIndex()
	if err != nil {
		return fmt.Errorf("failed to fetch registry index: %w", err)
	}

	for _, m := range manifests {
		pkg := Package{
			ID:          m.ID,
			Name:        m.Name,
			Version:     m.Version,
			Author:      m.Author,
			Description: m.Description,
			Tags:        m.Tags,
			Repository:  m.Repository,
		}

		found := false
		for i, p := range index.Packages {
			if p.ID == pkg.ID {
				index.Packages[i] = pkg
				found = true
				break
			}
		}
		if !found {
			index.Packages = append(index.Packages, pkg)
		}
	}

	tmpDir, err := os.MkdirTemp("", "ken-registry-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	registryURL := "https://github.com/diamondBelema/ken-registry.git"
	cmd := exec.Command("git", "clone", registryURL, tmpDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone registry: %w", err)
	}

	indexPath := filepath.Join(tmpDir, "registry.json")
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	if err := os.WriteFile(indexPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write index: %w", err)
	}

	branchName := fmt.Sprintf("update-packages-%s", time.Now().Format("20060102-150405"))

	run := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmpDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	if err := run("checkout", "-b", branchName); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	if err := run("add", "registry.json"); err != nil {
		return fmt.Errorf("failed to stage: %w", err)
	}

	var title, body string
	if len(manifests) == 1 {
		m := manifests[0]
		title = fmt.Sprintf("Update %s to v%s", m.Name, m.Version)
		body = fmt.Sprintf("Updates package `%s` to version `%s`.\n\nAuthor: %s\nDescription: %s", m.ID, m.Version, m.Author, m.Description)
	} else {
		names := make([]string, len(manifests))
		for i, m := range manifests {
			names[i] = fmt.Sprintf("%s v%s", m.Name, m.Version)
		}
		title = fmt.Sprintf("Update %d packages", len(manifests))
		body = fmt.Sprintf("Updates packages:\n")
		for _, m := range manifests {
			body += fmt.Sprintf("- `%s` to v%s\n", m.ID, m.Version)
		}
	}

	if err := run("commit", "-m", title); err != nil {
		fmt.Println("No changes to commit")
		return nil
	}

	if err := run("push", "-u", "origin", branchName); err != nil {
		return fmt.Errorf("failed to push branch: %w", err)
	}

	prCmd := exec.Command("gh", "pr", "create",
		"--repo", "diamondBelema/ken-registry",
		"--title", title,
		"--body", body,
		"--head", branchName,
	)
	prCmd.Stdout = os.Stdout
	prCmd.Stderr = os.Stderr
	if err := prCmd.Run(); err != nil {
		return fmt.Errorf("failed to create PR: %w", err)
	}

	return nil
}

func sanitize(s string) string {
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, " ", "-")
	return s
}
