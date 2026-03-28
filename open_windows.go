//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// hiddenSysProcAttr prevents console window flash for console apps (cmd.exe, powershell).
// Do NOT use on GUI apps like explorer.exe — SW_HIDE would suppress their window.
func hiddenSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}

func (a *App) OpenFile(path string) error {
	cmd := exec.Command("cmd", "/c", "start", "", path)
	cmd.SysProcAttr = hiddenSysProcAttr()
	return cmd.Start()
}

func (a *App) OpenURL(url string) error {
	cmd := exec.Command("cmd", "/c", "start", "", url)
	cmd.SysProcAttr = hiddenSysProcAttr()
	return cmd.Start()
}

func (a *App) OpenFolder(path string) error {
	cmd := exec.Command("explorer", path)
	return cmd.Start()
}

// OpenFileInFolder opens the containing folder and highlights the file.
// Uses SysProcAttr.CmdLine to bypass Go's default argument quoting,
// which would wrap the entire "/select,<path>" in quotes and break explorer.
func (a *App) OpenFileInFolder(filePath string) error {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	// If exact file is gone (e.g. deleted after conversion), open the parent folder
	if _, statErr := os.Stat(absPath); os.IsNotExist(statErr) {
		dir := filepath.Dir(absPath)
		if err := exec.Command("explorer", dir).Start(); err != nil {
			return fmt.Errorf("explorer failed to open directory %q: %w", dir, err)
		}
		return nil
	}

	cmdLine := `explorer /select,"` + absPath + `"`
	cmd := exec.Command("explorer")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine: cmdLine,
	}
	if startErr := cmd.Start(); startErr != nil {
		return fmt.Errorf("explorer failed (cmdline=%q): %w", cmdLine, startErr)
	}
	return nil
}
