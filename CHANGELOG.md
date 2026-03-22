# Changelog

All notable changes to YT-DLP Made Easy will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- Download queue now persists across app restarts — your queue is saved to disk and restored on launch
- "Open Folder" button on completed downloads opens the file manager and highlights the file
- Per-item remove button (✕) on completed/stopped downloads and "Clear Queue" button to bulk-remove finished items
- Audio/video badge on each download queue item so you can tell at a glance what type it is
- Verbose logging toggle in Settings (disabled by default, persistent) for debugging
- FFmpeg converter tab — convert media files with presets (Video to MP3, MP4, MKV, WebM, FLAC, WAV), codec options, custom args, and real-time progress
- Automatic FFmpeg download when opening the Convert tab for the first time
- "Pick a recent download" dropdown in converter to convert files straight from the queue

### Changed
- Default yt-dlp build switched from Stable to Nightly for access to the latest fixes
- Update check replaced the blocking popup (`confirm()` dialog) with a non-intrusive inline indicator in the header bar
- Download queue cards now show only the video/song title instead of the full URL

### Fixed
- Shell/console windows no longer flash on screen during background operations (downloads, updates, runtime checks)
- "Open Folder" now correctly handles file paths with spaces on Windows
- "Open Folder" now points to the final converted file instead of intermediate files

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
