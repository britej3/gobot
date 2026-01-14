package scheduler

import (
	"context"
	"sync"
	"time"
)

type Config struct {
	Workers   int
	QueueSize int
}

type Scheduler struct {
	cfg            Config
	mu             sync.RWMutex
	running        bool
	queue          chan Task
	scheduledTasks []Task
	stopCh         chan struct{}
	wg             sync.WaitGroup
}

type Task struct {
	ID         string
	Type       string
	Payload    interface{}
	Execute    func(ctx context.Context, payload interface{}) error
	Scheduled  time.Time
	RunAt      time.Time
	Retries    int
	MaxRetries int
}

type TaskHandler interface {
	HandleTask(ctx context.Context, task Task) error
}

func New(cfg Config) *Scheduler {
	if cfg.Workers <= 0 {
		cfg.Workers = 4
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 100
	}

	return &Scheduler{
		cfg:    cfg,
		queue:  make(chan Task, cfg.QueueSize),
		stopCh: make(chan struct{}),
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.mu.Unlock()

	for i := 0; i < s.cfg.Workers; i++ {
		s.wg.Add(1)
		go s.worker(ctx, i)
	}

	go s.processScheduledTasks(ctx)

	return nil
}

func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false
	close(s.stopCh)
	s.wg.Wait()

	return nil
}

func (s *Scheduler) Schedule(task Task) error {
	s.mu.RLock()
	if !s.running {
		s.mu.RUnlock()
		return nil
	}
	s.mu.RUnlock()

	select {
	case s.queue <- task:
		return nil
	default:
		return ErrQueueFull
	}
}

func (s *Scheduler) ScheduleAt(task Task, runAt time.Time) error {
	task.RunAt = runAt

	s.mu.RLock()
	if !s.running {
		s.mu.RUnlock()
		return nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.scheduledTasks = append(s.scheduledTasks, task)
	return nil
}

func (s *Scheduler) ScheduleInterval(task Task, interval time.Duration) error {
	task.RunAt = time.Now().Add(interval)

	s.mu.Lock()
	defer s.mu.Unlock()

	task.Retries = 0
	s.scheduledTasks = append(s.scheduledTasks, task)
	return nil
}

func (s *Scheduler) worker(ctx context.Context, id int) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case task, ok := <-s.queue:
			if !ok {
				return
			}
			s.executeTask(ctx, task)
		}
	}
}

func (s *Scheduler) executeTask(ctx context.Context, task Task) {
	err := task.Execute(ctx, task.Payload)
	if err != nil {
		task.Retries++
		if task.Retries < task.MaxRetries {
			task.RunAt = time.Now().Add(time.Duration(task.Retries) * time.Second)
			s.mu.Lock()
			s.scheduledTasks = append(s.scheduledTasks, task)
			s.mu.Unlock()
		}
	}
}

func (s *Scheduler) processScheduledTasks(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.processDueTasks(ctx)
		}
	}
}

func (s *Scheduler) processDueTasks(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	var remaining []Task

	for _, task := range s.scheduledTasks {
		if task.RunAt.After(now) {
			remaining = append(remaining, task)
			continue
		}

		select {
		case s.queue <- task:
		default:
			remaining = append(remaining, task)
		}
	}

	s.scheduledTasks = remaining
}

func (s *Scheduler) Cancel(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, task := range s.scheduledTasks {
		if task.ID == id {
			s.scheduledTasks = append(s.scheduledTasks[:i], s.scheduledTasks[i+1:]...)
			return true
		}
	}
	return false
}

func (s *Scheduler) Stats() SchedulerStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return SchedulerStats{
		QueueLength:    len(s.queue),
		ScheduledCount: len(s.scheduledTasks),
		Workers:        s.cfg.Workers,
		Running:        s.running,
	}
}

type SchedulerStats struct {
	QueueLength    int
	ScheduledCount int
	Workers        int
	Running        bool
}

var (
	ErrQueueFull = &SchedulerError{Message: "task queue is full"}
)

type SchedulerError struct {
	Message string
}

func (e *SchedulerError) Error() string {
	return e.Message
}
