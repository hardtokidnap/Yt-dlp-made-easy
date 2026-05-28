package converter

import (
	"strings"
	"testing"
)

func TestApplyQualityTier_LibX264(t *testing.T) {
	cases := []struct {
		tier       string
		wantCRF    int
		wantPreset string
	}{
		{"low", 28, "fast"},
		{"medium", 23, "medium"},
		{"high", 18, "slow"},
	}
	for _, tc := range cases {
		t.Run(tc.tier, func(t *testing.T) {
			opts := ConversionOptions{QualityTier: tc.tier, VideoCodec: "libx264", OutputFormat: "mp4"}
			warnings := applyQualityTier(&opts)
			if opts.CRF != tc.wantCRF {
				t.Errorf("CRF: want %d, got %d", tc.wantCRF, opts.CRF)
			}
			if opts.Preset != tc.wantPreset {
				t.Errorf("Preset: want %q, got %q", tc.wantPreset, opts.Preset)
			}
			if len(warnings) != 0 {
				t.Errorf("unexpected warnings: %v", warnings)
			}
		})
	}
}

func TestApplyQualityTier_LosslessAudioForcesFlac(t *testing.T) {
	opts := ConversionOptions{QualityTier: "lossless", OutputFormat: "mp3"}
	warnings := applyQualityTier(&opts)
	if opts.OutputFormat != "flac" {
		t.Errorf("OutputFormat: want flac, got %q", opts.OutputFormat)
	}
	if opts.AudioCodec != "flac" {
		t.Errorf("AudioCodec: want flac, got %q", opts.AudioCodec)
	}
	if len(warnings) == 0 {
		t.Errorf("expected a warning about format override")
	}
}

func TestApplyQualityTier_LosslessAv1IsRejected(t *testing.T) {
	opts := ConversionOptions{QualityTier: "lossless", VideoCodec: "libaom-av1", OutputFormat: "mp4"}
	warnings := applyQualityTier(&opts)
	if len(warnings) == 0 {
		t.Errorf("expected a warning that AV1 lossless is unsupported")
	}
}

func TestApplyQualityTier_CustomTierIsNoOp(t *testing.T) {
	opts := ConversionOptions{QualityTier: "custom", CRF: 19, AudioBitrate: "320k"}
	applyQualityTier(&opts)
	if opts.CRF != 19 {
		t.Errorf("custom tier should not modify CRF, got %d", opts.CRF)
	}
	if opts.AudioBitrate != "320k" {
		t.Errorf("custom tier should not modify AudioBitrate, got %q", opts.AudioBitrate)
	}
}

func TestBuildArgs_LosslessLibX264_EmitsLosslessFlag(t *testing.T) {
	opts := ConversionOptions{
		InputFile:    "x.mkv",
		OutputFormat: "mp4",
		VideoCodec:   "libx264",
		QualityTier:  "lossless",
	}
	args := BuildArgs(opts)
	joined := strings.Join(args, " ")
	// Either -qp 0 or -crf 0 must appear; the gate on opts.CRF > 0 must not
	// silently drop the lossless instruction.
	if !strings.Contains(joined, "-qp 0") && !strings.Contains(joined, "-crf 0") {
		t.Errorf("libx264 lossless must emit -qp 0 or -crf 0, got: %s", joined)
	}
}

func TestBuildArgs_LosslessVP9_EmitsLosslessFlag(t *testing.T) {
	opts := ConversionOptions{
		InputFile:    "x.mkv",
		OutputFormat: "webm",
		VideoCodec:   "libvpx-vp9",
		QualityTier:  "lossless",
	}
	args := BuildArgs(opts)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-lossless 1") {
		t.Errorf("libvpx-vp9 lossless must emit -lossless 1, got: %s", joined)
	}
}

func TestBuildArgs_AppliesTier(t *testing.T) {
	opts := ConversionOptions{
		InputFile:    "input.mkv",
		OutputFormat: "mp4",
		VideoCodec:   "libx264",
		QualityTier:  "high",
	}
	args := BuildArgs(opts)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-crf 18") {
		t.Errorf("expected -crf 18 in args, got: %s", joined)
	}
	if !strings.Contains(joined, "-preset slow") {
		t.Errorf("expected -preset slow in args, got: %s", joined)
	}
	if !strings.Contains(joined, "-b:a 256k") {
		t.Errorf("expected -b:a 256k in args, got: %s", joined)
	}
}
