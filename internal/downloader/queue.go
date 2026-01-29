package downloader

import (
	"context"
	"sync"

	"ytdlp-easy/internal/config"
)

// Queue coordinates concurrent downloads using a semaphore pattern.
// Downloads exceeding MaxConcurrentDownloads block until a slot opens.
type Queue struct {
	items       map[string]*Item
	downloaders map[string]*Downloader
	settings    *config.Settings
	mu          sync.RWMutex
	sem         chan struct{} // Bounded semaphore for concurrency control
	ctx         context.Context
	cancel      context.CancelFunc

	OnItemUpdate  func(*Item)
	OnQueueUpdate func()
	OnLog         func(itemID, line string)
}

func NewQueue(settings *config.Settings) *Queue {
	ctx, cancel := context.WithCancel(context.Background())

	q := &Queue{
		items:       make(map[string]*Item),
		downloaders: make(map[string]*Downloader),
		settings:    settings,
		sem:         make(chan struct{}, settings.General.MaxConcurrentDownloads),
		ctx:         ctx,
		cancel:      cancel,
	}

	return q
}

func (q *Queue) Add(url string, isAudioOnly bool, quality, format string) *Item {
	item := NewItem(url, isAudioOnly, quality, format)

	q.mu.Lock()
	q.items[item.ID] = item
	q.mu.Unlock()

	go q.processItem(item)
	q.notifyQueueUpdate()
	return item
}

func (q *Queue) processItem(item *Item) {
	// Block until a concurrency slot is available
	select {
	case q.sem <- struct{}{}:
	case <-q.ctx.Done():
		item.SetStatus(StatusStopped)
		q.notifyItemUpdate(item)
		return
	}
	defer func() { <-q.sem }()

	d := NewDownloader(item, q.settings)
	if q.OnLog != nil {
		d.OnLog = func(line string) {
			q.OnLog(item.ID, line)
		}
	}

	q.mu.Lock()
	q.downloaders[item.ID] = d
	q.mu.Unlock()

	if err := d.Start(q.ctx); err != nil {
		item.SetError(err)
		q.notifyItemUpdate(item)
		return
	}

	q.notifyItemUpdate(item)

	if err := d.Wait(); err != nil {
		q.notifyItemUpdate(item) // Error already set by Wait()
	}

	q.mu.Lock()
	delete(q.downloaders, item.ID)
	q.mu.Unlock()

	q.notifyItemUpdate(item)
	q.notifyQueueUpdate()
}

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

func (q *Queue) Resume(id string) error {
	q.mu.RLock()
	d, ok := q.downloaders[id]
	item := q.items[id]
	q.mu.RUnlock()

	if ok && d != nil {
		err := d.Resume()
		if err == nil {
			q.notifyItemUpdate(d.item)
		}
		return err
	}

	// Stopped items need a fresh download attempt
	if item != nil && item.Status == StatusStopped {
		item.Status = StatusPending
		go q.processItem(item)
		return nil
	}

	return nil
}

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

func (q *Queue) Remove(id string) {
	q.mu.Lock()
	if d, ok := q.downloaders[id]; ok {
		d.Stop()
		delete(q.downloaders, id)
	}
	delete(q.items, id)
	q.mu.Unlock() // Release before callback to avoid deadlock

	q.notifyQueueUpdate()
}

func (q *Queue) GetItem(id string) *Item {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.items[id]
}

func (q *Queue) GetAll() []*Item {
	q.mu.RLock()
	defer q.mu.RUnlock()

	items := make([]*Item, 0, len(q.items))
	for _, item := range q.items {
		items = append(items, item)
	}
	return items
}

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

func (q *Queue) ClearCompleted() int {
	q.mu.Lock()

	count := 0
	for id, item := range q.items {
		if item.Status == StatusCompleted || item.Status == StatusError || item.Status == StatusStopped {
			delete(q.items, id)
			count++
		}
	}

	q.mu.Unlock() // Release before callback to avoid deadlock

	if count > 0 {
		q.notifyQueueUpdate()
	}
	return count
}

// UpdateMaxConcurrent replaces the semaphore with a new capacity.
// Active downloads continue; the new limit applies to future downloads.
func (q *Queue) UpdateMaxConcurrent(max int) {
	q.sem = make(chan struct{}, max)
}

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
