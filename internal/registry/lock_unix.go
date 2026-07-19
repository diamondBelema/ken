//go:build !windows

package registry

import (
	"os"
	"syscall"
)

func withStateLock(fn func() error) error {
	statePath, err := StatePath()
	if err != nil {
		return err
	}
	lockPath := statePath + ".lock"

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	return fn()
}
