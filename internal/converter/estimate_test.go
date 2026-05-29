package converter

import "testing"

func TestEstimateOutputSize_AudioBitrate_Exact(t *testing.T) {
	opts := ConversionOptions{
		OutputFormat: "mp3",
		AudioBitrate: "192k",
	}
	info := &MediaInfo{DurationSec: 60}
	est := EstimateOutputSize(opts, info)
	wantBytes := int64(192_000 / 8 * 60)
	if est.Bytes != wantBytes {
		t.Errorf("Bytes: want %d, got %d", wantBytes, est.Bytes)
	}
	if est.Confidence != "exact" {
		t.Errorf("Confidence: want exact, got %q", est.Confidence)
	}
}

func TestEstimateOutputSize_VideoBitrate_Exact(t *testing.T) {
	opts := ConversionOptions{
		OutputFormat: "mp4",
		VideoCodec:   "libx264",
		VideoBitrate: "5M",
		AudioBitrate: "192k",
	}
	info := &MediaInfo{DurationSec: 60, Width: 1920, Height: 1080, FPS: 30}
	est := EstimateOutputSize(opts, info)
	want := int64((5_000_000 + 192_000) / 8 * 60)
	if est.Bytes != want {
		t.Errorf("Bytes: want %d, got %d", want, est.Bytes)
	}
	if est.Confidence != "exact" {
		t.Errorf("Confidence: want exact, got %q", est.Confidence)
	}
}

func TestEstimateOutputSize_VideoCRF_Estimate(t *testing.T) {
	opts := ConversionOptions{
		OutputFormat: "mp4",
		VideoCodec:   "libx264",
		CRF:          23,
		AudioBitrate: "192k",
	}
	info := &MediaInfo{DurationSec: 60, Width: 1280, Height: 720, FPS: 30}
	est := EstimateOutputSize(opts, info)
	if est.Confidence != "estimate" {
		t.Errorf("Confidence: want estimate, got %q", est.Confidence)
	}
	if est.Bytes < 18_000_000 || est.Bytes > 28_000_000 {
		t.Errorf("Bytes for 720p30 CRF23 libx264 out of plausible range, got %d", est.Bytes)
	}
}

func TestEstimateOutputSize_Lossless_Unknown(t *testing.T) {
	opts := ConversionOptions{
		OutputFormat: "mp4",
		VideoCodec:   "libx264",
		QualityTier:  "lossless",
	}
	info := &MediaInfo{DurationSec: 60, Width: 1280, Height: 720, FPS: 30}
	est := EstimateOutputSize(opts, info)
	if est.Confidence != "unknown" {
		t.Errorf("Confidence: want unknown, got %q", est.Confidence)
	}
}

func TestEstimateOutputSize_NoInfo_Unknown(t *testing.T) {
	opts := ConversionOptions{OutputFormat: "mp4", VideoCodec: "libx264", CRF: 23}
	est := EstimateOutputSize(opts, nil)
	if est.Confidence != "unknown" {
		t.Errorf("Confidence: want unknown, got %q", est.Confidence)
	}
}

// "Auto" video re-encode (no bitrate/CRF/tier) should still give a ballpark from
// the source bitrate rather than "unknown", so the UI shows a number pre-convert.
func TestEstimateOutputSize_AutoReencode_SourceFallback(t *testing.T) {
	opts := ConversionOptions{OutputFormat: "mp4", VideoCodec: "libx264"}
	info := &MediaInfo{DurationSec: 60, Width: 1920, Height: 1080, FPS: 30, Bitrate: "5.0 Mbps"}
	est := EstimateOutputSize(opts, info)
	if est.Confidence != "estimate" {
		t.Errorf("Confidence: want estimate, got %q", est.Confidence)
	}
	want := int64(5_000_000 / 8 * 60)
	if est.Bytes != want {
		t.Errorf("Bytes: want %d, got %d", want, est.Bytes)
	}
}

func TestEstimateOutputSize_Trim_ScalesDuration(t *testing.T) {
	opts := ConversionOptions{
		OutputFormat: "mp3",
		AudioBitrate: "192k",
		StartTime:    "00:00:10",
		EndTime:      "00:00:40",
	}
	info := &MediaInfo{DurationSec: 120}
	est := EstimateOutputSize(opts, info)
	want := int64(192_000 / 8 * 30)
	if est.Bytes != want {
		t.Errorf("Bytes (trimmed): want %d, got %d", want, est.Bytes)
	}
}

func TestParseBitrateBps(t *testing.T) {
	cases := []struct {
		input string
		want  int64
	}{
		{"192k", 192_000},
		{"5M", 5_000_000},
		{"1000000", 1_000_000},
		{"", 0},
		{"garbage", 0},
		{"320K", 320_000},
		{"1.5M", 1_500_000},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			if got := parseBitrateBps(tc.input); got != tc.want {
				t.Errorf("parseBitrateBps(%q) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}
