//go:build windows

package registry

import (
	"fmt"
	"os"
	"time"
)

func withStateLock(fn func() error) error {
	statePath, err := StatePath()
	if err != nil {
		return err
	}
	lockPath := statePath + ".lock"

	for i := 0; i < 50; i++ {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err == nil {
			f.WriteString(fmt.Sprintf("%d", os.Getpid()))
			f.Close()
			defer os.Remove(lockPath)
			return fn()
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for lock on %s", lockPath)
}
