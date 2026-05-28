//go:build !windows

package spotify

import (
	"context"
	"errors"
	"os/exec"
)

// ProgressFn receives free-form status messages during install.
type ProgressFn func(msg string)

// InstallRuntime is not yet implemented on non-Windows platforms.
// See docs/superpowers/specs/2026-05-28-spotdl-and-converter-design.md for
// the python-build-standalone approach planned for macOS and Linux.
func InstallRuntime(ctx context.Context, progress ProgressFn) error {
	_ = ctx
	_ = progress
	return errors.New("Spotify runtime install not yet implemented for this OS")
}

// hiddenCmd on non-Windows is just plain exec.Command (no console to hide).
func hiddenCmd(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

// hiddenCmdCtx is exec.CommandContext on non-Windows (no console window to hide).
func hiddenCmdCtx(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}
