---
title: Race Conditions and Mutex Usage
impact: HIGH
category: correctness
tags: mutex, sync, goroutine, race, settings
---

# Race Conditions and Mutex Usage

All shared state must be accessed through proper synchronization. This app has multiple goroutines accessing settings, queue, and history concurrently.

## Why This Matters

Go's race detector (`go run -race`) will catch most of these, but only at runtime on exercised code paths. Undetected races cause:
- Silent data corruption (settings half-written)
- Panics (`concurrent map read and map write`)
- Undefined behavior that only appears under load

## ❌ Incorrect

### Direct Field Access Across Goroutines
```go
// Race: download goroutine reads settings field directly
func (d *Downloader) getQuality() string {
    return d.settings.Download.Quality // data race with UI goroutine calling SaveSettings
}
```

### Unprotected Map Access
```go
// Panic: concurrent map writes
func (q *Queue) addItem(item *Item) {
    q.items[item.ID] = item // no lock
}

func (q *Queue) removeItem(id string) {
    delete(q.items, id) // no lock, concurrent with addItem
}
```

### Check-Then-Act Without Lock
```go
// Race: status can change between check and use
func (q *Queue) resumeIfStopped(id string) {
    item := q.items[id]
    if item.Status == StatusStopped { // check
        item.Status = StatusDownloading // act — another goroutine may have changed it
        go q.startDownload(item)
    }
}
```

## ✅ Correct

### Thread-Safe Accessors (Settings Pattern)
```go
// settings.go provides Get*/Set* methods with internal RWMutex
type Settings struct {
    mu   sync.RWMutex
    data SettingsData
}

func (s *Settings) GetQuality() string {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.data.Download.Quality
}

func (s *Settings) SetQuality(q string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.data.Download.Quality = q
}

// Usage: ALWAYS use the accessor
func (d *Downloader) getQuality() string {
    return d.settings.GetQuality()
}
```

### Protected Map Access
```go
type Queue struct {
    mu    sync.RWMutex
    items map[string]*Item
}

func (q *Queue) addItem(item *Item) {
    q.mu.Lock()
    defer q.mu.Unlock()
    q.items[item.ID] = item
}

func (q *Queue) getItem(id string) (*Item, bool) {
    q.mu.RLock()
    defer q.mu.RUnlock()
    item, ok := q.items[id]
    return item, ok
}
```

### Atomic Check-Then-Act
```go
func (q *Queue) resumeIfStopped(id string) {
    q.mu.Lock()
    defer q.mu.Unlock()

    item, ok := q.items[id]
    if !ok || item.Status != StatusStopped {
        return
    }
    item.Status = StatusDownloading
    go q.startDownload(item) // safe: status is set under lock
}
```

## Common Race Scenarios in This App

| Scenario | Goroutines Involved | Shared State |
|----------|-------------------|--------------|
| User changes settings while download is running | UI goroutine, download goroutine | `Settings` struct |
| Multiple downloads complete simultaneously | Multiple download goroutines | `Queue.items` map, `History` |
| Queue persistence during status change | Download goroutine, persist goroutine | `Queue` state |
| Clipboard monitoring + manual URL paste | Clipboard goroutine, UI goroutine | Download queue |

## Lock Ordering

When multiple locks are needed, always acquire in the same order to prevent deadlock:
1. `Settings.mu`
2. `Queue.mu`
3. `History.mu`

```go
// ❌ Deadlock risk: different lock order in different code paths
func pathA() {
    queue.mu.Lock()
    settings.mu.RLock() // lock order: queue → settings
}

func pathB() {
    settings.mu.RLock()
    queue.mu.Lock() // lock order: settings → queue (deadlock!)
}
```

## Testing for Races

```bash
# Run with race detector
go run -race ./...
go test -race ./...

# Stress test concurrent paths
go test -race -count=100 -run TestConcurrentDownloads
```

## Best Practices

- [ ] Always use `Get*()`/`Set*()` for settings access, never direct field reads
- [ ] Protect all map access with a mutex
- [ ] Use `sync.RWMutex` when reads vastly outnumber writes
- [ ] Always `defer mu.Unlock()` immediately after `mu.Lock()`
- [ ] Keep critical sections short (don't do I/O under lock)
- [ ] Use `go run -race` during development
- [ ] Document lock ordering if multiple locks are involved
