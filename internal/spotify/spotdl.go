package spotify

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"ytdlp-easy/internal/jsruntime"
	"ytdlp-easy/internal/util"
)

// envWithBundledDeno returns a copy of os.Environ() with the directory holding
// the app's bundled deno.exe prepended to PATH, so spotdl's transitive yt-dlp
// finds the same Deno the main downloader uses instead of failing on JS
// challenges (e.g. "made for kids" videos).
func envWithBundledDeno() []string {
	deno := jsruntime.BundledDenoPath()
	if _, err := os.Stat(deno); err != nil {
		return os.Environ()
	}
	dir := filepath.Dir(deno)
	env := os.Environ()
	for i, kv := range env {
		if strings.HasPrefix(strings.ToUpper(kv), "PATH=") {
			env[i] = "PATH=" + dir + string(os.PathListSeparator) + kv[len("PATH="):]
			return env
		}
	}
	return append(env, "PATH="+dir)
}

// Track is one Spotify track resolved by spotdl.
type Track struct {
	URL         string  `json:"url"`
	Title       string  `json:"title"`
	Artist      string  `json:"artist"`
	Album       string  `json:"album"`
	DurationSec float64 `json:"duration_sec"`
}

// Options for a Spotify download job.
type Options struct {
	URLs         []string `json:"urls"`
	OutputFolder string   `json:"output_folder"`
	Format       string   `json:"format"`
	Bitrate      string   `json:"bitrate"`
	Threads      int      `json:"threads"`
	FFmpegPath   string   `json:"ffmpeg_path"`
	// AudioProvider lets us route around YouTube Music blocks. Accepted by
	// spotdl --audio: "youtube-music" (default), "youtube", "piped",
	// "soundcloud", "bandcamp", "slider-kz". Empty = spotdl default.
	AudioProvider string `json:"audio_provider"`
	// CookieFile is a yt-dlp cookies.txt path. Forwarded to spotdl via
	// --cookie-file. Reuses the same Auth.CookiesFile the main downloader
	// uses; spotdl consumes it independently.
	CookieFile string `json:"cookie_file"`
	// Proxy is an HTTP/HTTPS proxy URL. Forwarded to spotdl via --proxy.
	// Reuses Network.Proxy from the main settings.
	Proxy string `json:"proxy"`
}

// Job represents an in-flight spotdl download.
type Job struct {
	ID        string    `json:"id"`
	URLs      []string  `json:"urls"`
	Status    string    `json:"status"` // pending | running | completed | failed | cancelled
	Completed int       `json:"completed"`
	Failed    int       `json:"failed"`
	Total     int       `json:"total"`
	StartedAt time.Time `json:"started_at"`
	LastLine  string    `json:"last_line"`
	mu        sync.Mutex
	cancel    context.CancelFunc
}

// PreviewOptions controls how PreviewURL contacts external services. Same
// audio-provider / cookie / proxy knobs as a Download job, since spotdl runs
// the same ytmusic-connection probe during 'save' as it does during 'download'.
type PreviewOptions struct {
	AudioProvider string
	CookieFile    string
	Proxy         string
}

// PreviewURL runs `spotdl save` on the given URL and returns the resolved tracks
// without downloading audio. Used to populate a track-picker UI.
func PreviewURL(ctx context.Context, url string, prevOpts PreviewOptions) ([]Track, error) {
	tmpFile, err := os.CreateTemp("", "spotdl-preview-*.json")
	if err != nil {
		return nil, err
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	args := []string{"-m", "spotdl", "save", url, "--save-file", tmpPath}
	if prevOpts.AudioProvider != "" {
		args = append(args, "--audio", prevOpts.AudioProvider)
	}
	if prevOpts.CookieFile != "" {
		args = append(args, "--cookie-file", prevOpts.CookieFile)
	}
	if prevOpts.Proxy != "" {
		args = append(args, "--proxy", prevOpts.Proxy)
	}
	cmd := hiddenCmd(util.PythonExe, args...)
	cmd.Env = envWithBundledDeno()
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("spotdl save failed: %v: %s", err, string(out))
	}

	raw, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, err
	}
	// spotdl save emits a JSON array of song dicts. Field names follow its own
	// internal schema; we extract a best-effort subset and silently skip fields
	// we cannot parse rather than erroring on schema drift.
	var rawTracks []map[string]interface{}
	if err := json.Unmarshal(raw, &rawTracks); err != nil {
		return nil, fmt.Errorf("parse spotdl save output: %w", err)
	}
	tracks := make([]Track, 0, len(rawTracks))
	for _, rt := range rawTracks {
		t := Track{}
		if v, ok := rt["url"].(string); ok {
			t.URL = v
		}
		if v, ok := rt["name"].(string); ok {
			t.Title = v
		}
		if v, ok := rt["artist"].(string); ok {
			t.Artist = v
		} else if arr, ok := rt["artists"].([]interface{}); ok && len(arr) > 0 {
			if a0, ok := arr[0].(string); ok {
				t.Artist = a0
			}
		}
		if v, ok := rt["album_name"].(string); ok {
			t.Album = v
		}
		if v, ok := rt["duration"].(float64); ok {
			t.DurationSec = v
		}
		tracks = append(tracks, t)
	}
	return tracks, nil
}

