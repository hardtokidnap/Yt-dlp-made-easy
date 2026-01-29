package downloader

import (
	"regexp"
)

// ErrorType represents a classified error category
type ErrorType string

const (
	ErrorUnknown       ErrorType = "unknown"
	ErrorForbidden403  ErrorType = "forbidden_403"
	ErrorRateLimit429  ErrorType = "rate_limit_429"
	ErrorAgeRestricted ErrorType = "age_restricted"
	ErrorGeoBlocked    ErrorType = "geo_blocked"
	ErrorNotAvailable  ErrorType = "not_available"
	ErrorNetwork       ErrorType = "network"
	ErrorExtractor     ErrorType = "extractor_outdated"
	ErrorSignIn        ErrorType = "sign_in_required"
	ErrorCookieDB      ErrorType = "cookie_database"
)

// Solution represents a suggested fix for an error
type Solution struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Action      string `json:"action"`      // "apply_setting", "open_settings", "open_link", "retry"
	ActionData  string `json:"action_data"` // Setting key, URL, or empty
	Priority    int    `json:"priority"`    // Lower = higher priority
}

// ClassifiedError contains the error type and suggested solutions
type ClassifiedError struct {
	Type        ErrorType  `json:"type"`
	RawMessage  string     `json:"raw_message"`
	UserMessage string     `json:"user_message"` // Friendly description
	Suggestions []Solution `json:"suggestions"`
}

// errorPattern defines a regex pattern and its error type
type errorPattern struct {
	pattern   *regexp.Regexp
	errorType ErrorType
}

// Compiled patterns for error classification
var errorPatterns = []errorPattern{
	// 403 Forbidden errors
	{regexp.MustCompile(`(?i)403\s*Forbidden`), ErrorForbidden403},
	{regexp.MustCompile(`(?i)HTTP\s*Error\s*403`), ErrorForbidden403},
	{regexp.MustCompile(`(?i)Access\s*Denied`), ErrorForbidden403},
	{regexp.MustCompile(`(?i)unable to download video data.*403`), ErrorForbidden403},

	// Rate limiting
	{regexp.MustCompile(`(?i)429`), ErrorRateLimit429},
	{regexp.MustCompile(`(?i)Too\s*Many\s*Requests`), ErrorRateLimit429},
	{regexp.MustCompile(`(?i)throttled`), ErrorRateLimit429},
	{regexp.MustCompile(`(?i)rate.?limit`), ErrorRateLimit429},

	// Age restriction
	{regexp.MustCompile(`(?i)age.?restrict`), ErrorAgeRestricted},
	{regexp.MustCompile(`(?i)confirm your age`), ErrorAgeRestricted},
	{regexp.MustCompile(`(?i)age.?gate`), ErrorAgeRestricted},
	{regexp.MustCompile(`(?i)mature content`), ErrorAgeRestricted},

	// Geo blocking
	{regexp.MustCompile(`(?i)not available in your country`), ErrorGeoBlocked},
	{regexp.MustCompile(`(?i)geo.?block`), ErrorGeoBlocked},
	{regexp.MustCompile(`(?i)blocked in your`), ErrorGeoBlocked},
	{regexp.MustCompile(`(?i)not available in your region`), ErrorGeoBlocked},

	// Not available
	{regexp.MustCompile(`(?i)Private video`), ErrorNotAvailable},
	{regexp.MustCompile(`(?i)Video unavailable`), ErrorNotAvailable},
	{regexp.MustCompile(`(?i)video has been removed`), ErrorNotAvailable},
	{regexp.MustCompile(`(?i)video is no longer available`), ErrorNotAvailable},
	{regexp.MustCompile(`(?i)This video is private`), ErrorNotAvailable},

	// Sign-in required
	{regexp.MustCompile(`(?i)Sign in to confirm`), ErrorSignIn},
	{regexp.MustCompile(`(?i)requires.*login`), ErrorSignIn},
	{regexp.MustCompile(`(?i)sign.?in required`), ErrorSignIn},

	// Network errors
	{regexp.MustCompile(`(?i)timed?\s*out`), ErrorNetwork},
	{regexp.MustCompile(`(?i)Connection refused`), ErrorNetwork},
	{regexp.MustCompile(`(?i)Connection reset`), ErrorNetwork},
	{regexp.MustCompile(`(?i)Network is unreachable`), ErrorNetwork},
	{regexp.MustCompile(`(?i)Name or service not known`), ErrorNetwork},
	{regexp.MustCompile(`(?i)Could not resolve`), ErrorNetwork},

	// Extractor/signature issues
	{regexp.MustCompile(`(?i)js_url`), ErrorExtractor},
	{regexp.MustCompile(`(?i)signature extraction`), ErrorExtractor},
	{regexp.MustCompile(`(?i)nsig extraction`), ErrorExtractor},
	{regexp.MustCompile(`(?i)Extractor.*broken`), ErrorExtractor},
	{regexp.MustCompile(`(?i)Unable to extract`), ErrorExtractor},

	// Cookie database issues (browser is open or locked)
	{regexp.MustCompile(`(?i)Could not copy.*cookie database`), ErrorCookieDB},
	{regexp.MustCompile(`(?i)cookie.*database.*locked`), ErrorCookieDB},
}

