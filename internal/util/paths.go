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
	// QueueFile is the path to persisted download queue
	QueueFile string
	// YtDlpPath is the path to yt-dlp.exe
	YtDlpPath string
	// DefaultDownloadFolder is the default download location
	DefaultDownloadFolder string
	// PythonDir is the root of the portable Python runtime used for spotdl.
	PythonDir string
	// PythonExe is the full path to the bundled python executable.
	PythonExe string
	// DefaultSpotifyFolder is the default output folder for Spotify downloads.
	DefaultSpotifyFolder string
)

func init() {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		localAppData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
	}

	AppDataDir = filepath.Join(localAppData, "ytdlp-easy")
	os.MkdirAll(AppDataDir, 0755)

	SettingsFile = filepath.Join(AppDataDir, "settings.json")
	HistoryFile = filepath.Join(AppDataDir, "history.jsonl")
	LogFile = filepath.Join(AppDataDir, "ytdlp.log")
	QueueFile = filepath.Join(AppDataDir, "queue.json")
	YtDlpPath = filepath.Join(AppDataDir, "yt-dlp.exe")
	PythonDir = filepath.Join(AppDataDir, "python")
	PythonExe = filepath.Join(PythonDir, "python.exe")

	userProfile := os.Getenv("USERPROFILE")
	DefaultDownloadFolder = filepath.Join(userProfile, "Downloads")
	DefaultSpotifyFolder = filepath.Join(userProfile, "Music", "spotdl")
}

// EnsureAppDir creates the application directory if it doesn't exist
func EnsureAppDir() error {
	return os.MkdirAll(AppDataDir, 0755)
}
