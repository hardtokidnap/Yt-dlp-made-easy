---
title: Error Handling (Go + JS)
impact: HIGH
category: correctness
tags: errors, go, javascript, reliability
---

# Error Handling

Always handle errors explicitly. In Go, wrap errors with context. In JavaScript, catch async errors and surface them to the UI.

## Go Error Handling

### ❌ Incorrect

```go
// Discarded error
data, _ := os.ReadFile(path)

// Bare return without context
func loadQueue() ([]*Item, error) {
    data, err := os.ReadFile(queuePath)
    if err != nil {
        return nil, err // which file? which operation?
    }
    var items []*Item
    err = json.Unmarshal(data, &items)
    return items, err // same problem
}

// Logging instead of returning
func saveSettings(s *Settings) {
    data, err := json.Marshal(s)
    if err != nil {
        log.Printf("error: %v", err)
        return // caller has no idea it failed
    }
    os.WriteFile(settingsPath, data, 0644) // error ignored
}
```

### ✅ Correct

```go
// Wrapped with context using %w for error chain
func loadQueue() ([]*Item, error) {
    data, err := os.ReadFile(queuePath)
    if err != nil {
        return nil, fmt.Errorf("reading queue from %s: %w", queuePath, err)
    }
    var items []*Item
    if err := json.Unmarshal(data, &items); err != nil {
        return nil, fmt.Errorf("parsing queue JSON: %w", err)
    }
    return items, nil
}

// Return errors to caller, let them decide
func saveSettings(s *Settings) error {
    data, err := json.MarshalIndent(s, "", "  ")
    if err != nil {
        return fmt.Errorf("marshalling settings: %w", err)
    }
    if err := os.WriteFile(settingsPath, data, 0644); err != nil {
        return fmt.Errorf("writing settings to %s: %w", settingsPath, err)
    }
    return nil
}
```

### Error Wrapping Rules
```go
// Use %w when callers might need errors.Is() or errors.As()
return fmt.Errorf("loading settings: %w", err)

// Use %v when wrapping an error you don't want callers to unwrap
return fmt.Errorf("internal error during init: %v", err)
```

### This App's Patterns

**History rollback on write failure:**
```go
// history.go pattern: mutate in-memory, then write to disk.
// If disk write fails, roll back the in-memory state.
func (h *History) Remove(id string) error {
    // Save old state for rollback
    old := h.entries
    h.entries = removeByID(h.entries, id)

    if err := h.writeToDisk(); err != nil {
        h.entries = old // rollback
        return fmt.Errorf("removing history entry %s: %w", id, err)
    }
    return nil
}
```

**Updater retry with context:**
```go
// updater.go pattern: retry rename because antivirus holds file locks
for i := 0; i < 5; i++ {
    if err := os.Rename(tmpPath, finalPath); err == nil {
        return nil
    }
    time.Sleep(200 * time.Millisecond)
}
return fmt.Errorf("rename %s to %s failed after 5 retries: %w", tmpPath, finalPath, err)
```

## JavaScript Error Handling

### ❌ Incorrect

```javascript
// Unhandled rejection
async function loadSettings() {
    const settings = await window.go.main.App.GetSettings();
    applySettings(settings);
}

// Silent catch
try {
    await startConversion(opts);
} catch (e) {
    console.log(e); // user sees nothing
}

// Catch-all that hides the real error
try {
    await complexOperation();
} catch {
    showError("Something went wrong"); // useless
}
```

### ✅ Correct

```javascript
// Catch and show meaningful error
async function loadSettings() {
    try {
        const settings = await window.go.main.App.GetSettings();
        applySettings(settings);
    } catch (err) {
        showError(`Failed to load settings: ${err.message}`);
        applyDefaults();
    }
}

// Specific error handling per operation
async function startConversion(opts) {
    try {
        await window.go.main.App.StartConversion(opts);
    } catch (err) {
        if (err.message.includes('ffmpeg not found')) {
            promptFFmpegInstall();
        } else {
            showError(`Conversion failed: ${err.message}`);
        }
    }
}
```

### Wails Event Error Handling
```javascript
// Always handle the 'error' event for fatal startup errors
runtime.EventsOn('error', (message) => {
    showFatalError(message);
});

// Handle conversion errors
runtime.EventsOn('convert:error', (message) => {
    showConversionError(message);
});
```

## Best Practices

- [ ] Never discard errors with `_` unless there's a comment explaining why
- [ ] Wrap Go errors with `fmt.Errorf("context: %w", err)`
- [ ] Return errors to callers, don't just log them
- [ ] Catch all `await` calls in JS, surface errors to UI
- [ ] Roll back in-memory state if disk writes fail (history pattern)
- [ ] Retry with backoff for transient failures (file locks, network)
