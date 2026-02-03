package engine

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/verustcode/verustcode/internal/model"
)

// createTestTask creates a task for testing
func createTestTask(reviewID string, repoURL string) *Task {
	return &Task{
		Review: &model.Review{
			ID:      reviewID,
			RepoURL: repoURL,
		},
	}
}

// TestNewRepoTaskQueue tests creating a new queue
func TestNewRepoTaskQueue(t *testing.T) {
	ctx := context.Background()
	q := NewRepoTaskQueue(ctx)

	if q == nil {
		t.Fatal("NewRepoTaskQueue() returned nil")
	}

	if q.queues == nil {
		t.Error("queues map is nil")
	}

	if q.tasksByID == nil {
		t.Error("tasksByID map is nil")
	}

	if q.taskReady == nil {
		t.Error("taskReady channel is nil")
	}

	if q.ctx == nil {
		t.Error("context is nil")
	}
}

// TestRepoTaskQueue_Enqueue tests enqueueing tasks
func TestRepoTaskQueue_Enqueue(t *testing.T) {
	ctx := context.Background()
	q := NewRepoTaskQueue(ctx)

	t.Run("enqueue single task", func(t *testing.T) {
		task := createTestTask("1", "https://github.com/test/repo1")

		result := q.Enqueue(task)
		if !result {
			t.Error("Enqueue() returned false for new task")
		}

		stats := q.GetStats()
		if stats.TotalPending != 1 {
			t.Errorf("TotalPending = %d, want 1", stats.TotalPending)
		}
	})

	t.Run("enqueue duplicate task", func(t *testing.T) {
		task := createTestTask("1", "https://github.com/test/repo1")

		result := q.Enqueue(task)
		if result {
			t.Error("Enqueue() returned true for duplicate task")
		}
	})

	t.Run("enqueue nil task", func(t *testing.T) {
		result := q.Enqueue(nil)
		if result {
			t.Error("Enqueue() returned true for nil task")
		}
	})

	t.Run("enqueue task with nil review", func(t *testing.T) {
		task := &Task{}
		result := q.Enqueue(task)
		if result {
			t.Error("Enqueue() returned true for task with nil review")
		}
	})
}

// TestRepoTaskQueue_Dequeue tests dequeueing tasks
func TestRepoTaskQueue_Dequeue(t *testing.T) {
	ctx := context.Background()
	q := NewRepoTaskQueue(ctx)

	t.Run("dequeue from empty queue", func(t *testing.T) {
		task := q.Dequeue()
		if task != nil {
			t.Error("Dequeue() should return nil for empty queue")
		}
	})

	t.Run("dequeue single task", func(t *testing.T) {
		task := createTestTask("10", "https://github.com/test/repo")
		q.Enqueue(task)

		dequeued := q.Dequeue()
		if dequeued == nil {
			t.Fatal("Dequeue() returned nil")
		}

		if dequeued.Review.ID != "10" {
			t.Errorf("Dequeued task ID = %s, want 10", dequeued.Review.ID)
		}

		// After dequeue, repo should be running
		stats := q.GetStats()
		if stats.TotalRunning != 1 {
			t.Errorf("TotalRunning = %d, want 1", stats.TotalRunning)
		}
	})

	t.Run("cannot dequeue while repo is running", func(t *testing.T) {
		// Enqueue another task for the same repo
		task2 := createTestTask("11", "https://github.com/test/repo")
		q.Enqueue(task2)

		// Should not dequeue because repo is running
		dequeued := q.Dequeue()
		if dequeued != nil {
			t.Error("Dequeue() should return nil when repo is running")
		}
	})
}

// TestRepoTaskQueue_MarkComplete tests marking tasks as complete
func TestRepoTaskQueue_MarkComplete(t *testing.T) {
	ctx := context.Background()
	q := NewRepoTaskQueue(ctx)

	repoURL := "https://github.com/test/repo"

	// Enqueue and dequeue a task
	task1 := createTestTask("20", repoURL)
	task2 := createTestTask("21", repoURL)
	q.Enqueue(task1)
	q.Enqueue(task2)

	dequeued := q.Dequeue()
	if dequeued == nil {
		t.Fatal("Dequeue() returned nil")
	}

	t.Run("mark complete allows next task", func(t *testing.T) {
		q.MarkComplete(repoURL, dequeued.Review.ID)

		// Now we should be able to dequeue the next task
		next := q.Dequeue()
		if next == nil {
			t.Fatal("Dequeue() should return next task after MarkComplete")
		}

		if next.Review.ID != "21" {
			t.Errorf("Next task ID = %s, want 21", next.Review.ID)
		}
	})

	t.Run("mark complete unknown repo", func(t *testing.T) {
		// Should not panic
		q.MarkComplete("unknown-repo", "999")
	})
}

