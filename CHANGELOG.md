# Changelog

All notable changes to YT-DLP Made Easy will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

## [V3.1.1] - 2026-03-28

### Fixed
- "Open in Folder" now correctly reveals the file in Explorer instead of opening it directly ([#16](https://github.com/hardtokidnap/Yt-dlp-made-easy/pull/16))
- File existence checks now run consistently across all history methods, not just the download tab ([#16](https://github.com/hardtokidnap/Yt-dlp-made-easy/pull/16))
- History entries with deleted files are now grayed out with a "File missing" label ([#16](https://github.com/hardtokidnap/Yt-dlp-made-easy/pull/16))
- Extra args autocomplete now keeps suggesting after the first selection — replaced native `<datalist>` with a custom multi-token dropdown ([#18](https://github.com/hardtokidnap/Yt-dlp-made-easy/pull/18))

### Changed
- CodeQL workflow split into separate Go and JS analysis jobs; Go job runs on `windows-latest` to resolve `golang.org/x/sys/windows` import errors
- CodeQL actions bumped from v3 to v4

## [V3.1.0] - 2026-03-26

### Added
- Download queue now persists across app restarts — your queue is saved to disk and restored on launch
- "Open Folder" button on completed downloads opens the file manager and highlights the file
- Per-item remove button (✕) on completed/stopped downloads and "Clear Queue" button to bulk-remove finished items
- Audio/video badge on each download queue item so you can tell at a glance what type it is
- Verbose logging toggle in Settings (disabled by default, persistent) for debugging
- FFmpeg converter tab — convert media files with presets (Video to MP3, MP4, MKV, WebM, FLAC, WAV), codec options, custom args, and real-time progress
- Automatic FFmpeg download when opening the Convert tab for the first time
- "Pick a recent download" dropdown in converter to convert files straight from the queue
- JSONL-backed download history with search, export, and individual delete ([#12](https://github.com/hardtokidnap/Yt-dlp-made-easy/pull/12))
- Re-download button on completed and history items ([#12](https://github.com/hardtokidnap/Yt-dlp-made-easy/pull/12))
- Dismiss (hide) individual history entries from the download tab without deleting them ([#12](https://github.com/hardtokidnap/Yt-dlp-made-easy/pull/12))
- Quote-aware custom FFmpeg argument parsing — quoted values like `-metadata title="My Video"` are handled correctly ([#9](https://github.com/hardtokidnap/Yt-dlp-made-easy/pull/9))

### Changed
- Default yt-dlp build switched from Stable to Nightly for access to the latest fixes
- Update check replaced the blocking popup (`confirm()` dialog) with a non-intrusive inline indicator in the header bar
- Download queue cards now show only the video/song title instead of the full URL
- Queue now holds only active downloads — completed/errored items move to history automatically ([#12](https://github.com/hardtokidnap/Yt-dlp-made-easy/pull/12))
- Download tab shows active queue plus 3 most recent history entries ([#12](https://github.com/hardtokidnap/Yt-dlp-made-easy/pull/12))
- Converter completion now shows a persistent result panel with "Open in Folder" / "Convert Another" instead of a 3-second auto-dismiss ([#8](https://github.com/hardtokidnap/Yt-dlp-made-easy/pull/8))
- Regex patterns compiled once at package level instead of per yt-dlp output line ([#14](https://github.com/hardtokidnap/Yt-dlp-made-easy/pull/14))
- History mutations roll back in-memory state on disk write failure ([#14](https://github.com/hardtokidnap/Yt-dlp-made-easy/pull/14))

### Fixed
- Shell/console windows no longer flash on screen during background operations (downloads, updates, runtime checks)
- "Open Folder" now correctly handles file paths with spaces on Windows
- "Open Folder" now points to the final converted file instead of intermediate files
- FFmpeg converter progress was stuck at 0% because progress uses `\r` carriage returns — added custom scanner ([#8](https://github.com/hardtokidnap/Yt-dlp-made-easy/pull/8))
- FFmpeg duration parsing now handles variable fractional digits across different builds ([#8](https://github.com/hardtokidnap/Yt-dlp-made-easy/pull/8))
- Log panel no longer jumps to bottom while scrolling through earlier entries ([#10](https://github.com/hardtokidnap/Yt-dlp-made-easy/pull/10))
- Title extraction now works for all yt-dlp output formats (Merger, ExtractAudio, MoveFiles, already downloaded) ([#11](https://github.com/hardtokidnap/Yt-dlp-made-easy/pull/11))
- Fixed nil slice → JSON null crash when history is empty ([#12](https://github.com/hardtokidnap/Yt-dlp-made-easy/pull/12))
- XSS via special characters in URL/path onclick handlers — proper JS string escaping ([#14](https://github.com/hardtokidnap/Yt-dlp-made-easy/pull/14))
- `resetConvertUI` not accessible from inline onclick handlers ([#14](https://github.com/hardtokidnap/Yt-dlp-made-easy/pull/14))
- Dismiss button not updating UI immediately ([#14](https://github.com/hardtokidnap/Yt-dlp-made-easy/pull/14))

### Security
- Bumped golang.org/x/net from 0.35.0 to 0.38.0 ([#4](https://github.com/hardtokidnap/Yt-dlp-made-easy/pull/4))
- Bumped vite from 3.2.11 to 5.4.21 ([#6](https://github.com/hardtokidnap/Yt-dlp-made-easy/pull/6))

## [V2] - 2026-01-29

### Added
- Smart error classification with guided recovery suggestions for common download failures
- JavaScript runtime detection and automatic management (Deno) for sites that require it
- Autocomplete for yt-dlp command-line arguments
- Concurrent download queue with pause, resume, and cancel support
- Theme support (Dark, Light, Nord, Dracula, Solarized, Monokai)
- Clipboard monitoring for automatic URL detection
- Configurable maximum concurrent downloads
- Desktop notifications for completed downloads

### Changed
- Complete rewrite from Python to Go with a native desktop UI using Wails
- Modern single-binary desktop application replacing the previous command-line Python tool

### Removed
- Python dependency and command-line interface

## [Release] - 2025-04-27

### Added
- Initial release of YT-DLP Made Easy
- Basic video and audio downloading via yt-dlp
- Simple graphical interface for yt-dlp
