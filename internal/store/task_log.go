// Package store provides data access operations for all models.
package store

import (
	"gorm.io/gorm"

	"github.com/verustcode/verustcode/internal/model"
)

// TaskLogStore defines operations for TaskLog model.
// Note: This store uses a separate database connection (task_logs.db)
// instead of the main application database.
// TaskLogStore also implements the logger.TaskLogWriter interface.
type TaskLogStore interface {
	// Write implements logger.TaskLogWriter interface for batch writing logs.
	// This is used by the logger hook for dual-write mode.
	Write(logs []model.TaskLog) error

	// Create creates a new task log entry
	Create(log *model.TaskLog) error

	// BatchCreate creates multiple task log entries in a single transaction
	BatchCreate(logs []model.TaskLog) error

	// GetByTaskID retrieves all logs for a specific task
	GetByTaskID(taskType model.TaskType, taskID string) ([]model.TaskLog, error)

	// GetByTaskIDWithPagination retrieves logs for a task with pagination
	GetByTaskIDWithPagination(taskType model.TaskType, taskID string, page, pageSize int) ([]model.TaskLog, int64, error)

	// GetByTaskIDWithLevel retrieves logs for a task filtered by log level
	GetByTaskIDWithLevel(taskType model.TaskType, taskID string, level model.LogLevel) ([]model.TaskLog, error)

	// GetByTaskIDAndLevel retrieves logs for a task filtered by log level with pagination
	GetByTaskIDAndLevel(taskType model.TaskType, taskID string, level model.LogLevel, page, pageSize int) ([]model.TaskLog, int64, error)

	// GetByTaskIDWithLevelAndAbove retrieves logs for a task at or above a specified level
	GetByTaskIDWithLevelAndAbove(taskType model.TaskType, taskID string, level model.LogLevel) ([]model.TaskLog, error)

	// GetLatestByTaskID retrieves the latest N logs for a task
	GetLatestByTaskID(taskType model.TaskType, taskID string, limit int) ([]model.TaskLog, error)

	// DeleteByTaskID deletes all logs for a specific task
	DeleteByTaskID(taskType model.TaskType, taskID string) error

	// DeleteOlderThan deletes logs older than a specified duration (for cleanup)
	DeleteOlderThan(days int) (int64, error)

	// CountByTaskID returns the total count of logs for a task
	CountByTaskID(taskType model.TaskType, taskID string) (int64, error)
}

// taskLogStore implements TaskLogStore using GORM.
type taskLogStore struct {
	db *gorm.DB
}

// NewTaskLogStore creates a new TaskLogStore with the provided database connection.
// Note: This should be called with the task log database connection, not the main database.
func NewTaskLogStore(db *gorm.DB) TaskLogStore {
	return &taskLogStore{db: db}
}

// Write implements logger.TaskLogWriter interface for batch writing logs.
// This is used by the logger hook for dual-write mode.
func (s *taskLogStore) Write(logs []model.TaskLog) error {
	return s.BatchCreate(logs)
}

// Create creates a new task log entry.
func (s *taskLogStore) Create(log *model.TaskLog) error {
	return s.db.Create(log).Error
}

// BatchCreate creates multiple task log entries in a single transaction.
func (s *taskLogStore) BatchCreate(logs []model.TaskLog) error {
	if len(logs) == 0 {
		return nil
	}
	return s.db.Create(&logs).Error
}

// GetByTaskID retrieves all logs for a specific task, ordered by creation time ascending.
func (s *taskLogStore) GetByTaskID(taskType model.TaskType, taskID string) ([]model.TaskLog, error) {
	var logs []model.TaskLog
	err := s.db.Where("task_type = ? AND task_id = ?", taskType, taskID).
		Order("created_at ASC").
		Find(&logs).Error
	return logs, err
}

