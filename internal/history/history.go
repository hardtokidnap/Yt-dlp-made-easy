package history

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"

	"ytdlp-easy/internal/util"
)

// Entry represents a single history entry
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
}

// History manages download history with JSON Lines storage
type History struct {
	entries []Entry
	mu      sync.RWMutex
}

// NewHistory creates a new history manager and loads existing entries
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

// load reads all entries from the JSONL file
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
			continue // Skip malformed lines
		}
		h.entries = append(h.entries, entry)
	}

	return scanner.Err()
}

// Add appends a new entry to history (crash-safe append)
func (h *History) Add(entry Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Set date if not set
	if entry.Date.IsZero() {
		entry.Date = time.Now()
	}

	// Append to in-memory list
	h.entries = append(h.entries, entry)

	// Append to file (crash-safe)
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

// GetAll returns all history entries (newest first)
func (h *History) GetAll() []Entry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Return in reverse order (newest first)
	result := make([]Entry, len(h.entries))
	for i, entry := range h.entries {
		result[len(h.entries)-1-i] = entry
	}
	return result
}

// Search filters history by query and status
func (h *History) Search(query, status string) []Entry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	queryLower := strings.ToLower(query)
	var results []Entry

	// Iterate in reverse for newest first
	for i := len(h.entries) - 1; i >= 0; i-- {
		entry := h.entries[i]

		// Filter by status
		if status != "" && status != "all" && entry.Status != status {
			continue
		}

		// Filter by query
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

// GetByID returns a specific entry
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

// Clear removes all history
func (h *History) Clear() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.entries = make([]Entry, 0)

	// Truncate the file
	return os.Truncate(util.HistoryFile, 0)
}

// ClearOld removes entries older than days
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
	h.entries = newEntries

	// Rewrite file
	if err := h.rewriteFile(); err != nil {
		return 0, err
	}

	return removed, nil
}

// Remove removes a specific entry
func (h *History) Remove(id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	var newEntries []Entry
	for _, entry := range h.entries {
		if entry.ID != id {
			newEntries = append(newEntries, entry)
		}
	}

	h.entries = newEntries
	return h.rewriteFile()
}

// rewriteFile rewrites the entire history file (called after deletions)
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
		file.WriteString(string(data) + "\n")
	}

	return nil
}

// ExportCSV exports history to a CSV file
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

	// Header
	writer.Write([]string{"Title", "URL", "Status", "Date", "File Path", "File Size", "Format"})

	// Data
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

// Count returns the number of entries
func (h *History) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.entries)
}

// Stats returns history statistics
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
		return string(rune(b)) + " B"
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return string(rune(b/div)) + " " + "KMGTPE"[exp:exp+1] + "iB"
}
