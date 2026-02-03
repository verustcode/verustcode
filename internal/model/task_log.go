// Package model provides database model definitions.
package model

import (
	"time"
)

// LogLevel represents the log level
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
)

// TaskType represents the type of task that generated the log
type TaskType string

const (
	TaskTypeReview TaskType = "review"
	TaskTypeReport TaskType = "report"
)

// TaskLog represents a log entry associated with a specific task (review or report)
type TaskLog struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`

	// Task identification
	TaskType TaskType `gorm:"size:20;not null;index" json:"task_type"` // review or report
	TaskID   string   `gorm:"size:20;not null;index" json:"task_id"`   // review_id or report_id

	// Log content
	Level   LogLevel `gorm:"size:10;not null;index" json:"level"`
	Message string   `gorm:"type:text;not null" json:"message"`
	Fields  JSONMap  `gorm:"type:text" json:"fields,omitempty"` // structured log fields as JSON

	// Source information
	Caller string `gorm:"size:255" json:"caller,omitempty"` // file:line of the log call
}

// TableName specifies the table name for TaskLog
func (TaskLog) TableName() string {
	return "task_logs"
}

// TaskLogQuery represents query parameters for listing task logs
type TaskLogQuery struct {
	TaskType TaskType `json:"task_type"`
	TaskID   string   `json:"task_id"`
	Level    LogLevel `json:"level,omitempty"`
	Limit    int      `json:"limit,omitempty"`
	Offset   int      `json:"offset,omitempty"`
}
