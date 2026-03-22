package main

import (
	"context"
	"fmt"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"ytdlp-easy/internal/config"
	"ytdlp-easy/internal/converter"
	"ytdlp-easy/internal/downloader"
	"ytdlp-easy/internal/history"
	"ytdlp-easy/internal/jsruntime"
	"ytdlp-easy/internal/updater"
	"ytdlp-easy/internal/util"
)

// App is the Wails bridge between Go backend and the frontend.
// All exported methods become callable from JavaScript.
type App struct {
	ctx       context.Context
	settings  *config.Settings
	queue     *downloader.Queue
	history   *history.History
	updater   *updater.Updater
	converter *converter.Converter
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	settings, err := config.Load()
	if err != nil {
		settings = config.DefaultSettings()
	}
	a.settings = settings

	a.queue = downloader.NewQueue(settings)
	a.queue.LoadPersistedItems()
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

	hist, err := history.NewHistory()
	if err != nil {
		hist = &history.History{}
	}
	a.history = hist

	a.updater = updater.NewUpdater(settings.Advanced.UseNightly)
	a.updater.OnProgress = func(msg string) {
		wailsruntime.EventsEmit(a.ctx, "updater:progress", msg)
	}

	go func() {
		if err := a.updater.EnsureInstalled(); err != nil {
			wailsruntime.EventsEmit(a.ctx, "error", "Failed to install yt-dlp: "+err.Error())
		}

		if settings.General.CheckUpdatesOnStart {
			wailsruntime.EventsEmit(a.ctx, "update:checking", nil)
			if info, err := a.updater.CheckForUpdate(); err == nil {
				if info.UpdateAvailable {
					wailsruntime.EventsEmit(a.ctx, "update:available", info)
				} else {
					wailsruntime.EventsEmit(a.ctx, "update:none", nil)
				}
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

	if !a.updater.IsInstalled() {
		return "", fmt.Errorf("yt-dlp is not installed yet, please wait for installation to complete")
	}

	item := a.queue.Add(url, opts.IsAudioOnly, opts.Quality, opts.Format)

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
func (a *App) SaveSettings(s *config.Settings) error {
	a.settings = s

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

// --- FFmpeg Converter ---

func (a *App) BrowseInputFile() string {
	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select Input File",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Media Files", Pattern: "*.mp4;*.mkv;*.webm;*.avi;*.mov;*.flv;*.wmv;*.mp3;*.m4a;*.aac;*.ogg;*.opus;*.flac;*.wav"},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})
	if err != nil {
		return ""
	}
	return path
}

func (a *App) BrowseOutputFile(defaultName string) string {
	path, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "Save Output File",
		DefaultFilename: defaultName,
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Media Files", Pattern: "*.mp4;*.mkv;*.webm;*.avi;*.mov;*.mp3;*.m4a;*.aac;*.ogg;*.opus;*.flac;*.wav"},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})
	if err != nil {
		return ""
	}
	return path
}

func (a *App) StartConversion(opts converter.ConversionOptions) (*converter.ConversionJob, error) {
	if !converter.IsFFmpegInstalled() {
		return nil, fmt.Errorf("FFmpeg is not installed. Please download it first.")
	}

	if opts.InputFile == "" {
		return nil, fmt.Errorf("no input file selected")
	}

	// Only one conversion at a time — stop the previous one
	if a.converter != nil {
		a.converter.Cancel()
	}

	job := &converter.ConversionJob{
		ID:        fmt.Sprintf("conv_%d", time.Now().UnixMilli()),
		InputFile: opts.InputFile,
		Status:    converter.StatusPending,
	}

	c := converter.NewConverter(job)
	c.OnProgress = func(j *converter.ConversionJob) {
		wailsruntime.EventsEmit(a.ctx, "convert:progress", j)
	}
	c.OnLog = func(line string) {
		wailsruntime.EventsEmit(a.ctx, "convert:log", line)
	}
	a.converter = c

	go func() {
		if err := c.Start(context.Background(), opts); err != nil {
			if job.Status != converter.StatusCancelled {
				wailsruntime.EventsEmit(a.ctx, "convert:error", err.Error())
			}
		}
	}()

	return job, nil
}

func (a *App) CancelConversion() {
	if a.converter != nil {
		a.converter.Cancel()
		a.converter = nil
	}
}

func (a *App) GetConversionPresets() []converter.Preset {
	return converter.GetPresets()
}

func (a *App) IsFFmpegInstalled() bool {
	return converter.IsFFmpegInstalled()
}

func (a *App) GetFFmpegVersion() string {
	ver, err := converter.GetFFmpegVersion()
	if err != nil {
		return ""
	}
	return ver
}

func (a *App) DownloadFFmpeg() error {
	dl := &converter.FFmpegDownloader{
		OnProgress: func(msg string) {
			wailsruntime.EventsEmit(a.ctx, "ffmpeg:progress", msg)
		},
	}
	return dl.Download()
}

// GetRecentCompletedDownloads returns completed queue items with file paths for the converter's "recent downloads" dropdown.
func (a *App) GetRecentCompletedDownloads() []map[string]string {
	items := a.queue.GetAll()
	var result []map[string]string
	for _, item := range items {
		if item.Status == downloader.StatusCompleted && item.FilePath != "" {
			result = append(result, map[string]string{
				"id":        item.ID,
				"title":     item.Title,
				"file_path": item.FilePath,
			})
		}
	}
	return result
}
