# Code Review Guidelines

**A review guide for Go + vanilla JS + Wails v2 desktop applications**, organized by priority and impact. This app spawns external processes, has no database, and targets Windows.

---

## Table of Contents

### Security — **CRITICAL**
1. [Command Injection Prevention](#command-injection-prevention)
2. [XSS Prevention in Wails](#xss-prevention-in-wails)
3. [Path Traversal Prevention](#path-traversal-prevention)

### Performance — **HIGH**
4. [Goroutine and Channel Safety](#goroutine-and-channel-safety)

### Correctness — **HIGH**
5. [Error Handling (Go + JS)](#error-handling)
6. [Race Conditions and Mutex Usage](#race-conditions-and-mutex-usage)

### Maintainability — **MEDIUM**
7. [Naming and Code Style](#naming-and-code-style)

---

## Security

### Command Injection Prevention

**Impact: CRITICAL** | **Category: security** | **Tags:** exec, process, injection, yt-dlp, ffmpeg

Never pass unsanitized user input as arguments to `exec.Command` or `exec.CommandContext`. This app spawns yt-dlp and FFmpeg processes with user-controlled parameters (URLs, file paths, extra args).

#### Why This Matters

Unlike web apps where SQL injection is the top risk, desktop apps that spawn processes face command injection. An attacker-crafted URL or filename can inject shell metacharacters or additional arguments.

#### ❌ Incorrect

```go
// Dangerous: user-controlled string interpolated into command
func runYtDlp(url string) error {
    cmd := exec.Command("sh", "-c", "yt-dlp " + url)
    return cmd.Run()
}
// Input: "https://example.com; rm -rf /"
// Executes both commands!
```

```go
// Dangerous: unsanitized extra args passed directly
func buildArgs(extraArgs string) []string {
    return append(baseArgs, strings.Split(extraArgs, " ")...)
}
// Input: "--exec rm -rf /"
// yt-dlp's --exec flag runs arbitrary commands!
```

#### ✅ Correct

```go
// Safe: pass URL as a single argument, not through shell
func runYtDlp(url string) error {
    cmd := exec.Command(ytDlpPath, "--no-exec", url)
    return cmd.Run()
}

// Safe: quote-aware split and validate args
func buildArgs(extraArgs string) []string {
    args := splitQuotedArgs(extraArgs)
    // Validate no dangerous flags
    for _, arg := range args {
        if strings.HasPrefix(arg, "--exec") {
            continue // skip dangerous flags
        }
    }
    return append(baseArgs, args...)
}
```

**Key principle:** Always use `exec.Command(binary, arg1, arg2)` with separate arguments. Never use `exec.Command("sh", "-c", concatenatedString)`. Go's exec passes args directly to the process without shell interpretation.

[➡️ Full details: security-command-injection.md](rules/security-command-injection.md)

---

### XSS Prevention in Wails

**Impact: CRITICAL** | **Category: security** | **Tags:** xss, wails, innerHTML, onclick, javascript

Never insert unsanitized file paths, URLs, or user input into HTML via `innerHTML` or inline event handlers. This was a **real past bug** in this codebase.

#### ❌ Incorrect

```javascript
// Dangerous: file path in inline onclick
const html = `<button onclick="openFile('${filePath}')">Open</button>`;
container.innerHTML = html;
// filePath: "file'); alert('xss"
// Breaks out of the string and executes arbitrary JS
```

```javascript
// Dangerous: URL injected into innerHTML
const html = `<a href="${url}">${title}</a>`;
listContainer.innerHTML += html;
```

#### ✅ Correct

```javascript
// Safe: create elements programmatically
const btn = document.createElement('button');
btn.textContent = 'Open';
btn.addEventListener('click', () => openFile(filePath));
container.appendChild(btn);

// Safe: if innerHTML is unavoidable, escape the values
function escapeJS(str) {
    return str.replace(/\\/g, '\\\\')
              .replace(/'/g, "\\'")
              .replace(/"/g, '\\"')
              .replace(/\n/g, '\\n');
}
const html = `<button onclick="openFile('${escapeJS(filePath)}')">Open</button>`;
```

**This app's specific risk:** File paths from yt-dlp output and download URLs are user-influenced and can contain quotes, backslashes, and special characters. Any inline `onclick` handler using these values must escape them properly.

[➡️ Full details: security-xss-wails.md](rules/security-xss-wails.md)

---

### Path Traversal Prevention

**Impact: CRITICAL** | **Category: security** | **Tags:** path, traversal, file, directory

Validate that file operations stay within expected directories. User-controlled paths (save folder, output template, cookies file) can escape intended boundaries.

#### ❌ Incorrect

```go
// Dangerous: user controls the path, no validation
func saveFile(userPath string, data []byte) error {
    return os.WriteFile(userPath, data, 0644)
}
// Input: "../../../etc/passwd" or "C:\Windows\System32\something"
```

#### ✅ Correct

```go
// Safe: resolve and validate the path is within allowed directory
func saveFile(baseDir, filename string, data []byte) error {
    absBase, _ := filepath.Abs(baseDir)
    fullPath := filepath.Join(absBase, filepath.Clean(filename))

    // Verify it's still under the base directory
    if !strings.HasPrefix(fullPath, absBase) {
        return fmt.Errorf("path escapes base directory: %s", filename)
    }
    return os.WriteFile(fullPath, data, 0644)
}
```

[➡️ Full details: security-path-traversal.md](rules/security-path-traversal.md)

---

## Performance

### Goroutine and Channel Safety

**Impact: HIGH** | **Category: performance** | **Tags:** goroutine, channel, leak, process, context

Goroutine leaks and zombie processes are the "N+1 queries" of desktop apps. Every spawned yt-dlp or FFmpeg process must be tracked and cleanable.

#### ❌ Incorrect

```go
// Leak: goroutine blocks forever if channel is never read
func startDownload(url string) {
    ch := make(chan string)
    go func() {
        result := download(url)
        ch <- result // blocks forever if nobody reads ch
    }()
    // ch is never read on error paths
}
```

```go
// Leak: process outlives its parent context
func convert(path string) {
    cmd := exec.Command("ffmpeg", "-i", path, "out.mp4")
    cmd.Start()
    // If function returns early, ffmpeg is orphaned
}
```

#### ✅ Correct

```go
// Safe: use context for cancellation, buffered channel
func startDownload(ctx context.Context, url string) (string, error) {
    ch := make(chan string, 1) // buffered so goroutine doesn't block
    go func() {
        ch <- download(url)
    }()

    select {
    case result := <-ch:
        return result, nil
    case <-ctx.Done():
        return "", ctx.Err()
    }
}
```

```go
// Safe: track process and kill on cancel
func convert(ctx context.Context, path string) error {
    cmd := exec.CommandContext(ctx, "ffmpeg", "-i", path, "out.mp4")
    if err := cmd.Start(); err != nil {
        return err
    }
    return cmd.Wait() // CommandContext kills process when ctx cancels
}
```

**This app's specific patterns:**
- Download queue uses a channel semaphore for concurrency. New channels must be buffered to match `MaxConcurrentDownloads`.
- `NtSuspendProcess`/`NtResumeProcess` must always be paired. A suspended process that's never resumed is a zombie.
- FFmpeg scanner goroutines reading stdout/stderr must exit when the process exits.

[➡️ Full details: performance-goroutine-safety.md](rules/performance-goroutine-safety.md)

---

## Correctness

### Error Handling

**Impact: HIGH** | **Category: correctness** | **Tags:** errors, go, javascript, reliability

#### Go: Wrap errors with context, never discard them

##### ❌ Incorrect

```go
// Bad: error discarded silently
data, _ := os.ReadFile(path)

// Bad: error returned without context
func loadSettings() (*Settings, error) {
    data, err := os.ReadFile(settingsPath)
    if err != nil {
        return nil, err // caller has no idea what file or why
    }
}
```

##### ✅ Correct

```go
// Good: wrapped with context
func loadSettings() (*Settings, error) {
    data, err := os.ReadFile(settingsPath)
    if err != nil {
        return nil, fmt.Errorf("loading settings from %s: %w", settingsPath, err)
    }
    var s Settings
    if err := json.Unmarshal(data, &s); err != nil {
        return nil, fmt.Errorf("parsing settings JSON: %w", err)
    }
    return &s, nil
}
```

#### JavaScript: Handle async errors, don't swallow rejections

##### ❌ Incorrect

```javascript
// Bad: unhandled promise rejection
async function startDownload(url) {
    const result = await window.go.main.App.AddDownload(url);
    updateUI(result);
    // If AddDownload throws, the UI shows nothing
}
```

##### ✅ Correct

```javascript
// Good: catch and handle
async function startDownload(url) {
    try {
        const result = await window.go.main.App.AddDownload(url);
        updateUI(result);
    } catch (err) {
        showError(`Download failed: ${err.message}`);
    }
}
```

[➡️ Full details: correctness-error-handling.md](rules/correctness-error-handling.md)

---

### Race Conditions and Mutex Usage

**Impact: HIGH** | **Category: correctness** | **Tags:** mutex, sync, goroutine, race, settings

This app has multiple goroutines accessing shared state: settings, download queue, history. All shared state access must use proper synchronization.

#### ❌ Incorrect

```go
// Race: reading settings field directly from another goroutine
func (q *Queue) getMaxConcurrent() int {
    return q.settings.General.MaxConcurrentDownloads // data race!
}
```

```go
// Race: map accessed from multiple goroutines without lock
func (q *Queue) addItem(item *Item) {
    q.items[item.ID] = item // concurrent map write = panic
}
```

#### ✅ Correct

```go
// Safe: use the thread-safe accessor
func (q *Queue) getMaxConcurrent() int {
    return q.settings.GetMaxConcurrentDownloads()
}
```

```go
// Safe: protect map access with mutex
func (q *Queue) addItem(item *Item) {
    q.mu.Lock()
    defer q.mu.Unlock()
    q.items[item.ID] = item
}
```

**This app's rule:** Always use the `Get*()`/`Set*()` methods on `*config.Settings`. Never read or write struct fields directly across goroutines.

[➡️ Full details: correctness-race-conditions.md](rules/correctness-race-conditions.md)

---

## Maintainability

### Naming and Code Style

**Impact: MEDIUM** | **Category: maintainability** | **Tags:** naming, go, javascript, readability

#### Go Conventions

```go
// Exported: PascalCase
func BuildArgs(s *Settings) []string { ... }
type DownloadItem struct { ... }

// Unexported: camelCase (no underscores)
func parseOutput(line string) { ... }
var maxRetries = 5

// Acronyms: all caps
var httpClient *http.Client
func (d *Downloader) GetID() string { ... }
type URLParser struct { ... }

// Errors: "Err" prefix for sentinels
var ErrNotFound = errors.New("not found")
```

#### JavaScript Conventions

```javascript
// Variables and functions: camelCase
const downloadQueue = [];
function updateProgressBar(item) { ... }

// Constants: UPPER_SNAKE_CASE
const MAX_CONCURRENT_DOWNLOADS = 3;

// DOM IDs in HTML: kebab-case
document.getElementById('download-progress');

// Wails event names: colon-separated, lowercase
runtime.EventsOn('download:update', callback);
```

#### Comments

```go
// ❌ What (redundant)
// increment counter by 1
counter++

// ✅ Why (useful)
// Retry rename up to 5 times because antivirus may hold a file lock
// on the freshly downloaded binary.
for i := 0; i < 5; i++ {
    err = os.Rename(tmpPath, finalPath)
    if err == nil { break }
    time.Sleep(200 * time.Millisecond)
}
```

[➡️ Full details: maintainability-naming.md](rules/maintainability-naming.md)

---

## Quick Reference

### Review Checklist

**Security (CRITICAL, review first)**
- [ ] No command injection in exec.Command calls
- [ ] File paths and URLs escaped in inline JS handlers
- [ ] Path traversal checked on user-supplied file paths
- [ ] Credentials (cookies, PO tokens) not logged or exposed
- [ ] `SysProcAttr.CmdLine` usage reviewed for injection

**Performance (HIGH)**
- [ ] No goroutine leaks (channels closed, contexts cancelled)
- [ ] No zombie processes (yt-dlp/FFmpeg cleaned up)
- [ ] Queue persistence not triggered on every progress tick
- [ ] Buffered channels used where appropriate

**Correctness (HIGH)**
- [ ] Go errors wrapped with `fmt.Errorf("context: %w", err)`
- [ ] JS async errors caught and surfaced to UI
- [ ] Shared state accessed through mutex or accessor methods
- [ ] New settings fields have `mergeDefaults()` entries
- [ ] Event names match the contract exactly

**Maintainability (MEDIUM)**
- [ ] Go: PascalCase exports, camelCase internal
- [ ] JS: camelCase, UPPER_SNAKE for constants
- [ ] Comments explain "why" not "what"
- [ ] `frontend/wailsjs/` not manually edited

---

## Severity Levels

| Level | Description | Examples | Action |
|-------|-------------|----------|--------|
| **CRITICAL** | Security vulnerabilities, data loss risks | Command injection, XSS, credential exposure | Block merge, fix immediately |
| **HIGH** | Process leaks, correctness bugs | Goroutine leak, race condition, wrong event name | Fix before merge |
| **MEDIUM** | Maintainability, code quality | Naming, comments, thread-safe accessor usage | Fix or accept with TODO |
| **LOW** | Style preferences, minor improvements | Formatting, minor refactoring | Optional |
