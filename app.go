package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"ytdlp-easy/internal/config"
	"ytdlp-easy/internal/converter"
	"ytdlp-easy/internal/downloader"
	"ytdlp-easy/internal/history"
	"ytdlp-easy/internal/jsruntime"
	"ytdlp-easy/internal/spotify"
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
	converter   *converter.Converter
	batchQueue  *converter.BatchQueue
	spotifyJob  *spotify.Job
	spotifyMu   sync.Mutex
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

	// History must init before queue — the OnItemUpdate callback references a.history
	hist, err := history.NewHistory()
	if err != nil {
		hist = &history.History{}
	}
	a.history = hist

	a.queue = downloader.NewQueue(settings)
	a.queue.LoadPersistedItems()
	a.queue.OnItemUpdate = func(item *downloader.Item) {
		wailsruntime.EventsEmit(a.ctx, "download:update", item)

		if item.Status == downloader.StatusCompleted || item.Status == downloader.StatusError {
			if err := a.history.Add(history.Entry{
				ID:          item.ID,
				URL:         item.URL,
				Title:       item.Title,
				Status:      string(item.Status),
				FilePath:    item.FilePath,
				FileSize:    item.FileSize,
				Error:       item.Error,
				IsAudioOnly: item.IsAudioOnly,
				Quality:     item.Quality,
				Format:      item.Format,
				FileExt:     filepath.Ext(item.FilePath),
			}); err != nil {
				wailsruntime.LogError(a.ctx, "Failed to persist history: "+err.Error())
				return
			}
			// Completed/errored items live in history now — remove from active queue.
			// Queue.Remove already emits queue:update via notifyQueueUpdate.
			a.queue.Remove(item.ID)
			wailsruntime.EventsEmit(a.ctx, "history:update", a.history.GetRecent(3))
		}
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
	if a.batchQueue != nil {
		a.batchQueue.Cancel()
	}
	if a.converter != nil {
		a.converter.Cancel()
	}
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

func (a *App) GetHistory(query, status string) []history.Entry {
	return a.history.Search(query, status)
}

func (a *App) GetRecentHistory(limit int) []history.Entry {
	return a.history.GetRecent(limit)
}

func (a *App) ClearHistory() error          { return a.history.Clear() }
func (a *App) ClearOldHistory(days int) int { count, _ := a.history.ClearOld(days); return count }

func (a *App) HideFromQueue(id string) error {
	if err := a.history.HideFromQueue(id); err != nil {
		return err
	}
	wailsruntime.EventsEmit(a.ctx, "history:update", a.history.GetRecent(3))
	return nil
}

func (a *App) RemoveHistoryEntry(id string) error {
	if err := a.history.Remove(id); err != nil {
		return err
	}
	wailsruntime.EventsEmit(a.ctx, "history:update", a.history.GetRecent(3))
	return nil
}

func (a *App) ExportHistory() (string, error) {
	outputPath, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "Export History",
		DefaultFilename: "download_history.csv",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "CSV Files", Pattern: "*.csv"},
		},
	})
	if err != nil || outputPath == "" {
		return "", err
	}

	return outputPath, a.history.ExportCSV(outputPath)
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

