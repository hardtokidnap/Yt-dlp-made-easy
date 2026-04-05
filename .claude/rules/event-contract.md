# Wails event contract

All events emitted by Go (`wailsruntime.EventsEmit`) and consumed by `frontend/src/main.js`. Do not rename these.

| Event                    | Payload                                | Emitted when                                                              |
| ------------------------ | -------------------------------------- | ------------------------------------------------------------------------- |
| `download:update`        | `downloader.Item`                      | Any state change on a download item (progress tick, status change, error) |
| `download:log`           | `{id: string, line: string}`           | Each yt-dlp stdout/stderr line for a specific item                        |
| `queue:update`           | `[]*downloader.Item`                   | Queue composition changes (add, remove, complete)                         |
| `history:update`         | `[]history.Entry` (last 3, non-hidden) | A download completes, errors, or history is modified                      |
| `convert:progress`       | `*converter.ConversionJob`             | FFmpeg progress tick (only on lines containing `time=`)                   |
| `convert:log`            | `string`                               | Raw FFmpeg stderr line                                                    |
| `convert:error`          | `string`                               | Conversion or batch failed                                                |
| `convert:batch:progress` | `*converter.BatchSnapshot`             | Batch conversion progress tick                                            |
| `updater:progress`       | `string`                               | yt-dlp download/install progress message                                  |
| `update:checking`        | `nil`                                  | Starting update check on startup                                          |
| `update:available`       | `*updater.UpdateInfo`                  | Newer yt-dlp version found                                                |
| `update:none`            | `nil`                                  | yt-dlp is up to date                                                      |
| `jsruntime:progress`     | `string`                               | Deno download progress message                                            |
| `ffmpeg:progress`        | `string`                               | FFmpeg binary download progress message                                   |
| `error`                  | `string`                               | Fatal startup error (e.g. yt-dlp install failed)                          |