// TestRepoTaskQueue_EnqueueAsRunning tests enqueueing as running (recovery)
func TestRepoTaskQueue_EnqueueAsRunning(t *testing.T) {
	ctx := context.Background()
	q := NewRepoTaskQueue(ctx)

	t.Run("enqueue as running", func(t *testing.T) {
		task := createTestTask("30", "https://github.com/test/repo")

		result := q.EnqueueAsRunning(task)
		if !result {
			t.Error("EnqueueAsRunning() returned false")
		}

		stats := q.GetStats()
		if stats.TotalRunning != 1 {
			t.Errorf("TotalRunning = %d, want 1", stats.TotalRunning)
		}

		// Pending should be 0 (task is marked as running, not pending)
		if stats.TotalPending != 0 {
			t.Errorf("TotalPending = %d, want 0", stats.TotalPending)
		}
	})

	t.Run("enqueue as running nil task", func(t *testing.T) {
		result := q.EnqueueAsRunning(nil)
		if result {
			t.Error("EnqueueAsRunning() returned true for nil task")
		}
	})

	t.Run("enqueue as running duplicate", func(t *testing.T) {
		task := createTestTask("30", "https://github.com/test/repo")
		result := q.EnqueueAsRunning(task)
		if result {
			t.Error("EnqueueAsRunning() returned true for duplicate task")
		}
	})
}

// TestRepoTaskQueue_GetStats tests getting queue statistics
func TestRepoTaskQueue_GetStats(t *testing.T) {
	ctx := context.Background()
	q := NewRepoTaskQueue(ctx)

	t.Run("empty queue stats", func(t *testing.T) {
		stats := q.GetStats()

		if stats.TotalPending != 0 {
			t.Errorf("TotalPending = %d, want 0", stats.TotalPending)
		}

		if stats.TotalRunning != 0 {
			t.Errorf("TotalRunning = %d, want 0", stats.TotalRunning)
		}

		if stats.RepoCount != 0 {
			t.Errorf("RepoCount = %d, want 0", stats.RepoCount)
		}
	})

	t.Run("stats with tasks", func(t *testing.T) {
		// Add tasks for different repos
		q.Enqueue(createTestTask("40", "repo1"))
		q.Enqueue(createTestTask("41", "repo1"))
		q.Enqueue(createTestTask("42", "repo2"))

		stats := q.GetStats()

		if stats.TotalPending != 3 {
			t.Errorf("TotalPending = %d, want 3", stats.TotalPending)
		}

		if stats.RepoCount != 2 {
			t.Errorf("RepoCount = %d, want 2", stats.RepoCount)
		}
	})
}

// TestRepoTaskQueue_IsEmpty tests the IsEmpty method
func TestRepoTaskQueue_IsEmpty(t *testing.T) {
	ctx := context.Background()
	q := NewRepoTaskQueue(ctx)

	t.Run("empty queue", func(t *testing.T) {
		if !q.IsEmpty() {
			t.Error("IsEmpty() should return true for empty queue")
		}
	})

	t.Run("queue with pending task", func(t *testing.T) {
		q.Enqueue(createTestTask("50", "repo"))

		if q.IsEmpty() {
			t.Error("IsEmpty() should return false when tasks exist")
		}
	})
}

// TestRepoTaskQueue_HasPendingTasks tests the HasPendingTasks method
func TestRepoTaskQueue_HasPendingTasks(t *testing.T) {
	ctx := context.Background()
	q := NewRepoTaskQueue(ctx)

	t.Run("no pending tasks", func(t *testing.T) {
		if q.HasPendingTasks() {
			t.Error("HasPendingTasks() should return false for empty queue")
		}
	})

	t.Run("has pending tasks", func(t *testing.T) {
		q.Enqueue(createTestTask("60", "repo"))

		if !q.HasPendingTasks() {
			t.Error("HasPendingTasks() should return true when tasks exist")
		}
	})

	t.Run("no pending when all repos running", func(t *testing.T) {
		// Dequeue to mark repo as running
		q.Dequeue()

		if q.HasPendingTasks() {
			t.Error("HasPendingTasks() should return false when all repos are running")
		}
	})
}

