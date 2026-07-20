package registry

import (
	"fmt"
	"os"
	"time"

	"github.com/diamondBelema/ken/internal/parser"
)

func UpdatePackage(pkgID, subjectsDir string) (parser.Manifest, string, error) {
	var manifest parser.Manifest
	var msg string
	err := withStateLock(func() error {
		var err error
		manifest, msg, err = updatePackageLocked(pkgID, subjectsDir)
		return err
	})
	return manifest, msg, err
}

func updatePackageLocked(pkgID, subjectsDir string) (parser.Manifest, string, error) {
	state, err := LoadInstalled()
	if err != nil {
		return parser.Manifest{}, "", err
	}

	installed, ok := state.Installed[pkgID]
	if !ok {
		return parser.Manifest{}, "", fmt.Errorf("package %s is not installed — use 'ken add %s' first", pkgID, pkgID)
	}

	index, err := FetchIndex()
	if err != nil {
		return parser.Manifest{}, "", fmt.Errorf("failed to fetch registry: %w", err)
	}

	var pkg *Package
	for _, p := range index.Packages {
		if p.ID == pkgID {
			pkg = &p
			break
		}
	}
	if pkg == nil {
		return parser.Manifest{}, "", fmt.Errorf("package %s not found in registry", pkgID)
	}

	latestVersion := pkg.Version
	if latestVersion == "" {
		latestVersion = "latest"
	}

	if installed.Version == latestVersion && latestVersion != "latest" {
		return parser.Manifest{}, fmt.Sprintf("package %s is already up to date (v%s)", pkgID, installed.Version), nil
	}

	repoOwner, repoName := parseGitHubRepo(pkg.Repository)
	if repoOwner == "" || repoName == "" {
		return parser.Manifest{}, "", fmt.Errorf("invalid repository URL for %s: %s", pkgID, pkg.Repository)
	}

	fmt.Printf("Updating %s from v%s to v%s...\n", pkgID, installed.Version, versionLabel(latestVersion))

	reader, err := DownloadTarball(repoOwner, repoName, latestVersion)
	if err != nil {
		return parser.Manifest{}, "", err
	}
	defer reader.Close()

	tmpDir, err := os.MkdirTemp("", "ken-update-*")
	if err != nil {
		return parser.Manifest{}, "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := ExtractTarball(reader, tmpDir, true); err != nil {
		return parser.Manifest{}, "", fmt.Errorf("failed to extract package: %w", err)
	}

	manifest, err := findManifest(tmpDir, installed.SubjectDir)
	if err != nil {
		return parser.Manifest{}, "", err
	}

	if err := copyContent(tmpDir, subjectsDir, manifest); err != nil {
		return parser.Manifest{}, "", err
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

	msg := fmt.Sprintf("Updated %s from v%s to v%s", pkgID, installed.Version, manifest.Version)
	return manifest, msg, nil
}

func UpdateAll(subjectsDir string) ([]UpdateResult, error) {
	var results []UpdateResult
	err := withStateLock(func() error {
		var err error
		results, err = updateAllLocked(subjectsDir)
		return err
	})
	return results, err
}

type UpdateResult struct {
	PkgID      string
	OldVersion string
	NewVersion string
	Status     string
}

func updateAllLocked(subjectsDir string) ([]UpdateResult, error) {
	state, err := LoadInstalled()
	if err != nil {
		return nil, err
	}

	if len(state.Installed) == 0 {
		return nil, fmt.Errorf("no packages installed — use 'ken add' to install some first")
	}

	index, err := FetchIndex()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch registry: %w", err)
	}

	registryMap := make(map[string]Package)
	for _, p := range index.Packages {
		registryMap[p.ID] = p
	}

	var results []UpdateResult
	for pkgID, installed := range state.Installed {
		pkg, ok := registryMap[pkgID]
		if !ok {
			results = append(results, UpdateResult{
				PkgID:      pkgID,
				OldVersion: installed.Version,
				Status:     "skipped (not in registry)",
			})
			continue
		}

		latestVersion := pkg.Version
		if latestVersion == "" {
			latestVersion = "latest"
		}

		if installed.Version == latestVersion && latestVersion != "latest" {
			results = append(results, UpdateResult{
				PkgID:      pkgID,
				OldVersion: installed.Version,
				NewVersion: installed.Version,
				Status:     "up to date",
			})
			continue
		}

		repoOwner, repoName := parseGitHubRepo(pkg.Repository)
		if repoOwner == "" || repoName == "" {
			results = append(results, UpdateResult{
				PkgID:      pkgID,
				OldVersion: installed.Version,
				Status:     "skipped (invalid repo URL)",
			})
			continue
		}

		fmt.Printf("Downloading %s v%s...\n", pkgID, versionLabel(latestVersion))

		reader, err := DownloadTarball(repoOwner, repoName, latestVersion)
		if err != nil {
			results = append(results, UpdateResult{
				PkgID:      pkgID,
				OldVersion: installed.Version,
				Status:     fmt.Sprintf("error: %v", err),
			})
			continue
		}

		tmpDir, err := os.MkdirTemp("", "ken-update-*")
		if err != nil {
			reader.Close()
			results = append(results, UpdateResult{
				PkgID:      pkgID,
				OldVersion: installed.Version,
				Status:     fmt.Sprintf("error: %v", err),
			})
			continue
		}

		if err := ExtractTarball(reader, tmpDir, true); err != nil {
			reader.Close()
			os.RemoveAll(tmpDir)
			results = append(results, UpdateResult{
				PkgID:      pkgID,
				OldVersion: installed.Version,
				Status:     fmt.Sprintf("error: %v", err),
			})
			continue
		}
		reader.Close()

		manifest, err := findManifest(tmpDir, installed.SubjectDir)
		if err != nil {
			os.RemoveAll(tmpDir)
			results = append(results, UpdateResult{
				PkgID:      pkgID,
				OldVersion: installed.Version,
				Status:     fmt.Sprintf("error: %v", err),
			})
			continue
		}

		if err := copyContent(tmpDir, subjectsDir, manifest); err != nil {
			os.RemoveAll(tmpDir)
			results = append(results, UpdateResult{
				PkgID:      pkgID,
				OldVersion: installed.Version,
				Status:     fmt.Sprintf("error: %v", err),
			})
			continue
		}
		os.RemoveAll(tmpDir)

		now := time.Now().UTC().Format(time.RFC3339)
		state.Installed[pkgID] = InstalledPackage{
			Version:     manifest.Version,
			InstalledAt: now,
			SubjectDir:  manifest.Subjects[0],
		}

		results = append(results, UpdateResult{
			PkgID:      pkgID,
			OldVersion: installed.Version,
			NewVersion: manifest.Version,
			Status:     "updated",
		})
	}

	if err := SaveInstalled(state); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to save state: %v\n", err)
	}

	return results, nil
}
