// Package recovery provides services for recovering pending/running reviews on startup.
package recovery

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/agent/base"
	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/engine/task"
	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/idgen"
	"github.com/verustcode/verustcode/pkg/logger"
)

// TaskEnqueuer allows components to enqueue tasks for processing.
type TaskEnqueuer interface {
	Enqueue(task *task.Task) bool
	EnqueueAsRunning(task *task.Task) bool
}

// ProviderResolver provides access to git providers.
type ProviderResolver interface {
	Get(name string) provider.Provider
	DetectFromURL(url string) string
}

// Service handles recovery of pending and running reviews on startup.
type Service struct {
	cfg              *config.Config
	store            store.Store
	providerResolver ProviderResolver
	taskEnqueuer     TaskEnqueuer
}

// NewService creates a new RecoveryService instance.
func NewService(cfg *config.Config, s store.Store, providerResolver ProviderResolver, taskEnqueuer TaskEnqueuer) *Service {
	return &Service{
		cfg:              cfg,
		store:            s,
		providerResolver: providerResolver,
		taskEnqueuer:     taskEnqueuer,
	}
}

// RecoverToQueue recovers pending and running reviews from the database and re-queues them.
// This is called during engine startup to ensure no reviews are lost across restarts.
func (s *Service) RecoverToQueue(ctx context.Context) {
	reviews, err := s.store.Review().ListPendingOrRunning()
	if err != nil {
		logger.Error("Failed to query pending/running reviews for recovery",
			zap.Error(err),
		)
		return
	}

	if len(reviews) == 0 {
		logger.Info("No pending reviews to recover")
		return
	}

	logger.Info("Recovering pending/running reviews to memory queue",
		zap.Int("count", len(reviews)),
	)

	pendingRecovered := 0
	runningRecovered := 0
	failed := 0

	for i := range reviews {
		review := &reviews[i]

		// Check if the review should be recovered or marked as failed
		shouldRecover, reason := s.shouldRecover(review)
		if !shouldRecover {
			logger.Warn("Review cannot be recovered, marking as failed",
				zap.String("review_id", review.ID),
				zap.String("status", string(review.Status)),
				zap.String("reason", reason),
			)
			failed++

			// Mark review as failed with reason
			s.store.Review().UpdateStatusWithError(review.ID, model.ReviewStatusFailed, fmt.Sprintf("recovery failed: %s", reason))
			continue
		}

		task := s.buildRecoveryTask(ctx, review)
		if task == nil {
			logger.Error("Failed to build recovery task",
				zap.String("review_id", review.ID),
				zap.String("status", string(review.Status)),
			)
			failed++

			// Mark review as failed if recovery fails
			s.store.Review().UpdateStatusWithError(review.ID, model.ReviewStatusFailed, "recovery failed: could not build recovery task")
			continue
		}

		// Add task to memory queue
		// Running tasks are marked as running in the queue (repo is busy)
		// Pending tasks are queued normally
		if review.Status == model.ReviewStatusRunning {
			// This repo has a running task, enqueue as running
			if s.taskEnqueuer.EnqueueAsRunning(task) {
				runningRecovered++
				logger.Info("Running review recovered to queue",
					zap.String("review_id", review.ID),
					zap.String("repo_url", review.RepoURL),
				)
			}
		} else {
			// Pending task, add to queue normally
			if s.taskEnqueuer.Enqueue(task) {
				pendingRecovered++
				logger.Info("Pending review recovered to queue",
					zap.String("review_id", review.ID),
					zap.String("repo_url", review.RepoURL),
				)
			}
		}
	}

	logger.Info("Review recovery completed",
		zap.Int("total", len(reviews)),
		zap.Int("pending_recovered", pendingRecovered),
		zap.Int("running_recovered", runningRecovered),
		zap.Int("failed", failed),
	)
}

// BuildRecoveryTask creates a Task from a persisted Review for recovery.
// This is a public method that implements the TaskBuilder interface for retry handler.
func (s *Service) BuildRecoveryTask(ctx context.Context, review *model.Review) *task.Task {
	return s.buildRecoveryTask(ctx, review)
}

// buildRecoveryTask creates a Task from a persisted Review for recovery.
// Note: ReviewRulesConfig is nil here and will be dynamically loaded in processTask
// after cloning the repository, following the priority order:
// 1. .verust-review.yaml at repository root
// 2. Database configured review file
// 3. config/reviews/default.yaml
func (s *Service) buildRecoveryTask(ctx context.Context, review *model.Review) *task.Task {
	// Detect provider from RepoURL
	providerName := s.providerResolver.DetectFromURL(review.RepoURL)
	if providerName == "" {
		logger.Warn("Cannot detect provider for review, skipping recovery",
			zap.String("review_id", review.ID),
			zap.String("repo_url", review.RepoURL),
		)
		return nil
	}

	// Validate provider
	prov := s.providerResolver.Get(providerName)
	if prov == nil {
		logger.Warn("Provider not configured, skipping recovery",
			zap.String("review_id", review.ID),
			zap.String("provider", providerName),
		)
		return nil
	}

	// Extract owner and repo from URL using provider-specific parsing
	owner, repoName, err := prov.ParseRepoPath(review.RepoURL)
	if err != nil {
		logger.Warn("Failed to parse repository URL, skipping recovery",
			zap.String("review_id", review.ID),
			zap.String("repo_url", review.RepoURL),
			zap.Error(err),
		)
		return nil
	}

	// Create task for recovery (task is identified by Review.ID, no separate task ID)
	// ReviewRulesConfig is nil - it will be loaded dynamically in processTask
	return &task.Task{
		Review:            review,
		ProviderName:      providerName,
		CreatedAt:         time.Now(),
		ReviewRulesConfig: nil, // Will be loaded dynamically in processTask
		Request: &base.ReviewRequest{
			RequestID:    idgen.NewRequestID(),
			RepoURL:      review.RepoURL,
			Owner:        owner,
			RepoName:     repoName,
			Ref:          review.Ref,
			CommitSHA:    review.CommitSHA,
			PRNumber:     review.PRNumber,
			PRTitle:      "",
			PRBody:       "",
			ChangedFiles: []string{},
		},
	}
}

// shouldRecover checks if a review should be recovered or marked as failed.
// Returns (true, "") if the review should be recovered, or (false, reason) if it should be marked as failed.
func (s *Service) shouldRecover(review *model.Review) (bool, string) {
	// Check retry count
	maxRetryCount := s.cfg.Recovery.MaxRetryCount
	if maxRetryCount <= 0 {
		maxRetryCount = 3 // Default
	}
	if review.RetryCount >= maxRetryCount {
		return false, fmt.Sprintf("max retry count exceeded: %d", review.RetryCount)
	}

	// Check timeout (only for running tasks)
	if review.Status == model.ReviewStatusRunning && review.StartedAt != nil {
		timeoutHours := s.cfg.Recovery.TaskTimeoutHours
		if timeoutHours <= 0 {
			timeoutHours = 24 // Default 24 hours
		}

		elapsed := time.Since(*review.StartedAt)
		timeout := time.Duration(timeoutHours) * time.Hour

		if elapsed > timeout {
			return false, fmt.Sprintf("task timeout: started %v ago (timeout: %v)", elapsed.Round(time.Minute), timeout)
		}
	}

	return true, ""
}
