package converter

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// BatchJob tracks one file in a batch conversion.
type BatchJob struct {
	ID         string    `json:"id"`
	InputFile  string    `json:"input_file"`
	OutputFile string    `json:"output_file"`
	Status     JobStatus `json:"status"`
	Progress   float64   `json:"progress"`
	Error      string    `json:"error"`
}

// BatchQueue manages sequential conversion of multiple files.
type BatchQueue struct {
	Jobs       []*BatchJob `json:"jobs"`
	Status     JobStatus   `json:"status"`
	CurrentIdx int         `json:"current_idx"`
	TotalFiles int         `json:"total_files"`
	Completed  int         `json:"completed"`
	Failed     int         `json:"failed"`
	Progress   float64     `json:"progress"`

	mu         sync.Mutex
	ctx        context.Context
	cancel     context.CancelFunc
	converter  *Converter
	OnProgress func(snap *BatchSnapshot)
	OnLog      func(line string)
}

// NewBatchQueue creates a batch queue for the given input files.
func NewBatchQueue(files []string, opts ConversionOptions) *BatchQueue {
	ext := opts.OutputFormat
	if ext == "" {
		ext = "mp4"
	}

	jobs := make([]*BatchJob, len(files))
	for i, f := range files {
		base := strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))
		output := filepath.Join(filepath.Dir(f), base+"_converted."+ext)

		jobs[i] = &BatchJob{
			ID:         fmt.Sprintf("batch_%d_%d", time.Now().UnixMilli(), i),
			InputFile:  f,
			OutputFile: output,
			Status:     StatusPending,
		}
	}

	return &BatchQueue{
		Jobs:       jobs,
		Status:     StatusPending,
		TotalFiles: len(jobs),
	}
}

// Start processes all jobs sequentially. Blocks until all are done or cancelled.
func (bq *BatchQueue) Start(ctx context.Context, opts ConversionOptions) {
	ctx, cancel := context.WithCancel(ctx)
	bq.mu.Lock()
	bq.ctx = ctx
	bq.cancel = cancel
	bq.Status = StatusRunning
	bq.mu.Unlock()

	defer cancel()

	for i, bj := range bq.Jobs {
		bq.mu.Lock()
		if bq.ctx.Err() != nil {
			// Mark remaining jobs as cancelled
			for j := i; j < len(bq.Jobs); j++ {
				bq.Jobs[j].Status = StatusCancelled
			}
			bq.Status = StatusCancelled
			bq.mu.Unlock()
			bq.emitProgress()
			return
		}

		bq.CurrentIdx = i
		bj.Status = StatusRunning
		bq.mu.Unlock()
		bq.emitProgress()

		fileOpts := opts
		fileOpts.InputFile = bj.InputFile
		fileOpts.OutputFile = bj.OutputFile

		job := &ConversionJob{
			ID:        bj.ID,
			InputFile: bj.InputFile,
			Status:    StatusRunning,
			StartedAt: time.Now(),
		}

		c := NewConverter(job)
		c.OnProgress = func(j *ConversionJob) {
			bq.mu.Lock()
			bj.Progress = j.Progress
			bq.updateOverallProgress()
			bq.mu.Unlock()
			bq.emitProgress()
		}
		c.OnLog = func(line string) {
			if bq.OnLog != nil {
				bq.OnLog(line)
			}
		}

		bq.mu.Lock()
		bq.converter = c
		bq.mu.Unlock()

		err := c.Start(bq.ctx, fileOpts)

		bq.mu.Lock()
		bq.converter = nil
		if err != nil {
			if bq.ctx.Err() != nil {
				bj.Status = StatusCancelled
			} else {
				bj.Status = StatusFailed
				bj.Error = err.Error()
				bq.Failed++
			}
		} else {
			bj.Status = StatusCompleted
			bj.Progress = 100
			bj.OutputFile = job.OutputFile
			bq.Completed++
		}
		bq.updateOverallProgress()
		bq.mu.Unlock()
		bq.emitProgress()
	}

	bq.mu.Lock()
	if bq.Status == StatusRunning {
		bq.Status = StatusCompleted
	}
	bq.mu.Unlock()
	bq.emitProgress()
}

// Cancel stops the current conversion and marks remaining jobs as cancelled.
func (bq *BatchQueue) Cancel() {
	bq.mu.Lock()
	defer bq.mu.Unlock()
	if bq.cancel != nil {
		bq.cancel()
	}
}

// updateOverallProgress recalculates total progress. Must be called with mu held.
func (bq *BatchQueue) updateOverallProgress() {
	if bq.TotalFiles == 0 {
		return
	}
	total := 0.0
	for _, j := range bq.Jobs {
		total += j.Progress
	}
	bq.Progress = total / float64(bq.TotalFiles)
}

// BatchSnapshot is a safe-to-serialize copy of the batch state.
type BatchSnapshot struct {
	Jobs       []BatchJob `json:"jobs"`
	Status     JobStatus  `json:"status"`
	CurrentIdx int        `json:"current_idx"`
	TotalFiles int        `json:"total_files"`
	Completed  int        `json:"completed"`
	Failed     int        `json:"failed"`
	Progress   float64    `json:"progress"`
}

// snapshot creates a deep copy of the batch state. Must be called with mu held.
func (bq *BatchQueue) snapshot() BatchSnapshot {
	jobs := make([]BatchJob, len(bq.Jobs))
	for i, j := range bq.Jobs {
		jobs[i] = *j
	}
	return BatchSnapshot{
		Jobs:       jobs,
		Status:     bq.Status,
		CurrentIdx: bq.CurrentIdx,
		TotalFiles: bq.TotalFiles,
		Completed:  bq.Completed,
		Failed:     bq.Failed,
		Progress:   bq.Progress,
	}
}

func (bq *BatchQueue) emitProgress() {
	if bq.OnProgress == nil {
		return
	}
	bq.mu.Lock()
	snap := bq.snapshot()
	bq.mu.Unlock()
	bq.OnProgress(&snap)
}
