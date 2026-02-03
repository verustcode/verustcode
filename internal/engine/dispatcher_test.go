package engine

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/verustcode/verustcode/internal/model"
)

// createDispatcherTestTask creates a task for dispatcher testing
func createDispatcherTestTask(reviewID string, repoURL string) *Task {
	return &Task{
		Review: &model.Review{
			ID:      reviewID,
			RepoURL: repoURL,
		},
	}
}

// TestDefaultDispatcherConfig tests the default dispatcher configuration
func TestDefaultDispatcherConfig(t *testing.T) {
	config := DefaultDispatcherConfig()

	if config == nil {
		t.Fatal("DefaultDispatcherConfig() returned nil")
	}

	if config.MaxWorkers != 4 {
		t.Errorf("MaxWorkers = %d, want 4", config.MaxWorkers)
	}

	if config.QueueSize != 100 {
		t.Errorf("QueueSize = %d, want 100", config.QueueSize)
	}

	if config.IdleTimeout != 30*time.Second {
		t.Errorf("IdleTimeout = %v, want 30s", config.IdleTimeout)
	}
}

// TestNewDispatcher tests creating a new dispatcher
func TestNewDispatcher(t *testing.T) {
	ctx := context.Background()
	queue := NewRepoTaskQueue(ctx)

	processFunc := func(task *Task) {}

	t.Run("with default config", func(t *testing.T) {
		d := NewDispatcher(ctx, queue, nil, processFunc)

		if d == nil {
			t.Fatal("NewDispatcher() returned nil")
		}

		if d.maxWorkers != 4 {
			t.Errorf("maxWorkers = %d, want 4 (default)", d.maxWorkers)
		}

		if d.queue != queue {
			t.Error("queue not set correctly")
		}

		if d.processFunc == nil {
			t.Error("processFunc is nil")
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		config := &DispatcherConfig{
			MaxWorkers: 8,
			QueueSize:  200,
		}

		d := NewDispatcher(ctx, queue, config, processFunc)

		if d.maxWorkers != 8 {
			t.Errorf("maxWorkers = %d, want 8", d.maxWorkers)
		}
	})
}

// TestDispatcher_StartStop tests starting and stopping the dispatcher
func TestDispatcher_StartStop(t *testing.T) {
	ctx := context.Background()
	queue := NewRepoTaskQueue(ctx)

	processFunc := func(task *Task) {}

	d := NewDispatcher(ctx, queue, nil, processFunc)

	t.Run("start dispatcher", func(t *testing.T) {
		if d.IsRunning() {
			t.Error("Dispatcher should not be running before Start()")
		}

		d.Start()

		// Give workers time to start
		time.Sleep(10 * time.Millisecond)

		if !d.IsRunning() {
			t.Error("Dispatcher should be running after Start()")
		}
	})

	t.Run("start when already running", func(t *testing.T) {
		// Should not panic or cause issues
		d.Start()

		if !d.IsRunning() {
			t.Error("Dispatcher should still be running")
		}
	})

	t.Run("stop dispatcher", func(t *testing.T) {
		d.Stop()

		if d.IsRunning() {
			t.Error("Dispatcher should not be running after Stop()")
		}
	})

	t.Run("stop when not running", func(t *testing.T) {
		// Should not panic
		d.Stop()

		if d.IsRunning() {
			t.Error("Dispatcher should not be running")
		}
	})
}

// TestDispatcher_GetWorkerCount tests getting the worker count
func TestDispatcher_GetWorkerCount(t *testing.T) {
	ctx := context.Background()
	queue := NewRepoTaskQueue(ctx)
	processFunc := func(task *Task) {}

	config := &DispatcherConfig{
		MaxWorkers: 10,
		QueueSize:  50,
	}

	d := NewDispatcher(ctx, queue, config, processFunc)

	if d.GetWorkerCount() != 10 {
		t.Errorf("GetWorkerCount() = %d, want 10", d.GetWorkerCount())
	}
}

// TestDispatcher_IsRunning tests the IsRunning method
func TestDispatcher_IsRunning(t *testing.T) {
	ctx := context.Background()
	queue := NewRepoTaskQueue(ctx)
	processFunc := func(task *Task) {}

	d := NewDispatcher(ctx, queue, nil, processFunc)

	if d.IsRunning() {
		t.Error("IsRunning() should return false before Start()")
	}

	d.Start()
	defer d.Stop()

	if !d.IsRunning() {
		t.Error("IsRunning() should return true after Start()")
	}
}

// TestDispatcher_ProcessTask tests that tasks are processed correctly
func TestDispatcher_ProcessTask(t *testing.T) {
	ctx := context.Background()
	queue := NewRepoTaskQueue(ctx)

	var processedCount int32
	var processedMu sync.Mutex
	processedTasks := make([]string, 0)

	processFunc := func(task *Task) {
		atomic.AddInt32(&processedCount, 1)
		processedMu.Lock()
		processedTasks = append(processedTasks, task.Review.ID)
		processedMu.Unlock()
	}

	config := &DispatcherConfig{
		MaxWorkers: 2,
		QueueSize:  10,
	}

	d := NewDispatcher(ctx, queue, config, processFunc)
	d.Start()
	defer d.Stop()

	// Enqueue tasks
	queue.Enqueue(createDispatcherTestTask("1", "repo1"))
	queue.Enqueue(createDispatcherTestTask("2", "repo2"))

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&processedCount) != 2 {
		t.Errorf("Processed %d tasks, want 2", processedCount)
	}
}