// ClassifyError analyzes an error message and returns a classified error with solutions
func ClassifyError(errorMsg string) ClassifiedError {
	errorType := ErrorUnknown

	// Find matching pattern
	for _, ep := range errorPatterns {
		if ep.pattern.MatchString(errorMsg) {
			errorType = ep.errorType
			break
		}
	}

	return ClassifiedError{
		Type:        errorType,
		RawMessage:  errorMsg,
		UserMessage: getUserMessage(errorType),
		Suggestions: GetSolutions(errorType),
	}
}

// getUserMessage returns a user-friendly description for an error type
func getUserMessage(errorType ErrorType) string {
	messages := map[ErrorType]string{
		ErrorForbidden403:  "YouTube is blocking this download. This is a common issue that can usually be fixed.",
		ErrorRateLimit429:  "You've been rate-limited by YouTube. Too many requests in a short time.",
		ErrorAgeRestricted: "This video requires age verification.",
		ErrorGeoBlocked:    "This video is not available in your region.",
		ErrorNotAvailable:  "This video is private, removed, or no longer available.",
		ErrorNetwork:       "Network connection issue. Check your internet connection.",
		ErrorExtractor:     "The yt-dlp extractor needs to be updated.",
		ErrorSignIn:        "This video requires you to be signed in to YouTube.",
		ErrorCookieDB:      "Cannot access browser cookies. The browser is open or the database is locked.",
		ErrorUnknown:       "An error occurred while downloading.",
	}

	if msg, ok := messages[errorType]; ok {
		return msg
	}
	return messages[ErrorUnknown]
}

