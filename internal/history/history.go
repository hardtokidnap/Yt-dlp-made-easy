package history

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"ytdlp-easy/internal/util"
)

type Entry struct {
	ID          string    `json:"id"`
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Status      string    `json:"status"`
	FilePath    string    `json:"file_path"`
	FileSize    int64     `json:"file_size"`
	Error       string    `json:"error,omitempty"`
	Date        time.Time `json:"date"`
	IsAudioOnly bool      `json:"is_audio_only"`
	Quality     string    `json:"quality"`
	Format      string    `json:"format"`
	HideInQueue bool      `json:"hide_in_queue"`
}

// History persists download records to a JSON Lines file for crash-safe appends.
type History struct {
	entries []Entry
	mu      sync.RWMutex
}

func NewHistory() (*History, error) {
	h := &History{
		entries: make([]Entry, 0),
	}

	if err := h.load(); err != nil {
		// File doesn't exist yet is okay
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return h, nil
}

func (h *History) load() error {
	file, err := os.Open(util.HistoryFile)
	if err != nil {
		return err
	}
	defer file.Close()

	h.mu.Lock()
	defer h.mu.Unlock()

	h.entries = make([]Entry, 0)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip malformed lines gracefully
		}
		h.entries = append(h.entries, entry)
	}

	return scanner.Err()
}

// Add appends atomically - even if the app crashes mid-write, previous entries survive.
func (h *History) Add(entry Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if entry.Date.IsZero() {
		entry.Date = time.Now()
	}

	h.entries = append(h.entries, entry)

	file, err := os.OpenFile(util.HistoryFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	_, err = file.WriteString(string(data) + "\n")
	return err
}

func (h *History) GetRecent(limit int) []Entry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if limit < 0 {
		limit = 0
	}
	if limit > len(h.entries) {
		limit = len(h.entries)
	}

	results := make([]Entry, 0, limit)
	for i := len(h.entries) - 1; i >= 0 && len(results) < limit; i-- {
		if !h.entries[i].HideInQueue {
			results = append(results, h.entries[i])
		}
	}
	return results
}

func (h *History) GetRecentCompleted(limit int) []Entry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if limit < 0 {
		limit = 0
	}
	if limit > len(h.entries) {
		limit = len(h.entries)
	}

	results := make([]Entry, 0, limit)
	for i := len(h.entries) - 1; i >= 0 && len(results) < limit; i-- {
		e := h.entries[i]
		if e.Status == "completed" && e.FilePath != "" {
			results = append(results, e)
		}
	}
	return results
}

func (h *History) GetAll() []Entry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.entries) == 0 {
		return make([]Entry, 0)
	}
	result := make([]Entry, len(h.entries))
	for i, entry := range h.entries {
		result[len(h.entries)-1-i] = entry // Newest first
	}
	return result
}

func (h *History) Search(query, status string) []Entry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	queryLower := strings.ToLower(query)
	results := make([]Entry, 0)

	for i := len(h.entries) - 1; i >= 0; i-- {
		entry := h.entries[i]

		if status != "" && status != "all" && entry.Status != status {
			continue
		}

		if queryLower != "" {
			titleLower := strings.ToLower(entry.Title)
			urlLower := strings.ToLower(entry.URL)
			if !strings.Contains(titleLower, queryLower) && !strings.Contains(urlLower, queryLower) {
				continue
			}
		}

		results = append(results, entry)
	}

	return results
}

func (h *History) GetByID(id string) *Entry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, entry := range h.entries {
		if entry.ID == id {
			return &entry
		}
	}
	return nil
}

func (h *History) Clear() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.entries = make([]Entry, 0)
	return os.Truncate(util.HistoryFile, 0)
}

func (h *History) ClearOld(days int) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -days)
	var newEntries []Entry

	for _, entry := range h.entries {
		if entry.Date.After(cutoff) {
			newEntries = append(newEntries, entry)
		}
	}

	removed := len(h.entries) - len(newEntries)
	if newEntries == nil {
		newEntries = make([]Entry, 0)
	}
	h.entries = newEntries

	if err := h.rewriteFile(); err != nil {
		return 0, err
	}

	return removed, nil
}

func (h *History) HideFromQueue(id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	for i := range h.entries {
		if h.entries[i].ID == id {
			h.entries[i].HideInQueue = true
			if err := h.rewriteFile(); err != nil {
				h.entries[i].HideInQueue = false
				return err
			}
			return nil
		}
	}
	return nil
}

func (h *History) Remove(id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	newEntries := make([]Entry, 0, len(h.entries))
	for _, entry := range h.entries {
		if entry.ID != id {
			newEntries = append(newEntries, entry)
		}
	}

	h.entries = newEntries
	return h.rewriteFile()
}

// rewriteFile rebuilds the file after deletions. Unlike Add(), this is not atomic.
func (h *History) rewriteFile() error {
	file, err := os.Create(util.HistoryFile)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, entry := range h.entries {
		data, err := json.Marshal(entry)
		if err != nil {
			continue
		}
		if _, err := file.WriteString(string(data) + "\n"); err != nil {
			return err
		}
	}

	return nil
}

func (h *History) ExportCSV(filepath string) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"Title", "URL", "Status", "Date", "File Path", "File Size", "Format"})

	for i := len(h.entries) - 1; i >= 0; i-- {
		entry := h.entries[i]
		writer.Write([]string{
			entry.Title,
			entry.URL,
			entry.Status,
			entry.Date.Format("2006-01-02 15:04:05"),
			entry.FilePath,
			formatBytes(entry.FileSize),
			entry.Format,
		})
	}

	return nil
}

func (h *History) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.entries)
}

func (h *History) Stats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	completed := 0
	failed := 0
	var totalSize int64

	for _, entry := range h.entries {
		switch entry.Status {
		case "completed":
			completed++
			totalSize += entry.FileSize
		case "error":
			failed++
		}
	}

	return map[string]interface{}{
		"total":      len(h.entries),
		"completed":  completed,
		"failed":     failed,
		"total_size": formatBytes(totalSize),
	}
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %siB", float64(b)/float64(div), string("KMGTPE"[exp]))
}
