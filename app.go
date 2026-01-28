package main

import (
	"context"
	"os/exec"
	goruntime "runtime"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"ytdlp-easy/internal/config"
	"ytdlp-easy/internal/downloader"
	"ytdlp-easy/internal/history"
	"ytdlp-easy/internal/updater"
	"ytdlp-easy/internal/util"
)

// App struct
type App struct {
	ctx      context.Context
	settings *config.Settings
	queue    *downloader.Queue
	history  *history.History
	updater  *updater.Updater
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Load settings
	settings, err := config.Load()
	if err != nil {
		settings = config.DefaultSettings()
	}
	a.settings = settings

	// Create download queue
	a.queue = downloader.NewQueue(settings)
	a.queue.OnItemUpdate = func(item *downloader.Item) {
		wailsruntime.EventsEmit(a.ctx, "download:update", item)
	}
	a.queue.OnQueueUpdate = func() {
		wailsruntime.EventsEmit(a.ctx, "queue:update", a.GetQueueStatus())
	}

	// Load history
	hist, err := history.NewHistory()
	if err != nil {
		hist = &history.History{}
	}
	a.history = hist

	// Create updater
	a.updater = updater.NewUpdater(settings.Advanced.UseNightly)
	a.updater.OnProgress = func(msg string) {
		wailsruntime.EventsEmit(a.ctx, "updater:progress", msg)
	}

	// Ensure yt-dlp is installed
	go func() {
		if err := a.updater.EnsureInstalled(); err != nil {
			wailsruntime.EventsEmit(a.ctx, "error", "Failed to install yt-dlp: "+err.Error())
		}

		// Check for updates if enabled
		if settings.General.CheckUpdates {
			if info, err := a.updater.CheckForUpdate(); err == nil && info.UpdateAvailable {
				wailsruntime.EventsEmit(a.ctx, "update:available", info)
			}
		}
	}()
}

// shutdown is called when the app closes
func (a *App) shutdown(ctx context.Context) {
	if a.queue != nil {
		a.queue.Shutdown()
	}
}

// ========== Download Operations ==========

// DownloadOptions specifies download configuration
type DownloadOptions struct {
	IsAudioOnly bool   `json:"is_audio_only"`
	Quality     string `json:"quality"`
	Format      string `json:"format"`
}

// AddDownload adds a URL to the download queue
func (a *App) AddDownload(url string, opts DownloadOptions) string {
	item := a.queue.Add(url, opts.IsAudioOnly, opts.Quality, opts.Format)
	return item.ID
}

// PauseDownload pauses a download
func (a *App) PauseDownload(id string) error {
	return a.queue.Pause(id)
}

// ResumeDownload resumes a download
func (a *App) ResumeDownload(id string) error {
	return a.queue.Resume(id)
}

// StopDownload stops a download
func (a *App) StopDownload(id string) error {
	return a.queue.Stop(id)
}

// RemoveDownload removes a download from queue
func (a *App) RemoveDownload(id string) {
	a.queue.Remove(id)
}

// GetQueueStatus returns all queue items
func (a *App) GetQueueStatus() []*downloader.Item {
	return a.queue.GetAll()
}

// PauseAllDownloads pauses all active downloads
func (a *App) PauseAllDownloads() int {
	return a.queue.PauseAll()
}

// ResumeAllDownloads resumes all paused downloads
func (a *App) ResumeAllDownloads() int {
	return a.queue.ResumeAll()
}

// StopAllDownloads stops all downloads
func (a *App) StopAllDownloads() int {
	return a.queue.StopAll()
}

// ClearCompletedDownloads clears finished downloads
func (a *App) ClearCompletedDownloads() int {
	return a.queue.ClearCompleted()
}

// ========== Settings Operations ==========

// GetSettings returns current settings
func (a *App) GetSettings() *config.Settings {
	return a.settings
}

// SaveSettings saves settings
func (a *App) SaveSettings(s config.Settings) error {
	a.settings.General = s.General
	a.settings.Download = s.Download
	a.settings.Network = s.Network
	a.settings.Auth = s.Auth
	a.settings.Advanced = s.Advanced

	// Update queue concurrency if changed
	a.queue.UpdateMaxConcurrent(s.General.MaxConcurrent)

	// Update updater nightly setting
	a.updater.UseNightly = s.Advanced.UseNightly

	return a.settings.Save()
}

// BrowseFolder opens a folder picker dialog
func (a *App) BrowseFolder() string {
	folder, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:            "Select Download Folder",
		DefaultDirectory: a.settings.General.SaveFolder,
	})
	if err != nil {
		return ""
	}
	return folder
}

// ========== History Operations ==========

// HistoryFilter specifies search criteria
type HistoryFilter struct {
	Query  string `json:"query"`
	Status string `json:"status"`
}

// GetHistory returns filtered history
func (a *App) GetHistory(filter HistoryFilter) []history.Entry {
	return a.history.Search(filter.Query, filter.Status)
}

// ClearHistory clears all history
func (a *App) ClearHistory() error {
	return a.history.Clear()
}

// ClearOldHistory clears history older than days
func (a *App) ClearOldHistory(days int) int {
	count, _ := a.history.ClearOld(days)
	return count
}

// ExportHistory exports history to CSV
func (a *App) ExportHistory() (string, error) {
	filepath, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "Export History",
		DefaultFilename: "download_history.csv",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "CSV Files", Pattern: "*.csv"},
		},
	})
	if err != nil || filepath == "" {
		return "", err
	}

	return filepath, a.history.ExportCSV(filepath)
}

// RedownloadFromHistory starts a re-download from history
func (a *App) RedownloadFromHistory(id string) string {
	entry := a.history.GetByID(id)
	if entry == nil {
		return ""
	}

	item := a.queue.Add(entry.URL, entry.IsAudioOnly, entry.Quality, entry.Format)
	return item.ID
}

// GetHistoryStats returns history statistics
func (a *App) GetHistoryStats() map[string]interface{} {
	return a.history.Stats()
}

// ========== File Operations ==========

// OpenFile opens a file with the default application
func (a *App) OpenFile(path string) error {
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

// OpenURL opens a URL in the default browser
func (a *App) OpenURL(url string) error {
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// OpenFolder opens a folder in file explorer
func (a *App) OpenFolder(path string) error {
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

// ========== Updater Operations ==========

// CheckForUpdates checks for yt-dlp updates
func (a *App) CheckForUpdates() (*updater.UpdateInfo, error) {
	return a.updater.CheckForUpdate()
}

// UpdateYtDlp updates yt-dlp
func (a *App) UpdateYtDlp() error {
	return a.updater.Update()
}

// GetYtDlpVersion returns current yt-dlp version
func (a *App) GetYtDlpVersion() string {
	return a.updater.GetCurrentVersion()
}

// ========== Clipboard Operations ==========

// GetClipboard returns clipboard contents
func (a *App) GetClipboard() string {
	text, err := wailsruntime.ClipboardGetText(a.ctx)
	if err != nil {
		return ""
	}
	return text
}

// ========== Utility ==========

// GetAppInfo returns application info
func (a *App) GetAppInfo() map[string]string {
	return map[string]string{
		"app_name":    "YT-DLP Made Easy",
		"app_version": "2.0.0",
		"ytdlp_path":  util.YtDlpPath,
		"config_path": util.AppDataDir,
	}
}
