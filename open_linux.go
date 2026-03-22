//go:build linux

package main

import (
	"os/exec"
	"path/filepath"
)

func (a *App) OpenFile(path string) error {
	return exec.Command("xdg-open", path).Start()
}

func (a *App) OpenURL(url string) error {
	return exec.Command("xdg-open", url).Start()
}

func (a *App) OpenFolder(path string) error {
	return exec.Command("xdg-open", path).Start()
}

func (a *App) OpenFileInFolder(filePath string) error {
	return exec.Command("xdg-open", filepath.Dir(filePath)).Start()
}
