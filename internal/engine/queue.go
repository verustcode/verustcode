// Package engine provides the core review engine for VerustCode.
// This file implements the RepoTaskQueue for memory-based task queue management.
package engine

import (
	"container/list"
	"context"
	"sync"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/pkg/logger"
)

// RepoTaskQueue manages task queues per repository.
// Each repository has its own FIFO queue, and only one task per repo can run at a time.
// This ensures repository-level serialization while allowing concurrent processing across repos.
type RepoTaskQueue struct {
	mu sync.RWMutex

	// queues holds per-repo task queues, key is repo_url
	queues map[string]*repoQueue

	// tasksByID allows quick lookup by review ID (UUID) to prevent duplicates
	tasksByID map[string]*Task

	// taskReady signals that there are tasks ready to be processed
	taskReady chan struct{}

	// ctx and cancel for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// repoQueue represents a single repository's task queue
type repoQueue struct {
	// tasks is a FIFO list of pending tasks
	tasks *list.List

	// running indicates if a task is currently running for this repo
	running bool

	// currentTaskID is the review ID (UUID) of the currently running task ("" if none)
	currentTaskID string

	// dequeued indicates if the recovered running task has already been dequeued.
	// This prevents the same recovered task from being returned multiple times by Dequeue().
	dequeued bool
}

// NewRepoTaskQueue creates a new RepoTaskQueue instance
func NewRepoTaskQueue(ctx context.Context) *RepoTaskQueue {
	queueCtx, cancel := context.WithCancel(ctx)

	q := &RepoTaskQueue{
		queues:    make(map[string]*repoQueue),
		tasksByID: make(map[string]*Task),
		taskReady: make(chan struct{}, 100), // buffered to avoid blocking
		ctx:       queueCtx,
		cancel:    cancel,
	}

	logger.Info("RepoTaskQueue initialized")
	return q
}

// Enqueue adds a task to the queue for its repository.
// Returns true if the task was added, false if it already exists.
func (q *RepoTaskQueue) Enqueue(task *Task) bool {
	if task == nil || task.Review == nil {
		logger.Warn("Attempted to enqueue nil task or task with nil review")
		return false
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	reviewID := task.Review.ID
	repoURL := task.Review.RepoURL

	// Check if task already exists (prevent duplicates)
	if _, exists := q.tasksByID[reviewID]; exists {
		logger.Debug("Task already in queue, skipping",
			zap.String("review_id", reviewID),
			zap.String("repo_url", repoURL),
		)
		return false
	}

	// Get or create repo queue
	rq, ok := q.queues[repoURL]
	if !ok {
		rq = &repoQueue{
			tasks:   list.New(),
			running: false,
		}
		q.queues[repoURL] = rq
	}

	// Add task to the end of the queue (FIFO)
	rq.tasks.PushBack(task)
	q.tasksByID[reviewID] = task

	// Signal that a task is ready (non-blocking)
	q.signalTaskReady()

	return true
}

// EnqueueAsRunning adds a task to the queue and marks its repo as running.
// This is used during recovery for tasks that were in "running" state.
// Returns true if the task was added, false if it already exists.
func (q *RepoTaskQueue) EnqueueAsRunning(task *Task) bool {
	if task == nil || task.Review == nil {
		logger.Warn("Attempted to enqueue nil task or task with nil review")
		return false
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	reviewID := task.Review.ID
	repoURL := task.Review.RepoURL

	// Check if task already exists
	if _, exists := q.tasksByID[reviewID]; exists {
		logger.Debug("Task already in queue, skipping",
			zap.String("review_id", reviewID),
			zap.String("repo_url", repoURL),
		)
		return false
	}

	// Get or create repo queue
	rq, ok := q.queues[repoURL]
	if !ok {
		rq = &repoQueue{
			tasks:   list.New(),
			running: false,
		}
		q.queues[repoURL] = rq
	}

	// Mark repo as running and set current task
	rq.running = true
	rq.currentTaskID = reviewID
	rq.dequeued = false // Reset dequeued flag for new recovered task

	// Add to tasksByID for tracking
	q.tasksByID[reviewID] = task

	logger.Info("Task enqueued as running (recovery)",
		zap.String("review_id", reviewID),
		zap.String("repo_url", repoURL),
		zap.Int("pending_count", rq.tasks.Len()),
	)

	// Signal that a task is ready (the running task needs to be re-processed)
	q.signalTaskReady()

	return true
}

// Dequeue returns the next task that can be processed.
// Returns nil if no tasks are available or all repos have running tasks.
func (q *RepoTaskQueue) Dequeue() *Task {
	q.mu.Lock()
	defer q.mu.Unlock()

	// First, check if there are running tasks that need to be re-processed (recovery case)
	// When a server restarts, EnqueueAsRunning marks tasks as running but doesn't put them
	// in rq.tasks. We need to return these tasks for re-processing.
	for repoURL, rq := range q.queues {
		// Skip if already dequeued - this prevents the same recovered task from being
		// returned multiple times when tryDispatch() calls Dequeue() in a loop
		if rq.running && rq.currentTaskID != "" && !rq.dequeued {
			// Check if the running task exists in tasksByID (recovery case)
			// Normal running tasks are removed from tasksByID when dequeued,
			// but recovered running tasks remain in tasksByID
			if task, exists := q.tasksByID[rq.currentTaskID]; exists {
				// This is a recovered running task, return it for re-processing
				logger.Info("Dequeuing recovered running task",
					zap.String("review_id", rq.currentTaskID),
					zap.String("repo_url", repoURL),
				)
				// Mark as dequeued to prevent returning it again in this dispatch cycle
				rq.dequeued = true
				// Remove from tasksByID so we don't dequeue it again
				delete(q.tasksByID, rq.currentTaskID)
				return task
			}
		}
	}

	// Find a repo with pending tasks that doesn't have a running task
	for _, rq := range q.queues {
		if rq.running {
			// This repo already has a task running, skip it
			continue
		}

		if rq.tasks.Len() == 0 {
			// No pending tasks for this repo
			continue
		}

		// Get the first task (FIFO)
		elem := rq.tasks.Front()
		if elem == nil {
			continue
		}

		task := elem.Value.(*Task)
		rq.tasks.Remove(elem)

		// Mark repo as running
		rq.running = true
		rq.currentTaskID = task.Review.ID

		// Remove from tasksByID - this distinguishes normal dequeue from recovery case
		// Recovery tasks remain in tasksByID until they are dequeued via the recovery path
		delete(q.tasksByID, task.Review.ID)

		return task
	}

	return nil
}

// MarkComplete marks a task as complete and triggers scheduling of next task.
// This should be called when a task finishes (success or failure).
func (q *RepoTaskQueue) MarkComplete(repoURL string, reviewID string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Remove from tasksByID
	delete(q.tasksByID, reviewID)

	rq, ok := q.queues[repoURL]
	if !ok {
		logger.Warn("MarkComplete called for unknown repo",
			zap.String("repo_url", repoURL),
			zap.String("review_id", reviewID),
		)
		return
	}

	// Clear running state
	rq.running = false
	rq.currentTaskID = ""
	rq.dequeued = false // Reset dequeued flag for next task

	logger.Info("Task marked complete",
		zap.String("review_id", reviewID),
		zap.String("repo_url", repoURL),
		zap.Int("pending_count", rq.tasks.Len()),
	)

	// Clean up empty repo queue
	if rq.tasks.Len() == 0 {
		delete(q.queues, repoURL)
		logger.Debug("Removed empty repo queue",
			zap.String("repo_url", repoURL),
		)
	}

	// Signal that next task can be processed
	q.signalTaskReady()
}

// TaskReady returns the channel that signals when tasks are ready
func (q *RepoTaskQueue) TaskReady() <-chan struct{} {
	return q.taskReady
}

// signalTaskReady sends a non-blocking signal to the taskReady channel
func (q *RepoTaskQueue) signalTaskReady() {
	select {
	case q.taskReady <- struct{}{}:
	default:
		// Channel is full, which means there's already a pending signal
	}
}

// GetStats returns queue statistics
func (q *RepoTaskQueue) GetStats() QueueStats {
	q.mu.RLock()
	defer q.mu.RUnlock()

	stats := QueueStats{
		TotalPending: 0,
		TotalRunning: 0,
		RepoCount:    len(q.queues),
		RepoStats:    make(map[string]RepoQueueStats),
	}

	for repoURL, rq := range q.queues {
		pendingCount := rq.tasks.Len()
		stats.TotalPending += pendingCount

		if rq.running {
			stats.TotalRunning++
		}

		stats.RepoStats[repoURL] = RepoQueueStats{
			PendingCount: pendingCount,
			Running:      rq.running,
			CurrentTask:  rq.currentTaskID,
		}
	}

	return stats
}

// QueueStats holds queue statistics
type QueueStats struct {
	TotalPending int                       // Total pending tasks across all repos
	TotalRunning int                       // Number of repos with running tasks
	RepoCount    int                       // Number of repos with queued tasks
	RepoStats    map[string]RepoQueueStats // Per-repo statistics
}

// RepoQueueStats holds per-repo queue statistics
type RepoQueueStats struct {
	PendingCount int    // Number of pending tasks
	Running      bool   // Whether a task is running
	CurrentTask  string // Review ID (UUID) of running task ("" if none)
}

// IsEmpty returns true if there are no tasks in any queue
func (q *RepoTaskQueue) IsEmpty() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, rq := range q.queues {
		if rq.tasks.Len() > 0 || rq.running {
			return false
		}
	}
	return true
}

// HasPendingTasks returns true if there are pending tasks that can be processed
func (q *RepoTaskQueue) HasPendingTasks() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, rq := range q.queues {
		if !rq.running && rq.tasks.Len() > 0 {
			return true
		}
	}
	return false
}

// GetPendingCount returns the total number of pending tasks
func (q *RepoTaskQueue) GetPendingCount() int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	count := 0
	for _, rq := range q.queues {
		count += rq.tasks.Len()
	}
	return count
}

