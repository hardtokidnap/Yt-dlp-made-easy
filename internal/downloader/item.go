package downloader

import (
	"time"

	"github.com/google/uuid"
)

// Status represents download status
type Status string

const (
	StatusPending     Status = "pending"
	StatusDownloading Status = "downloading"
	StatusPaused      Status = "paused"
	StatusStopped     Status = "stopped"
	StatusCompleted   Status = "completed"
	StatusError       Status = "error"
)

// Item represents a single download
type Item struct {
	ID            string    `json:"id"`
	URL           string    `json:"url"`
	Title         string    `json:"title"`
	Status        Status    `json:"status"`
	Progress      float64   `json:"progress"` // 0-100
	Speed         string    `json:"speed"`
	ETA           string    `json:"eta"`
	FilePath      string    `json:"file_path"`
	FileSize      int64     `json:"file_size"`
	Error         string    `json:"error"`
	CreatedAt     time.Time `json:"created_at"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	IsAudioOnly   bool      `json:"is_audio_only"`
	Quality       string    `json:"quality"`
	Format        string    `json:"format"`
	CurrentItem   int       `json:"current_item"` // For playlists
	TotalItems    int       `json:"total_items"`
	ProcessPID    int       `json:"-"` // Not serialized
}

// NewItem creates a new download item
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
	}
}

// SetStatus updates the status and timestamps
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
	}
}

// SetError sets error status and message
func (i *Item) SetError(err error) {
	i.Status = StatusError
	i.Error = err.Error()
}

// CanPause returns true if item can be paused
func (i *Item) CanPause() bool {
	return i.Status == StatusDownloading
}

// CanResume returns true if item can be resumed
func (i *Item) CanResume() bool {
	return i.Status == StatusPaused || i.Status == StatusStopped
}

// CanStop returns true if item can be stopped
func (i *Item) CanStop() bool {
	return i.Status == StatusDownloading || i.Status == StatusPaused
}

// IsActive returns true if download is active
func (i *Item) IsActive() bool {
	return i.Status == StatusDownloading || i.Status == StatusPaused
}

// DisplayTitle returns title or truncated URL
func (i *Item) DisplayTitle() string {
	if i.Title != "" {
		return i.Title
	}
	if len(i.URL) > 60 {
		return i.URL[:57] + "..."
	}
	return i.URL
}
