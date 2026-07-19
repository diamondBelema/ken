//go:build windows

package system

import "os/exec"

func OpenURL(url string) error {
	return exec.Command("cmd", "/c", "start", url).Start()
}

func OpenFile(path string) error {
	return exec.Command("cmd", "/c", "start", "", path).Start()
}
