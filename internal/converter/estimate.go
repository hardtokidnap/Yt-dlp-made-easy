package converter

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// SizeEstimate is the predicted output file size for a conversion job.
type SizeEstimate struct {
	Bytes      int64  `json:"bytes"`
	Confidence string `json:"confidence"` // "exact" | "estimate" | "unknown"
	Note       string `json:"note"`       // human-readable explanation for the UI
}

// codecCRFAnchor holds the bits-per-pixel anchor for a codec at a known CRF.
// bpp at any other CRF: anchor.bpp * 2^((anchor.crf - currentCRF) / 6)
// (well-known x264 doubling rule per 6 CRF steps, applied approximately to other codecs).
type codecCRFAnchor struct {
	anchorCRF int
	anchorBPP float64
}

var codecAnchors = map[string]codecCRFAnchor{
	"libx264":    {anchorCRF: 23, anchorBPP: 0.1},
	"libx265":    {anchorCRF: 28, anchorBPP: 0.05},
	"libvpx-vp9": {anchorCRF: 31, anchorBPP: 0.07},
	"libaom-av1": {anchorCRF: 30, anchorBPP: 0.04},
}

// EstimateOutputSize predicts the output file size for the given options + probed media info.
// Confidence levels: "exact" (math is deterministic from inputs), "estimate" (CRF heuristic
// or stream-copy approximation), "unknown" (lossless, missing probe, unsupported combination).
func EstimateOutputSize(opts ConversionOptions, info *MediaInfo) SizeEstimate {
	if info == nil || info.DurationSec <= 0 {
		return SizeEstimate{Confidence: "unknown", Note: "Source media not probed yet."}
	}

	// Tier mutations must match what BuildArgs will do, so the estimate stays honest.
	applyQualityTier(&opts)

	duration := info.DurationSec
	if opts.StartTime != "" || opts.EndTime != "" {
		start := 0.0
		end := duration
		if opts.StartTime != "" {
			start = ParseTimeToSeconds(opts.StartTime)
		}
		if opts.EndTime != "" {
			end = ParseTimeToSeconds(opts.EndTime)
		}
		if end > start {
			duration = end - start
		}
	}

	isAudioOnly := audioOnlyFormats[opts.OutputFormat]
	audioBps := parseBitrateBps(opts.AudioBitrate)

	if isAudioOnly {
		switch opts.AudioCodec {
		case "flac":
			return SizeEstimate{
				Confidence: "unknown",
				Note:       "FLAC output size depends on content (typically 3-6x of source MP3).",
			}
		case "pcm_s16le":
			return SizeEstimate{
				Confidence: "unknown",
				Note:       "Uncompressed WAV size depends on sample rate and channels.",
			}
		}
		if opts.OutputFormat == "wav" {
			return SizeEstimate{
				Confidence: "unknown",
				Note:       "Uncompressed WAV size depends on sample rate and channels.",
			}
		}
		if audioBps <= 0 {
			return SizeEstimate{Confidence: "unknown", Note: "Audio bitrate not set."}
		}
		bytes := int64(float64(audioBps) / 8.0 * duration)
		return SizeEstimate{Bytes: bytes, Confidence: "exact", Note: formatSizeNote(bytes, "exact", "")}
	}

	// Lossless video: real size is content-dependent (5-10x typical).
	if opts.QualityTier == "lossless" || (opts.CRF == 0 && opts.VideoCodec != "copy" && strings.Contains(opts.CustomArgs, "lossless")) {
		return SizeEstimate{
			Confidence: "unknown",
			Note:       "Lossless output, size depends on content (typically 5-10x source for video).",
		}
	}

	// Stream copy: approximate using source total bitrate if available.
	if opts.VideoCodec == "copy" {
		if info.Bitrate != "" {
			bps := parseBitrateString(info.Bitrate)
			if bps > 0 {
				bytes := int64(float64(bps) / 8.0 * duration)
				return SizeEstimate{Bytes: bytes, Confidence: "estimate", Note: formatSizeNote(bytes, "estimate", "source-rate copy")}
			}
		}
		return SizeEstimate{Confidence: "unknown", Note: "Stream copy: source bitrate not available."}
	}

	// Video bitrate mode: deterministic.
	videoBps := parseBitrateBps(opts.VideoBitrate)
	if videoBps > 0 {
		bytes := int64(float64(videoBps+audioBps) / 8.0 * duration)
		return SizeEstimate{Bytes: bytes, Confidence: "exact", Note: formatSizeNote(bytes, "exact", "")}
	}

	// Video CRF mode: bits-per-pixel heuristic per codec.
	if opts.CRF > 0 && info.Width > 0 && info.Height > 0 {
		codec := opts.VideoCodec
		if codec == "" {
			codec = "libx264"
		}
		anchor, ok := codecAnchors[codec]
		if !ok {
			return SizeEstimate{Confidence: "unknown", Note: "Codec " + codec + " not supported for size estimation."}
		}
		fps := info.FPS
		if fps <= 0 {
			fps = 30
		}
		bpp := anchor.anchorBPP * math.Pow(2, float64(anchor.anchorCRF-opts.CRF)/6.0)
		videoBpsEst := bpp * float64(info.Width) * float64(info.Height) * fps
		totalBps := videoBpsEst + float64(audioBps)
		bytes := int64(totalBps / 8.0 * duration)
		return SizeEstimate{Bytes: bytes, Confidence: "estimate", Note: formatSizeNote(bytes, "estimate", "CRF mode, ±20%")}
	}

	// Re-encode with no explicit bitrate/CRF/tier ("Auto"): ffmpeg picks codec
	// defaults, so the exact size is not derivable. Rather than show nothing,
	// fall back to the source total bitrate as a rough ballpark.
	if info.Bitrate != "" {
		if bps := parseBitrateString(info.Bitrate); bps > 0 {
			bytes := int64(float64(bps) / 8.0 * duration)
			return SizeEstimate{Bytes: bytes, Confidence: "estimate", Note: formatSizeNote(bytes, "estimate", "rough, based on source")}
		}
	}

	return SizeEstimate{Confidence: "unknown", Note: "Not enough information to estimate."}
}

