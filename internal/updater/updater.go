package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"ytdlp-easy/internal/util"
)

const (
	StableURL  = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe"
	NightlyURL = "https://github.com/yt-dlp/yt-dlp-nightly-builds/releases/latest/download/yt-dlp.exe"
	StableAPI  = "https://api.github.com/repos/yt-dlp/yt-dlp/releases/latest"
	NightlyAPI = "https://api.github.com/repos/yt-dlp/yt-dlp-nightly-builds/releases/latest"
)

// UpdateInfo contains version information
type UpdateInfo struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	UpdateAvailable bool   `json:"update_available"`
	IsNightly       bool   `json:"is_nightly"`
}

// Updater manages yt-dlp binary updates
type Updater struct {
	UseNightly bool
	OnProgress func(message string)
}

// NewUpdater creates a new updater
func NewUpdater(useNightly bool) *Updater {
	return &Updater{
		UseNightly: useNightly,
	}
}

// GetCurrentVersion returns the installed yt-dlp version
func (u *Updater) GetCurrentVersion() string {
	if _, err := os.Stat(util.YtDlpPath); os.IsNotExist(err) {
		return ""
	}

	cmd := exec.Command(util.YtDlpPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// GetLatestVersion fetches the latest version from GitHub API
func (u *Updater) GetLatestVersion() (string, error) {
	apiURL := StableAPI
	if u.UseNightly {
		apiURL = NightlyAPI
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var data struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	return data.TagName, nil
}

// CheckForUpdate checks if an update is available
func (u *Updater) CheckForUpdate() (*UpdateInfo, error) {
	current := u.GetCurrentVersion()
	latest, err := u.GetLatestVersion()
	if err != nil {
		return nil, err
	}

	return &UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   latest,
		UpdateAvailable: current != latest && current != "",
		IsNightly:       u.UseNightly,
	}, nil
}

// IsInstalled checks if yt-dlp is installed
func (u *Updater) IsInstalled() bool {
	_, err := os.Stat(util.YtDlpPath)
	return err == nil
}

// Download downloads yt-dlp binary
func (u *Updater) Download() error {
	downloadURL := StableURL
	if u.UseNightly {
		downloadURL = NightlyURL
	}

	u.notify("Downloading yt-dlp...")

	// Create temp file with unique name
	tempPath := util.YtDlpPath + ".tmp"

	// Clean up any stale temp file
	os.Remove(tempPath)

	// Download
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	// Create output file
	out, err := os.Create(tempPath)
	if err != nil {
		return err
	}

	// Copy with progress (throttled to only update on % change)
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
					u.notify(fmt.Sprintf("Downloading... %d%%", percent))
					lastPercent = percent
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			out.Close()
			os.Remove(tempPath)
			return err
		}
	}

	// Close file before rename
	out.Close()

	// Try to replace old file with retries (handle file lock)
	var renameErr error
	for i := 0; i < 5; i++ {
		// Remove old file if exists
		if _, err := os.Stat(util.YtDlpPath); err == nil {
			if err := os.Remove(util.YtDlpPath); err != nil {
				u.notify(fmt.Sprintf("Waiting for file lock... attempt %d", i+1))
				time.Sleep(time.Second)
				continue
			}
		}

		// Rename temp to final
		renameErr = os.Rename(tempPath, util.YtDlpPath)
		if renameErr == nil {
			break
		}
		u.notify(fmt.Sprintf("Waiting for file lock... attempt %d", i+1))
		time.Sleep(time.Second)
	}

	if renameErr != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to install yt-dlp (file locked): %w", renameErr)
	}

	u.notify("Download complete!")
	return nil
}

// Update updates yt-dlp using its built-in update
func (u *Updater) Update() error {
	if !u.IsInstalled() {
		return u.Download()
	}

	u.notify("Updating yt-dlp...")

	var args []string
	if u.UseNightly {
		args = []string{"--update-to", "nightly"}
	} else {
		args = []string{"-U"}
	}

	cmd := exec.Command(util.YtDlpPath, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// If self-update fails, try downloading fresh
		u.notify("Self-update failed, downloading fresh...")
		return u.Download()
	}

	u.notify("Update complete: " + string(output))
	return nil
}

// EnsureInstalled makes sure yt-dlp is available
func (u *Updater) EnsureInstalled() error {
	// Clean up any stale temp file from previous failed attempts
	tempPath := util.YtDlpPath + ".tmp"
	os.Remove(tempPath)

	if u.IsInstalled() {
		return nil
	}
	return u.Download()
}

func (u *Updater) notify(msg string) {
	if u.OnProgress != nil {
		u.OnProgress(msg)
	}
}
