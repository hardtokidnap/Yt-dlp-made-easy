//go:build windows

package converter

import (
	"os/exec"
	"syscall"
)

// hiddenCmd creates an exec.Cmd that won't flash a console window on Windows.
func hiddenCmd(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
	return cmd
}