// GetRunningCount returns the number of repos with running tasks
func (q *RepoTaskQueue) GetRunningCount() int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	count := 0
	for _, rq := range q.queues {
		if rq.running {
			count++
		}
	}
	return count
}

// Stop stops the queue and cancels the context
func (q *RepoTaskQueue) Stop() {

	q.cancel()

	close(q.taskReady)

	q.mu.Lock()
	defer q.mu.Unlock()

	logger.Info("RepoTaskQueue stopped")

}

// Context returns the queue's context
func (q *RepoTaskQueue) Context() context.Context {
	return q.ctx
}

// HasTask checks if a task with the given review ID exists in the queue (pending or running)
func (q *RepoTaskQueue) HasTask(reviewID string) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()

	// Check if in tasksByID (pending tasks and recovered running tasks)
	if _, exists := q.tasksByID[reviewID]; exists {
		return true
	}

	// Check if it's a currently running task (dequeued tasks are removed from tasksByID)
	for _, rq := range q.queues {
		if rq.currentTaskID == reviewID {
			return true
		}
	}

	return false
}

// RemoveTask removes a task from the queue (e.g., when cancelled)
func (q *RepoTaskQueue) RemoveTask(reviewID string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	// First, check if this is a running task (dequeued tasks are removed from tasksByID)
	for repoURL, rq := range q.queues {
		if rq.currentTaskID == reviewID {
			// This is a running task, mark as not running
			rq.running = false
			rq.currentTaskID = ""
			rq.dequeued = false           // Reset dequeued flag
			delete(q.tasksByID, reviewID) // Delete if exists (recovery case)
			logger.Info("Running task removed",
				zap.String("review_id", reviewID),
				zap.String("repo_url", repoURL),
			)
			q.signalTaskReady() // Signal to process next task
			return true
		}
	}

	// Check if this is a pending task in tasksByID
	task, exists := q.tasksByID[reviewID]
	if !exists {
		return false
	}

	repoURL := task.Review.RepoURL
	rq, ok := q.queues[repoURL]
	if !ok {
		delete(q.tasksByID, reviewID)
		return true
	}

	// Find and remove from pending list
	for elem := rq.tasks.Front(); elem != nil; elem = elem.Next() {
		t := elem.Value.(*Task)
		if t.Review.ID == reviewID {
			rq.tasks.Remove(elem)
			delete(q.tasksByID, reviewID)
			logger.Info("Pending task removed from queue",
				zap.String("review_id", reviewID),
				zap.String("repo_url", repoURL),
			)
			return true
		}
	}

	delete(q.tasksByID, reviewID)
	return true
}
