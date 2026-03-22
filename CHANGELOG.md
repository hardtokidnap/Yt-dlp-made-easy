# Changelog

All notable changes to YT-DLP Made Easy will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- Download queue persists across app restarts
- "Open Folder" button in the download queue highlights the downloaded file in your file manager
- Inline update indicator in the header bar shows update status without interrupting your workflow
- Per-item remove button and "Clear Queue" button for download queue management
- Audio/video badge on download queue items
- Verbose logging toggle (disabled by default, persistent)
- Platform-specific file/folder opening (Windows, macOS, Linux)

### Changed
- YT-DLP now defaults to the Nightly build instead of Stable for access to the latest fixes
- Download queue now shows only the video/song title instead of the full URL
- Update notifications replaced with a non-intrusive inline header indicator

### Fixed
- Shell/console windows no longer flash on screen during background operations (downloads, updates, runtime checks)
- "Open Folder" now correctly handles file paths with spaces on Windows
- Downloaded file path now tracks through post-processing steps (merging, audio extraction, file moves)

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
