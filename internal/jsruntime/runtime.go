package jsruntime

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"ytdlp-easy/internal/util"
)

// Runtime holds executable info for yt-dlp's --js-runtimes flag.
// yt-dlp needs a JS runtime to decrypt certain YouTube videos that use
// signature ciphers implemented in JavaScript.
type Runtime struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Version string `json:"version"`
}

// RuntimeInfo is exposed to the frontend so users can see what's available
// and decide whether to install a bundled runtime.
type RuntimeInfo struct {
	Detected     *Runtime  `json:"detected"`
	Available    []Runtime `json:"available"`
	Recommended  string    `json:"recommended"`
	DenoPath     string    `json:"deno_path"`
	NeedsInstall bool      `json:"needs_install"`
}

// DenoDownloadURL provides the GitHub releases URL for the user's platform.
// Deno is recommended by the yt-dlp team for best compatibility.
func DenoDownloadURL() string {
	if runtime.GOOS == "windows" {
		return "https://github.com/denoland/deno/releases/latest/download/deno-x86_64-pc-windows-msvc.zip"
	}
	return ""
}

// BundledDenoPath returns where we store a self-managed Deno executable,
// separate from any system-installed version to avoid conflicts.
func BundledDenoPath() string {
	return filepath.Join(util.AppDataDir, "deno.exe")
}

// DetectRuntimes scans for JS runtimes in order of yt-dlp's preference:
// bundled Deno first (we control the version), then system PATH.
func DetectRuntimes() RuntimeInfo {
	info := RuntimeInfo{
		Available:   make([]Runtime, 0),
		Recommended: "deno",
		DenoPath:    BundledDenoPath(),
	}

	// Check for bundled Deno first
	if r := checkRuntime("deno", BundledDenoPath()); r != nil {
		info.Available = append(info.Available, *r)
	}

	// Check system PATH for runtimes in order of preference
	runtimes := []string{"deno", "node", "bun"}
	for _, name := range runtimes {
		if r := checkRuntimeInPath(name); r != nil {
			// Avoid duplicates (bundled vs system)
			isDuplicate := false
			for _, existing := range info.Available {
				if existing.Path == r.Path {
					isDuplicate = true
					break
				}
			}
			if !isDuplicate {
				info.Available = append(info.Available, *r)
			}
		}
	}

	// Set detected to first available (highest priority)
	if len(info.Available) > 0 {
		info.Detected = &info.Available[0]
		info.NeedsInstall = false
	} else {
		info.NeedsInstall = true
	}

	return info
}

func checkRuntime(name, path string) *Runtime {
	if _, err := os.Stat(path); err != nil {
		return nil
	}

	version := getRuntimeVersion(name, path)
	if version == "" {
		return nil
	}

	return &Runtime{
		Name:    name,
		Path:    path,
		Version: version,
	}
}

func checkRuntimeInPath(name string) *Runtime {
	path, err := exec.LookPath(name)
	if err != nil {
		// exec.LookPath on Windows doesn't always add .exe suffix
		if runtime.GOOS == "windows" {
			path, err = exec.LookPath(name + ".exe")
			if err != nil {
				return nil
			}
		} else {
			return nil
		}
	}

	version := getRuntimeVersion(name, path)
	if version == "" {
		return nil
	}

	return &Runtime{
		Name:    name,
		Path:    path,
		Version: version,
	}
}

func getRuntimeVersion(name, path string) string {
	var cmd *exec.Cmd
	switch name {
	case "deno", "node", "bun":
		cmd = exec.Command(path, "--version")
	default:
		return ""
	}

	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) == 0 {
		return ""
	}

	// Each runtime has a different output format that needs normalization
	version := strings.TrimSpace(lines[0])
	if name == "deno" && strings.HasPrefix(version, "deno ") {
		version = strings.TrimPrefix(version, "deno ")
	}
	return version
}

// GetYtDlpRuntimeArgs builds the --js-runtimes flag based on user config.
// When "auto" (default), we detect available runtimes and only pass the flag
// when necessary - yt-dlp already defaults to system Deno if present.
func GetYtDlpRuntimeArgs(configured string) []string {
	if configured != "" && configured != "auto" {
		return []string{"--js-runtimes", configured}
	}

	info := DetectRuntimes()
	if info.Detected == nil {
		return nil
	}

	if info.Detected.Name == "deno" {
		// Bundled Deno needs explicit path; system Deno is yt-dlp's default
		if info.Detected.Path == BundledDenoPath() {
			return []string{"--js-runtimes", fmt.Sprintf("deno:%s", info.Detected.Path)}
		}
		return nil
	}

	return []string{"--js-runtimes", info.Detected.Name}
}

type Downloader struct {
	OnProgress func(message string)
}

// DownloadDeno fetches Deno from GitHub releases and extracts it to our app
// data folder. This gives users a working JS runtime without requiring them
// to install anything system-wide.
func (d *Downloader) DownloadDeno() error {
	url := DenoDownloadURL()
	if url == "" {
		return fmt.Errorf("Deno download not available for this platform")
	}

	d.notify("Downloading Deno...")

	// Download the zip file
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download Deno: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Create temp file for the zip
	tempZip := filepath.Join(util.AppDataDir, "deno-temp.zip")
	out, err := os.Create(tempZip)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempZip) // Clean up temp file

	// Copy with progress
	totalSize := resp.ContentLength
	written := int64(0)
	buf := make([]byte, 32*1024)
	lastPercent := -1

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
			written += int64(n)

			if totalSize > 0 {
				percent := int(float64(written) / float64(totalSize) * 100)
				if percent != lastPercent {
					d.notify(fmt.Sprintf("Downloading Deno... %d%%", percent))
					lastPercent = percent
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			out.Close()
			return fmt.Errorf("download interrupted: %w", err)
		}
	}
	out.Close()

	d.notify("Extracting Deno...")

	// Extract deno.exe from the zip
	if err := extractDenoFromZip(tempZip, BundledDenoPath()); err != nil {
		return fmt.Errorf("failed to extract Deno: %w", err)
	}

	d.notify("Deno installed successfully!")
	return nil
}

func (d *Downloader) notify(msg string) {
	if d.OnProgress != nil {
		d.OnProgress(msg)
	}
}

// extractDenoFromZip uses PowerShell on Windows because Go's archive/zip
// requires more boilerplate for a simple single-file extraction.
func extractDenoFromZip(zipPath, destPath string) error {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("powershell", "-Command",
			fmt.Sprintf(`Expand-Archive -Path '%s' -DestinationPath '%s' -Force; Move-Item -Path '%s\deno.exe' -Destination '%s' -Force`,
				zipPath,
				filepath.Dir(destPath),
				filepath.Dir(destPath),
				destPath))
		return cmd.Run()
	}
	return fmt.Errorf("extraction not implemented for this platform")
}

func GetDenoLatestVersion() (string, error) {
	resp, err := http.Get("https://api.github.com/repos/denoland/deno/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return strings.TrimPrefix(release.TagName, "v"), nil
}