// TestRepoTaskQueue_GetPendingCount tests the GetPendingCount method
func TestRepoTaskQueue_GetPendingCount(t *testing.T) {
	ctx := context.Background()
	q := NewRepoTaskQueue(ctx)

	if q.GetPendingCount() != 0 {
		t.Error("GetPendingCount() should return 0 for empty queue")
	}

	q.Enqueue(createTestTask("70", "repo1"))
	q.Enqueue(createTestTask("71", "repo2"))

	if q.GetPendingCount() != 2 {
		t.Errorf("GetPendingCount() = %d, want 2", q.GetPendingCount())
	}
}

// TestRepoTaskQueue_GetRunningCount tests the GetRunningCount method
func TestRepoTaskQueue_GetRunningCount(t *testing.T) {
	ctx := context.Background()
	q := NewRepoTaskQueue(ctx)

	if q.GetRunningCount() != 0 {
		t.Error("GetRunningCount() should return 0 for empty queue")
	}

	q.Enqueue(createTestTask("80", "repo1"))
	q.Enqueue(createTestTask("81", "repo2"))

	// Dequeue one task from each repo
	q.Dequeue()
	q.Dequeue()

	if q.GetRunningCount() != 2 {
		t.Errorf("GetRunningCount() = %d, want 2", q.GetRunningCount())
	}
}

// TestRepoTaskQueue_RemoveTask tests removing tasks from the queue
func TestRepoTaskQueue_RemoveTask(t *testing.T) {
	ctx := context.Background()
	q := NewRepoTaskQueue(ctx)

	t.Run("remove pending task", func(t *testing.T) {
		task := createTestTask("90", "repo")
		q.Enqueue(task)

		result := q.RemoveTask("90")
		if !result {
			t.Error("RemoveTask() returned false for existing task")
		}

		if !q.IsEmpty() {
			t.Error("Queue should be empty after removing only task")
		}
	})

	t.Run("remove running task", func(t *testing.T) {
		task := createTestTask("91", "repo")
		q.Enqueue(task)
		q.Dequeue() // Mark as running

		result := q.RemoveTask("91")
		if !result {
			t.Error("RemoveTask() returned false for running task")
		}

		stats := q.GetStats()
		if stats.TotalRunning != 0 {
			t.Error("Running count should be 0 after removing running task")
		}
	})

	t.Run("remove non-existent task", func(t *testing.T) {
		result := q.RemoveTask("999")
		if result {
			t.Error("RemoveTask() returned true for non-existent task")
		}
	})
}

// TestRepoTaskQueue_TaskReady tests the TaskReady channel
func TestRepoTaskQueue_TaskReady(t *testing.T) {
	ctx := context.Background()
	q := NewRepoTaskQueue(ctx)

	ch := q.TaskReady()
	if ch == nil {
		t.Fatal("TaskReady() returned nil channel")
	}

	// Enqueue should signal
	go func() {
		time.Sleep(10 * time.Millisecond)
		q.Enqueue(createTestTask("100", "repo"))
	}()

	select {
	case <-ch:
		// Signal received
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive signal on TaskReady channel")
	}
}

// TestRepoTaskQueue_Context tests the Context method
func TestRepoTaskQueue_Context(t *testing.T) {
	ctx := context.Background()
	q := NewRepoTaskQueue(ctx)

	qCtx := q.Context()
	if qCtx == nil {
		t.Error("Context() returned nil")
	}
}

// TestRepoTaskQueue_FIFOOrder tests that tasks are processed in FIFO order
func TestRepoTaskQueue_FIFOOrder(t *testing.T) {
	ctx := context.Background()
	q := NewRepoTaskQueue(ctx)

	repoURL := "https://github.com/test/repo"

	// Enqueue tasks in order
	q.Enqueue(createTestTask("1", repoURL))
	q.Enqueue(createTestTask("2", repoURL))
	q.Enqueue(createTestTask("3", repoURL))

	// Dequeue and verify order
	expectedOrder := []string{"1", "2", "3"}

	for _, expectedID := range expectedOrder {
		task := q.Dequeue()
		if task == nil {
			t.Fatalf("Dequeue() returned nil, expected task with ID %s", expectedID)
		}

		if task.Review.ID != expectedID {
			t.Errorf("Task ID = %s, want %s (FIFO order violated)", task.Review.ID, expectedID)
		}

		// Mark complete to allow next dequeue
		q.MarkComplete(repoURL, task.Review.ID)
	}
}

