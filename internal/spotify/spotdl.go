package spotify

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

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

// AuthOptions selects how spotdl authenticates to the Spotify Web API.
type AuthOptions struct {
	ClientID     string
	ClientSecret string
	// UserAuth runs the OAuth (Authorization Code) flow instead of client
	// credentials. Required to read playlist items and the user library, which
	// Spotify no longer exposes to app-only tokens. The first call opens a
	// browser; the token is cached at CachePath so later calls are silent.
	UserAuth  bool
	CachePath string
}

// authArgs returns the spotdl Spotify-auth flags for the given mode.
//   - No creds: best-effort SpotipyFree (only --no-cache); user-auth impossible.
//   - Client credentials (tracks/albums): official API + creds + --no-cache, so
//     a stale spotipy token cache cannot mask the supplied credentials.
//   - User auth (playlists/liked): official API + creds + --user-auth, and the
//     OAuth token IS cached (no --no-cache) so the user does not re-login.
func authArgs(o AuthOptions) []string {
	if o.ClientID == "" || o.ClientSecret == "" {
		return []string{"--no-cache"}
	}
	args := []string{"--use-official-api", "--client-id", o.ClientID, "--client-secret", o.ClientSecret}
	if o.UserAuth {
		args = append(args, "--user-auth")
		if o.CachePath != "" {
			args = append(args, "--cache-path", o.CachePath)
		}
		return args
	}
	return append(args, "--no-cache")
}

// redactCmd renders a spotdl arg list as a loggable command string with the
// client-id / client-secret values masked. Never log raw args directly.
func redactCmd(args []string) string {
	out := make([]string, len(args))
	copy(out, args)
	for i := 0; i < len(out)-1; i++ {
		if out[i] == "--client-id" || out[i] == "--client-secret" {
			out[i+1] = "***"
		}
	}
	return "$ python " + strings.Join(out, " ")
}

// parseResolvedURL extracts the matched audio URL from `spotdl url` stdout.
// spotdl prints status/warning lines plus one bare URL per resolved track; we
// take the first line that is itself a URL (not a line that merely contains
// one, like "Processing query: https://open.spotify.com/...").
func parseResolvedURL(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
			return line
		}
	}
	return ""
}

// Track is one Spotify track resolved by spotdl.
type Track struct {
	URL         string  `json:"url"`
	Title       string  `json:"title"`
	Artist      string  `json:"artist"`
	Album       string  `json:"album"`
	DurationSec float64 `json:"duration_sec"`
}

// PreviewOptions controls how PreviewURL contacts external services. Same
// audio-provider / cookie / proxy knobs as a Download job, since spotdl runs
// the same ytmusic-connection probe during 'save' as it does during 'download'.
type PreviewOptions struct {
	AudioProvider string
	CookieFile    string
	Proxy         string
	ClientID      string
	ClientSecret  string
	// UserAuth / CachePath enable the OAuth flow for playlist + liked-songs URLs
	// (app-only tokens cannot read playlist items). Ignored for track/album URLs.
	UserAuth  bool
	CachePath string
}

