package converter

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// MediaInfo holds parsed ffprobe output for display in the UI.
type MediaInfo struct {
	Duration    string  `json:"duration"`
	DurationSec float64 `json:"duration_sec"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	VideoCodec  string  `json:"video_codec"`
	AudioCodec  string  `json:"audio_codec"`
	Bitrate     string  `json:"bitrate"`
	FileSize    string  `json:"file_size"`
	HasVideo    bool    `json:"has_video"`
	HasAudio    bool    `json:"has_audio"`
}

// ffprobeOutput mirrors the JSON structure from ffprobe -show_format -show_streams.
type ffprobeOutput struct {
	Format  ffprobeFormat   `json:"format"`
	Streams []ffprobeStream `json:"streams"`
}

type ffprobeFormat struct {
	Duration string `json:"duration"`
	Size     string `json:"size"`
	BitRate  string `json:"bit_rate"`
}

type ffprobeStream struct {
	CodecType string `json:"codec_type"`
	CodecName string `json:"codec_name"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

// ProbeFile runs ffprobe on the given file and returns parsed media information.
func ProbeFile(filePath string) (*MediaInfo, error) {
	cmd := hiddenCmd(FFprobePath(),
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	var probe ffprobeOutput
	if err := json.Unmarshal(out, &probe); err != nil {
		return nil, fmt.Errorf("parse ffprobe output: %w", err)
	}

	info := &MediaInfo{}

	// Parse duration
	if probe.Format.Duration != "" {
		secs, _ := strconv.ParseFloat(probe.Format.Duration, 64)
		info.DurationSec = secs
		info.Duration = formatDuration(secs)
	}

	// Parse file size
	if probe.Format.Size != "" {
		bytes, _ := strconv.ParseInt(probe.Format.Size, 10, 64)
		info.FileSize = formatFileSize(bytes)
	}

	// Parse bitrate
	if probe.Format.BitRate != "" {
		bps, _ := strconv.ParseInt(probe.Format.BitRate, 10, 64)
		info.Bitrate = formatBitrate(bps)
	}

	// Find video and audio streams
	for _, s := range probe.Streams {
		switch s.CodecType {
		case "video":
			if !info.HasVideo {
				info.HasVideo = true
				info.VideoCodec = strings.ToUpper(s.CodecName)
				info.Width = s.Width
				info.Height = s.Height
			}
		case "audio":
			if !info.HasAudio {
				info.HasAudio = true
				info.AudioCodec = strings.ToUpper(s.CodecName)
			}
		}
	}

	return info, nil
}

func formatFileSize(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.0f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func formatBitrate(bps int64) string {
	if bps >= 1_000_000 {
		return fmt.Sprintf("%.1f Mbps", float64(bps)/1_000_000)
	}
	return fmt.Sprintf("%.0f kbps", float64(bps)/1_000)
}
