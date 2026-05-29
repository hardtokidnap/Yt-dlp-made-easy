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

// Compile-time check that hiddenCmdCtx exists on every platform via the
// build-tagged install_windows.go / install_other.go files.
var _ = hiddenCmdCtx

// spotdlVersionFloor is the minimum spotdl version with the genres .get fix
// (Spotify deprecated the artist genres field for new apps) and the
// --use-official-api flag. Keep in sync with the pin in install_windows.go.
const spotdlVersionFloor = "4.5.0"

// RuntimeInfo describes the install state of the portable Python and spotdl.
type RuntimeInfo struct {
	PythonInstalled bool   `json:"python_installed"`
	PythonVersion   string `json:"python_version"`
	PythonPath      string `json:"python_path"`
	SpotdlInstalled bool   `json:"spotdl_installed"`
	SpotdlVersion   string `json:"spotdl_version"`
	SpotdlOutdated  bool   `json:"spotdl_outdated"`
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
	if info.SpotdlInstalled {
		// runVersion returns a bare "4.2.11" for `-m spotdl --version`.
		info.SpotdlOutdated = spotdlBelow(info.SpotdlVersion, spotdlVersionFloor)
	}
	return info
}

// runVersion runs the given executable with the given args and returns the
// first line of combined output, or "" on failure. 5-second timeout.
// Uses hiddenCmdCtx on Windows so probing for python / spotdl versions does
// not flash a console window every time the user clicks the Spotify tab.
func runVersion(exe string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := hiddenCmdCtx(ctx, exe, args...).CombinedOutput()
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