// GetByTaskIDWithPagination retrieves logs for a task with pagination.
func (s *taskLogStore) GetByTaskIDWithPagination(taskType model.TaskType, taskID string, page, pageSize int) ([]model.TaskLog, int64, error) {
	var logs []model.TaskLog
	var total int64

	query := s.db.Model(&model.TaskLog{}).Where("task_type = ? AND task_id = ?", taskType, taskID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_at ASC").Offset(offset).Limit(pageSize).Find(&logs).Error
	return logs, total, err
}

// GetByTaskIDWithLevel retrieves logs for a task filtered by log level.
func (s *taskLogStore) GetByTaskIDWithLevel(taskType model.TaskType, taskID string, level model.LogLevel) ([]model.TaskLog, error) {
	var logs []model.TaskLog
	err := s.db.Where("task_type = ? AND task_id = ? AND level = ?", taskType, taskID, level).
		Order("created_at ASC").
		Find(&logs).Error
	return logs, err
}

// GetByTaskIDAndLevel retrieves logs for a task filtered by log level with pagination.
func (s *taskLogStore) GetByTaskIDAndLevel(taskType model.TaskType, taskID string, level model.LogLevel, page, pageSize int) ([]model.TaskLog, int64, error) {
	var logs []model.TaskLog
	var total int64

	query := s.db.Model(&model.TaskLog{}).Where("task_type = ? AND task_id = ? AND level = ?", taskType, taskID, level)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_at ASC").Offset(offset).Limit(pageSize).Find(&logs).Error
	return logs, total, err
}

// GetByTaskIDWithLevelAndAbove retrieves logs at or above a specified level.
// Level priority: debug < info < warn < error < fatal
func (s *taskLogStore) GetByTaskIDWithLevelAndAbove(taskType model.TaskType, taskID string, level model.LogLevel) ([]model.TaskLog, error) {
	var logs []model.TaskLog
	levels := getLevelsAtAndAbove(level)
	err := s.db.Where("task_type = ? AND task_id = ? AND level IN ?", taskType, taskID, levels).
		Order("created_at ASC").
		Find(&logs).Error
	return logs, err
}

// GetLatestByTaskID retrieves the latest N logs for a task.
func (s *taskLogStore) GetLatestByTaskID(taskType model.TaskType, taskID string, limit int) ([]model.TaskLog, error) {
	var logs []model.TaskLog
	err := s.db.Where("task_type = ? AND task_id = ?", taskType, taskID).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error

	// Reverse the slice to return in chronological order
	for i, j := 0, len(logs)-1; i < j; i, j = i+1, j-1 {
		logs[i], logs[j] = logs[j], logs[i]
	}
	return logs, err
}

// DeleteByTaskID deletes all logs for a specific task.
func (s *taskLogStore) DeleteByTaskID(taskType model.TaskType, taskID string) error {
	return s.db.Where("task_type = ? AND task_id = ?", taskType, taskID).
		Delete(&model.TaskLog{}).Error
}

// DeleteOlderThan deletes logs older than a specified number of days.
func (s *taskLogStore) DeleteOlderThan(days int) (int64, error) {
	result := s.db.Exec(
		"DELETE FROM task_logs WHERE created_at < datetime('now', '-' || ? || ' days')",
		days,
	)
	return result.RowsAffected, result.Error
}

// CountByTaskID returns the total count of logs for a task.
func (s *taskLogStore) CountByTaskID(taskType model.TaskType, taskID string) (int64, error) {
	var count int64
	err := s.db.Model(&model.TaskLog{}).
		Where("task_type = ? AND task_id = ?", taskType, taskID).
		Count(&count).Error
	return count, err
}

// getLevelsAtAndAbove returns all log levels at or above the specified level.
func getLevelsAtAndAbove(level model.LogLevel) []model.LogLevel {
	switch level {
	case model.LogLevelDebug:
		return []model.LogLevel{model.LogLevelDebug, model.LogLevelInfo, model.LogLevelWarn, model.LogLevelError, model.LogLevelFatal}
	case model.LogLevelInfo:
		return []model.LogLevel{model.LogLevelInfo, model.LogLevelWarn, model.LogLevelError, model.LogLevelFatal}
	case model.LogLevelWarn:
		return []model.LogLevel{model.LogLevelWarn, model.LogLevelError, model.LogLevelFatal}
	case model.LogLevelError:
		return []model.LogLevel{model.LogLevelError, model.LogLevelFatal}
	case model.LogLevelFatal:
		return []model.LogLevel{model.LogLevelFatal}
	default:
		return []model.LogLevel{model.LogLevelDebug, model.LogLevelInfo, model.LogLevelWarn, model.LogLevelError, model.LogLevelFatal}
	}
}