// parseBitrateBps parses ffmpeg-style bitrate strings ("192k", "5M", "1000000") into bits/sec.
func parseBitrateBps(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	mult := int64(1)
	last := s[len(s)-1]
	switch last {
	case 'k', 'K':
		mult = 1_000
		s = s[:len(s)-1]
	case 'm', 'M':
		mult = 1_000_000
		s = s[:len(s)-1]
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return int64(n * float64(mult))
}

// parseBitrateString parses human-readable bitrate strings ("5.0 Mbps") back to bps.
// Used for reading MediaInfo.Bitrate which is pre-formatted.
func parseBitrateString(s string) int64 {
	s = strings.TrimSpace(s)
	lower := strings.ToLower(s)
	switch {
	case strings.HasSuffix(lower, "mbps"):
		n, _ := strconv.ParseFloat(strings.TrimSpace(s[:len(s)-4]), 64)
		return int64(n * 1_000_000)
	case strings.HasSuffix(lower, "kbps"):
		n, _ := strconv.ParseFloat(strings.TrimSpace(s[:len(s)-4]), 64)
		return int64(n * 1_000)
	}
	n, _ := strconv.ParseFloat(s, 64)
	return int64(n)
}

func formatSizeNote(bytes int64, confidence, tag string) string {
	pretty := formatFileSize(bytes)
	switch confidence {
	case "exact":
		return pretty
	case "estimate":
		if tag != "" {
			return fmt.Sprintf("~%s (%s)", pretty, tag)
		}
		return "~" + pretty
	}
	return pretty
}