// Download spawns spotdl to download the given URLs. progress is called on
// every parsed line + on every status transition. Returns when the process
// exits or context is cancelled.
func Download(ctx context.Context, opts Options, progress func(*Job)) (*Job, error) {
	ctx, cancel := context.WithCancel(ctx)
	job := &Job{
		ID:        strconv.FormatInt(time.Now().UnixNano(), 36),
		URLs:      opts.URLs,
		Status:    "running",
		Total:     len(opts.URLs),
		StartedAt: time.Now(),
		cancel:    cancel,
	}
	emit := func() {
		if progress != nil {
			progress(job)
		}
	}
	emit()

	args := []string{"-m", "spotdl", "download"}
	args = append(args, opts.URLs...)
	if opts.OutputFolder != "" {
		if err := os.MkdirAll(opts.OutputFolder, 0755); err != nil {
			job.Status = "failed"
			emit()
			return job, fmt.Errorf("create output folder: %w", err)
		}
		args = append(args, "--output", filepath.Join(opts.OutputFolder, "{artists} - {title}.{output-ext}"))
	}
	if opts.Format != "" {
		args = append(args, "--format", opts.Format)
	}
	if opts.Bitrate != "" && opts.Bitrate != "auto" {
		args = append(args, "--bitrate", opts.Bitrate)
	}
	if opts.Threads > 0 {
		args = append(args, "--threads", strconv.Itoa(opts.Threads))
	}
	if opts.FFmpegPath != "" {
		args = append(args, "--ffmpeg", opts.FFmpegPath)
	}
	if opts.AudioProvider != "" {
		args = append(args, "--audio", opts.AudioProvider)
	}
	if opts.CookieFile != "" {
		args = append(args, "--cookie-file", opts.CookieFile)
	}
	if opts.Proxy != "" {
		args = append(args, "--proxy", opts.Proxy)
	}

	cmd := hiddenCmdCtx(ctx, util.PythonExe, args...)
	cmd.Env = envWithBundledDeno()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		job.Status = "failed"
		emit()
		return job, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		job.Status = "failed"
		emit()
		return job, err
	}
	if err := cmd.Start(); err != nil {
		job.Status = "failed"
		emit()
		return job, err
	}

	scanLines := func(s *bufio.Scanner) {
		for s.Scan() {
			line := s.Text()
			parseSpotdlLine(line, job)
			emit()
		}
	}
	go scanLines(bufio.NewScanner(stdout))
	go scanLines(bufio.NewScanner(stderr))

	err = cmd.Wait()
	if ctx.Err() == context.Canceled {
		job.Status = "cancelled"
		emit()
		return job, ctx.Err()
	}
	if err != nil {
		job.Status = "failed"
		emit()
		return job, err
	}
	job.Status = "completed"
	emit()
	return job, nil
}

// Cancel a running job. Safe to call multiple times.
func (j *Job) Cancel() {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.cancel != nil {
		j.cancel()
	}
}

// parseSpotdlLine looks for known stdout/stderr patterns and updates the job
// counters. Conservative on purpose. Unknown lines just update LastLine so the
// UI shows fresh status without us inventing fake state transitions.
func parseSpotdlLine(line string, job *Job) {
	job.mu.Lock()
	defer job.mu.Unlock()
	job.LastLine = line
	low := strings.ToLower(line)
	switch {
	case strings.Contains(low, "downloaded "), strings.Contains(low, "downloaded:"):
		job.Completed++
	case strings.Contains(low, "skipped "):
		job.Completed++
	case strings.Contains(low, "error:"), strings.Contains(low, "lookuperror"):
		job.Failed++
	}
}