// GetSolutions returns suggested solutions for an error type, ordered by priority
func GetSolutions(errorType ErrorType) []Solution {
	switch errorType {
	case ErrorForbidden403:
		return []Solution{
			{
				ID:          "try_mweb",
				Title:       "Try Mobile Web Player",
				Description: "Switch to mobile web client which often bypasses 403 errors",
				Action:      "apply_setting",
				ActionData:  "player_client:mweb",
				Priority:    1,
			},
			{
				ID:          "add_cookies",
				Title:       "Add Browser Cookies",
				Description: "Use your browser's YouTube cookies for authentication",
				Action:      "open_settings",
				ActionData:  "auth",
				Priority:    2,
			},
			{
				ID:          "try_nightly",
				Title:       "Use Nightly Build",
				Description: "Nightly builds have the latest fixes for YouTube issues",
				Action:      "apply_setting",
				ActionData:  "use_nightly:true",
				Priority:    3,
			},
			{
				ID:          "help_403",
				Title:       "View Help Guide",
				Description: "Learn more about fixing 403 errors",
				Action:      "open_link",
				ActionData:  "https://github.com/yt-dlp/yt-dlp/wiki/FAQ#http-error-403-forbidden-when-downloading-a-video",
				Priority:    4,
			},
		}

	case ErrorAgeRestricted:
		return []Solution{
			{
				ID:          "add_po_token",
				Title:       "Add PO Token",
				Description: "PO Tokens allow downloading age-restricted videos",
				Action:      "open_settings",
				ActionData:  "auth",
				Priority:    1,
			},
			{
				ID:          "add_cookies",
				Title:       "Add Browser Cookies",
				Description: "Use cookies from a browser where you're logged in to YouTube",
				Action:      "open_settings",
				ActionData:  "auth",
				Priority:    2,
			},
			{
				ID:          "po_token_guide",
				Title:       "PO Token Guide",
				Description: "Step-by-step guide to get a PO Token",
				Action:      "open_link",
				ActionData:  "https://github.com/yt-dlp/yt-dlp/wiki/PO-Token-Guide",
				Priority:    3,
			},
		}

	case ErrorRateLimit429:
		return []Solution{
			{
				ID:          "reduce_concurrent",
				Title:       "Reduce Concurrent Downloads",
				Description: "Download one video at a time to avoid rate limits",
				Action:      "apply_setting",
				ActionData:  "max_concurrent:1",
				Priority:    1,
			},
			{
				ID:          "wait_retry",
				Title:       "Wait and Retry",
				Description: "Wait a few minutes before trying again",
				Action:      "retry",
				ActionData:  "",
				Priority:    2,
			},
			{
				ID:          "use_proxy",
				Title:       "Use a Proxy",
				Description: "Configure a proxy to use a different IP address",
				Action:      "open_settings",
				ActionData:  "network",
				Priority:    3,
			},
		}

	case ErrorExtractor:
		return []Solution{
			{
				ID:          "try_nightly",
				Title:       "Switch to Nightly Build",
				Description: "Nightly builds have the latest extractor fixes",
				Action:      "apply_setting",
				ActionData:  "use_nightly:true",
				Priority:    1,
			},
			{
				ID:          "update_ytdlp",
				Title:       "Update yt-dlp",
				Description: "Check for and install the latest yt-dlp version",
				Action:      "update_ytdlp",
				ActionData:  "",
				Priority:    2,
			},
		}

	case ErrorNetwork:
		return []Solution{
			{
				ID:          "check_connection",
				Title:       "Check Internet Connection",
				Description: "Verify your internet connection is working",
				Action:      "retry",
				ActionData:  "",
				Priority:    1,
			},
			{
				ID:          "increase_retries",
				Title:       "Increase Retries",
				Description: "Increase the number of retry attempts",
				Action:      "open_settings",
				ActionData:  "network",
				Priority:    2,
			},
			{
				ID:          "use_proxy",
				Title:       "Use a Proxy",
				Description: "Try using a proxy server",
				Action:      "open_settings",
				ActionData:  "network",
				Priority:    3,
			},
		}

	case ErrorGeoBlocked:
		return []Solution{
			{
				ID:          "use_proxy",
				Title:       "Use a Proxy/VPN",
				Description: "Use a proxy or VPN from a region where the video is available",
				Action:      "open_settings",
				ActionData:  "network",
				Priority:    1,
			},
			{
				ID:          "geo_help",
				Title:       "Learn About Geo-Blocking",
				Description: "Understand why videos are blocked by region",
				Action:      "open_link",
				ActionData:  "https://github.com/yt-dlp/yt-dlp#geo-restriction",
				Priority:    2,
			},
		}

	case ErrorSignIn:
		return []Solution{
			{
				ID:          "add_cookies",
				Title:       "Add Browser Cookies",
				Description: "Use cookies from a browser where you're logged in",
				Action:      "open_settings",
				ActionData:  "auth",
				Priority:    1,
			},
		}

	case ErrorCookieDB:
		return []Solution{
			{
				ID:          "use_cookies_file",
				Title:       "Use Cookies File Instead",
				Description: "Export cookies to a file - works even when browser is open",
				Action:      "open_settings",
				ActionData:  "auth",
				Priority:    1,
			},
			{
				ID:          "close_browser",
				Title:       "Close Browser & Retry",
				Description: "Close the browser completely, then retry the download",
				Action:      "retry",
				ActionData:  "",
				Priority:    2,
			},
			{
				ID:          "cookie_help",
				Title:       "Cookie Export Guide",
				Description: "Learn how to export cookies to a file",
				Action:      "open_link",
				ActionData:  "https://github.com/yt-dlp/yt-dlp/wiki/FAQ#how-do-i-pass-cookies-to-yt-dlp",
				Priority:    3,
			},
		}

	case ErrorNotAvailable:
		return []Solution{
			{
				ID:          "check_url",
				Title:       "Check the URL",
				Description: "Verify the video URL is correct and the video exists",
				Action:      "retry",
				ActionData:  "",
				Priority:    1,
			},
		}

	default:
		return []Solution{
			{
				ID:          "retry",
				Title:       "Retry Download",
				Description: "Try downloading again",
				Action:      "retry",
				ActionData:  "",
				Priority:    1,
			},
			{
				ID:          "view_log",
				Title:       "View Full Log",
				Description: "Check the log for more details",
				Action:      "open_log",
				ActionData:  "",
				Priority:    2,
			},
		}
	}
}

// IsRetryable returns true if the error type might succeed on retry
func IsRetryable(errorType ErrorType) bool {
	switch errorType {
	case ErrorNetwork, ErrorRateLimit429:
		return true
	case ErrorNotAvailable:
		return false
	default:
		return true // Most errors might be fixed with settings changes
	}
}
