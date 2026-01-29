package main

import (
	"context"
	"fmt"
	"os/exec"
	goruntime "runtime"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"ytdlp-easy/internal/config"
	"ytdlp-easy/internal/downloader"
	"ytdlp-easy/internal/history"
	"ytdlp-easy/internal/jsruntime"
	"ytdlp-easy/internal/updater"
	"ytdlp-easy/internal/util"
)

// App is the Wails bridge between Go backend and the frontend.
// All exported methods become callable from JavaScript.
type App struct {
	ctx      context.Context
	settings *config.Settings
	queue    *downloader.Queue
	history  *history.History
	updater  *updater.Updater
}

func NewApp() *App {
	return &App{}
}

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
	a.queue.OnLog = func(itemID, line string) {
		wailsruntime.EventsEmit(a.ctx, "download:log", map[string]string{
			"id":   itemID,
			"line": line,
		})
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
		if settings.General.CheckUpdatesOnStart {
			if info, err := a.updater.CheckForUpdate(); err == nil && info.UpdateAvailable {
				wailsruntime.EventsEmit(a.ctx, "update:available", info)
			}
		}
	}()
}

func (a *App) shutdown(ctx context.Context) {
	if a.queue != nil {
		a.queue.Shutdown()
	}
}

// DownloadOptions matches the frontend's download request format.
type DownloadOptions struct {
	IsAudioOnly bool   `json:"is_audio_only"`
	Quality     string `json:"quality"`
	Format      string `json:"format"`
}

func (a *App) AddDownload(url string, opts DownloadOptions) (string, error) {
	wailsruntime.LogDebug(a.ctx, "Adding download: "+url)

	// Ensure yt-dlp is installed
	if !a.updater.IsInstalled() {
		return "", fmt.Errorf("yt-dlp is not installed yet, please wait for installation to complete")
	}

	item := a.queue.Add(url, opts.IsAudioOnly, opts.Quality, opts.Format)

	// Emit initial state
	wailsruntime.EventsEmit(a.ctx, "download:update", item)

	return item.ID, nil
}

func (a *App) PauseDownload(id string) error  { return a.queue.Pause(id) }
func (a *App) ResumeDownload(id string) error { return a.queue.Resume(id) }
func (a *App) StopDownload(id string) error   { return a.queue.Stop(id) }
func (a *App) RemoveDownload(id string)       { a.queue.Remove(id) }
func (a *App) GetQueueStatus() []*downloader.Item { return a.queue.GetAll() }
func (a *App) PauseAllDownloads() int  { return a.queue.PauseAll() }
func (a *App) ResumeAllDownloads() int { return a.queue.ResumeAll() }
func (a *App) StopAllDownloads() int   { return a.queue.StopAll() }
func (a *App) ClearCompletedDownloads() int { return a.queue.ClearCompleted() }

func (a *App) GetSettings() *config.Settings { return a.settings }

// SaveSettings persists settings and propagates changes to running components.
// Takes a pointer to avoid copying the embedded mutex (go vet warning).
func (a *App) SaveSettings(s *config.Settings) error {
	a.settings.General = s.General
	a.settings.Download = s.Download
	a.settings.Network = s.Network
	a.settings.Auth = s.Auth
	a.settings.Advanced = s.Advanced

	a.queue.UpdateMaxConcurrent(s.General.MaxConcurrentDownloads)
	a.updater.UseNightly = s.Advanced.UseNightly

	return a.settings.Save()
}

// BrowseFolder opens a native folder picker and persists the selection.
func (a *App) BrowseFolder() string {
	folder, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:            "Select Download Folder",
		DefaultDirectory: a.settings.General.SaveFolder,
	})
	if err != nil || folder == "" {
		return a.settings.General.SaveFolder
	}

	// Update and save settings
	a.settings.General.SaveFolder = folder
	if err := a.settings.Save(); err != nil {
		wailsruntime.LogError(a.ctx, "Failed to save settings: "+err.Error())
	}

	return folder
}

type HistoryFilter struct {
	Query  string `json:"query"`
	Status string `json:"status"`
}

func (a *App) GetHistory(filter HistoryFilter) []history.Entry {
	return a.history.Search(filter.Query, filter.Status)
}

func (a *App) ClearHistory() error           { return a.history.Clear() }
func (a *App) ClearOldHistory(days int) int  { count, _ := a.history.ClearOld(days); return count }

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

func (a *App) RedownloadFromHistory(id string) string {
	entry := a.history.GetByID(id)
	if entry == nil {
		return ""
	}

	item := a.queue.Add(entry.URL, entry.IsAudioOnly, entry.Quality, entry.Format)
	return item.ID
}

func (a *App) GetHistoryStats() map[string]interface{} { return a.history.Stats() }

// OpenFile launches the OS default handler for a file path.
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

func (a *App) CheckForUpdates() (*updater.UpdateInfo, error) { return a.updater.CheckForUpdate() }
func (a *App) UpdateYtDlp() error                            { return a.updater.Update() }
func (a *App) GetYtDlpVersion() string                       { return a.updater.GetCurrentVersion() }

func (a *App) GetClipboard() string {
	text, err := wailsruntime.ClipboardGetText(a.ctx)
	if err != nil {
		return ""
	}
	return text
}

func (a *App) GetJSRuntimeInfo() jsruntime.RuntimeInfo { return jsruntime.DetectRuntimes() }

func (a *App) DownloadDeno() error {
	dl := &jsruntime.Downloader{
		OnProgress: func(msg string) {
			wailsruntime.EventsEmit(a.ctx, "jsruntime:progress", msg)
		},
	}
	return dl.DownloadDeno()
}

func (a *App) GetAppInfo() map[string]string {
	return map[string]string{
		"app_name":    "YT-DLP Made Easy",
		"app_version": "2.0.0",
		"ytdlp_path":  util.YtDlpPath,
		"config_path": util.AppDataDir,
	}
}
