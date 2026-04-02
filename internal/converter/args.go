package converter

import (
	"path/filepath"
	"strconv"
	"strings"
)

// ConversionOptions holds all parameters for a conversion job.
type ConversionOptions struct {
	InputFile    string `json:"input_file"`
	OutputFile   string `json:"output_file"`
	OutputFormat string `json:"output_format"`
	VideoCodec   string `json:"video_codec"`
	AudioCodec   string `json:"audio_codec"`
	Preset       string `json:"preset"`
	VideoBitrate string `json:"video_bitrate"`
	AudioBitrate string `json:"audio_bitrate"`
	Resolution   string `json:"resolution"`
	CRF          int    `json:"crf"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
	CustomArgs   string `json:"custom_args"`
}

// Preset is a named collection of conversion settings.
type Preset struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// Settings applied when this preset is selected
	OutputFormat string `json:"output_format"`
	VideoCodec   string `json:"video_codec"`
	AudioCodec   string `json:"audio_codec"`
	Preset       string `json:"preset"`
	AudioBitrate string `json:"audio_bitrate"`
	AudioOnly    bool   `json:"audio_only"`
}

// GetPresets returns the built-in conversion presets.
func GetPresets() []Preset {
	return []Preset{
		{
			ID:           "video_to_mp3",
			Name:         "Video to MP3",
			Description:  "Extract audio as MP3",
			OutputFormat: "mp3",
			AudioCodec:   "libmp3lame",
			AudioBitrate: "192k",
			AudioOnly:    true,
		},
		{
			ID:           "convert_mp4",
			Name:         "Convert to MP4",
			Description:  "Re-encode to MP4 (H.264 + AAC)",
			OutputFormat: "mp4",
			VideoCodec:   "libx264",
			AudioCodec:   "aac",
			Preset:       "medium",
			AudioBitrate: "192k",
		},
		{
			ID:           "convert_mkv",
			Name:         "Convert to MKV",
			Description:  "Re-encode to MKV (H.264 + AAC)",
			OutputFormat: "mkv",
			VideoCodec:   "libx264",
			AudioCodec:   "aac",
			Preset:       "medium",
			AudioBitrate: "192k",
		},
		{
			ID:           "convert_webm",
			Name:         "Convert to WebM",
			Description:  "Re-encode to WebM (VP9 + Opus)",
			OutputFormat: "webm",
			VideoCodec:   "libvpx-vp9",
			AudioCodec:   "libopus",
			AudioBitrate: "128k",
		},
		{
			ID:           "extract_audio",
			Name:         "Extract Audio",
			Description:  "Copy audio stream without re-encoding",
			OutputFormat: "m4a",
			AudioCodec:   "copy",
			AudioOnly:    true,
		},
		{
			ID:           "convert_flac",
			Name:         "Convert to FLAC",
			Description:  "Lossless audio conversion",
			OutputFormat: "flac",
			AudioCodec:   "flac",
			AudioOnly:    true,
		},
		{
			ID:           "convert_wav",
			Name:         "Convert to WAV",
			Description:  "Uncompressed audio",
			OutputFormat: "wav",
			AudioCodec:   "pcm_s16le",
			AudioOnly:    true,
		},
	}
}

// audioOnlyFormats are output formats where video stream should be stripped.
var audioOnlyFormats = map[string]bool{
	"mp3": true, "m4a": true, "aac": true, "ogg": true,
	"opus": true, "flac": true, "wav": true,
}

// BuildArgs constructs the ffmpeg command-line arguments.
// Arg order: [-y] [-ss start] [-i input] [-t duration] [codec/quality args] [output]
func BuildArgs(opts ConversionOptions) []string {
	args := []string{"-y"}

	// Pre-input: -ss before -i for fast seek to nearest keyframe
	if opts.StartTime != "" {
		args = append(args, "-ss", opts.StartTime)
	}

	args = append(args, "-i", opts.InputFile)

	// Post-input: -to for end time (interpreted relative to -ss when -ss is before -i)
	if opts.EndTime != "" {
		args = append(args, "-to", opts.EndTime)
	}

	isAudioOnly := audioOnlyFormats[opts.OutputFormat]

	if isAudioOnly {
		args = append(args, "-vn")
	}

	// Video codec
	if !isAudioOnly && opts.VideoCodec != "" {
		args = append(args, "-c:v", opts.VideoCodec)
	}

	// Audio codec
	if opts.AudioCodec != "" {
		args = append(args, "-c:a", opts.AudioCodec)
	}

	// Encoder preset (ultrafast, fast, medium, slow, etc.)
	if opts.Preset != "" && !isAudioOnly {
		args = append(args, "-preset", opts.Preset)
	}

	// Quality: CRF takes priority over video bitrate (they are mutually exclusive in ffmpeg)
	if opts.CRF > 0 && !isAudioOnly {
		args = append(args, "-crf", strconv.Itoa(opts.CRF))
		// VP9 requires -b:v 0 for constant quality mode
		if opts.VideoCodec == "libvpx-vp9" {
			args = append(args, "-b:v", "0")
		}
	} else if opts.VideoBitrate != "" && !isAudioOnly {
		args = append(args, "-b:v", opts.VideoBitrate)
	}

	// Audio bitrate
	if opts.AudioBitrate != "" {
		args = append(args, "-b:a", opts.AudioBitrate)
	}

	// Resolution scaling
	if opts.Resolution != "" && !isAudioOnly {
		args = append(args, "-vf", "scale="+opts.Resolution)
	}

	// Custom args — split on spaces but respect quoted values
	if opts.CustomArgs != "" {
		args = append(args, splitArgs(opts.CustomArgs)...)
	}

	// Output file — derive from input if not specified
	output := opts.OutputFile
	if output == "" {
		ext := opts.OutputFormat
		if ext == "" {
			ext = "mp4"
		}
		base := strings.TrimSuffix(filepath.Base(opts.InputFile), filepath.Ext(opts.InputFile))
		output = filepath.Join(filepath.Dir(opts.InputFile), base+"_converted."+ext)
	}
	args = append(args, output)

	return args
}

// ParseTimeToSeconds parses HH:MM:SS, MM:SS, or SS format to seconds.
func ParseTimeToSeconds(t string) float64 {
	parts := strings.Split(t, ":")
	switch len(parts) {
	case 3:
		h, _ := strconv.ParseFloat(parts[0], 64)
		m, _ := strconv.ParseFloat(parts[1], 64)
		s, _ := strconv.ParseFloat(parts[2], 64)
		return h*3600 + m*60 + s
	case 2:
		m, _ := strconv.ParseFloat(parts[0], 64)
		s, _ := strconv.ParseFloat(parts[1], 64)
		return m*60 + s
	case 1:
		s, _ := strconv.ParseFloat(parts[0], 64)
		return s
	default:
		return 0
	}
}

// splitArgs splits a string into arguments, respecting single and double quotes.
// e.g. `-metadata title="My Video" -ss 00:01:00` → ["-metadata", "title=My Video", "-ss", "00:01:00"]
func splitArgs(s string) []string {
	var args []string
	var cur []byte
	var quote byte

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			} else {
				cur = append(cur, c)
			}
		case c == '"' || c == '\'':
			quote = c
		case c == ' ' || c == '\t':
			if len(cur) > 0 {
				args = append(args, string(cur))
				cur = cur[:0]
			}
		default:
			cur = append(cur, c)
		}
	}
	if len(cur) > 0 {
		args = append(args, string(cur))
	}
	return args
}