func (a *App) GetAppInfo() map[string]interface{} {
	return map[string]interface{}{
		"app_name":     util.AppName,
		"app_version":  util.AppVersion,
		"ytdlp_path":   util.YtDlpPath,
		"config_path":  util.AppDataDir,
		"contributors": util.Contributors,
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

func (a *App) ProbeFile(filePath string) (*converter.MediaInfo, error) {
	if !converter.IsFFmpegInstalled() {
		return nil, fmt.Errorf("FFmpeg is not installed. Please download it first.")
	}
	if filePath == "" {
		return nil, fmt.Errorf("no input file selected")
	}
	return converter.ProbeFile(filePath)
}

// EstimateConversionSize predicts the output file size for the given conversion
// options. Frontend calls this on every option change (debounced) to keep the
// size hint live. Returns Confidence "unknown" when probe fails rather than an
// error so the UI can simply show a blank/neutral hint.
func (a *App) EstimateConversionSize(opts converter.ConversionOptions) (converter.SizeEstimate, error) {
	if opts.InputFile == "" {
		return converter.SizeEstimate{Confidence: "unknown", Note: "No input file selected."}, nil
	}
	if !converter.IsFFmpegInstalled() {
		return converter.SizeEstimate{Confidence: "unknown", Note: "FFmpeg not installed."}, nil
	}
	info, err := converter.ProbeFile(opts.InputFile)
	if err != nil {
		return converter.SizeEstimate{Confidence: "unknown", Note: "Could not probe input file."}, nil
	}
	return converter.EstimateOutputSize(opts, info), nil
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

	// When trimming, set the effective duration so progress is based on clip length.
	// With -ss before -i, ffmpeg's time= output starts from 0 and counts up to the clip length.
	if opts.EndTime != "" {
		startSec := 0.0
		if opts.StartTime != "" {
			startSec = converter.ParseTimeToSeconds(opts.StartTime)
		}
		endSec := converter.ParseTimeToSeconds(opts.EndTime)
		if endSec > startSec {
			c.SetEffectiveDuration(endSec - startSec)
		}
	}

	a.converter = c

	go func() {
		if err := c.Start(a.ctx, opts); err != nil {
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

func (a *App) BrowseMultipleInputFiles() []string {
	paths, err := wailsruntime.OpenMultipleFilesDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select Input Files",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Media Files", Pattern: "*.mp4;*.mkv;*.webm;*.avi;*.mov;*.flv;*.wmv;*.mp3;*.m4a;*.aac;*.ogg;*.opus;*.flac;*.wav"},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})
	if err != nil {
		return nil
	}
	return paths
}

func (a *App) StartBatchConversion(files []string, opts converter.ConversionOptions) error {
	if !converter.IsFFmpegInstalled() {
		return fmt.Errorf("FFmpeg is not installed. Please download it first.")
	}
	if len(files) == 0 {
		return fmt.Errorf("no input files selected")
	}

	// Only one conversion pipeline at a time — stop anything in-flight
	if a.converter != nil {
		a.converter.Cancel()
		a.converter = nil
	}
	if a.batchQueue != nil {
		a.batchQueue.Cancel()
	}

	bq := converter.NewBatchQueue(files, opts)
	bq.OnProgress = func(snap *converter.BatchSnapshot) {
		wailsruntime.EventsEmit(a.ctx, "convert:batch:progress", snap)
	}
	bq.OnLog = func(line string) {
		wailsruntime.EventsEmit(a.ctx, "convert:log", line)
	}
	a.batchQueue = bq

	go func() {
		bq.Start(a.ctx, opts)
		// Surface any per-job failures that weren't already emitted via progress events
		if bq.Failed > 0 {
			wailsruntime.EventsEmit(a.ctx, "convert:error",
				fmt.Sprintf("Batch finished with %d failed file(s)", bq.Failed))
		}
	}()

	return nil
}

func (a *App) CancelBatchConversion() {
	if a.batchQueue != nil {
		a.batchQueue.Cancel()
		a.batchQueue = nil
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

// GetRecentCompletedDownloads returns up to 20 history entries with files that still exist on disk.
func (a *App) GetRecentCompletedDownloads() []map[string]string {
	entries := a.history.GetRecentCompleted(20)
	result := make([]map[string]string, 0, len(entries))
	for _, entry := range entries {
		if _, err := os.Stat(entry.FilePath); err != nil {
			continue
		}
		result = append(result, map[string]string{
			"id":        entry.ID,
			"title":     entry.Title,
			"file_path": entry.FilePath,
		})
	}
	return result
}

// --- Spotify subsystem ---
// Fully isolated: no shared yt-dlp binary, queue, history, or cookies.
// Shares only the managed FFmpeg via the --ffmpeg flag.

// GetSpotifyRuntimeInfo reports the install state of the portable Python and spotdl.
// Cheap call, frontend uses it on every Spotify-tab open to decide which UI to show.
func (a *App) GetSpotifyRuntimeInfo() spotify.RuntimeInfo {
	return spotify.DetectRuntime()
}

// InstallSpotifyRuntime downloads Python embeddable + spotdl into the app data
// folder. Progress is emitted as "spotify:runtime:progress" events.
func (a *App) InstallSpotifyRuntime() error {
	return spotify.InstallRuntime(a.ctx, func(msg string) {
		wailsruntime.EventsEmit(a.ctx, "spotify:runtime:progress", msg)
	})
}

// PreviewSpotifyURL resolves track metadata for a Spotify URL (track / playlist
// / album) without downloading audio. 30-second timeout in case spotdl hangs.
// Passes through AudioProvider / cookies / proxy so the ytmusic-connection
// probe spotdl runs on 'save' uses the same workaround the download flow will.
func (a *App) PreviewSpotifyURL(url string) ([]spotify.Track, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	prevOpts := spotify.PreviewOptions{
		AudioProvider: a.settings.GetSpotify().AudioProvider,
	}
	sp := a.settings.GetSpotify()
	if sp.UseAuthCookies {
		prevOpts.CookieFile = a.settings.GetAuth().CookiesFile
	}
	if sp.UseProxy {
		prevOpts.Proxy = a.settings.GetNetwork().Proxy
	}
	return spotify.PreviewURL(ctx, url, prevOpts)
}

// DownloadSpotifyTracks starts a spotdl download in the background. Progress
// is emitted as "spotify:progress" + "spotify:complete" events.
func (a *App) DownloadSpotifyTracks(urls []string) (string, error) {
	if len(urls) == 0 {
		return "", fmt.Errorf("no URLs provided")
	}
	a.spotifyMu.Lock()
	if a.spotifyJob != nil && a.spotifyJob.Status == "running" {
		a.spotifyMu.Unlock()
		return "", fmt.Errorf("a Spotify download is already running")
	}
	a.spotifyMu.Unlock()

	settings := a.settings.GetSpotify()
	opts := spotify.Options{
		URLs:          urls,
		OutputFolder:  settings.OutputFolder,
		Format:        settings.Format,
		Bitrate:       settings.Bitrate,
		Threads:       settings.Threads,
		FFmpegPath:    converter.FFmpegPath(),
		AudioProvider: settings.AudioProvider,
	}
	if settings.UseAuthCookies {
		opts.CookieFile = a.settings.GetAuth().CookiesFile
	}
	if settings.UseProxy {
		opts.Proxy = a.settings.GetNetwork().Proxy
	}
	go func() {
		ctx := context.Background()
		_, err := spotify.Download(ctx, opts, func(job *spotify.Job) {
			a.spotifyMu.Lock()
			a.spotifyJob = job
			a.spotifyMu.Unlock()
			wailsruntime.EventsEmit(a.ctx, "spotify:progress", job)
			if job.Status == "completed" || job.Status == "failed" || job.Status == "cancelled" {
				wailsruntime.EventsEmit(a.ctx, "spotify:complete", job)
			}
		})
		if err != nil {
			wailsruntime.EventsEmit(a.ctx, "spotify:error", err.Error())
		}
	}()
	return "started", nil
}

// CancelSpotifyDownload kills the in-flight spotdl process if any.
func (a *App) CancelSpotifyDownload() {
	a.spotifyMu.Lock()
	defer a.spotifyMu.Unlock()
	if a.spotifyJob != nil {
		a.spotifyJob.Cancel()
	}
}
