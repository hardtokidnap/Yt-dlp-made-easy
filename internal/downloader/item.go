package downloader

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending     Status = "pending"
	StatusDownloading Status = "downloading"
	StatusPaused      Status = "paused"
	StatusStopped     Status = "stopped"
	StatusCompleted   Status = "completed"
	StatusError       Status = "error"
)

// Item tracks state for a single download, including progress from yt-dlp
// and classified errors with suggested fixes.
type Item struct {
	ID            string     `json:"id"`
	URL           string     `json:"url"`
	Title         string     `json:"title"`
	Status        Status     `json:"status"`
	Progress      float64    `json:"progress"` // 0-100
	Speed         string     `json:"speed"`
	ETA           string     `json:"eta"`
	FilePath      string     `json:"file_path"`
	FileSize      int64      `json:"file_size"`
	Error         string     `json:"error"`
	ErrorType     ErrorType  `json:"error_type"`              // Classified error type
	Suggestions   []Solution `json:"suggestions,omitempty"`   // Suggested fixes
	CreatedAt     time.Time  `json:"created_at"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	IsAudioOnly   bool       `json:"is_audio_only"`
	Quality       string     `json:"quality"`
	Format        string     `json:"format"`
	CurrentItem   int        `json:"current_item"` // For playlists
	TotalItems    int        `json:"total_items"`
	FileExists    bool       `json:"file_exists"`
	ProcessPID    int        `json:"-"` // Not serialized
}

func NewItem(url string, isAudioOnly bool, quality, format string) *Item {
	return &Item{
		ID:          uuid.New().String(),
		URL:         url,
		Status:      StatusPending,
		Progress:    0,
		CreatedAt:   time.Now(),
		IsAudioOnly: isAudioOnly,
		Quality:     quality,
		Format:      format,
		CurrentItem: 1,
		TotalItems:  1,
		FileExists:  true,
	}
}

func (i *Item) SetStatus(status Status) {
	i.Status = status
	now := time.Now()

	switch status {
	case StatusDownloading:
		if i.StartedAt == nil {
			i.StartedAt = &now
		}
	case StatusCompleted:
		i.Progress = 100
		i.CompletedAt = &now
		i.FileExists = true
	}
}

// SetError classifies the error and populates fix suggestions.
func (i *Item) SetError(err error) {
	i.Status = StatusError
	i.Error = err.Error()
	classified := ClassifyError(i.Error)
	i.ErrorType = classified.Type
	i.Suggestions = classified.Suggestions
}

// SetErrorFromString classifies the error string and populates fix suggestions.
func (i *Item) SetErrorFromString(errMsg string) {
	i.Status = StatusError
	i.Error = errMsg
	classified := ClassifyError(errMsg)
	i.ErrorType = classified.Type
	i.Suggestions = classified.Suggestions
}

func (i *Item) CanPause() bool  { return i.Status == StatusDownloading }
func (i *Item) CanResume() bool { return i.Status == StatusPaused || i.Status == StatusStopped }
func (i *Item) CanStop() bool   { return i.Status == StatusDownloading || i.Status == StatusPaused }
func (i *Item) IsActive() bool  { return i.Status == StatusDownloading || i.Status == StatusPaused }

// DisplayTitle returns video title, falling back to truncated URL if unknown.
func (i *Item) DisplayTitle() string {
	if i.Title != "" {
		return i.Title
	}
	if len(i.URL) > 60 {
		return i.URL[:57] + "..."
	}
	return i.URL
}
