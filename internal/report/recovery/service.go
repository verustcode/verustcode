// Package recovery provides services for recovering pending/processing reports on startup.
package recovery

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/logger"
)

// TaskEnqueuer allows components to enqueue report tasks for processing.
type TaskEnqueuer interface {
	Enqueue(report *model.Report, callback func(*model.Report, error)) bool
}

// Service handles recovery of pending and processing reports on startup.
type Service struct {
	cfg          *config.Config
	store        store.Store
	taskEnqueuer TaskEnqueuer
}

// NewService creates a new RecoveryService instance.
func NewService(cfg *config.Config, s store.Store, taskEnqueuer TaskEnqueuer) *Service {
	return &Service{
		cfg:          cfg,
		store:        s,
		taskEnqueuer: taskEnqueuer,
	}
}

// RecoverToQueue recovers pending and processing reports from the database and re-queues them.
// This is called during engine startup to ensure no reports are lost across restarts.
func (s *Service) RecoverToQueue(ctx context.Context) {
	reports, err := s.store.Report().ListPendingOrProcessing()
	if err != nil {
		logger.Error("Failed to query pending/processing reports for recovery",
			zap.Error(err),
		)
		return
	}

	if len(reports) == 0 {
		logger.Info("No pending reports to recover")
		return
	}

	logger.Info("Recovering pending/processing reports to memory queue",
		zap.Int("count", len(reports)),
	)

	recovered := 0
	failed := 0

	for i := range reports {
		report := &reports[i]

		// Check if the report should be recovered or marked as failed
		shouldRecover, reason := s.shouldRecover(report)
		if !shouldRecover {
			logger.Warn("Report cannot be recovered, marking as failed",
				zap.String("report_id", report.ID),
				zap.String("status", string(report.Status)),
				zap.String("reason", reason),
			)
			failed++

			// Mark report as failed with reason
			s.store.Report().UpdateStatusWithError(report.ID, model.ReportStatusFailed, fmt.Sprintf("recovery failed: %s", reason))
			continue
		}

		// Enqueue the report for processing
		if s.taskEnqueuer.Enqueue(report, nil) {
			recovered++
			logger.Info("Report recovered to queue",
				zap.String("report_id", report.ID),
				zap.String("repo_url", report.RepoURL),
				zap.String("status", string(report.Status)),
			)
		} else {
			logger.Warn("Failed to enqueue recovered report",
				zap.String("report_id", report.ID),
			)
			failed++
			s.store.Report().UpdateStatusWithError(report.ID, model.ReportStatusFailed, "recovery failed: could not enqueue task")
		}
	}

	logger.Info("Report recovery completed",
		zap.Int("total", len(reports)),
		zap.Int("recovered", recovered),
		zap.Int("failed", failed),
	)
}

// shouldRecover checks if a report should be recovered or marked as failed.
// Returns (true, "") if the report should be recovered, or (false, reason) if it should be marked as failed.
func (s *Service) shouldRecover(report *model.Report) (bool, string) {
	// Check retry count
	maxRetryCount := s.cfg.Recovery.MaxRetryCount
	if maxRetryCount <= 0 {
		maxRetryCount = 3 // Default
	}
	if report.RetryCount >= maxRetryCount {
		return false, fmt.Sprintf("max retry count exceeded: %d", report.RetryCount)
	}

	// Check timeout (only for started tasks)
	if report.StartedAt != nil {
		timeoutHours := s.cfg.Recovery.TaskTimeoutHours
		if timeoutHours <= 0 {
			timeoutHours = 24 // Default 24 hours
		}

		elapsed := time.Since(*report.StartedAt)
		timeout := time.Duration(timeoutHours) * time.Hour

		if elapsed > timeout {
			return false, fmt.Sprintf("task timeout: started %v ago (timeout: %v)", elapsed.Round(time.Minute), timeout)
		}
	}

	return true, ""
}
