package converter

import (
	"bufio"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusRunning    JobStatus = "running"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
	StatusCancelled  JobStatus = "cancelled"
)

// ConversionJob tracks a single ffmpeg conversion.
type ConversionJob struct {
	ID         string    `json:"id"`
	InputFile  string    `json:"input_file"`
	OutputFile string    `json:"output_file"`
	Status     JobStatus `json:"status"`
	Progress   float64   `json:"progress"`
	Duration   string    `json:"duration"`
	Speed      string    `json:"speed"`
	Error      string    `json:"error"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
}

// Converter manages a single ffmpeg conversion process.
type Converter struct {
	job        *ConversionJob
	cancel     context.CancelFunc
	totalSecs  float64
	OnProgress func(job *ConversionJob)
	OnLog      func(line string)
}

func NewConverter(job *ConversionJob) *Converter {
	return &Converter{job: job}
}

func (c *Converter) Start(ctx context.Context, opts ConversionOptions) error {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	args := BuildArgs(opts)
	c.job.OutputFile = args[len(args)-1]
	c.job.Status = StatusRunning
	c.job.StartedAt = time.Now()
	c.emitProgress()

	cmd := hiddenCmd(FFmpegPath(), args...)

	// ffmpeg writes progress to stderr
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ffmpeg: %w", err)
	}

	// Parse stderr for duration and progress
	go func() {
		scanner := bufio.NewScanner(stderr)
		scanner.Buffer(make([]byte, 64*1024), 64*1024)
		for scanner.Scan() {
			line := scanner.Text()
			c.emitLog(line)
			c.parseOutput(line)
		}
	}()

	// Wait for completion in a goroutine so we can listen for cancellation
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		c.job.Status = StatusCancelled
		c.job.FinishedAt = time.Now()
		c.emitProgress()
		return ctx.Err()

	case err := <-done:
		c.job.FinishedAt = time.Now()
		if err != nil {
			c.job.Status = StatusFailed
			c.job.Error = err.Error()
		} else {
			c.job.Status = StatusCompleted
			c.job.Progress = 100
		}
		c.emitProgress()
		return err
	}
}

func (c *Converter) Cancel() {
	if c.cancel != nil {
		c.cancel()
	}
}

func (c *Converter) Job() *ConversionJob {
	return c.job
}

var (
	// "  Duration: 00:03:45.12, ..."
	durationRe = regexp.MustCompile(`Duration:\s+(\d{2}):(\d{2}):(\d{2})\.(\d{2})`)
	// "frame=  123 ... time=00:01:23.45 ... speed=2.1x"
	timeRe  = regexp.MustCompile(`time=(\d{2}):(\d{2}):(\d{2})\.(\d{2})`)
	speedRe = regexp.MustCompile(`speed=\s*([\d.]+)x`)
)

func (c *Converter) parseOutput(line string) {
	// Capture total duration
	if c.totalSecs == 0 {
		if m := durationRe.FindStringSubmatch(line); m != nil {
			c.totalSecs = parseDuration(m)
		}
	}

	// Capture current time position for progress
	if m := timeRe.FindStringSubmatch(line); m != nil {
		currentSecs := parseDuration(m)
		if c.totalSecs > 0 {
			c.job.Progress = (currentSecs / c.totalSecs) * 100
			if c.job.Progress > 100 {
				c.job.Progress = 100
			}
		}
		c.job.Duration = formatDuration(currentSecs)
	}

	if m := speedRe.FindStringSubmatch(line); m != nil {
		c.job.Speed = m[1] + "x"
	}

	// Only emit on lines that contain progress info
	if strings.Contains(line, "time=") {
		c.emitProgress()
	}
}

func (c *Converter) emitProgress() {
	if c.OnProgress != nil {
		c.OnProgress(c.job)
	}
}

func (c *Converter) emitLog(line string) {
	if c.OnLog != nil {
		c.OnLog(line)
	}
}

// parseDuration converts regex match groups [full, HH, MM, SS, cc] to seconds.
func parseDuration(m []string) float64 {
	h, _ := strconv.ParseFloat(m[1], 64)
	min, _ := strconv.ParseFloat(m[2], 64)
	s, _ := strconv.ParseFloat(m[3], 64)
	cs, _ := strconv.ParseFloat(m[4], 64)
	return h*3600 + min*60 + s + cs/100
}

func formatDuration(secs float64) string {
	h := int(secs) / 3600
	m := (int(secs) % 3600) / 60
	s := int(secs) % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
