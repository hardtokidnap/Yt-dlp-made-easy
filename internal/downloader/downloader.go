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

// Downloader wraps a yt-dlp process with pause/resume support via Windows NT APIs.
// Each download gets its own Downloader instance managed by the Queue.
type Downloader struct {
	item     *Item
	settings *config.Settings
	cmd      *exec.Cmd
	cancel   context.CancelFunc
	logFile  *os.File
	OnLog    func(line string)
}

func NewDownloader(item *Item, settings *config.Settings) *Downloader {
	return &Downloader{
		item:     item,
		settings: settings,
	}
}

func (d *Downloader) Start(ctx context.Context) error {
	args := BuildArgs(d.item.URL, d.settings, d.item.IsAudioOnly)

	ctx, cancel := context.WithCancel(ctx)
	d.cancel = cancel

	d.cmd = exec.CommandContext(ctx, args[0], args[1:]...)

	// Prevent console window flash on Windows
	d.cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}

	logFile, err := os.OpenFile(util.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	d.logFile = logFile

	stdout, err := d.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := d.cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := d.cmd.Start(); err != nil {
		return err
	}

	d.item.ProcessPID = d.cmd.Process.Pid
	d.item.SetStatus(StatusDownloading)

	// Stream stdout to log file, frontend, and progress parser
	go func() {
		defer stdout.Close()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if d.logFile != nil {
				d.logFile.WriteString(line + "\n")
			}
			d.emitLog(line)
			d.parseOutput(line)
		}
	}()

	// Stream stderr - yt-dlp writes errors here
	go func() {
		defer stderr.Close()
		defer func() {
			if d.logFile != nil {
				d.logFile.Close()
			}
		}()

		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if d.logFile != nil {
				d.logFile.WriteString("[stderr] " + line + "\n")
			}
			d.emitLog(line)
			d.parseOutput(line)
		}
	}()

	return nil
}

func (d *Downloader) emitLog(line string) {
	if d.OnLog != nil {
		d.OnLog(line)
	}
}

// Wait blocks until yt-dlp exits, then updates item status based on exit code.
func (d *Downloader) Wait() error {
	if d.cmd == nil {
		return fmt.Errorf("no process running")
	}

	err := d.cmd.Wait()

	if err == nil {
		d.item.SetStatus(StatusCompleted)
	} else if d.item.Status != StatusStopped {
		// Preserve yt-dlp error message if we captured one, otherwise use Go error
		if d.item.Error == "" {
			d.item.SetErrorFromString(err.Error())
		} else {
			// Error was already captured from output, just ensure status is set
			d.item.Status = StatusError
			// Re-classify if not already done
			if d.item.ErrorType == "" || d.item.ErrorType == ErrorUnknown {
				classified := ClassifyError(d.item.Error)
				d.item.ErrorType = classified.Type
				d.item.Suggestions = classified.Suggestions
			}
		}
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

func (d *Downloader) Stop() error {
	if d.cancel != nil {
		d.cancel()
	}

	d.item.SetStatus(StatusStopped)
	return nil
}

// parseOutput extracts progress, filenames, and errors from yt-dlp's stdout/stderr.
// yt-dlp uses a specific format: [download] 50.0% of 100.00MiB at 5.00MiB/s ETA 00:10
func (d *Downloader) parseOutput(line string) {
	line = strings.TrimSpace(line)

	// Progress: [download] 50.0% of 100.00MiB at 5.00MiB/s ETA 00:10
	progressRe := regexp.MustCompile(`\[download\]\s+(\d+\.?\d*)%\s+of\s+~?\s*(\S+)\s+at\s+(\S+)\s+ETA\s+(\S+)`)
	if matches := progressRe.FindStringSubmatch(line); matches != nil {
		if percent, err := strconv.ParseFloat(matches[1], 64); err == nil {
			d.item.Progress = percent
			d.item.Speed = matches[3]
			d.item.ETA = matches[4]
		}
		return
	}

	// Destination: [download] Destination: filename.mp4
	destRe := regexp.MustCompile(`\[download\]\s+Destination:\s+(.+)`)
	if matches := destRe.FindStringSubmatch(line); matches != nil {
		d.item.FilePath = strings.TrimSpace(matches[1])
		if d.item.Title == "" {
			title := strings.TrimSuffix(matches[1], ".mp4")
			title = strings.TrimSuffix(title, ".webm")
			title = strings.TrimSuffix(title, ".mkv")
			d.item.Title = title
		}
		return
	}

	// Playlist: [download] Downloading item 1 of 10
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

	// Errors come in various formats - capture and classify them
	if strings.Contains(line, "ERROR:") || strings.Contains(line, "error:") {
		var errMsg string
		if idx := strings.Index(line, "ERROR:"); idx != -1 {
			errMsg = strings.TrimSpace(line[idx:])
		} else if idx := strings.Index(line, "error:"); idx != -1 {
			errMsg = strings.TrimSpace(line[idx:])
		} else {
			errMsg = line
		}
		d.item.SetErrorFromString(errMsg)
		return
	}

	// Some failures don't have ERROR: prefix
	if strings.Contains(line, "Unable to") {
		d.item.SetErrorFromString(line)
		return
	}

	// HTTP errors sometimes appear without ERROR: prefix
	if strings.Contains(line, "HTTP Error") || strings.Contains(line, "403") || strings.Contains(line, "429") {
		d.item.SetErrorFromString(line)
		return
	}

	if strings.Contains(line, "has already been downloaded") {
		d.item.Progress = 100
	}
}
