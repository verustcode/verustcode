// Package engine provides the core review engine for VerustCode.
// This file implements the Dispatcher for event-driven task scheduling.
package engine

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/pkg/logger"
)

// Dispatcher handles event-driven task dispatching from the queue to workers.
// It listens for task ready signals and dispatches tasks to available workers.
type Dispatcher struct {
	queue      *RepoTaskQueue
	taskQueue  chan *Task // output channel to workers
	maxWorkers int
	workerWg   sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc

	// processFunc is the function called to process each task
	processFunc func(*Task)

	// running indicates if the dispatcher is running
	running bool
	mu      sync.Mutex
}

// DispatcherConfig holds configuration for the Dispatcher
type DispatcherConfig struct {
	MaxWorkers  int           // Maximum number of concurrent workers
	QueueSize   int           // Size of the task channel buffer
	IdleTimeout time.Duration // Timeout for idle workers (not used currently)
}

// DefaultDispatcherConfig returns default dispatcher configuration
func DefaultDispatcherConfig() *DispatcherConfig {
	return &DispatcherConfig{
		MaxWorkers:  4,
		QueueSize:   100,
		IdleTimeout: 30 * time.Second,
	}
}

// NewDispatcher creates a new Dispatcher
func NewDispatcher(ctx context.Context, queue *RepoTaskQueue, config *DispatcherConfig, processFunc func(*Task)) *Dispatcher {
	if config == nil {
		config = DefaultDispatcherConfig()
	}

	dispatcherCtx, cancel := context.WithCancel(ctx)

	d := &Dispatcher{
		queue:       queue,
		taskQueue:   make(chan *Task, config.QueueSize),
		maxWorkers:  config.MaxWorkers,
		ctx:         dispatcherCtx,
		cancel:      cancel,
		processFunc: processFunc,
	}

	logger.Info("Dispatcher created",
		zap.Int("max_workers", config.MaxWorkers),
		zap.Int("queue_size", config.QueueSize),
	)

	return d
}

// Start starts the dispatcher and workers
func (d *Dispatcher) Start() {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return
	}
	d.running = true
	d.mu.Unlock()

	logger.Info("Starting Dispatcher", zap.Int("workers", d.maxWorkers))

	// Start workers
	for i := 0; i < d.maxWorkers; i++ {
		d.workerWg.Add(1)
		go d.worker(i)
	}

	// Start the main dispatch loop
	go d.dispatchLoop()
}

// dispatchLoop is the main loop that listens for task ready signals
// and dispatches tasks to workers
func (d *Dispatcher) dispatchLoop() {
	logger.Info("Dispatch loop started")

	for {
		select {
		case <-d.ctx.Done():
			logger.Info("Dispatch loop stopping")
			return

		case <-d.queue.TaskReady():
			// Task ready signal received, try to dispatch
			d.tryDispatch()
		}
	}
}

// tryDispatch attempts to dispatch available tasks from the queue
func (d *Dispatcher) tryDispatch() {
	// Keep dispatching until no more tasks can be dispatched
	for {
		task := d.queue.Dequeue()
		if task == nil {
			// No more tasks available
			return
		}

		// Send task to worker channel
		select {
		case d.taskQueue <- task:
			logger.Debug("Task dispatched to worker",
				zap.String("review_id", task.Review.ID),
				zap.String("repo_url", task.Review.RepoURL),
			)
		case <-d.ctx.Done():
			// Context cancelled, put task back
			// Note: In a real scenario, we might want to handle this differently
			logger.Warn("Context cancelled while dispatching task",
				zap.String("review_id", task.Review.ID),
			)
			return
		}
	}
}

// worker is a goroutine that processes tasks from the task channel
func (d *Dispatcher) worker(id int) {
	defer d.workerWg.Done()

	logger.Debug("Worker started", zap.Int("worker_id", id))

	for {
		select {
		case <-d.ctx.Done():
			logger.Debug("Worker stopping", zap.Int("worker_id", id))
			return

		case task, ok := <-d.taskQueue:
			if !ok {
				// Channel closed
				logger.Debug("Worker task channel closed", zap.Int("worker_id", id))
				return
			}

			if task == nil {
				continue
			}

			logger.Info("Worker processing task",
				zap.Int("worker_id", id),
				zap.String("review_id", task.Review.ID),
				zap.String("repo_url", task.Review.RepoURL),
			)

			// Process the task
			startTime := time.Now()
			d.processFunc(task)
			duration := time.Since(startTime)

			logger.Info("Worker completed task",
				zap.Int("worker_id", id),
				zap.String("review_id", task.Review.ID),
				zap.Duration("duration", duration),
			)

			// Mark task as complete in the queue
			// This will trigger scheduling of next task for this repo
			d.queue.MarkComplete(task.Review.RepoURL, task.Review.ID)
		}
	}
}

// Stop stops the dispatcher and waits for all workers to finish
func (d *Dispatcher) Stop() {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return
	}
	d.running = false
	d.mu.Unlock()

	logger.Info("Stopping Dispatcher")

	// Cancel context to signal workers to stop
	d.cancel()

	// Close task channel
	close(d.taskQueue)

	// Wait for all workers to finish
	d.workerWg.Wait()

	logger.Info("Dispatcher stopped")
}

// IsRunning returns true if the dispatcher is running
func (d *Dispatcher) IsRunning() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.running
}

// GetWorkerCount returns the number of workers
func (d *Dispatcher) GetWorkerCount() int {
	return d.maxWorkers
}

// TriggerDispatch manually triggers a dispatch attempt
// This can be used when tasks are added externally
func (d *Dispatcher) TriggerDispatch() {
	d.tryDispatch()
}
