---
title: Goroutine and Channel Safety
impact: HIGH
category: performance
tags: goroutine, channel, leak, process, context
---

# Goroutine and Channel Safety

Every goroutine must have a clear exit path. Every spawned process must be tracked and cleanable. Goroutine leaks are memory leaks that also hold OS resources.

## Why This Matters

This app spawns multiple goroutines per download (stdout scanner, stderr scanner, progress emitter) and manages a pool of concurrent yt-dlp/FFmpeg processes. A leaked goroutine holds:
- Memory for its stack (2-8KB minimum)
- An OS thread if it's blocked on syscall
- Potentially a child process (yt-dlp/FFmpeg)
- A file descriptor for the pipe

Over time, leaked goroutines degrade performance and can hit OS limits.

## ❌ Incorrect

### Unbuffered Channel, No Reader on Error Path
```go
func startDownload(url string) (*Result, error) {
    ch := make(chan *Result)
    go func() {
        result := doDownload(url)
        ch <- result // blocks forever if caller returns early
    }()

    if err := validate(url); err != nil {
        return nil, err // goroutine leaked, blocked on ch
    }
    return <-ch, nil
}
```

### Process Started Without Wait
```go
func probe(path string) {
    cmd := exec.Command("ffprobe", "-show_format", path)
    cmd.Start()
    // cmd.Wait() never called — zombie process
}
```

### Scanner Goroutine Outlives Process
```go
func scanOutput(pipe io.Reader) {
    go func() {
        scanner := bufio.NewScanner(pipe)
        for scanner.Scan() {
            process(scanner.Text())
        }
        // If pipe is never closed (process hangs), this goroutine hangs too
    }()
}
```

## ✅ Correct

### Buffered Channel + Context
```go
func startDownload(ctx context.Context, url string) (*Result, error) {
    ch := make(chan *Result, 1) // buffered: goroutine can always send
    go func() {
        ch <- doDownload(url)
    }()

    select {
    case result := <-ch:
        return result, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}
```

### Process With Proper Cleanup
```go
func probe(ctx context.Context, path string) (*MediaInfo, error) {
    cmd := exec.CommandContext(ctx, "ffprobe", "-show_format", path)
    output, err := cmd.Output() // Start + Wait in one call
    if err != nil {
        return nil, fmt.Errorf("ffprobe %s: %w", path, err)
    }
    return parseMediaInfo(output)
}
```

### Scanner With Done Signal
```go
func scanOutput(pipe io.Reader, done chan struct{}) {
    go func() {
        defer close(done)
        scanner := bufio.NewScanner(pipe)
        for scanner.Scan() {
            process(scanner.Text())
        }
    }()
}

// Caller waits for scanner to finish
<-done
```

## Patterns Specific to This App

### Channel Semaphore (Download Queue)
```go
// The queue uses a channel as a semaphore to limit concurrency.
// UpdateMaxConcurrent replaces the channel wholesale.
// Active downloads continue unaffected; new limit applies to future downloads.
semaphore := make(chan struct{}, maxConcurrent)

// Acquire slot
semaphore <- struct{}{}
// Release slot
<-semaphore

// CRITICAL: always release in defer to prevent deadlock
func (q *Queue) processItem(item *Item) {
    q.semaphore <- struct{}{} // acquire
    defer func() { <-q.semaphore }() // release

    // ... download logic
}
```

### NtSuspendProcess / NtResumeProcess Pairing
```go
// ALWAYS pair suspend with resume. A suspended process is a zombie
// that holds memory and can't be killed normally.
func (d *Downloader) Pause() {
    ntSuspendProcess.Call(d.processHandle)
    d.status = StatusPaused
}

func (d *Downloader) Resume() {
    ntResumeProcess.Call(d.processHandle)
    d.status = StatusDownloading
}

// On Stop/Cancel, resume before kill (suspended processes may not respond to TerminateProcess)
func (d *Downloader) Stop() {
    if d.status == StatusPaused {
        ntResumeProcess.Call(d.processHandle) // resume first
    }
    d.cmd.Process.Kill()
}
```

## Best Practices

- [ ] Every `go func()` has a documented exit condition
- [ ] Channels are buffered unless there's a clear reason not to
- [ ] `exec.CommandContext` used for all long-running processes
- [ ] `cmd.Wait()` always called after `cmd.Start()`
- [ ] Scanner goroutines on stdout/stderr have a done channel or context
- [ ] Semaphore slots always released via `defer`
- [ ] Suspended processes resumed before kill
