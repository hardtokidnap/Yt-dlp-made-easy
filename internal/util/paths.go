package util

import (
	"os"
	"path/filepath"
)

var (
	// AppDataDir is the application data directory
	AppDataDir string
	// SettingsFile is the path to settings.json
	SettingsFile string
	// HistoryFile is the path to history.jsonl
	HistoryFile string
	// LogFile is the path to log file
	LogFile string
	// YtDlpPath is the path to yt-dlp.exe
	YtDlpPath string
	// DefaultDownloadFolder is the default download location
	DefaultDownloadFolder string
)

func init() {
	// Get AppData/Local directory
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		localAppData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
	}

	// Application directory
	AppDataDir = filepath.Join(localAppData, "ytdlp-easy")

	// Ensure directory exists
	os.MkdirAll(AppDataDir, 0755)

	// File paths
	SettingsFile = filepath.Join(AppDataDir, "settings.json")
	HistoryFile = filepath.Join(AppDataDir, "history.jsonl")
	LogFile = filepath.Join(AppDataDir, "ytdlp.log")
	YtDlpPath = filepath.Join(AppDataDir, "yt-dlp.exe")

	// Default download folder
	DefaultDownloadFolder = filepath.Join(os.Getenv("USERPROFILE"), "Downloads")
}

// EnsureAppDir creates the application directory if it doesn't exist
func EnsureAppDir() error {
	return os.MkdirAll(AppDataDir, 0755)
}
