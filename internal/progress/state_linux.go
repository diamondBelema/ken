//go:build linux

package progress

import (
	"fmt"
	"os"
	"path/filepath"
)

func stateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", stateDirName), nil
}