// TestRepoTaskQueue_ConcurrentEnqueue tests concurrent enqueue operations
func TestRepoTaskQueue_ConcurrentEnqueue(t *testing.T) {
	ctx := context.Background()
	q := NewRepoTaskQueue(ctx)

	var wg sync.WaitGroup
	numGoroutines := 10
	tasksPerGoroutine := 10

	// Concurrent enqueue
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			for j := 0; j < tasksPerGoroutine; j++ {
				id := base*tasksPerGoroutine + j + 1
				task := createTestTask(fmt.Sprintf("%d", id), "repo")
				q.Enqueue(task)
			}
		}(i)
	}

	wg.Wait()

	// All unique tasks should be enqueued
	expectedCount := numGoroutines * tasksPerGoroutine
	if q.GetPendingCount() != expectedCount {
		t.Errorf("GetPendingCount() = %d, want %d", q.GetPendingCount(), expectedCount)
	}
}

// TestRepoTaskQueue_ConcurrentDequeue tests concurrent dequeue operations
func TestRepoTaskQueue_ConcurrentDequeue(t *testing.T) {
	ctx := context.Background()
	q := NewRepoTaskQueue(ctx)

	// Enqueue tasks for multiple repos
	numRepos := 5
	for i := 0; i < numRepos; i++ {
		repoURL := "repo" + string(rune('A'+i))
		q.Enqueue(createTestTask(fmt.Sprintf("%d", i+1), repoURL))
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	dequeuedTasks := make([]*Task, 0)

	// Concurrent dequeue
	for i := 0; i < numRepos; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			task := q.Dequeue()
			if task != nil {
				mu.Lock()
				dequeuedTasks = append(dequeuedTasks, task)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// All tasks should be dequeued (each repo has one task)
	if len(dequeuedTasks) != numRepos {
		t.Errorf("Dequeued %d tasks, want %d", len(dequeuedTasks), numRepos)
	}
}

// TestRepoTaskQueue_MultipleRepos tests handling multiple repositories
func TestRepoTaskQueue_MultipleRepos(t *testing.T) {
	ctx := context.Background()
	q := NewRepoTaskQueue(ctx)

	// Enqueue tasks for different repos
	q.Enqueue(createTestTask("1", "repo1"))
	q.Enqueue(createTestTask("2", "repo1"))
	q.Enqueue(createTestTask("3", "repo2"))
	q.Enqueue(createTestTask("4", "repo2"))

	stats := q.GetStats()
	if stats.RepoCount != 2 {
		t.Errorf("RepoCount = %d, want 2", stats.RepoCount)
	}

	// Can dequeue from both repos simultaneously
	task1 := q.Dequeue()
	task2 := q.Dequeue()

	if task1 == nil || task2 == nil {
		t.Fatal("Should be able to dequeue from both repos")
	}

	// Tasks should be from different repos
	if task1.Review.RepoURL == task2.Review.RepoURL {
		t.Error("Tasks should be from different repos")
	}
}

// TestRepoTaskQueue_HasTask tests the HasTask method
func TestRepoTaskQueue_HasTask(t *testing.T) {
	ctx := context.Background()
	q := NewRepoTaskQueue(ctx)

	t.Run("non-existent task", func(t *testing.T) {
		if q.HasTask("999") {
			t.Error("HasTask() should return false for non-existent task")
		}
	})

	t.Run("pending task", func(t *testing.T) {
		task := createTestTask("200", "https://github.com/test/repo")
		q.Enqueue(task)

		if !q.HasTask("200") {
			t.Error("HasTask() should return true for pending task")
		}
	})

	t.Run("running task", func(t *testing.T) {
		// Dequeue to mark as running
		dequeued := q.Dequeue()
		if dequeued == nil {
			t.Fatal("Dequeue() returned nil")
		}

		// After dequeue, task is removed from tasksByID but should still be found via currentTaskID
		if !q.HasTask("200") {
			t.Error("HasTask() should return true for running task")
		}
	})

	t.Run("completed task", func(t *testing.T) {
		// Mark complete
		q.MarkComplete("https://github.com/test/repo", "200")

		// Task should no longer exist
		if q.HasTask("200") {
			t.Error("HasTask() should return false for completed task")
		}
	})
}
