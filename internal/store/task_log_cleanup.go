// Package store provides data access operations for all models.
package store

import (
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/verustcode/verustcode/pkg/logger"
)

const (
	// DefaultTaskLogRetentionDays is the default number of days to retain task logs
	DefaultTaskLogRetentionDays = 30
	// TaskLogCleanupSchedule is the cron schedule for task log cleanup (daily at 2 AM)
	TaskLogCleanupSchedule = "0 2 * * *" // Every day at 2:00 AM
)

// TaskLogCleanupService manages periodic cleanup of old task logs
type TaskLogCleanupService struct {
	store         TaskLogStore
	cron          *cron.Cron
	retentionDays int
	entryID       cron.EntryID
	mu            sync.RWMutex
}

// NewTaskLogCleanupService creates a new task log cleanup service
func NewTaskLogCleanupService(store TaskLogStore, retentionDays int) *TaskLogCleanupService {
	if retentionDays <= 0 {
		retentionDays = DefaultTaskLogRetentionDays
	}

	return &TaskLogCleanupService{
		store:         store,
		cron:          cron.New(),
		retentionDays: retentionDays,
	}
}

// Start starts the cleanup service with scheduled cleanup tasks
func (s *TaskLogCleanupService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Add cleanup job
	entryID, err := s.cron.AddFunc(TaskLogCleanupSchedule, s.cleanup)
	if err != nil {
		logger.Error("Failed to schedule task log cleanup", zap.Error(err))
		return err
	}

	s.entryID = entryID

	// Start cron scheduler
	s.cron.Start()

	logger.Info("Task log cleanup service started",
		zap.String("schedule", TaskLogCleanupSchedule),
		zap.Int("retention_days", s.retentionDays),
	)

	// Run initial cleanup immediately (non-blocking)
	go s.cleanup()

	return nil
}

// Stop stops the cleanup service gracefully
func (s *TaskLogCleanupService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cron != nil {
		logger.Info("Stopping task log cleanup service")
		ctx := s.cron.Stop()
		<-ctx.Done()
		logger.Info("Task log cleanup service stopped")
	}
}

// cleanup performs the actual cleanup of old task logs
func (s *TaskLogCleanupService) cleanup() {
	logger.Info("Starting task log cleanup",
		zap.Int("retention_days", s.retentionDays),
	)

	startTime := time.Now()
	deletedCount, err := s.store.DeleteOlderThan(s.retentionDays)
	if err != nil {
		logger.Error("Failed to cleanup old task logs",
			zap.Int("retention_days", s.retentionDays),
			zap.Error(err),
		)
		return
	}

	duration := time.Since(startTime)
	logger.Info("Task log cleanup completed",
		zap.Int64("deleted_count", deletedCount),
		zap.Int("retention_days", s.retentionDays),
		zap.Duration("duration", duration),
	)
}

// SetRetentionDays updates the retention period (takes effect on next cleanup)
func (s *TaskLogCleanupService) SetRetentionDays(days int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if days <= 0 {
		days = DefaultTaskLogRetentionDays
	}

	s.retentionDays = days
	logger.Info("Task log retention days updated",
		zap.Int("retention_days", days),
	)
}
