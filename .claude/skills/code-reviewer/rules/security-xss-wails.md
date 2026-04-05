---
title: XSS Prevention in Wails
impact: CRITICAL
category: security
tags: xss, wails, innerHTML, onclick, javascript
---

# XSS Prevention in Wails

Never insert unsanitized file paths, URLs, or user input into HTML via `innerHTML` or inline event handlers. This was a **real past bug** in this codebase.

## Why This Matters

Even though this is a desktop app (not a public website), XSS is still dangerous:
- Wails renders a webview, so injected JS executes with full access to the Wails runtime
- An attacker can craft a video title or filename containing JS that executes when rendered
- The injected code can call any `window.go.main.App.*` method (download files, change settings, access cookies)

## The Real Bug

File paths and URLs from yt-dlp output were used in inline `onclick` handlers without escaping. A filename containing a single quote broke out of the JS string and allowed arbitrary code execution.

```javascript
// ❌ This was the actual bug pattern
const html = `<span onclick="openFile('${filePath}')">${fileName}</span>`;
container.innerHTML = html;
// filePath: "C:\Users\test\Downloads\video'); alert('pwned"
// Result: onclick="openFile('C:\Users\test\Downloads\video'); alert('pwned')"
```

## ❌ Incorrect Patterns

### Inline Event Handlers with Unescaped Values
```javascript
// ❌ Any of these are vulnerable
const html = `<button onclick="openFile('${path}')">Open</button>`;
const html = `<a onclick="openURL('${url}')">Link</a>`;
const html = `<div onclick="selectItem('${id}')">Item</div>`;
```

### innerHTML with User-Influenced Content
```javascript
// ❌ Title comes from yt-dlp, controlled by video uploader
const html = `<div class="title">${videoTitle}</div>`;
container.innerHTML = html;
// videoTitle: "<img src=x onerror='fetch(document.cookie)'>"
```

### Template Literal HTML Building
```javascript
// ❌ Building HTML strings with unescaped interpolation
function renderDownloadItem(item) {
    return `
        <div class="item">
            <span>${item.title}</span>
            <button onclick="openFile('${item.filePath}')">Open</button>
        </div>
    `;
}
```

## ✅ Correct Patterns

### Option 1: Programmatic DOM (Preferred)
```javascript
function renderDownloadItem(item) {
    const div = document.createElement('div');
    div.className = 'item';

    const title = document.createElement('span');
    title.textContent = item.title; // textContent auto-escapes
    div.appendChild(title);

    const btn = document.createElement('button');
    btn.textContent = 'Open';
    btn.addEventListener('click', () => openFile(item.filePath));
    div.appendChild(btn);

    return div;
}
```

### Option 2: Escape When innerHTML Is Unavoidable
```javascript
// JS string escaping for onclick handlers
function escapeJS(str) {
    return String(str)
        .replace(/\\/g, '\\\\')
        .replace(/'/g, "\\'")
        .replace(/"/g, '\\"')
        .replace(/\n/g, '\\n')
        .replace(/\r/g, '\\r');
}

// HTML entity escaping for content
function escapeHTML(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}

// Usage
const html = `
    <span onclick="openFile('${escapeJS(filePath)}')">
        ${escapeHTML(fileName)}
    </span>
`;
```

### Option 3: Data Attributes + Delegated Events
```javascript
// Set data attributes (auto-escaped by the browser)
const html = `<button data-action="open" data-path="${escapeHTML(filePath)}">Open</button>`;

// Single delegated handler
container.addEventListener('click', (e) => {
    const btn = e.target.closest('[data-action="open"]');
    if (btn) {
        openFile(btn.dataset.path);
    }
});
```

## Sources of Untrusted Content in This App

| Source | Risk | Example |
|--------|------|---------|
| Video title | HIGH | Controlled by uploader, can contain any characters |
| File path | HIGH | Derived from title, may contain quotes and backslashes |
| Download URL | MEDIUM | User-pasted, could contain crafted fragments |
| Error messages | MEDIUM | yt-dlp error output may echo parts of the URL |
| History entries | MEDIUM | Persisted to disk, replayed on UI load |

## Best Practices

- [ ] Prefer `textContent` over `innerHTML` for text display
- [ ] Prefer `addEventListener` over inline `onclick`
- [ ] If `innerHTML` is unavoidable, escape ALL interpolated values
- [ ] Use separate escaping for JS string context vs HTML content context
- [ ] Treat ALL data from yt-dlp output as untrusted
- [ ] Test with filenames containing `' " < > & \` characters
