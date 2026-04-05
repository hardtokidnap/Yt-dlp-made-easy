---
globs:
  - "*.go"
  - "internal/**"
---

# Backend reference

## Key file map

```
app.go                              Wails bridge — every exported method here becomes callable
                                    from JS via the auto-generated wailsjs/ bindings
main.go                             Bootstrap, Wails window config (1024x768, min 800x600),
                                    drag-and-drop enabled, frontend assets embedded via go:embed
open_windows.go                     OpenFile/OpenFolder/OpenFileInFolder/OpenURL for Windows.
                                    OpenFileInFolder uses raw SysProcAttr.CmdLine to bypass
                                    Go's argument quoting, which breaks explorer /select
open_darwin.go / open_linux.go      Same interface, xdg-open / open implementations

internal/config/settings.go         Settings struct, Load(), Save(), mergeDefaults(),
                                    thread-safe Get*/Set* accessors
internal/downloader/args.go         BuildArgs() — constructs yt-dlp CLI args from Settings.
                                    CookiesFile takes priority over CookiesBrowser.
                                    PO Token formatting logic lives here.
internal/downloader/downloader.go   Spawns yt-dlp process. Pause/Resume use Windows NT APIs
                                    NtSuspendProcess/NtResumeProcess via ntdll.dll.
                                    parseOutput() extracts progress, title, file path, errors
                                    from yt-dlp stdout/stderr line by line.
internal/downloader/errors.go       ClassifyError() — regex-based classification of yt-dlp
                                    error strings into ErrorType constants with Solution lists.
                                    Patterns are compiled once at package level.
internal/downloader/item.go         Item struct and Status constants. SetErrorFromString()
                                    auto-classifies and populates Suggestions.
internal/downloader/queue.go        Concurrent download queue using a channel semaphore.
                                    Downloads exceeding MaxConcurrentDownloads block until
                                    a slot opens. Persists to queue.json on status changes
                                    (not on every progress tick). LoadPersistedItems() resets
                                    mid-download items to StatusStopped on restart.
internal/converter/args.go          BuildArgs() for FFmpeg. ConversionOptions struct.
                                    Preset definitions (GetPresets()). ParseTimeToSeconds().
                                    Uses -ss before -i (input-side seek) for fast trim.
                                    Uses -t (duration) not -to (absolute) for trim end.
internal/converter/batch.go         BatchQueue — sequential conversion of multiple files.
                                    Job IDs include random component to avoid millisecond collisions.
internal/converter/converter.go     Single-file FFmpeg conversion. Contains scanCRLF (critical).
                                    SetEffectiveDuration() overrides progress denominator for trims.
                                    Progress only emitted on lines containing "time=".
internal/converter/ffmpeg.go        FFmpeg/ffprobe binary detection and auto-download from
                                    BtbN/FFmpeg-Builds. Extracts to temp file first to prevent
                                    partial installs from being seen as valid by IsFFmpegInstalled().
internal/converter/probe.go         ProbeFile() — runs ffprobe -show_format -show_streams,
                                    parses JSON output into MediaInfo struct.
internal/converter/cmd_windows.go   hiddenCmd() — HideWindow + CREATE_NO_WINDOW for all
                                    FFmpeg/ffprobe processes to suppress console flash.
internal/converter/cmd_other.go     Plain exec.Command for non-Windows.
internal/history/history.go         JSONL-backed history. Add() is append-only (crash-safe).
                                    Remove()/HideFromQueue()/ClearOld() rewrite the file.
                                    Mutations roll back in-memory state on disk write failure.
                                    stampFileExists() checks disk on every read result.
internal/jsruntime/runtime.go       Detects Deno/Node/Bun. Bundled Deno at AppDataDir/deno.exe
                                    takes priority over system PATH.
                                    GetYtDlpRuntimeArgs() only passes --js-runtimes when needed.
                                    Deno extraction on Windows uses PowerShell Expand-Archive.
internal/updater/updater.go         yt-dlp version management. Downloads to .tmp first,
                                    retries rename up to 5 times (antivirus file locks).
                                    Update() tries self-update first, falls back to fresh download.
internal/util/meta.go               AppName, AppVersion, Contributors slice
internal/util/paths.go              All app data path constants, initialized in init()
```

## Settings struct (quick reference)

```go
Settings {
  General: {
    SaveFolder             string  // default: %USERPROFILE%\Downloads
    Theme                  string  // "system" | "light" | "dark"
    MaxConcurrentDownloads int     // default: 3
    ClipboardMonitoring    bool    // default: true
    NotificationsEnabled   bool    // default: true
    CheckUpdatesOnStart    bool    // default: true
    VerboseLogging         bool    // default: false
  }
  Download: {
    Quality        string  // "best"|"4K"|"1440p"|"1080p"|"720p"|"480p"|"360p"
    Format         string  // "mp4" etc — merge output format
    AudioFormat    string  // "mp3" etc
    AudioQuality   string  // "192" (kbps)
    EmbedThumbnail bool
    EmbedMetadata  bool
    EmbedChapters  bool
    Sponsorblock   bool    // default: false
  }
  Network: {
    RateLimit string  // yt-dlp --limit-rate value, empty = unlimited
    Proxy     string
    Retries   int     // default: 10
  }
  Auth: {
    CookiesBrowser string  // "none"|"chrome"|"firefox"|"edge"|"brave"
    CookiesFile    string  // path to cookies.txt — takes priority over CookiesBrowser
    POToken        string
    PlayerClient   string  // "default"|"mweb"|"web_creator"|"ios"|"android"
  }
  Advanced: {
    UseNightly     bool    // default: TRUE — do not change default
    OutputTemplate string  // default: "%(title)s.%(ext)s"
    ExtraArgs      string  // raw yt-dlp args, quote-aware split
    JSRuntime      string  // "auto"|"deno"|"node"|"bun"|path
  }
}
```