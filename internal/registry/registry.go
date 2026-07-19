package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	registryIndexURL = "https://raw.githubusercontent.com/diamondBelema/ken-registry/main/registry.json"
	stateFileName    = "registry.json"
)

type Package struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Repository  string   `json:"repository"`
	Stars       int      `json:"stars"`
	Downloads   int      `json:"downloads"`
}

type RegistryIndex struct {
	Version  int       `json:"version"`
	Packages []Package `json:"packages"`
}

type InstalledPackage struct {
	Version     string `json:"version"`
	InstalledAt string `json:"installed_at"`
	SubjectDir  string `json:"subject_dir"`
}

type InstalledState struct {
	Installed map[string]InstalledPackage `json:"installed"`
}

func StatePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}

	// Use platform-appropriate state dir
	stateDir := filepath.Join(home, ".local", "share", "ken")
	if isWindows() {
		stateDir = filepath.Join(home, "AppData", "Local", "ken")
	}

	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return "", fmt.Errorf("cannot create state directory: %w", err)
	}

	return filepath.Join(stateDir, stateFileName), nil
}

func LoadInstalled() (InstalledState, error) {
	var state InstalledState
	state.Installed = make(map[string]InstalledPackage)

	path, err := StatePath()
	if err != nil {
		return state, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, fmt.Errorf("failed to read registry state: %w", err)
	}

	if err := json.Unmarshal(data, &state); err != nil {
		return state, fmt.Errorf("failed to parse registry state: %w", err)
	}
	if state.Installed == nil {
		state.Installed = make(map[string]InstalledPackage)
	}
	return state, nil
}

func SaveInstalled(state InstalledState) error {
	path, err := StatePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry state: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

func isWindows() bool {
	return filepath.Separator == '\\'
}
