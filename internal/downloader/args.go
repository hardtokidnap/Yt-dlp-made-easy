package downloader

import (
	"fmt"
	"strings"

	"ytdlp-easy/internal/config"
	"ytdlp-easy/internal/jsruntime"
	"ytdlp-easy/internal/util"
)

// BuildArgs constructs yt-dlp command line arguments.
// URL is appended last as required by yt-dlp.
func BuildArgs(url string, settings *config.Settings, isAudioOnly bool) []string {
	args := []string{util.YtDlpPath}

	args = append(args, "-P", settings.General.SaveFolder)

	if settings.Advanced.OutputTemplate != "" {
		args = append(args, "-o", settings.Advanced.OutputTemplate)
	}

	// Resume partial downloads
	args = append(args, "--continue", "--newline")

	if isAudioOnly {
		args = append(args, "-x")
		args = append(args, "--audio-format", settings.Download.AudioFormat)
		args = append(args, "--audio-quality", settings.Download.AudioQuality)
	} else {
		formatStr := buildFormatString(settings.Download.Quality)
		if formatStr != "" {
			args = append(args, "-f", formatStr)
		}
		if settings.Download.Format != "" && settings.Download.Format != "best" {
			args = append(args, "--merge-output-format", settings.Download.Format)
		}
	}

	if settings.Download.EmbedThumbnail {
		args = append(args, "--embed-thumbnail")
	}
	if settings.Download.EmbedMetadata {
		args = append(args, "--embed-metadata")
	}
	if settings.Download.EmbedChapters {
		args = append(args, "--embed-chapters")
	}
	if settings.Download.Sponsorblock {
		args = append(args, "--sponsorblock-remove", "all")
	}

	if settings.Network.RateLimit != "" {
		args = append(args, "--limit-rate", settings.Network.RateLimit)
	}
	if settings.Network.Proxy != "" {
		args = append(args, "--proxy", settings.Network.Proxy)
	}
	if settings.Network.Retries > 0 {
		args = append(args, "--retries", fmt.Sprintf("%d", settings.Network.Retries))
		args = append(args, "--fragment-retries", fmt.Sprintf("%d", settings.Network.Retries))
	}

	// Cookies file takes priority - it's explicit and doesn't require browser to be closed
	if settings.Auth.CookiesFile != "" {
		args = append(args, "--cookies", settings.Auth.CookiesFile)
	} else if settings.Auth.CookiesBrowser != "" && settings.Auth.CookiesBrowser != "none" {
		args = append(args, "--cookies-from-browser", settings.Auth.CookiesBrowser)
	}

	// JavaScript runtime for YouTube extraction
	if runtimeArgs := jsruntime.GetYtDlpRuntimeArgs(settings.Advanced.JSRuntime); runtimeArgs != nil {
		args = append(args, runtimeArgs...)
	}

	// Build YouTube extractor args for player client and PO token
	var ytExtractorArgs []string

	// Player client selection (helps with 403 errors)
	playerClient := settings.Auth.PlayerClient
	if playerClient != "" && playerClient != "default" {
		ytExtractorArgs = append(ytExtractorArgs, fmt.Sprintf("player-client=%s", playerClient))
	}

	// PO Token support (format: CLIENT.gvs+TOKEN or just TOKEN for auto-detect)
	if settings.Auth.POToken != "" {
		// If token already has format (contains + or .), use as-is
		// Otherwise, format it for the selected client
		token := settings.Auth.POToken
		if playerClient != "" && playerClient != "default" && !strings.Contains(token, "+") && !strings.Contains(token, ".") {
			token = fmt.Sprintf("%s.gvs+%s", playerClient, token)
		}
		ytExtractorArgs = append(ytExtractorArgs, fmt.Sprintf("po_token=%s", token))
	}

	if len(ytExtractorArgs) > 0 {
		args = append(args, "--extractor-args", "youtube:"+strings.Join(ytExtractorArgs, ";"))
	}

	if settings.Advanced.ExtraArgs != "" {
		args = append(args, splitArgs(settings.Advanced.ExtraArgs)...)
	}

	args = append(args, url)
	return args
}

func buildFormatString(quality string) string {
	switch quality {
	case "best":
		return "bv*+ba/best"
	case "4K":
		return "bestvideo[height<=2160]+bestaudio/best[height<=2160]"
	case "1440p":
		return "bestvideo[height<=1440]+bestaudio/best[height<=1440]"
	case "1080p":
		return "bestvideo[height<=1080]+bestaudio/best[height<=1080]"
	case "720p":
		return "bestvideo[height<=720]+bestaudio/best[height<=720]"
	case "480p":
		return "bestvideo[height<=480]+bestaudio/best[height<=480]"
	case "360p":
		return "bestvideo[height<=360]+bestaudio/best[height<=360]"
	default:
		return "bv*+ba/best"
	}
}

// splitArgs handles quoted arguments properly
func splitArgs(s string) []string {
	if s == "" {
		return nil
	}
	var args []string
	var current string
	inQuote := false

	for _, c := range s {
		switch {
		case c == '"':
			inQuote = !inQuote
		case c == ' ' && !inQuote:
			if current != "" {
				args = append(args, current)
				current = ""
			}
		default:
			current += string(c)
		}
	}
	if current != "" {
		args = append(args, current)
	}
	return args
}

