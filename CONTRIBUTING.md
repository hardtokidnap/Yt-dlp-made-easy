# Contributing to YT-DLP Made Easy

Thanks for your interest in contributing. This is a hobby project so response times may vary, but all contributions are welcome.

## Getting Started

### Prerequisites

- Go 1.21+
- Node.js 18+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

### Setup

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest

git clone https://github.com/hardtokidnap/Yt-dlp-made-easy.git
cd Yt-dlp-made-easy

go mod tidy
cd frontend && npm install && cd ..
```

### Development

```bash
wails dev          # Hot reload dev mode
wails build        # Production binary
```

> **Important:** `go build` and `npm run build` alone won't produce a working desktop binary. You must use `wails build` or `wails dev`.

## Project Layout

```
app.go                    # Wails bridge — all methods exposed to the frontend
open_{windows,darwin,linux}.go  # Platform-specific file/folder operations
internal/
  config/                 # Settings persistence
  converter/              # FFmpeg conversion engine
  downloader/             # Queue, download items, error classification
  history/                # Download history
  jsruntime/              # JS runtime detection (Deno, Node, Bun)
  updater/                # yt-dlp version management
  util/                   # Shared paths and helpers
frontend/
  src/main.js             # All UI logic (vanilla JS, no framework)
  index.html              # Single-page shell
```

## How to Contribute

### Reporting Bugs

Open an issue with:
- What you did
- What you expected
- What happened instead
- Your Windows version and any relevant logs (enable Verbose Logging in Settings)

### Submitting Changes

1. Fork the repo and create a branch from `Development`
2. Make your changes
3. Run `go vet ./...` to check for issues
4. Test with `wails dev`
5. Open a PR against `Development` (not `main`)

### Conventions

- **Commits:** Use [Conventional Commits](https://www.conventionalcommits.org/) — `feat:`, `fix:`, `chore:`, `docs:`, etc.
- **Comments:** Explain *why*, not *what*. The code should be readable on its own.
- **Go:** Follow standard Go conventions. Run `go vet` before submitting.
- **Frontend:** Vanilla JS — no frameworks. Keep it simple.

### What's Useful

- Bug reports with reproduction steps
- Fixes for [known issues](https://github.com/hardtokidnap/Yt-dlp-made-easy/releases)
- Performance improvements
- Linux/macOS testing and fixes (currently Windows-only)
- UI/UX improvements

### AI-Assisted / Vibe Coding

AI-assisted contributions are welcome — how you write the code is your business. But you're responsible for what you submit. Every PR is reviewed the same way regardless of how it was written. If the code is clean, correct, and well-structured, it gets in. If it's not, it doesn't.

All PRs go through automated code analysis that checks for common AI-generated patterns — hallucinated APIs, unused imports, dead code paths, inconsistent error handling, and copy-pasted boilerplate that doesn't match the project's style. These checks run before human review even begins. On top of that, every PR is manually reviewed with attention to architectural fit, not just "does it compile."

If you're using AI tools, treat their output as a first draft, not a finished product. Read every line, test every path, and be ready to explain any decision in your code. If you can't explain why something is there, remove it.

We won't accept:
- Code that degrades app quality, performance, or stability
- Unreviewed AI output dumped into a PR
- Changes outside the scope of this app (this is a yt-dlp GUI, not a general media platform)

### Please Don't

- Add large dependencies without discussion first
- Refactor working code without a clear reason
- Add features without opening an issue to discuss scope
- Submit code you haven't tested or don't understand

## Architecture Notes

- **Backend → Frontend:** Wails exposes Go methods as JS functions. All methods in `app.go` are callable from `frontend/src/main.js` via auto-generated bindings in `frontend/wailsjs/`.
- **Frontend → Backend events:** Real-time updates (download progress, conversion progress) use `EventsEmit`/`EventsOn`.
- **Platform code:** Windows-specific code (console hiding, explorer integration) uses build tags (`//go:build windows`). The app is Windows-first but structured for future cross-platform support.
- **No database:** Settings are JSON, queue persistence is JSON, history is flat file. Intentionally simple.

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
