---
title: Naming and Code Style
impact: MEDIUM
category: maintainability
tags: naming, go, javascript, readability
---

# Naming and Code Style

Follow Go and JavaScript community conventions. Comments must always explain "why" we do something, not "what we did." The code should be readable on its own.

## Go Naming

### Exported vs Unexported
```go
// Exported (PascalCase): visible outside package
func BuildArgs(s *Settings) []string { ... }
type DownloadItem struct { ... }
const MaxRetries = 5

// Unexported (camelCase): package-internal
func parseOutput(line string) { ... }
var errNotFound = errors.New("not found")
const defaultTimeout = 30
```

### Acronyms: All Caps
```go
// ✅ Correct
var httpClient *http.Client
func GetHTTPStatus() int { ... }
func (d *Downloader) GetID() string { ... }
type URLParser struct { ... }

// ❌ Incorrect
var httpClient *http.Client // this one is actually fine
func GetHttpStatus() int { ... } // Http → HTTP
func (d *Downloader) GetId() string { ... } // Id → ID
type UrlParser struct { ... } // Url → URL
```

### Interfaces
```go
// Single-method interfaces: method name + "er"
type Reader interface { Read(p []byte) (n int, err error) }
type Converter interface { Convert(opts ConversionOptions) error }

// Multi-method: descriptive noun
type Downloader interface {
    Start(url string) error
    Pause() error
    Resume() error
}
```

### Error Variables
```go
// Sentinel errors: Err prefix
var ErrNotFound = errors.New("item not found")
var ErrAlreadyRunning = errors.New("download already running")

// Error types: Error suffix
type ValidationError struct { Field string; Message string }
func (e *ValidationError) Error() string { ... }
```

### Receivers
```go
// Short, consistent receiver names (not "this" or "self")
func (q *Queue) AddItem(item *Item) { ... }
func (q *Queue) RemoveItem(id string) { ... }

// ❌ Incorrect
func (queue *Queue) AddItem(item *Item) { ... }
func (this *Queue) RemoveItem(id string) { ... }
```

## JavaScript Naming

### Variables and Functions
```javascript
// camelCase for variables and functions
const downloadQueue = [];
function updateProgressBar(item) { ... }
let isDownloading = false;

// Boolean variables: is, has, can, should prefixes
const isActive = true;
const hasPermission = false;
const canResume = item.status === 'stopped';
```

### Constants
```javascript
// UPPER_SNAKE_CASE for true constants
const MAX_CONCURRENT_DOWNLOADS = 3;
const DEFAULT_QUALITY = 'best';
const STATUS_DOWNLOADING = 'downloading';
```

### DOM and Events
```javascript
// DOM IDs: kebab-case (matches HTML convention)
document.getElementById('download-progress');
document.querySelector('.queue-item');

// Wails event names: colon-separated, lowercase
runtime.EventsOn('download:update', callback);
runtime.EventsOn('convert:progress', callback);

// Do NOT invent new event names without updating the contract
```

## Comments

### ❌ What (Redundant)
```go
// increment counter
counter++

// check if error is nil
if err != nil {

// create new settings object
s := &Settings{}

// loop through items
for _, item := range items {
```

### ✅ Why (Useful)
```go
// Retry rename up to 5 times because antivirus may hold a file lock
// on the freshly downloaded binary.
for i := 0; i < 5; i++ {

// Use -ss before -i for input-side seek, which is much faster than
// output-side seek for large files.
args = append(args, "-ss", startTime, "-i", inputPath)

// Extract to temp file first to prevent IsFFmpegInstalled() from
// seeing a partially extracted binary as valid.
tmpPath := finalPath + ".tmp"

// CookiesFile takes priority over CookiesBrowser because
// file-based cookies are more reliable and don't require
// browser profile access.
if s.Auth.CookiesFile != "" {
```

### When Comments Are Mandatory
```go
// Document non-obvious platform-specific behavior
// OpenFileInFolder bypasses Go's argument quoting by setting
// SysProcAttr.CmdLine directly. Explorer's /select,<path> flag
// breaks when Go wraps the whole thing in double quotes.

// Document initialization order dependencies
// History must initialize before queue because OnItemUpdate
// closure references a.history. Do not reorder.

// Document intentional "wrong-looking" code
// CREATE_NO_WINDOW flag (0x08000000) must NOT be applied to
// explorer.exe — it would suppress the entire window, not just
// a console. This only applies to background processes.
```

## Best Practices

- [ ] Go exports: PascalCase. Go internal: camelCase. No underscores.
- [ ] JS: camelCase for variables/functions, UPPER_SNAKE for constants
- [ ] Acronyms fully capitalized (ID, URL, HTTP, JSON, API)
- [ ] Go receiver names: single letter matching type (q for Queue, d for Downloader)
- [ ] Comments explain intent, constraints, or non-obvious behavior
- [ ] No commented-out code in commits (use git history)
- [ ] Wails event names must match the contract, never ad-hoc
