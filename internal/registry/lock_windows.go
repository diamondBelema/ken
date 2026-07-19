//go:build windows

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

	h := syscall.Handle(f.Fd())
	if err := syscall.LockFile(h, 0, 0, 1, 0); err != nil {
		return err
	}
	defer syscall.UnlockFile(h, 0, 0, 1, 0)

	return fn()
}