// PreviewURL runs `spotdl save` on the given URL and returns the resolved tracks
// without downloading audio. Used to populate a track-picker UI. The onLog
// callback (optional) is invoked for every spotdl stdout/stderr line so the
// UI can stream progress instead of waiting for exit.
func PreviewURL(ctx context.Context, url string, prevOpts PreviewOptions, onLog func(string)) ([]Track, error) {
	if onLog == nil {
		onLog = func(string) {}
	}
	tmpFile, err := os.CreateTemp("", "spotdl-preview-*.spotdl")
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
	args = append(args, authArgs(AuthOptions{
		ClientID:     prevOpts.ClientID,
		ClientSecret: prevOpts.ClientSecret,
		UserAuth:     prevOpts.UserAuth,
		CachePath:    prevOpts.CachePath,
	})...)
	if prevOpts.CookieFile != "" {
		args = append(args, "--cookie-file", prevOpts.CookieFile)
	}
	if prevOpts.Proxy != "" {
		args = append(args, "--proxy", prevOpts.Proxy)
	}
	onLog(redactCmd(args))
	if _, err := runSpotdl(ctx, args, onLog); err != nil {
		return nil, fmt.Errorf("spotdl save failed: %w", err)
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

// ResolveOptions controls how ResolveURL contacts Spotify + the matching provider.
type ResolveOptions struct {
	ClientID      string
	ClientSecret  string
	AudioProvider string // matching provider; default "youtube-music"
	CookieFile    string
	Proxy         string
}

// ResolveURL runs `spotdl url` to match a single Spotify track URL to an
// audio-source URL (e.g. music.youtube.com) without downloading. The onLog
// callback (optional) receives the redacted command and every stdout/stderr
// line for verbose troubleshooting.
func ResolveURL(ctx context.Context, spotifyURL string, opts ResolveOptions, onLog func(string)) (string, error) {
	if onLog == nil {
		onLog = func(string) {}
	}
	provider := opts.AudioProvider
	if provider == "" {
		provider = "youtube-music"
	}
	args := []string{"-m", "spotdl", "url", spotifyURL, "--audio", provider}
	args = append(args, authArgs(AuthOptions{ClientID: opts.ClientID, ClientSecret: opts.ClientSecret})...)
	if opts.CookieFile != "" {
		args = append(args, "--cookie-file", opts.CookieFile)
	}
	if opts.Proxy != "" {
		args = append(args, "--proxy", opts.Proxy)
	}
	onLog(redactCmd(args))

	out, err := runSpotdl(ctx, args, onLog)
	if err != nil {
		return "", fmt.Errorf("spotdl url failed: %w", err)
	}
	url := parseResolvedURL(out)
	if url == "" {
		return "", fmt.Errorf("spotdl produced no match for %s", spotifyURL)
	}
	return url, nil
}

// MetaOptions controls the spotdl meta tagging pass.
type MetaOptions struct {
	ClientID     string
	ClientSecret string
	FFmpegPath   string
	CookieFile   string
	Proxy        string
}

// ApplyMetadata runs `spotdl meta` on an already-downloaded audio file to embed
// full Spotify metadata (tags + cover art + lyrics). spotdl matches the file to
// Spotify by filename, so the file should be named "<artist> - <title>". The
// onLog callback (optional) receives the redacted command + raw output.
func ApplyMetadata(ctx context.Context, filePath string, opts MetaOptions, onLog func(string)) error {
	if onLog == nil {
		onLog = func(string) {}
	}
	args := []string{"-m", "spotdl", "meta", filePath}
	args = append(args, authArgs(AuthOptions{ClientID: opts.ClientID, ClientSecret: opts.ClientSecret})...)
	if opts.FFmpegPath != "" {
		args = append(args, "--ffmpeg", opts.FFmpegPath)
	}
	if opts.CookieFile != "" {
		args = append(args, "--cookie-file", opts.CookieFile)
	}
	if opts.Proxy != "" {
		args = append(args, "--proxy", opts.Proxy)
	}
	onLog(redactCmd(args))
	if _, err := runSpotdl(ctx, args, onLog); err != nil {
		return fmt.Errorf("spotdl meta failed: %w", err)
	}
	return nil
}

// runSpotdl starts `python <args>` with the bundled-Deno PATH, streams every
// stdout/stderr line to onLog, and returns the combined output. Used by the
// url/meta/save wrappers. The mutex guards buf because both scan goroutines
// write to it concurrently.
func runSpotdl(ctx context.Context, args []string, onLog func(string)) (string, error) {
	cmd := hiddenCmdCtx(ctx, util.PythonExe, args...)
	cmd.Env = envWithBundledDeno()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}

	var buf strings.Builder
	var mu sync.Mutex
	scan := func(r *bufio.Scanner) {
		for r.Scan() {
			line := r.Text()
			onLog(line)
			mu.Lock()
			buf.WriteString(line)
			buf.WriteByte('\n')
			mu.Unlock()
		}
	}
	done := make(chan struct{}, 2)
	go func() { scan(bufio.NewScanner(stdout)); done <- struct{}{} }()
	go func() { scan(bufio.NewScanner(stderr)); done <- struct{}{} }()
	<-done
	<-done
	waitErr := cmd.Wait()
	mu.Lock()
	out := buf.String()
	mu.Unlock()
	if waitErr != nil {
		return out, fmt.Errorf("%v: %s", waitErr, out)
	}
	return out, nil
}
