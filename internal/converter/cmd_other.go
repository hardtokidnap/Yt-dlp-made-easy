//go:build !windows

package converter

import "os/exec"

func hiddenCmd(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
