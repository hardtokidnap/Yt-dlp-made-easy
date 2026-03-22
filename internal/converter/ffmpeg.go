package converter

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"ytdlp-easy/internal/util"
)

var (
	ffmpegDir  = filepath.Join(util.AppDataDir, "ffmpeg")
	ffmpegPath = filepath.Join(ffmpegDir, "ffmpeg.exe")
	ffprobePath = filepath.Join(ffmpegDir, "ffprobe.exe")
)

const ffmpegDownloadURL = "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-gpl.zip"

func FFmpegPath() string  { return ffmpegPath }
func FFprobePath() string { return ffprobePath }

func IsFFmpegInstalled() bool {
	_, err := os.Stat(ffmpegPath)
	return err == nil
}

func GetFFmpegVersion() (string, error) {
	if !IsFFmpegInstalled() {
		return "", fmt.Errorf("ffmpeg not installed")
	}

	cmd := hiddenCmd(ffmpegPath, "-version")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	// First line: "ffmpeg version N-xxxxx-gXXXXXXXX ..."
	line := strings.SplitN(string(out), "\n", 2)[0]
	return strings.TrimSpace(line), nil
}

// FFmpegDownloader handles downloading and extracting ffmpeg.
type FFmpegDownloader struct {
	OnProgress func(msg string)
}

func (d *FFmpegDownloader) emit(msg string) {
	if d.OnProgress != nil {
		d.OnProgress(msg)
	}
}

func (d *FFmpegDownloader) Download() error {
	d.emit("Downloading FFmpeg...")

	if err := os.MkdirAll(ffmpegDir, 0755); err != nil {
		return fmt.Errorf("create ffmpeg dir: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "ffmpeg-*.zip")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	resp, err := http.Get(ffmpegDownloadURL)
	if err != nil {
		return fmt.Errorf("download ffmpeg: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	totalSize := resp.ContentLength
	written := int64(0)
	buf := make([]byte, 64*1024)

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, wErr := tmpFile.Write(buf[:n]); wErr != nil {
				return fmt.Errorf("write temp file: %w", wErr)
			}
			written += int64(n)
			if totalSize > 0 {
				pct := float64(written) / float64(totalSize) * 100
				d.emit(fmt.Sprintf("Downloading FFmpeg... %.0f%%", pct))
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read response: %w", readErr)
		}
	}
	tmpFile.Close()

	d.emit("Extracting FFmpeg...")

	if err := d.extractFFmpeg(tmpFile.Name()); err != nil {
		return fmt.Errorf("extract ffmpeg: %w", err)
	}

	d.emit("FFmpeg installed successfully")
	return nil
}

// extractFFmpeg opens the zip and copies ffmpeg.exe + ffprobe.exe to ffmpegDir.
func (d *FFmpegDownloader) extractFFmpeg(zipPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	found := 0
	for _, f := range r.File {
		name := filepath.Base(f.Name)
		var destPath string

		switch strings.ToLower(name) {
		case "ffmpeg.exe":
			destPath = ffmpegPath
		case "ffprobe.exe":
			destPath = ffprobePath
		default:
			continue
		}

		d.emit(fmt.Sprintf("Extracting %s...", name))

		src, err := f.Open()
		if err != nil {
			return fmt.Errorf("open %s in zip: %w", name, err)
		}

		dst, err := os.Create(destPath)
		if err != nil {
			src.Close()
			return fmt.Errorf("create %s: %w", destPath, err)
		}

		if _, err := io.Copy(dst, src); err != nil {
			dst.Close()
			src.Close()
			return fmt.Errorf("extract %s: %w", name, err)
		}

		dst.Close()
		src.Close()
		found++

		if found == 2 {
			break
		}
	}

	if found < 2 {
		return fmt.Errorf("could not find ffmpeg.exe and ffprobe.exe in zip (found %d)", found)
	}
	return nil
}
