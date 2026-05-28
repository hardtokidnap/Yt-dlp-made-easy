package spotify

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"

	"ytdlp-easy/internal/util"
)

// RuntimeInfo describes the install state of the portable Python and spotdl.
type RuntimeInfo struct {
	PythonInstalled bool   `json:"python_installed"`
	PythonVersion   string `json:"python_version"`
	PythonPath      string `json:"python_path"`
	SpotdlInstalled bool   `json:"spotdl_installed"`
	SpotdlVersion   string `json:"spotdl_version"`
}

// DetectRuntime returns the current install state of the Spotify runtime.
// Does not download or modify anything. Safe to call repeatedly.
func DetectRuntime() RuntimeInfo {
	info := RuntimeInfo{PythonPath: util.PythonExe}
	if _, err := os.Stat(util.PythonExe); err != nil {
		return info
	}
	info.PythonInstalled = true
	info.PythonVersion = runVersion(util.PythonExe, "--version")
	info.SpotdlVersion = runVersion(util.PythonExe, "-m", "spotdl", "--version")
	info.SpotdlInstalled = info.SpotdlVersion != "" &&
		!strings.Contains(info.SpotdlVersion, "No module") &&
		!strings.Contains(strings.ToLower(info.SpotdlVersion), "error")
	return info
}

// runVersion runs the given executable with the given args and returns the
// first line of combined output, or "" on failure. 5-second timeout.
func runVersion(exe string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, exe, args...).CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return ""
		}
		// Fall through, exit-error may still carry useful output (e.g. spotdl
		// reporting "No module named spotdl" on stderr with non-zero exit).
	}
	line := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
	return line
}
