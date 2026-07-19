//go:build linux

package system

import "os/exec"

func OpenURL(url string) error {
	return exec.Command("xdg-open", url).Start()
}

func OpenFile(path string) error {
	return exec.Command("xdg-open", path).Start()
}
