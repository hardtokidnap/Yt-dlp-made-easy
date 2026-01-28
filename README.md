# YT-DLP Made Easy

> **Status:** 
WIP. This project is currently being rebuilt from the ground up using **Go** and **Wails**.

A modern, lightweight GUI wrapper for `yt-dlp` designed to provide a native experience on Windows 11.

> [!WARNING]
 This is a hobby project. I do not really care if it doesn't work for you. Sorry not sorry. Issues most likely won't be handled.



## About the project

**The original Python/CustomTkinter application is being deprecated** in favor of a robust Go backend coupled with a Wails frontend. This transition reduces the application size to a single \~8MB binary, eliminates external dependencies, and provides better performance through Go's native concurrency.
___

## Key features

The new architecture introduces several improvements over the previous version:

* **Better SponsorBlock integration:** automatically skip sponsored segments in videos

* **Build switching:** seamless toggling between stable and nightly `yt-dlp` builds to access the latest fixes

* **Concurrent downloads:** configurable queue allowing multiple simultaneous downloads

* **Process control:** ability to pause and resume downloads without connection drops

* **Advanced authentication:** support for browser cookies and PO tokens for restricted content

* **Visual queue:** real-time progress bars with detailed status information

* **History management:** search, filter, and re-download previously grabbed content

* **Native integration:** supports clipboard monitoring, drag-and-drop, and desktop notifications

* **JS External Runtime swap:** ability to swap between `Deno` (Default), `node.js`, `npm`, `QuickJS`, `QuickJS-ng`, and `Bun`

* **Custom args:** support for custom args with autocomplete (If i can do it without adding a million libraries)

More to come.

___

## Tech stack

* **Backend:** Go (handling process management and file I/O)

* **Frontend:** HTML5, JavaScript, Tailwind CSS

* **Build tool:** Wails (using system WebView2)

## Development

To work on this project, you will need Go 1.21+ and the Wails CLI installed.

> [!NOTE]
> The app won't run. It's still in it's infancy. I'm just adding it to have it here.
### Setup

1. Install Wails: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

2. Clone the repository

3. Initialize dependencies: `go mod tidy`

4. Install frontend dependencies: `cd frontend && npm install`

### Commands

* **Run in dev mode:** `wails dev` (enables hot reload)

* **Build for Windows:** `wails build -platform windows/amd64`

* **Production build:** `wails build -platform windows/amd64 -ldflags="-s -w"`

## Structure

* `app.go`: Main application logic and frontend bindings

* `frontend/`: Contains the HTML, Tailwind CSS, and JavaScript logic

* `internal/`: Core packages for the downloader, queue, and settings management
