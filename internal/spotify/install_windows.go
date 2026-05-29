//go:build windows

package spotify

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"ytdlp-easy/internal/util"
)

const (
	// Pinned to a known-good combination. Bump these together and re-test on update.
	pythonEmbedURL = "https://www.python.org/ftp/python/3.12.7/python-3.12.7-embed-amd64.zip"
	getPipURL      = "https://bootstrap.pypa.io/get-pip.py"
	// Pinned to the first version with the genres .get fix + --use-official-api.
	// Bump and re-test (resolve + meta + download) on update.
	spotdlVersion = "spotdl==4.5.0"
	pythonPthFile  = "python312._pth"
)

// ProgressFn receives free-form status messages during install.
type ProgressFn func(msg string)

// InstallRuntime downloads embeddable Python, patches the ._pth file to enable
// site-packages, installs pip, then installs spotdl. Idempotent. Re-running
// after a partial install resumes by checking each step.
func InstallRuntime(ctx context.Context, progress ProgressFn) error {
	if progress == nil {
		progress = func(string) {}
	}

	if err := os.MkdirAll(util.PythonDir, 0755); err != nil {
		return fmt.Errorf("create python dir: %w", err)
	}

	if _, err := os.Stat(util.PythonExe); os.IsNotExist(err) {
		progress("Downloading Python runtime...")
		zipPath := filepath.Join(util.PythonDir, "python.zip")
		if err := downloadFile(ctx, pythonEmbedURL, zipPath, progress); err != nil {
			return fmt.Errorf("download python: %w", err)
		}
		progress("Extracting Python runtime...")
		if err := unzip(zipPath, util.PythonDir); err != nil {
			return fmt.Errorf("extract python: %w", err)
		}
		os.Remove(zipPath)
	}

	if err := patchPthFile(filepath.Join(util.PythonDir, pythonPthFile)); err != nil {
		return fmt.Errorf("patch ._pth: %w", err)
	}

	if !pipInstalled() {
		progress("Installing pip...")
		getPipPath := filepath.Join(util.PythonDir, "get-pip.py")
		if err := downloadFile(ctx, getPipURL, getPipPath, progress); err != nil {
			return fmt.Errorf("download get-pip: %w", err)
		}
		cmd := hiddenCmd(util.PythonExe, getPipPath, "--no-warn-script-location")
		cmd.Stdout = newProgressWriter(progress)
		cmd.Stderr = newProgressWriter(progress)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("run get-pip: %w", err)
		}
		os.Remove(getPipPath)
	}

	// Install when missing, or upgrade when an older spotdl (e.g. 4.2.x, which
	// crashes on the deprecated Spotify artist 'genres' field) is present.
	if info := DetectRuntime(); !info.SpotdlInstalled || info.SpotdlOutdated {
		progress("Installing/updating spotdl (this can take a few minutes)...")
		cmd := hiddenCmd(util.PythonExe, "-m", "pip", "install", "--upgrade",
			"--no-warn-script-location", "--disable-pip-version-check", spotdlVersion)
		cmd.Stdout = newProgressWriter(progress)
		cmd.Stderr = newProgressWriter(progress)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("pip install spotdl: %w", err)
		}
	}

	progress("Spotify runtime ready.")
	return nil
}

// patchPthFile uncomments the "import site" line so pip-installed packages are
// visible. Embeddable Python ships with "#import site" commented out.
func patchPthFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	patched := strings.ReplaceAll(string(data), "#import site", "import site")
	if !strings.Contains(patched, "import site") {
		patched += "\nimport site\n"
	}
	if patched == string(data) {
		return nil
	}
	return os.WriteFile(path, []byte(patched), 0644)
}

func pipInstalled() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return hiddenCmdCtx(ctx, util.PythonExe, "-m", "pip", "--version").Run() == nil
}


// downloadFile streams URL to dest path with .tmp + rename for AV-safe writes.
func downloadFile(ctx context.Context, url, dest string, progress ProgressFn) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	tmpPath := dest + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	buf := make([]byte, 64*1024)
	var total int64
	var nextReport int64 = 1 << 20
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				f.Close()
				os.Remove(tmpPath)
				return werr
			}
			total += int64(n)
			if total >= nextReport {
				progress(fmt.Sprintf("Downloaded %.1f MB...", float64(total)/(1<<20)))
				nextReport += 1 << 20
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			os.Remove(tmpPath)
			return rerr
		}
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	// AV may briefly hold the new file. Retry rename up to 5 times.
	var lastErr error
	for i := 0; i < 5; i++ {
		if err := os.Rename(tmpPath, dest); err == nil {
			return nil
		} else {
			lastErr = err
			time.Sleep(time.Duration(200*(i+1)) * time.Millisecond)
		}
	}
	return fmt.Errorf("rename %s -> %s: %w", tmpPath, dest, lastErr)
}

// unzip extracts a zip archive to dest, with zip-slip protection.
func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	cleanDest := filepath.Clean(dest)
	for _, f := range r.File {
		outPath := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(outPath, cleanDest+string(os.PathSeparator)) && outPath != cleanDest {
			return errors.New("invalid zip entry path: " + f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(outPath, f.Mode()); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return err
		}
		in, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			in.Close()
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			in.Close()
			out.Close()
			return err
		}
		in.Close()
		out.Close()
	}
	return nil
}

// progressWriter adapts an io.Writer to call ProgressFn line-by-line.
type progressWriter struct {
	fn  ProgressFn
	buf []byte
}

func newProgressWriter(fn ProgressFn) *progressWriter {
	return &progressWriter{fn: fn}
}

func (p *progressWriter) Write(b []byte) (int, error) {
	p.buf = append(p.buf, b...)
	for {
		i := -1
		for j, c := range p.buf {
			if c == '\n' || c == '\r' {
				i = j
				break
			}
		}
		if i < 0 {
			break
		}
		line := strings.TrimSpace(string(p.buf[:i]))
		p.buf = p.buf[i+1:]
		if line != "" {
			p.fn(line)
		}
	}
	return len(b), nil
}

// hiddenCmd is the Windows equivalent of converter.hiddenCmd: hide the console
// window for child Python processes so users do not see a flash on every spawn.
// Mirrors internal/converter/cmd_windows.go.
func hiddenCmd(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	return cmd
}

// hiddenCmdCtx is hiddenCmd with context-bound cancellation. Used for long-
// running downloads where the user may cancel mid-flight.
func hiddenCmdCtx(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
	return cmd
}
