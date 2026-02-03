package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/database"
	"github.com/verustcode/verustcode/internal/engine/retry"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/errors"
	"github.com/verustcode/verustcode/pkg/logger"
)

// setupRetryTestDB initializes a test database with temporary path
func setupRetryTestDB(t *testing.T) (store.Store, func()) {
	t.Helper()

	// Reset database state from any previous tests
	database.ResetForTesting()

	// Initialize logger for testing
	logger.Init(logger.Config{
		Level:  "error",
		Format: "text",
		File:   "",
	})

	// Create temporary database file
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Initialize database with custom path for testing
	err := database.InitWithPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// Create store instance
	s := store.NewStore(database.Get())

	// Return store and cleanup function
	return s, func() {
		database.ResetForTesting()
		os.Remove(dbPath)
		logger.Sync()
	}
}

// createMinimalTestEngine creates an engine for testing with minimal config
func createMinimalTestEngine(t *testing.T, s store.Store) *Engine {
	t.Helper()

	cfg := config.Default()
	cfg.Review.MaxRetries = 3

	ctx := context.Background()

	// Create repo queue for task enqueuer
	repoQueue := NewRepoTaskQueue(ctx)

	// Create minimal retry handler - for testing we only need store and config
	// taskEnqueuer is needed for AlreadyInQueue test
	retryHandler := retry.NewHandler(
		cfg,
		s,
		nil,       // providerResolver - not needed for NotFound test
		repoQueue, // taskEnqueuer - needed for AlreadyInQueue test
		nil,       // taskBuilder - not needed for NotFound test
		nil,       // runner - not needed for NotFound test
		ctx,
	)

	engine := &Engine{
		cfg:          cfg,
		repoQueue:    repoQueue,
		ctx:          ctx,
		store:        s,
		retryHandler: retryHandler,
	}

	return engine
}

// TestEngine_Retry_NotFound tests retry when review doesn't exist
func TestEngine_Retry_NotFound(t *testing.T) {
	s, cleanup := setupRetryTestDB(t)
	defer cleanup()

	engine := createMinimalTestEngine(t, s)

	err := engine.Retry("nonexistent")
	if err == nil {
		t.Fatal("Retry() should return error for non-existent review")
	}

	appErr, ok := errors.AsAppError(err)
	if !ok {
		t.Fatalf("Expected AppError, got %T: %v", err, err)
	}

	if appErr.Code != errors.ErrCodeReviewNotFound {
		t.Errorf("Expected error code %s, got %s", errors.ErrCodeReviewNotFound, appErr.Code)
	}
}

// TestEngine_Retry_WrongStatus tests retry for non-failed reviews
func TestEngine_Retry_WrongStatus(t *testing.T) {
	s, cleanup := setupRetryTestDB(t)
	defer cleanup()

	engine := createMinimalTestEngine(t, s)

	testCases := []struct {
		name   string
		status model.ReviewStatus
	}{
		{"pending", model.ReviewStatusPending},
		{"running", model.ReviewStatusRunning},
		{"completed", model.ReviewStatusCompleted},
		{"cancelled", model.ReviewStatusCancelled},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Use unique commit_sha for each subtest to avoid unique constraint violation
			review := &model.Review{
				ID:        "test-" + tc.name,
				RepoURL:   "https://github.com/test/repo",
				Ref:       "main",
				CommitSHA: "abc123-" + tc.name, // Unique per test case
				Status:    tc.status,
			}

			if err := s.Review().Create(review); err != nil {
				t.Fatalf("Failed to create review: %v", err)
			}

			err := engine.Retry(review.ID)
			if err == nil {
				t.Fatal("Retry() should return error for non-failed review")
			}

			appErr, ok := errors.AsAppError(err)
			if !ok {
				t.Fatalf("Expected AppError, got %T: %v", err, err)
			}

			if appErr.Code != errors.ErrCodeValidation {
				t.Errorf("Expected error code %s, got %s", errors.ErrCodeValidation, appErr.Code)
			}
		})
	}
}

