package downloader

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"

	"ytdlp-easy/internal/config"
	"ytdlp-easy/internal/util"
)

var (
	ntdll              = syscall.NewLazyDLL("ntdll.dll")
	ntSuspendProcess   = ntdll.NewProc("NtSuspendProcess")
	ntResumeProcess    = ntdll.NewProc("NtResumeProcess")

	progressRe = regexp.MustCompile(`\[download\]\s+(\d+\.?\d*)%\s+of\s+~?\s*(\S+)\s+at\s+(\S+)\s+ETA\s+(\S+)`)
	destRe     = regexp.MustCompile(`\[download\]\s+Destination:\s+(.+)`)
	playlistRe = regexp.MustCompile(`\[download\]\s+Downloading\s+item\s+(\d+)\s+of\s+(\d+)`)
	alreadyRe  = regexp.MustCompile(`\[download\]\s+(.+?)\s+has already been downloaded`)
	mergerRe   = regexp.MustCompile(`\[Merger\]\s+Merging formats into "(.+)"`)
	extractRe  = regexp.MustCompile(`\[ExtractAudio\]\s+Destination:\s+(.+)`)
	moveRe     = regexp.MustCompile(`\[MoveFiles\]\s+Moving file ".+" to "(.+)"`)
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
	// OnProgress fires on progress ticks (throttled) so the UI can update the
	// live percentage/speed/ETA. Wired by the Queue to emit a download:update.
	OnProgress func()
	lastEmit   time.Time
}

func NewDownloader(item *Item, settings *config.Settings) *Downloader {
	return &Downloader{
		item:     item,
		settings: settings,
	}
}

func (d *Downloader) Start(ctx context.Context) error {
	args := BuildArgs(d.item.URL, d.settings, d.item.IsAudioOnly, d.item.AudioFormat, d.item.AudioQuality)

	ctx, cancel := context.WithCancel(ctx)
	d.cancel = cancel

	d.cmd = exec.CommandContext(ctx, args[0], args[1:]...)

	// Force yt-dlp's stdio to UTF-8. With CREATE_NO_WINDOW there is no console,
	// so Python falls back to the locale codepage (e.g. cp1252) and drops
	// non-ASCII filename characters (？ … etc.) from its reported paths. That
	// made the captured FilePath mismatch the real file -> spurious "file
	// missing" and broken open-file/folder. UTF-8 mode keeps paths intact.
	d.cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8", "PYTHONUTF8=1")

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
		d.logFile.Close()
		return err
	}

	stderr, err := d.cmd.StderrPipe()
	if err != nil {
		d.logFile.Close()
		return err
	}

	if err := d.cmd.Start(); err != nil {
		d.logFile.Close()
		return err
	}

	d.item.ProcessPID = d.cmd.Process.Pid
	d.item.SetStatus(StatusDownloading)

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

	// yt-dlp writes progress and errors to stderr
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
			// Error was already captured from output - re-classify to ensure suggestions
			d.item.SetErrorFromString(d.item.Error)
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
	if matches := progressRe.FindStringSubmatch(line); matches != nil {
		if percent, err := strconv.ParseFloat(matches[1], 64); err == nil {
			d.item.Progress = percent
			d.item.Speed = matches[3]
			d.item.ETA = matches[4]
			// yt-dlp emits progress many times per second; throttle the UI
			// update to ~4/s so we move the bar without flooding events.
			if d.OnProgress != nil && time.Since(d.lastEmit) >= 250*time.Millisecond {
				d.lastEmit = time.Now()
				d.OnProgress()
			}
		}
		return
	}

	// Destination: [download] Destination: filename.mp4
	if matches := destRe.FindStringSubmatch(line); matches != nil {
		d.setFileInfo(strings.TrimSpace(matches[1]))
		return
	}

	// Playlist: [download] Downloading item 1 of 10
	if matches := playlistRe.FindStringSubmatch(line); matches != nil {
		if current, err := strconv.Atoi(matches[1]); err == nil {
			d.item.CurrentItem = current
		}
		if total, err := strconv.Atoi(matches[2]); err == nil {
			d.item.TotalItems = total
		}
		return
	}

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

	// yt-dlp sometimes reports failures without the ERROR: prefix
	if strings.Contains(line, "Unable to") {
		d.item.SetErrorFromString(line)
		return
	}

	if strings.Contains(line, "HTTP Error") || strings.Contains(line, "403") || strings.Contains(line, "429") {
		d.item.SetErrorFromString(line)
		return
	}

	if strings.Contains(line, "has already been downloaded") {
		d.item.Progress = 100
		if matches := alreadyRe.FindStringSubmatch(line); matches != nil {
			d.setFileInfo(strings.TrimSpace(matches[1]))
		}
		return
	}

	// Post-processing updates the final file path after merge/convert/move
	// [Merger] Merging formats into "filename.mp4"
	if matches := mergerRe.FindStringSubmatch(line); matches != nil {
		d.setFileInfo(strings.TrimSpace(matches[1]))
		return
	}

	// [ExtractAudio] Destination: filename.mp3
	if matches := extractRe.FindStringSubmatch(line); matches != nil {
		d.setFileInfo(strings.TrimSpace(matches[1]))
		return
	}

	// [MoveFiles] Moving file "source" to "destination"
	if matches := moveRe.FindStringSubmatch(line); matches != nil {
		d.setFileInfo(strings.TrimSpace(matches[1]))
		return
	}
}

// setFileInfo sets the file path and extracts a title from the filename.
func (d *Downloader) setFileInfo(path string) {
	d.item.FilePath = path
	if d.item.Title == "" {
		name := filepath.Base(path)
		d.item.Title = strings.TrimSuffix(name, filepath.Ext(name))
	}
}
