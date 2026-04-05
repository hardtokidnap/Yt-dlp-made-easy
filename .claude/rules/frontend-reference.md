---
globs:
  - "frontend/**"
---

# Frontend reference

## Architecture

- No framework. No build step for JS logic. `frontend/src/main.js` is a single file with a global state object.
- Drag-and-drop uses Wails' `OnFileDrop` runtime API (configured in `main.go`), not browser drop events. Drop targets use the `--wails-drop-target` CSS variable / `.wails-drop-target-active` class.
- Clipboard monitoring polls `GetClipboard()` on a Go timer rather than the browser Clipboard API (which requires focus/permission).
- Custom multi-token autocomplete for yt-dlp extra args replaces native `<datalist>` (which stops suggesting after first selection).
- Custom context menu for text inputs (Wails restricts default browser right-click menu).
- XSS: file paths and URLs used in inline `onclick` handlers must be properly JS-string-escaped. This was a real past bug, don't regress it.

## Key frontend files

```
frontend/src/main.js                ALL UI logic — tabs, DOM state machine, event handlers.
                                    No framework. Global state object. Custom multi-token
                                    autocomplete for yt-dlp args (replaced native datalist).
                                    Custom context menu (Wails restricts default browser menus).
                                    Clipboard polling via GetClipboard() Go call (not browser API).
frontend/src/style.css              Tailwind base + custom component classes
frontend/index.html                 Single-page shell. Tab content rendered dynamically into
                                    #tab-content by main.js.
frontend/wailsjs/go/main/App.js     Auto-generated JS bindings — DO NOT EDIT
frontend/wailsjs/go/models.ts       Auto-generated TypeScript types — DO NOT EDIT
```

## Exposed JS methods (App.go exports)

All callable from JS via `window.go.main.App.*` or named imports from `wailsjs/go/main/App.js`:

**Downloads:** `AddDownload(url, DownloadOptions)` `PauseDownload(id)` `ResumeDownload(id)` `StopDownload(id)` `RemoveDownload(id)` `GetQueueStatus()` `PauseAllDownloads()` `ResumeAllDownloads()` `StopAllDownloads()` `ClearCompletedDownloads()`

**Settings:** `GetSettings()` `SaveSettings(Settings)`

**History:** `GetHistory(query, status)` `GetRecentHistory(limit)` `ClearHistory()` `ClearOldHistory(days)` `HideFromQueue(id)` `RemoveHistoryEntry(id)` `ExportHistory()` `GetHistoryStats()`

**Converter:** `ProbeFile(path)` `StartConversion(ConversionOptions)` `CancelConversion()` `StartBatchConversion(files[], ConversionOptions)` `CancelBatchConversion()` `GetConversionPresets()` `IsFFmpegInstalled()` `GetFFmpegVersion()` `DownloadFFmpeg()` `GetRecentCompletedDownloads()`

**File dialogs:** `BrowseFolder()` `BrowseInputFile()` `BrowseOutputFile(defaultName)` `BrowseMultipleInputFiles()`

**Shell:** `OpenFile(path)` `OpenFolder(path)` `OpenFileInFolder(path)` `OpenURL(url)`

**yt-dlp/runtime:** `CheckForUpdates()` `UpdateYtDlp()` `GetYtDlpVersion()` `GetJSRuntimeInfo()` `DownloadDeno()`

**Misc:** `GetClipboard()` `GetAppInfo()`