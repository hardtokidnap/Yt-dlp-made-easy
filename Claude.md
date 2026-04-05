# CLAUDE.md

## Project overview
Yt-dlp-made-easy is a Wails v2 desktop application providing a GUI for yt-dlp and FFmpeg. Backend is Go, frontend is vanilla JavaScript with Tailwind CSS. Output is a single ~8MB binary. yt-dlp and FFmpeg are not bundled, they are downloaded on demand to keep binary size minimal. Windows-first, but structured for cross-platform with build-tagged platform files.

---

## Absolute rules

- **Never edit `frontend/wailsjs/`**. Regenerated on every `wails dev` or `wails build`. Changes will be silently overwritten.
- **Never edit `go.sum`** manually.
- **Never bundle yt-dlp or FFmpeg** in the binary. Both are fetched at runtime into the app data directory.
- **Never use standard `bufio` line scanners for FFmpeg output.** FFmpeg uses `\r` to overwrite progress lines in-place. Use the `scanCRLF` function in `internal/converter/converter.go`. Standard scanners silently drop all progress events.
- **Never expose account ban risk silently.** Any UI flow involving browser cookies or PO tokens must warn users and suggest a throwaway account.
- **`go build` alone does not produce a working binary.** You must use `wails build` or `wails dev`, Wails embeds the frontend assets at build time.

---

## Mandatory development practices

- **Thread safety on settings:** always use the `Get*()`/`Set*()` methods on `*config.Settings`, which internally use `sync.RWMutex`. Never read or write struct fields directly across goroutines.
- **New settings fields require `mergeDefaults` updates:** add a corresponding zero-value check in `mergeDefaults()` in `internal/config/settings.go`. Older saved configs silently ignore new fields otherwise.
- **All Go-to-JS communication via Wails events:** use `wailsruntime.EventsEmit`. Do not invent new event names; the JS side hardcodes listeners. See `.claude/rules/event-contract.md` for the full contract.
- **Always default `UseNightly` to `true`:** stable yt-dlp releases frequently break YouTube extraction. This is intentional; do not change the default.
- **New contributors** must add their GitHub username to `Contributors` in `internal/util/meta.go` before opening a PR.

---

## Conventions

- **PRs** target the `Development` branch, not `main`.
- **Commits** follow [Conventional Commits](https://www.conventionalcommits.org/): `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `perf:`, `test:`.
- **Branches** are created from `Main`. Use descriptive names (e.g. `feat/oauth-login`, `fix/ffmpeg-progress-parsing`).
- **Comments** We want comments to explain *why*, as in why we did something, not *what we did*. The code should be readable on its own.
- **Go** follows standard Go conventions. Run `go vet` before submitting.
- **Frontend** is vanilla JS. No frameworks.

---

## Dev commands

```bash
wails dev                          # Hot reload dev mode
wails build                        # Production binary
wails build -ldflags="-s -w"       # Optimized/stripped release binary
go vet ./...                       # Required before submitting a PR
```

---

## Data storage paths (Windows)

All data lives under `%LOCALAPPDATA%\ytdlp-easy\` (resolved in `internal/util/paths.go`):

| Path constant       | File                 | Purpose                                                |
| ------------------- | -------------------- | ------------------------------------------------------ |
| `util.SettingsFile` | `settings.json`      | User preferences (`Settings` struct)                   |
| `util.HistoryFile`  | `history.jsonl`      | Download history (newline-delimited JSON, append-safe) |
| `util.QueueFile`    | `queue.json`         | Persisted download queue (survives restarts)           |
| `util.LogFile`      | `ytdlp.log`          | Debug log (only written when `VerboseLogging` is true) |
| `util.YtDlpPath`    | `yt-dlp.exe`         | Auto-downloaded yt-dlp binary                          |
| (none)              | `ffmpeg\ffmpeg.exe`  | Auto-downloaded FFmpeg                                 |
| (none)              | `ffmpeg\ffprobe.exe` | Auto-downloaded ffprobe                                |
| (none)              | `deno.exe`           | Bundled Deno JS runtime (optional)                     |

---

## Download item status flow

```
pending → downloading → completed  (moves to history, removed from queue)
                      → error      (moves to history, removed from queue)
                      → paused     (stays in queue, OS process suspended via NtSuspendProcess)
                      → stopped    (stays in queue, resume = fresh yt-dlp process)
```

`StatusStopped` items can be resumed: `Queue.Resume()` detects no active downloader and starts a fresh yt-dlp process. `StatusPaused` items have a live suspended process that gets `NtResumeProcess`.

---

## Platform-specific gotchas

- **Process suspension** uses `NtSuspendProcess`/`NtResumeProcess` from `ntdll.dll` via `syscall.NewLazyDLL`. Windows-only, lives in `downloader.go`.
  - Although, we at a later time want to add support for Linux and MacOS, so keep that in mind.
- **Console hiding:** `CREATE_NO_WINDOW` (0x08000000) + `HideWindow: true` on all spawned processes. Do NOT apply to GUI apps like explorer.exe.
- **`OpenFileInFolder`** bypasses Go's argument quoting by setting `SysProcAttr.CmdLine` directly. Explorer's `/select,<path>` flag breaks when Go wraps it in double quotes.
- **Deno extraction** uses PowerShell `Expand-Archive` rather than Go's `archive/zip`.
- **CodeQL** Go analysis runs on `windows-latest` because `golang.org/x/sys/windows` imports fail on Linux runners.
- **yt-dlp progress** comes from stdout, errors from stderr. Both scanned in separate goroutines. `parseOutput()` handles both pipes since yt-dlp sometimes sends errors to stdout.
- **XSS:** file paths and URLs in inline `onclick` handlers must be properly JS-string-escaped. This was a real past bug.
- **Security:** Keep in mind that this is a desktop application that will be run by users on their own computers. Users will insert links into the application, and we don't want a scenario where the application is used to download malicious code or other harmful content. 

---

## Known architectural decisions (don't "fix" these)

- **No database.** Settings = JSON, queue = JSON, history = JSONL. Intentionally simple.
- **History is separate from queue.** Completed/errored items move to history immediately. The download tab shows active queue + last 3 non-hidden history entries.
- **Queue persistence** saves on status transitions only, not every progress tick.
- **Single converter at a time.** `StartConversion` cancels any existing `a.converter` before starting. Same for batch.
- **`startup()` initialization order matters.** History must initialize before queue because `OnItemUpdate` closure references `a.history`. Do not reorder.
- **`UpdateMaxConcurrent`** replaces the semaphore channel wholesale. Active downloads continue unaffected; the new limit applies to future downloads only.

---

## Error classification

`ClassifyError()` in `internal/downloader/errors.go` runs compiled regex patterns against raw yt-dlp output. Returns `ErrorType` + `[]Solution`. Error types: `unknown`, `forbidden_403`, `rate_limit_429`, `age_restricted`, `geo_blocked`, `not_available`, `network`, `extractor_outdated`, `sign_in_required`, `cookie_database`.

Each `Solution.Action` is one of: `"apply_setting"`, `"open_settings"`, `"open_link"`, `"retry"`, `"update_ytdlp"`, `"open_log"`. The frontend handles these to apply fixes and retry automatically. `Solution.ActionData` carries the setting key or URL.