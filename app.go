package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	converter  *converter.Converter
	batchQueue *converter.BatchQueue
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
	// Spotify-origin items get full metadata + cover embedded after download.
	a.queue.OnSpotifyTag = func(item *downloader.Item) error {
		sp := a.settings.GetSpotify()
		// yt-dlp named the file from the YouTube title. spotdl meta matches by
		// filename, so rename to the Spotify "Artist - Title" first, otherwise
		// ambiguous/messy YouTube titles tag the wrong release.
		if item.Title != "" && item.FilePath != "" {
			if newPath, err := renameToTitle(item.FilePath, item.Title); err != nil {
				a.queue.OnLog(item.ID, "Could not rename before tagging, using original name: "+err.Error())
			} else {
				item.FilePath = newPath
			}
		}
		a.queue.OnLog(item.ID, "Tagging metadata via spotdl...")
		return spotify.ApplyMetadata(a.ctx, item.FilePath, spotify.MetaOptions{
			ClientID:     sp.ClientID,
			ClientSecret: sp.ClientSecret,
			FFmpegPath:   converter.FFmpegPath(),
			CookieFile:   spotifyCookieFile(a.settings),
			Proxy:        spotifyProxy(a.settings),
		}, func(line string) {
			wailsruntime.EventsEmit(a.ctx, "spotify:log", line) // verbose
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

// renameToTitle renames the downloaded file's base name to the given title
// (sanitized), preserving directory and extension. Returns the new path, or the
// original path on error. Used so spotdl meta matches the correct Spotify track
// by filename instead of the YouTube video title.
func renameToTitle(path, title string) (string, error) {
	base := sanitizeFilename(title)
	if base == "" {
		return path, nil
	}
	newPath := filepath.Join(filepath.Dir(path), base+filepath.Ext(path))
	if newPath == path {
		return path, nil
	}
	if err := os.Rename(path, newPath); err != nil {
		return path, err
	}
	return newPath, nil
}

// sanitizeFilename strips characters Windows disallows in filenames.
func sanitizeFilename(s string) string {
	r := strings.NewReplacer(
		"<", "-", ">", "-", ":", "-", "\"", "'",
		"/", "-", "\\", "-", "|", "-", "?", "", "*", "",
	)
	return strings.TrimSpace(r.Replace(s))
}

// spotifyAudioQuality maps a Settings.Spotify.Bitrate value to a yt-dlp
// --audio-quality value. "auto"/"best"/"" -> "0" (best). A specific bitrate
// ("192k", "320k") is passed through (yt-dlp accepts a bitrate string).
func spotifyAudioQuality(bitrate string) string {
	switch bitrate {
	case "", "auto", "best":
		return "0"
	default:
		return bitrate
	}
}

// spotifyCookieFile returns the cookies path when the Spotify settings opt in.
func spotifyCookieFile(s *config.Settings) string {
	if s.GetSpotify().UseAuthCookies {
		return s.GetAuth().CookiesFile
	}
	return ""
}

// spotifyProxy returns the proxy when the Spotify settings opt in.
func spotifyProxy(s *config.Settings) string {
	if s.GetSpotify().UseProxy {
		return s.GetNetwork().Proxy
	}
	return ""
}

// SpotifyCredsConfigured reports whether both Spotify API credentials are set.
// The frontend uses this to warn at URL-detect time that downloads need creds.
func (a *App) SpotifyCredsConfigured() bool {
	sp := a.settings.GetSpotify()
	return sp.ClientID != "" && sp.ClientSecret != ""
}

// PreviewSpotifyURL resolves track metadata for a Spotify URL (track / playlist
// / album) without downloading audio. 60-second timeout. Streams spotdl stdout
// and stderr to the frontend as 'spotify:log' events (routed to the verbose log)
// so the user can troubleshoot a failed resolve.
func (a *App) PreviewSpotifyURL(url string) ([]spotify.Track, error) {
	sp := a.settings.GetSpotify()
	// Playlists and liked songs require OAuth (app-only tokens cannot read
	// playlist items). The first such preview opens a browser to sign in, so it
	// gets a much longer timeout; the token is then cached for silent reuse.
	userAuth := spotifyNeedsUserAuth(url)
	timeout := 60 * time.Second
	if userAuth {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	prevOpts := spotify.PreviewOptions{
		AudioProvider: sp.AudioProvider,
		ClientID:      sp.ClientID,
		ClientSecret:  sp.ClientSecret,
		CookieFile:    spotifyCookieFile(a.settings),
		Proxy:         spotifyProxy(a.settings),
		UserAuth:      userAuth,
		CachePath:     spotifyOAuthCachePath(),
	}
	// spotdl retries on a 429 by sleeping out the (huge) retry-after, which would
	// hang until our timeout. Detect the rate-limit line in the live output and
	// cancel immediately so the user gets a fast, clear failure instead.
	var rateLimited bool
	tracks, err := spotify.PreviewURL(ctx, url, prevOpts, func(line string) {
		wailsruntime.EventsEmit(a.ctx, "spotify:log", line)
		if strings.Contains(line, "reached a rate/request limit") {
			rateLimited = true
			cancel()
		}
	})
	if err != nil && rateLimited {
		return nil, fmt.Errorf("Spotify rate-limited your app. Large playlists exceed a personal dev app's quota; try a smaller playlist, wait, or request Extended Quota Mode for your app")
	}
	return tracks, err
}

// spotifyNeedsUserAuth reports whether a Spotify URL points at content only
// readable with user OAuth (playlists, liked/saved songs) rather than an
// app-only token (tracks, albums, artists).
func spotifyNeedsUserAuth(url string) bool {
	return strings.Contains(url, "/playlist/") || strings.Contains(url, "/collection")
}

// spotifyOAuthCachePath is where spotipy stores the cached OAuth token so the
// user does not re-login on every playlist action.
func spotifyOAuthCachePath() string {
	return filepath.Join(util.AppDataDir, "spotify-oauth.cache")
}

// SpotifyTrackRequest is one track the user chose to download.
type SpotifyTrackRequest struct {
	URL   string `json:"url"`   // Spotify track URL
	Title string `json:"title"` // "Artist - Title" for queue/history display
}

// DownloadSpotifyTracks resolves each selected Spotify track to an audio-source
// URL via spotdl, then enqueues it as a first-class download. The existing
// queue handles the download; OnSpotifyTag embeds metadata on completion.
// Returns the number of tracks successfully enqueued.
func (a *App) DownloadSpotifyTracks(tracks []SpotifyTrackRequest) (int, error) {
	if len(tracks) == 0 {
		return 0, fmt.Errorf("no tracks selected")
	}
	if !a.updater.IsInstalled() {
		return 0, fmt.Errorf("yt-dlp is not installed yet, please wait")
	}
	sp := a.settings.GetSpotify()
	if sp.ClientID == "" || sp.ClientSecret == "" {
		return 0, fmt.Errorf("Spotify API credentials are required: add them in the Spotify panel (developer.spotify.com/dashboard)")
	}

	resolveOpts := spotify.ResolveOptions{
		ClientID:      sp.ClientID,
		ClientSecret:  sp.ClientSecret,
		AudioProvider: sp.AudioProvider,
		CookieFile:    spotifyCookieFile(a.settings),
		Proxy:         spotifyProxy(a.settings),
	}
	verbose := func(line string) { wailsruntime.EventsEmit(a.ctx, "spotify:log", line) }

	enqueued := 0
	for _, tr := range tracks {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		ytURL, err := spotify.ResolveURL(ctx, tr.URL, resolveOpts, verbose)
		cancel()
		if err != nil {
			wailsruntime.EventsEmit(a.ctx, "spotify:error", "Could not match "+tr.URL+": "+err.Error())
			continue
		}
		a.queue.AddSpotify(ytURL, tr.URL, tr.Title, sp.Format, spotifyAudioQuality(sp.Bitrate))
		enqueued++
	}
	if enqueued == 0 {
		return 0, fmt.Errorf("no tracks could be matched; see the Log tab")
	}
	return enqueued, nil
}
