package downloader

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"

	"ytdlp-easy/internal/config"
	"ytdlp-easy/internal/util"
)

var (
	ntdll              = syscall.NewLazyDLL("ntdll.dll")
	ntSuspendProcess   = ntdll.NewProc("NtSuspendProcess")
	ntResumeProcess    = ntdll.NewProc("NtResumeProcess")
)

// Downloader manages a single yt-dlp process
type Downloader struct {
	item     *Item
	settings *config.Settings
	cmd      *exec.Cmd
	cancel   context.CancelFunc
	logFile  *os.File
}

// NewDownloader creates a new downloader
func NewDownloader(item *Item, settings *config.Settings) *Downloader {
	return &Downloader{
		item:     item,
		settings: settings,
	}
}

func (d *Downloader) Start(ctx context.Context) error {
	args := BuildArgs(d.item.URL, d.settings, d.item.IsAudioOnly)

	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	d.cancel = cancel

	// Create command
	d.cmd = exec.CommandContext(ctx, args[0], args[1:]...)

	// Open log file
	logFile, err := os.OpenFile(util.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	d.logFile = logFile

	// Get stdout pipe
	stdout, err := d.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	// Start command
	if err := d.cmd.Start(); err != nil {
		return err
	}

	d.item.ProcessPID = d.cmd.Process.Pid
	d.item.SetStatus(StatusDownloading)

	// Parse output in goroutine
	go func() {
		defer stdout.Close()
		defer d.logFile.Close()

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()

			// Write to log
			d.logFile.WriteString(line + "\n")

			// Parse progress
			d.parseOutput(line)
		}
	}()

	return nil
}

// Wait waits for the download to complete
func (d *Downloader) Wait() error {
	if d.cmd == nil {
		return fmt.Errorf("no process running")
	}

	err := d.cmd.Wait()

	if err == nil {
		d.item.SetStatus(StatusCompleted)
	} else if d.item.Status != StatusStopped {
		d.item.SetError(err)
	}

	return err
}

func (d *Downloader) Pause() error {
	if d.cmd == nil || d.cmd.Process == nil {
		return fmt.Errorf("no process running")
	}
	if !d.item.CanPause() {
		return fmt.Errorf("cannot pause in current state")
	}

	handle, err := windows.OpenProcess(
		windows.PROCESS_SUSPEND_RESUME,
		false,
		uint32(d.cmd.Process.Pid),
	)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)

	r, _, _ := ntSuspendProcess.Call(uintptr(handle))
	if r != 0 {
		return fmt.Errorf("NtSuspendProcess failed: %x", r)
	}

	d.item.SetStatus(StatusPaused)
	return nil
}

func (d *Downloader) Resume() error {
	if d.cmd == nil || d.cmd.Process == nil {
		return fmt.Errorf("no process running")
	}
	if !d.item.CanResume() {
		return fmt.Errorf("cannot resume in current state")
	}

	handle, err := windows.OpenProcess(
		windows.PROCESS_SUSPEND_RESUME,
		false,
		uint32(d.cmd.Process.Pid),
	)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)

	r, _, _ := ntResumeProcess.Call(uintptr(handle))
	if r != 0 {
		return fmt.Errorf("NtResumeProcess failed: %x", r)
	}

	d.item.SetStatus(StatusDownloading)
	return nil
}

// Stop stops the download
func (d *Downloader) Stop() error {
	if d.cancel != nil {
		d.cancel()
	}

	d.item.SetStatus(StatusStopped)
	return nil
}

// parseOutput parses yt-dlp output for progress info
func (d *Downloader) parseOutput(line string) {
	line = strings.TrimSpace(line)

	// Parse download progress
	// Format: [download]  50.0% of 100.00MiB at 5.00MiB/s ETA 00:10
	progressRe := regexp.MustCompile(`\[download\]\s+(\d+\.?\d*)%\s+of\s+~?\s*(\S+)\s+at\s+(\S+)\s+ETA\s+(\S+)`)
	if matches := progressRe.FindStringSubmatch(line); matches != nil {
		if percent, err := strconv.ParseFloat(matches[1], 64); err == nil {
			d.item.Progress = percent
			d.item.Speed = matches[3]
			d.item.ETA = matches[4]
		}
		return
	}

	// Parse destination filename
	// [download] Destination: filename.mp4
	destRe := regexp.MustCompile(`\[download\]\s+Destination:\s+(.+)`)
	if matches := destRe.FindStringSubmatch(line); matches != nil {
		d.item.FilePath = strings.TrimSpace(matches[1])

		// Extract title from filename if not set
		if d.item.Title == "" {
			title := strings.TrimSuffix(matches[1], ".mp4")
			title = strings.TrimSuffix(title, ".webm")
			title = strings.TrimSuffix(title, ".mkv")
			d.item.Title = title
		}
		return
	}

	// Parse playlist progress
	// [download] Downloading item 1 of 10
	playlistRe := regexp.MustCompile(`\[download\]\s+Downloading\s+item\s+(\d+)\s+of\s+(\d+)`)
	if matches := playlistRe.FindStringSubmatch(line); matches != nil {
		if current, err := strconv.Atoi(matches[1]); err == nil {
			d.item.CurrentItem = current
		}
		if total, err := strconv.Atoi(matches[2]); err == nil {
			d.item.TotalItems = total
		}
		return
	}

	// Parse errors
	if strings.Contains(line, "ERROR:") {
		d.item.Error = line
	}

	// Already downloaded
	if strings.Contains(line, "has already been downloaded") {
		d.item.Progress = 100
	}
}