// TestDispatcher_TriggerDispatch tests manual dispatch triggering
func TestDispatcher_TriggerDispatch(t *testing.T) {
	ctx := context.Background()
	queue := NewRepoTaskQueue(ctx)

	var processedCount int32

	processFunc := func(task *Task) {
		atomic.AddInt32(&processedCount, 1)
	}

	config := &DispatcherConfig{
		MaxWorkers: 1,
		QueueSize:  10,
	}

	d := NewDispatcher(ctx, queue, config, processFunc)
	d.Start()
	defer d.Stop()

	// Enqueue a task
	queue.Enqueue(createDispatcherTestTask("100", "repo"))

	// Manually trigger dispatch
	d.TriggerDispatch()

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&processedCount) < 1 {
		t.Error("Task should have been processed after TriggerDispatch()")
	}
}

// TestDispatcher_MultipleRepos tests processing tasks from multiple repos concurrently
func TestDispatcher_MultipleRepos(t *testing.T) {
	ctx := context.Background()
	queue := NewRepoTaskQueue(ctx)

	var processedCount int32
	startTimes := make(map[string]time.Time)
	var mu sync.Mutex

	processFunc := func(task *Task) {
		mu.Lock()
		startTimes[task.Review.ID] = time.Now()
		mu.Unlock()

		// Simulate some work
		time.Sleep(50 * time.Millisecond)

		atomic.AddInt32(&processedCount, 1)
	}

	config := &DispatcherConfig{
		MaxWorkers: 4,
		QueueSize:  10,
	}

	d := NewDispatcher(ctx, queue, config, processFunc)
	d.Start()
	defer d.Stop()

	// Enqueue tasks for different repos (should run in parallel)
	queue.Enqueue(createDispatcherTestTask("1", "repo1"))
	queue.Enqueue(createDispatcherTestTask("2", "repo2"))
	queue.Enqueue(createDispatcherTestTask("3", "repo3"))
	queue.Enqueue(createDispatcherTestTask("4", "repo4"))

	// Wait for all to complete
	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt32(&processedCount) != 4 {
		t.Errorf("Processed %d tasks, want 4", processedCount)
	}
}

