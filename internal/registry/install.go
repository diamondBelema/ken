package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/diamondBelema/ken/internal/parser"
)

func InstallPackage(owner, repo, version, subjectsDir string) (manifest parser.Manifest, err error) {
	err = withStateLock(func() error {
		manifest, err = installPackageLocked(owner, repo, version, subjectsDir)
		return err
	})
	return
}

func installPackageLocked(owner, repo, version, subjectsDir string) (parser.Manifest, error) {
	state, err := LoadInstalled()
	if err != nil {
		return parser.Manifest{}, err
	}

	pkgID := owner + "/" + repo
	if existing, ok := state.Installed[pkgID]; ok {
		if version == "" || version == "latest" {
			return parser.Manifest{}, fmt.Errorf("package %s already installed (v%s) — use 'ken update %s' to update", pkgID, existing.Version, pkgID)
		}
		if existing.Version == version {
			return parser.Manifest{}, fmt.Errorf("package %s@%s already installed", pkgID, version)
		}
	}

	// Look up package in registry to find the actual repository URL
	index, err := FetchIndex()
	if err != nil {
		return parser.Manifest{}, fmt.Errorf("failed to fetch registry: %w", err)
	}

	var pkg *Package
	for _, p := range index.Packages {
		if p.ID == pkgID {
			pkg = &p
			break
		}
	}
	if pkg == nil {
		return parser.Manifest{}, fmt.Errorf("package %s not found in registry", pkgID)
	}

	// Parse the repository URL to get owner/repo for download
	repoOwner, repoName := parseGitHubRepo(pkg.Repository)
	if repoOwner == "" || repoName == "" {
		return parser.Manifest{}, fmt.Errorf("invalid repository URL for %s: %s", pkgID, pkg.Repository)
	}

	fmt.Printf("Downloading %s@%s...\n", pkgID, versionLabel(version))

	reader, err := DownloadTarball(repoOwner, repoName, version)
	if err != nil {
		return parser.Manifest{}, err
	}
	defer reader.Close()

	tmpDir, err := os.MkdirTemp("", "ken-pkg-*")
	if err != nil {
		return parser.Manifest{}, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := ExtractTarball(reader, tmpDir, true); err != nil {
		return parser.Manifest{}, fmt.Errorf("failed to extract package: %w", err)
	}

	manifest, err := findManifest(tmpDir, repo)
	if err != nil {
		return parser.Manifest{}, err
	}

	if err := copyContent(tmpDir, subjectsDir, manifest); err != nil {
		return parser.Manifest{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	state.Installed[pkgID] = InstalledPackage{
		Version:     manifest.Version,
		InstalledAt: now,
		SubjectDir:  manifest.Subjects[0],
	}
	if err := SaveInstalled(state); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to save install state: %v\n", err)
	}

	return manifest, nil
}

func findManifest(tmpDir, subjectHint string) (parser.Manifest, error) {
	// Single-subject repo: ken.yaml at root
	manifest, err := parser.LoadManifest(tmpDir)
	if err == nil {
		return manifest, nil
	}

	// Multi-subject repo: look for subdirectories with ken.yaml
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return parser.Manifest{}, fmt.Errorf("failed to read extracted package: %w", err)
	}

	var manifests []parser.Manifest
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		subDir := filepath.Join(tmpDir, e.Name())
		m, err := parser.LoadManifest(subDir)
		if err != nil {
			continue
		}
		manifests = append(manifests, m)
	}

	if len(manifests) == 0 {
		return parser.Manifest{}, fmt.Errorf("no ken.yaml found in package")
	}

	// If a subject hint was given, try to find it
	if subjectHint != "" {
		for _, m := range manifests {
			for _, s := range m.Subjects {
				if s == subjectHint {
					return m, nil
				}
			}
			if m.ID == subjectHint {
				return m, nil
			}
		}
	}

	if len(manifests) == 1 {
		return manifests[0], nil
	}

	// Multiple subjects — return first one
	fmt.Printf("Package contains %d subjects:\n", len(manifests))
	for _, m := range manifests {
		fmt.Printf("  - %s (%s)\n", m.Name, strings.Join(m.Subjects, ", "))
	}
	fmt.Println()

	return manifests[0], nil
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

func copyContent(tmpDir, subjectsDir string, manifest parser.Manifest) error {
	for _, subject := range manifest.Subjects {
		dest := filepath.Join(subjectsDir, subject)

		// Check if this subject exists as a subdirectory in the tarball
		subjectSrc := filepath.Join(tmpDir, subject)
		if info, err := os.Stat(subjectSrc); err == nil && info.IsDir() {
			// Multi-subject repo: copy the subject subdirectory
			if _, err := os.Stat(dest); err == nil {
				fmt.Printf("Updating existing subject: %s\n", subject)
			} else {
				fmt.Printf("Installing subject: %s\n", subject)
			}

			if err := os.MkdirAll(dest, 0755); err != nil {
				return fmt.Errorf("failed to create subject directory: %w", err)
			}

			if err := copyDir(subjectSrc, dest); err != nil {
				return fmt.Errorf("failed to copy %s/: %w", subject, err)
			}
		} else {
			// Single-subject repo: copy content subdirectories from root
			if _, err := os.Stat(dest); err == nil {
				fmt.Printf("Updating existing subject: %s\n", subject)
			} else {
				fmt.Printf("Installing subject: %s\n", subject)
			}

			if err := os.MkdirAll(dest, 0755); err != nil {
				return fmt.Errorf("failed to create subject directory: %w", err)
			}

			subdirs := []string{"concepts", "flashcards", "quizzes", "notes", "diagrams"}
			for _, sub := range subdirs {
				src := filepath.Join(tmpDir, sub)
				dst := filepath.Join(dest, sub)

				if _, err := os.Stat(src); os.IsNotExist(err) {
					continue
				}

				if err := copyDir(src, dst); err != nil {
					return fmt.Errorf("failed to copy %s/: %w", sub, err)
				}
			}

			// Copy ken.yaml into subject dir
			srcManifest := filepath.Join(tmpDir, "ken.yaml")
			dstManifest := filepath.Join(dest, "ken.yaml")
			if _, err := os.Stat(srcManifest); err == nil {
				copyFile(srcManifest, dstManifest)
			}
		}
	}

	return nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

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

		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func RemovePackage(pkgID, subjectsDir string) error {
	return withStateLock(func() error {
		return removePackageLocked(pkgID, subjectsDir)
	})
}

func removePackageLocked(pkgID, subjectsDir string) error {
	state, err := LoadInstalled()
	if err != nil {
		return err
	}

	pkg, ok := state.Installed[pkgID]
	if !ok {
		return fmt.Errorf("package %s is not installed", pkgID)
	}

	subjectDir := filepath.Join(subjectsDir, pkg.SubjectDir)
	if _, err := os.Stat(subjectDir); err == nil {
		if err := os.RemoveAll(subjectDir); err != nil {
			return fmt.Errorf("failed to remove subject directory: %w", err)
		}
		fmt.Printf("Removed subject: %s\n", pkg.SubjectDir)
	}

	delete(state.Installed, pkgID)
	if err := SaveInstalled(state); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

func ListInstalled() (InstalledState, error) {
	return LoadInstalled()
}

func versionLabel(v string) string {
	if v == "" {
		return "latest"
	}
	return strings.TrimPrefix(v, "v")
}
