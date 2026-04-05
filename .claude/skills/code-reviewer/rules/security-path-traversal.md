---
title: Path Traversal Prevention
impact: CRITICAL
category: security
tags: path, traversal, file, directory
---

# Path Traversal Prevention

Validate that file operations stay within expected directories. User-controlled paths (save folder, output template, cookies file) can escape intended boundaries.

## Why This Matters

This app writes files to user-specified directories and reads files from user-specified paths. Unchecked paths can:
- Write to system directories
- Overwrite critical files
- Read sensitive files (cookies.txt path pointing to system files)

## ❌ Incorrect

```go
// No validation on save path
func saveDownload(userDir, filename string, data []byte) error {
    path := filepath.Join(userDir, filename)
    return os.WriteFile(path, data, 0644)
}
// filename: "../../../Windows/System32/drivers/etc/hosts"
```

```go
// No validation on output template
template := settings.Advanced.OutputTemplate
outputPath := filepath.Join(saveFolder, template)
// template: "../../important_dir/%(title)s.%(ext)s"
```

## ✅ Correct

```go
func safePath(baseDir, userPath string) (string, error) {
    absBase, err := filepath.Abs(baseDir)
    if err != nil {
        return "", fmt.Errorf("resolving base dir: %w", err)
    }

    // Clean the path (resolves .., removes redundant separators)
    joined := filepath.Join(absBase, filepath.Clean(userPath))

    // Resolve symlinks
    resolved, err := filepath.EvalSymlinks(joined)
    if err != nil {
        // File may not exist yet, check the parent
        resolved = joined
    }

    // Verify it's still under the base directory
    if !strings.HasPrefix(resolved, absBase+string(filepath.Separator)) && resolved != absBase {
        return "", fmt.Errorf("path escapes base directory: %s", userPath)
    }
    return resolved, nil
}
```

## Paths to Validate in This App

| Setting | Risk | Mitigation |
|---------|------|------------|
| `General.SaveFolder` | User picks via dialog | Wails BrowseFolder validates existence |
| `Advanced.OutputTemplate` | Free text input | Reject `..`, absolute paths |
| `Auth.CookiesFile` | Free text path | Validate it's a regular file, warn user |
| Downloaded filenames | From yt-dlp output | yt-dlp handles this, but sanitize if displayed |

## Best Practices

- [ ] Use `filepath.Clean` before joining paths
- [ ] Verify the resolved path starts with the intended base directory
- [ ] Reject output templates containing `..` or absolute paths
- [ ] Use `filepath.EvalSymlinks` when checking existing files
- [ ] Log path validation failures for debugging
