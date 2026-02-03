# YT-DLP Made Easy

A modern, lightweight GUI wrapper for [yt-dlp](https://github.com/yt-dlp/yt-dlp) built with Go and Wails. Single ~8MB binary, no external dependencies.

> [!NOTE] 
> This is a hobby project. Issues may not be addressed.

> [!WARNING]
> Some antivirus software may flag this app due to browser cookie access. This is a false positive - the app reads cookies locally to authenticate with YouTube (standard yt-dlp feature). These cookies never leave your machine. They're already stored as plaintext SQLite files that any application can read. Verify via VirusTotal or build from source.

[VirusTotal scan](https://www.virustotal.com/gui/file/a9c2529b618c98145e3268104e3bd00bf7529e71f3b721c503f41e268d627e25?nocache=1)

---

## Features

### Download Management
- **Concurrent downloads** with configurable queue size
- **Pause and resume** downloads without losing progress
- **Real-time progress** with speed and ETA display
- **Playlist support** with individual item tracking
- **Quality selection** from 360p to 4K
- **Audio extraction** with format and quality options

### Smart Error Handling
- **Automatic error classification** detects 403 blocks, rate limits, age restrictions, and more
- **Guided recovery** with actionable fix suggestions
- **One-click fixes** that apply settings and retry automatically

### Authentication
- **Browser cookie import** from Chrome, Firefox, Edge, Brave
- **Cookies file support** for explicit authentication
- **PO Token support** for age-restricted content
- **Player client switching** (web, mweb, ios, android) to bypass blocks

### Advanced Options
- **SponsorBlock integration** to skip sponsored segments
- **Stable/Nightly builds** with automatic updates
- **JavaScript runtime management** (Deno, Node.js, Bun) with bundled Deno download
- **Custom yt-dlp arguments** with autocomplete suggestions
- **Clipboard monitoring** for automatic URL detection
- **Download history** with search, filter, and re-download
> [!NOTE]
> Download history is currently broken while I work on a solution that can use either sqlite or a simple csv due to bloating possibilites and JSON corruption for users with high usage.
---

## Installation

Download the latest release from the [Releases](https://github.com/hardtokidnap/Yt-dlp-made-easy/releases) page.

The application will automatically download yt-dlp on first launch.

---

## Building from Source

### Prerequisites

- Go 1.21+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)
- Node.js 18+

### Setup

```bash
# Install Wails
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Clone repository
git clone https://github.com/hardtokidnap/yt-dlp-easy.git
cd yt-dlp-easy

# Install dependencies
go mod tidy
cd frontend && npm install && cd ..
```

### Commands

| Command | Description |
|---------|-------------|
| `wails dev` | Run in development mode with hot reload |
| `wails build` | Build production binary |
| `wails build -ldflags="-s -w"` | Build optimized binary (smaller size) |

---

## Project Structure

```
.
├── app.go                 # Wails application entry and frontend bindings
├── main.go                # Application bootstrap
├── frontend/              # HTML, CSS, JavaScript UI
│   └── src/
│       ├── main.js        # Application logic
│       └── style.css      # Tailwind styles
└── internal/
    ├── config/            # Settings management
    ├── downloader/        # Queue, item, args, error handling
    ├── history/           # Download history tracking
    ├── jsruntime/         # JavaScript runtime detection and management
    ├── updater/           # yt-dlp version management
    └── util/              # Shared utilities and paths
```

---

## Configuration

Settings are stored in:
- **Windows:** `%APPDATA%\ytdlp-easy\settings.json`
- **macOS:** `~/Library/Application Support/ytdlp-easy/settings.json`
- **Linux:** `~/.config/ytdlp-easy/settings.json`

---

## Troubleshooting

### HTTP 403 Forbidden

YouTube frequently blocks downloads. Try these fixes in order:

1. **Switch to Mobile Web player** in Settings > Authentication > Player Client
2. **Add browser cookies** from a logged-in session
3. **Use the nightly build** which has the latest fixes
4. **Add a PO Token** for persistent access

### Age-Restricted Videos

Age-restricted videos require authentication:

1. **PO Token** (recommended) - Follow the in-app guide or see [yt-dlp PO Token Guide](https://github.com/yt-dlp/yt-dlp/wiki/PO-Token-Guide)
2. **Browser cookies** from a logged-in YouTube account

### Cookie Database Errors

If you see "Could not copy cookie database":

1. **Close your browser completely** before downloading
2. **Use a cookies.txt file** instead of browser extraction (works with browser open) see [How do I pass cookies to yt-dlp? - Second to last paragraph](https://github.com/yt-dlp/yt-dlp/wiki/FAQ#how-do-i-pass-cookies-to-yt-dlp)

---

## Security Notice
> [!NOTE]
> Using cookies or authentication may put your YouTube account at risk. Consider using a throwaway account for downloading if you are doing higher quantities like playlist.

---

## Tech Stack

| Component | Technology |
|-----------|------------|
| Backend | Go 1.21+ |
| Frontend | HTML5, JavaScript, Tailwind CSS |
| Framework | [Wails v2](https://wails.io) |
| Downloader | [yt-dlp](https://github.com/yt-dlp/yt-dlp) |

---

## License

MIT License. See [LICENSE](LICENSE) for details.
