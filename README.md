# YT-DLP Made Easy

A modern, lightweight GUI wrapper for [yt-dlp](https://github.com/yt-dlp/yt-dlp) built with Go and Wails. Single ~8MB binary, no external dependencies.

> [!NOTE] 
> This is a hobby project. Issues may not be addressed.

> [!WARNING]
> Some antivirus software may flag this app due to browser cookie access. This is a false positive - the app reads cookies locally to authenticate with YouTube (standard yt-dlp feature). These cookies never leave your machine. They're already stored as plaintext SQLite files that any application can read. Verify via VirusTotal or build from source.

[VirusTotal scan](https://www.virustotal.com/gui/file/c2d2e6a0440c1e0c9a87a007d49020b3c973ecf0450583f866121469db422675/details)

---


## Features

### Download Management
- **Concurrent downloads** with configurable queue size
- **Pause and resume** downloads without losing progress
- **Real-time progress** with speed and ETA display
- **Persistent queue** - downloads survive app restarts
- **Playlist support** with individual item tracking
- **Quality selection** from 360p to 4K
- **Audio extraction** with format and quality options
- **Open Folder** button to reveal downloaded files in your file manager
<img width="1016" height="1056" alt="image" src="https://github.com/user-attachments/assets/67a4ae3f-fe77-4899-a20f-d2eae47d6fe5" />
<img width="987" height="848" alt="image" src="https://github.com/user-attachments/assets/4d8f4feb-5279-46c5-9f8f-bd5b68bee5f9" />


### Media Conversion
- **Built-in FFmpeg converter** with automatic FFmpeg download
- **Media info preview** - see duration, resolution, codecs, bitrate, and file size before converting
- **Quick presets** - Video to MP3, Convert to MP4/MKV/WebM, Extract Audio, FLAC, WAV
- **Platform export presets** - one-click optimization for YouTube, Twitter/X, LinkedIn, and Web Embed
- **Quality slider** - CRF-based quality control from lossless to maximum compression
- **Trim and cut** - set start/end times to extract clips with frame-accurate seeking
- **Batch conversion** - drop or select multiple files, convert them sequentially with per-file progress
- **Drag-and-drop** - drop files directly onto the converter tab
- **Full codec control** - H.264, H.265, VP9, AAC, MP3, Opus, FLAC, and more
- **Custom FFmpeg arguments** for filters and advanced use
- **Real-time progress** with speed and duration tracking
- **Convert recent downloads** directly from the queue
<img width="970" height="1392" alt="image" src="https://github.com/user-attachments/assets/e654da25-727a-47b3-bb3e-f2a2c6402504" />


### Smart Error Handling
- **Automatic error classification** detects 403 blocks, rate limits, age restrictions, and more
- **Guided recovery** with actionable fix suggestions
- **One-click fixes** that apply settings and retry automatically


### Authentication
- **Browser cookie import** from Chrome, Firefox, Edge, Brave
- **Cookies file support** for explicit authentication
- **PO Token support** for age-restricted content
- **Player client switching** (web, mweb, ios, android) to bypass blocks
<img width="918" height="605" alt="image" src="https://github.com/user-attachments/assets/bd97994e-cc25-41b2-af3c-0e2db1638115" />
<img width="915" height="353" alt="image" src="https://github.com/user-attachments/assets/cf6886f1-efa6-49a3-bf73-55e7e7b673f6" />



### Advanced Options
- **SponsorBlock integration** to skip sponsored segments
- **Nightly builds by default** with inline update indicator
- **JavaScript runtime management** (Deno, Node.js, Bun) with bundled Deno download
- **Custom yt-dlp arguments** with autocomplete suggestions
- **Clipboard monitoring** for automatic URL detection
- **Verbose logging** toggle for debugging
- **Download history** with search, filter, and re-download
<img width="927" height="371" alt="image" src="https://github.com/user-attachments/assets/e3478045-0493-4314-93b8-2189b0695859" />


---

## Installation

Download the latest release from the [Releases](https://github.com/hardtokidnap/Yt-dlp-made-easy/releases) page.

The application will automatically download yt-dlp on first launch.

---

## Building from Source

### Prerequisites

- Go 1.23+
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

> [!NOTE]
> Project Structure provided by [Gitingest](https://gitingest.com/hardtokidnap/Yt-dlp-made-easy)

```
├── app.go                         # Wails bridge - all exported methods bind to JS
├── main.go                        # Application bootstrap and Wails config
├── open_windows.go                # Platform-specific file/folder opening (Windows)
├── open_darwin.go                 # Platform-specific file/folder opening (macOS)
├── open_linux.go                  # Platform-specific file/folder opening (Linux)
├── wails.json
├── frontend/
│   ├── index.html
│   ├── package.json
│   ├── postcss.config.js
│   ├── tailwind.config.js
│   ├── src/
│   │   ├── main.js                # UI logic (tabs, state, event handlers)
│   │   ├── style.css              # Tailwind + custom component styles
│   │   ├── app.css                # Base application styles
│   │   └── assets/
│   │       └── fonts/
│   └── wailsjs/                   # Auto-generated Wails bindings
│       ├── go/
│       │   ├── models.ts
│       │   └── main/
│       │       ├── App.d.ts
│       │       └── App.js
│       └── runtime/
├── internal/
│   ├── config/
│   │   └── settings.go            # Settings persistence (JSON)
│   ├── converter/
│   │   ├── args.go                # CLI arg builder, presets, time parsing
│   │   ├── batch.go               # Multi-file sequential batch queue
│   │   ├── converter.go           # Single-file conversion with progress
│   │   ├── ffmpeg.go              # FFmpeg/ffprobe path detection and download
│   │   ├── probe.go               # Media info extraction via ffprobe
│   │   ├── cmd_windows.go         # Hidden process creation (no console flash)
│   │   └── cmd_other.go           # Unix process creation
│   ├── downloader/
│   │   ├── args.go                # yt-dlp CLI arg builder
│   │   ├── downloader.go          # Download execution and progress parsing
│   │   ├── errors.go              # Error classification and recovery suggestions
│   │   ├── item.go                # Download item model
│   │   └── queue.go               # Concurrent download queue with persistence
│   ├── history/
│   │   └── history.go             # JSONL-backed download history
│   ├── jsruntime/
│   │   └── runtime.go             # JS runtime detection (Deno/Node/Bun)
│   ├── updater/
│   │   └── updater.go             # yt-dlp version management
│   └── util/
│       ├── meta.go                # App name, version, contributors
│       └── paths.go               # Platform-specific data paths
└── .github/
    ├── PULL_REQUEST_TEMPLATE.md
    ├── ISSUE_TEMPLATE/
    │   ├── bug_report.md
    │   └── feature-request.md
    └── workflows/
        ├── claude.yaml
        └── codeql.yml
```




---

## Configuration

Settings and data are stored in `%LOCALAPPDATA%\ytdlp-easy\`:

| File | Purpose |
|------|---------|
| `settings.json` | User preferences and configuration |
| `history.jsonl` | Download history |
| `queue.json` | Persisted download queue |
| `ytdlp.log` | Debug log (when verbose logging is enabled) |

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

### Conversion Issues

**"FFmpeg is not installed"**
- Open the Convert tab and click "Download FFmpeg" - both FFmpeg and FFprobe are downloaded automatically
- If the download fails, check your internet connection or firewall - the binaries are fetched from [BtbN/FFmpeg-Builds](https://github.com/BtbN/FFmpeg-Builds)
- Binaries are stored in `%LOCALAPPDATA%\ytdlp-easy\ffmpeg\`

**Conversion fails or produces corrupted output**
- Check the log panel at the bottom of the Convert tab for FFmpeg error messages
- Verify the input file isn't corrupted by playing it in a media player first
- Try a different output codec - some codec combinations are incompatible (e.g. VP9 video in an MP4 container)
- If using custom args, make sure they don't conflict with the preset settings

**Progress stuck at 0%**
- This can happen with very short files where FFmpeg finishes before reporting progress
- For trimmed clips, progress is based on the clip duration, not the full file - if the trim times are invalid, progress may not advance

**Batch conversion stops mid-way**
- Individual file failures don't stop the batch - check the batch progress panel for per-file status
- If all files fail, the issue is likely with the output settings rather than the input files

---

## Security Notice
> [!NOTE]
> Using cookies or authentication may put your YouTube account at risk. Consider using a throwaway account for downloading if you are doing higher quantities like playlist.

---

## Tech Stack

| Component | Technology |
|-----------|------------|
| Backend | Go 1.23+ |
| Frontend | HTML5, JavaScript, Tailwind CSS |
| Framework | [Wails v2](https://wails.io) |
| Downloader | [yt-dlp](https://github.com/yt-dlp/yt-dlp) |
| Converter | [FFmpeg](https://ffmpeg.org) + [FFprobe](https://ffmpeg.org) (auto-downloaded) |

---

## License

MIT License. See [LICENSE](LICENSE) for details.
