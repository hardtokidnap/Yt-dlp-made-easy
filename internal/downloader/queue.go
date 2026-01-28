package downloader

import (
	"context"
	"sync"

	"ytdlp-easy/internal/config"
)

// Queue manages concurrent downloads
type Queue struct {
	items       map[string]*Item
	downloaders map[string]*Downloader
	settings    *config.Settings
	mu          sync.RWMutex
	sem         chan struct{}
	ctx         context.Context
	cancel      context.CancelFunc

	// Callbacks for UI updates
	OnItemUpdate func(*Item)
	OnQueueUpdate func()
}

// NewQueue creates a new download queue
func NewQueue(settings *config.Settings) *Queue {
	ctx, cancel := context.WithCancel(context.Background())

	q := &Queue{
		items:       make(map[string]*Item),
		downloaders: make(map[string]*Downloader),
		settings:    settings,
		sem:         make(chan struct{}, settings.General.MaxConcurrent),
		ctx:         ctx,
		cancel:      cancel,
	}

	return q
}

// Add adds a new download to the queue
func (q *Queue) Add(url string, isAudioOnly bool, quality, format string) *Item {
	item := NewItem(url, isAudioOnly, quality, format)

	q.mu.Lock()
	q.items[item.ID] = item
	q.mu.Unlock()

	// Start processing
	go q.processItem(item)

	q.notifyQueueUpdate()
	return item
}

// processItem handles downloading a single item
func (q *Queue) processItem(item *Item) {
	// Acquire semaphore (blocks if at max concurrent)
	select {
	case q.sem <- struct{}{}:
		// Got slot
	case <-q.ctx.Done():
		return
	}
	defer func() { <-q.sem }() // Release slot when done

	// Create downloader
	d := NewDownloader(item, q.settings)

	q.mu.Lock()
	q.downloaders[item.ID] = d
	q.mu.Unlock()

	// Start download
	if err := d.Start(q.ctx); err != nil {
		item.SetError(err)
		q.notifyItemUpdate(item)
		return
	}

	q.notifyItemUpdate(item)

	// Wait for completion
	d.Wait()

	// Cleanup
	q.mu.Lock()
	delete(q.downloaders, item.ID)
	q.mu.Unlock()

	q.notifyItemUpdate(item)
	q.notifyQueueUpdate()
}

// Pause pauses a download
func (q *Queue) Pause(id string) error {
	q.mu.RLock()
	d, ok := q.downloaders[id]
	q.mu.RUnlock()

	if !ok {
		return nil
	}

	err := d.Pause()
	if err == nil {
		q.notifyItemUpdate(d.item)
	}
	return err
}

// Resume resumes a paused download
func (q *Queue) Resume(id string) error {
	q.mu.RLock()
	d, ok := q.downloaders[id]
	item := q.items[id]
	q.mu.RUnlock()

	// If we have an active downloader, resume it
	if ok && d != nil {
		err := d.Resume()
		if err == nil {
			q.notifyItemUpdate(d.item)
		}
		return err
	}

	// If item is stopped, restart the download
	if item != nil && item.Status == StatusStopped {
		item.Status = StatusPending
		go q.processItem(item)
		return nil
	}

	return nil
}

// Stop stops a download
func (q *Queue) Stop(id string) error {
	q.mu.RLock()
	d, ok := q.downloaders[id]
	q.mu.RUnlock()

	if !ok {
		return nil
	}

	err := d.Stop()
	if err == nil {
		q.notifyItemUpdate(d.item)
	}
	return err
}

// Remove removes an item from the queue
func (q *Queue) Remove(id string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Stop if active
	if d, ok := q.downloaders[id]; ok {
		d.Stop()
		delete(q.downloaders, id)
	}

	delete(q.items, id)
	q.notifyQueueUpdate()
}

// GetItem returns an item by ID
func (q *Queue) GetItem(id string) *Item {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.items[id]
}

// GetAll returns all items
func (q *Queue) GetAll() []*Item {
	q.mu.RLock()
	defer q.mu.RUnlock()

	items := make([]*Item, 0, len(q.items))
	for _, item := range q.items {
		items = append(items, item)
	}
	return items
}

// GetActive returns active downloads
func (q *Queue) GetActive() []*Item {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var items []*Item
	for _, item := range q.items {
		if item.IsActive() {
			items = append(items, item)
		}
	}
	return items
}

// PauseAll pauses all active downloads
func (q *Queue) PauseAll() int {
	q.mu.RLock()
	downloaders := make([]*Downloader, 0)
	for _, d := range q.downloaders {
		downloaders = append(downloaders, d)
	}
	q.mu.RUnlock()

	count := 0
	for _, d := range downloaders {
		if d.Pause() == nil {
			count++
		}
	}
	return count
}

// ResumeAll resumes all paused downloads
func (q *Queue) ResumeAll() int {
	q.mu.RLock()
	downloaders := make([]*Downloader, 0)
	for _, d := range q.downloaders {
		downloaders = append(downloaders, d)
	}
	q.mu.RUnlock()

	count := 0
	for _, d := range downloaders {
		if d.Resume() == nil {
			count++
		}
	}
	return count
}

// StopAll stops all downloads
func (q *Queue) StopAll() int {
	q.mu.RLock()
	downloaders := make([]*Downloader, 0)
	for _, d := range q.downloaders {
		downloaders = append(downloaders, d)
	}
	q.mu.RUnlock()

	count := 0
	for _, d := range downloaders {
		if d.Stop() == nil {
			count++
		}
	}
	return count
}

// ClearCompleted removes completed, stopped, and errored items
func (q *Queue) ClearCompleted() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	count := 0
	for id, item := range q.items {
		if item.Status == StatusCompleted || item.Status == StatusError || item.Status == StatusStopped {
			delete(q.items, id)
			count++
		}
	}

	if count > 0 {
		q.notifyQueueUpdate()
	}
	return count
}

// UpdateMaxConcurrent changes the concurrency limit
func (q *Queue) UpdateMaxConcurrent(max int) {
	// Create new semaphore with new size
	q.sem = make(chan struct{}, max)
}

// Shutdown stops all downloads and cleans up
func (q *Queue) Shutdown() {
	q.cancel()
	q.StopAll()
}

func (q *Queue) notifyItemUpdate(item *Item) {
	if q.OnItemUpdate != nil {
		q.OnItemUpdate(item)
	}
}

func (q *Queue) notifyQueueUpdate() {
	if q.OnQueueUpdate != nil {
		q.OnQueueUpdate()
	}
}
