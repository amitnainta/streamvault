package scheduler

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// TaskStatus is the last known state of a task.
type TaskStatus string

const (
	StatusIdle    TaskStatus = "idle"
	StatusRunning TaskStatus = "running"
	StatusSuccess TaskStatus = "success"
	StatusError   TaskStatus = "error"
)

// Task is a named, optionally scheduled background job.
type Task struct {
	ID       string
	Name     string
	Schedule string // cron expression, e.g. "0 3 * * *". Empty = manual only.
	Handler  func(ctx context.Context) error
	Timeout  time.Duration

	mu         sync.Mutex
	status     TaskStatus
	lastRun    time.Time
	lastError  string
}

func (t *Task) run(logger *zap.Logger) {
	t.mu.Lock()
	t.status = StatusRunning
	t.lastRun = time.Now()
	t.mu.Unlock()

	timeout := t.Timeout
	if timeout == 0 {
		timeout = time.Hour
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := t.Handler(ctx)

	t.mu.Lock()
	if err != nil {
		t.status = StatusError
		t.lastError = err.Error()
		logger.Error("task failed", zap.String("task", t.ID), zap.Error(err))
	} else {
		t.status = StatusSuccess
		t.lastError = ""
		logger.Info("task complete", zap.String("task", t.ID))
	}
	t.mu.Unlock()
}

// Scheduler runs registered tasks on their cron schedule or on demand.
type Scheduler struct {
	mu     sync.RWMutex
	tasks  map[string]*Task
	logger *zap.Logger
	stop   chan struct{}
}

func New(logger *zap.Logger) *Scheduler {
	return &Scheduler{
		tasks:  make(map[string]*Task),
		logger: logger,
		stop:   make(chan struct{}),
	}
}

// Register adds a task. Must be called before Start().
func (s *Scheduler) Register(t *Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t.status = StatusIdle
	s.tasks[t.ID] = t
}

// Start launches the scheduling loop.
func (s *Scheduler) Start() {
	go s.loop()
}

// Stop shuts down the scheduler gracefully.
func (s *Scheduler) Stop() {
	close(s.stop)
}

// RunNow triggers a task immediately in a goroutine.
func (s *Scheduler) RunNow(taskID string) bool {
	s.mu.RLock()
	t, ok := s.tasks[taskID]
	s.mu.RUnlock()
	if !ok {
		return false
	}
	go t.run(s.logger)
	return true
}

// GetStatus returns the current status of a task.
func (s *Scheduler) GetStatus(taskID string) (TaskStatus, time.Time, string) {
	s.mu.RLock()
	t, ok := s.tasks[taskID]
	s.mu.RUnlock()
	if !ok {
		return "", time.Time{}, ""
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.status, t.lastRun, t.lastError
}

// ListTasks returns a snapshot of all registered tasks.
func (s *Scheduler) ListTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, t)
	}
	return out
}

// loop checks every minute if any task is due.
// A simple cron implementation — parses "M H * * *" patterns.
func (s *Scheduler) loop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case now := <-ticker.C:
			s.mu.RLock()
			for _, t := range s.tasks {
				if t.Schedule != "" && isDue(t.Schedule, now) {
					go t.run(s.logger)
				}
			}
			s.mu.RUnlock()
		}
	}
}

// isDue is a minimal cron matcher for "M H * * *" patterns.
// For production: replace with a proper cron library like robfig/cron.
func isDue(schedule string, t time.Time) bool {
	// Placeholder: always return false (manual triggers only for MVP)
	// TODO: integrate robfig/cron v3 for proper cron parsing
	return false
}