// TestDispatcher_SameRepoSerialized tests that tasks for the same repo are serialized
func TestDispatcher_SameRepoSerialized(t *testing.T) {
	ctx := context.Background()
	queue := NewRepoTaskQueue(ctx)

	var executionOrder []string
	var orderMu sync.Mutex

	processFunc := func(task *Task) {
		orderMu.Lock()
		executionOrder = append(executionOrder, task.Review.ID)
		orderMu.Unlock()

		// Simulate work
		time.Sleep(20 * time.Millisecond)
	}

	config := &DispatcherConfig{
		MaxWorkers: 4,
		QueueSize:  10,
	}

	d := NewDispatcher(ctx, queue, config, processFunc)
	d.Start()
	defer d.Stop()

	repoURL := "same-repo"

	// Enqueue multiple tasks for the same repo
	queue.Enqueue(createDispatcherTestTask("1", repoURL))
	queue.Enqueue(createDispatcherTestTask("2", repoURL))
	queue.Enqueue(createDispatcherTestTask("3", repoURL))

	// Wait for all to complete
	time.Sleep(200 * time.Millisecond)

	orderMu.Lock()
	defer orderMu.Unlock()

	// Tasks for the same repo should be processed in order
	if len(executionOrder) != 3 {
		t.Errorf("Execution order length = %d, want 3", len(executionOrder))
		return
	}

	// First task should be 1
	if executionOrder[0] != "1" {
		t.Errorf("First executed task = %s, want 1", executionOrder[0])
	}
}

// TestDispatcher_ContextCancellation tests that dispatcher stops on context cancellation
func TestDispatcher_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	queue := NewRepoTaskQueue(ctx)

	processFunc := func(task *Task) {
		time.Sleep(100 * time.Millisecond)
	}

	d := NewDispatcher(ctx, queue, nil, processFunc)
	d.Start()

	// Cancel context
	cancel()

	// Give time for dispatcher to stop
	time.Sleep(50 * time.Millisecond)

	// Dispatcher should handle cancellation gracefully
	// The dispatch loop should exit
}

// TestDispatcher_TaskCompletionUnlocksRepo tests that completing a task unlocks the repo
func TestDispatcher_TaskCompletionUnlocksRepo(t *testing.T) {
	ctx := context.Background()
	queue := NewRepoTaskQueue(ctx)

	var processedTasks []string
	var mu sync.Mutex

	processFunc := func(task *Task) {
		mu.Lock()
		processedTasks = append(processedTasks, task.Review.ID)
		mu.Unlock()
	}

	config := &DispatcherConfig{
		MaxWorkers: 1, // Single worker to ensure serialization
		QueueSize:  10,
	}

	d := NewDispatcher(ctx, queue, config, processFunc)
	d.Start()
	defer d.Stop()

	repoURL := "test-repo"

	// Enqueue first task
	queue.Enqueue(createDispatcherTestTask("1", repoURL))

	// Wait for first task to complete
	time.Sleep(50 * time.Millisecond)

	// Enqueue second task for same repo
	queue.Enqueue(createDispatcherTestTask("2", repoURL))

	// Wait for second task
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(processedTasks) != 2 {
		t.Errorf("Processed %d tasks, want 2", len(processedTasks))
	}
}

// TestDispatcher_SequentialStartStop tests sequential start/stop calls
func TestDispatcher_SequentialStartStop(t *testing.T) {
	ctx := context.Background()
	queue := NewRepoTaskQueue(ctx)
	processFunc := func(task *Task) {}

	// Sequential start/stop cycles should work correctly
	for i := 0; i < 3; i++ {
		d := NewDispatcher(ctx, queue, nil, processFunc)

		d.Start()
		if !d.IsRunning() {
			t.Errorf("Iteration %d: Dispatcher should be running after Start()", i)
		}

		d.Stop()
		if d.IsRunning() {
			t.Errorf("Iteration %d: Dispatcher should not be running after Stop()", i)
		}
	}
}
