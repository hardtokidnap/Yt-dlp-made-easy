//go:build darwin

package main

import "os/exec"

func (a *App) OpenFile(path string) error {
	return exec.Command("open", path).Start()
}

func (a *App) OpenURL(url string) error {
	return exec.Command("open", url).Start()
}

func (a *App) OpenFolder(path string) error {
	return exec.Command("open", path).Start()
}

func (a *App) OpenFileInFolder(filePath string) error {
	return exec.Command("open", "-R", filePath).Start()
}