// TestEngine_Retry_Success tests successful retry state reset
// This test requires a fully configured engine with providers, so it's marked as integration test
func TestEngine_Retry_Success(t *testing.T) {
	// Skip in unit test mode - this requires full engine setup with providers
	// In a real integration test environment, you would configure providers properly
	t.Skip("Skipping integration test - requires full engine setup with providers")

	s, cleanup := setupRetryTestDB(t)
	defer cleanup()

	engine := createMinimalTestEngine(t, s)

	// Create a failed review with associated rules
	review := &model.Review{
		ID:           "test-success",
		RepoURL:      "https://github.com/test/repo",
		Ref:          "main",
		CommitSHA:    "abc123",
		Status:       model.ReviewStatusFailed,
		RetryCount:   0,
		ErrorMessage: "test error",
	}

	if err := s.Review().Create(review); err != nil {
		t.Fatalf("Failed to create review: %v", err)
	}

	// Create associated review rule
	rule := &model.ReviewRule{
		ReviewID:  review.ID,
		RuleIndex: 0,
		RuleID:    "test-rule",
		Status:    model.RuleStatusFailed,
	}

	if err := s.Review().CreateRule(rule); err != nil {
		t.Fatalf("Failed to create review rule: %v", err)
	}

	// Create associated review rule run
	run := &model.ReviewRuleRun{
		ReviewRuleID: rule.ID,
		RunIndex:     0,
		Agent:        "test-agent",
		Status:       model.RunStatusFailed,
	}

	if err := s.Review().CreateRun(run); err != nil {
		t.Fatalf("Failed to create review rule run: %v", err)
	}

	// Perform retry
	err := engine.Retry(review.ID)
	if err != nil {
		t.Fatalf("Retry() failed: %v", err)
	}

	// Verify review was reset
	updatedReview, err := s.Review().GetByID(review.ID)
	if err != nil {
		t.Fatalf("Failed to fetch updated review: %v", err)
	}

	if updatedReview.Status != model.ReviewStatusPending {
		t.Errorf("Review status = %s, want %s", updatedReview.Status, model.ReviewStatusPending)
	}

	if updatedReview.RetryCount != 1 {
		t.Errorf("Review retry_count = %d, want 1", updatedReview.RetryCount)
	}

	if updatedReview.CurrentRuleIndex != 0 {
		t.Errorf("Review current_rule_index = %d, want 0", updatedReview.CurrentRuleIndex)
	}

	if updatedReview.ErrorMessage != "" {
		t.Errorf("Review error_message = %q, want empty", updatedReview.ErrorMessage)
	}

	// Verify review rule was reset
	updatedRule, err := s.Review().GetRuleByID(rule.ID)
	if err != nil {
		t.Fatalf("Failed to fetch updated rule: %v", err)
	}

	if updatedRule.Status != model.RuleStatusPending {
		t.Errorf("Rule status = %s, want %s", updatedRule.Status, model.RuleStatusPending)
	}

	// Verify review rule runs were deleted
	runs, err := s.Review().GetRunsByRuleID(rule.ID)
	if err != nil {
		t.Fatalf("Failed to fetch review rule runs: %v", err)
	}
	if len(runs) != 0 {
		t.Errorf("ReviewRuleRun count = %d, want 0", len(runs))
	}

	// Verify task was enqueued
	if !engine.repoQueue.HasTask(review.ID) {
		t.Error("Review task should be in queue after retry")
	}
}

// TestEngine_Retry_AlreadyInQueue tests retry when task is already in queue
func TestEngine_Retry_AlreadyInQueue(t *testing.T) {
	s, cleanup := setupRetryTestDB(t)
	defer cleanup()

	engine := createMinimalTestEngine(t, s)

	review := &model.Review{
		ID:        "test-in-queue",
		RepoURL:   "https://github.com/test/repo",
		Ref:       "main",
		CommitSHA: "abc123-in-queue",
		Status:    model.ReviewStatusFailed,
	}

	if err := s.Review().Create(review); err != nil {
		t.Fatalf("Failed to create review: %v", err)
	}

	// Manually add the task to the queue to simulate already enqueued state
	// This avoids the need for full provider configuration
	task := &Task{
		Review: review,
	}
	engine.repoQueue.Enqueue(task)

	// Verify task is in queue
	if !engine.repoQueue.HasTask(review.ID) {
		t.Fatal("Task should be in queue after manual enqueue")
	}

	// Retry should fail because task is already in queue
	err := engine.Retry(review.ID)
	if err == nil {
		t.Fatal("Retry() should fail when task is already in queue")
	}

	appErr, ok := errors.AsAppError(err)
	if !ok {
		t.Fatalf("Expected AppError, got %T: %v", err, err)
	}

	if appErr.Code != errors.ErrCodeValidation {
		t.Errorf("Expected error code %s, got %s", errors.ErrCodeValidation, appErr.Code)
	}
}

// TestEngine_Retry_IncrementRetryCount tests retry count increment
// This test requires a fully configured engine with providers, so it's marked as integration test
func TestEngine_Retry_IncrementRetryCount(t *testing.T) {
	// Skip in unit test mode - this requires full engine setup with providers
	t.Skip("Skipping integration test - requires full engine setup with providers")

	s, cleanup := setupRetryTestDB(t)
	defer cleanup()

	engine := createMinimalTestEngine(t, s)
	engine.cfg.Review.MaxRetries = 5

	review := &model.Review{
		ID:         "test-increment",
		RepoURL:    "https://github.com/test/repo",
		Ref:        "main",
		CommitSHA:  "abc123-increment",
		Status:     model.ReviewStatusFailed,
		RetryCount: 2,
	}

	if err := s.Review().Create(review); err != nil {
		t.Fatalf("Failed to create review: %v", err)
	}

	err := engine.Retry(review.ID)
	if err != nil {
		t.Fatalf("Retry() failed: %v", err)
	}

	updatedReview, err := s.Review().GetByID(review.ID)
	if err != nil {
		t.Fatalf("Failed to fetch updated review: %v", err)
	}

	if updatedReview.RetryCount != 3 {
		t.Errorf("Review retry_count = %d, want 3", updatedReview.RetryCount)
	}
}

// timePtr returns a pointer to the given time
func timePtr(t time.Time) *time.Time {
	return &t
}
