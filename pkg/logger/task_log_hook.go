// Package logger provides structured logging capabilities for the application.
package logger

import (
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/zap/zapcore"

	"github.com/verustcode/verustcode/internal/model"
)

const (
	// FieldReviewID is the field key for review ID in log entries
	FieldReviewID = "review_id"
	// FieldReportID is the field key for report ID in log entries
	FieldReportID = "report_id"

	// bufferSize is the size of the log buffer before flushing to database
	bufferSize = 100
	// flushInterval is the interval for periodic buffer flushing
	flushInterval = 5 * time.Second
)

// TaskLogWriter defines the interface for writing task logs to storage.
// This abstraction allows the logger package to remain independent of database packages.
type TaskLogWriter interface {
	// Write writes a batch of task logs to storage
	Write(logs []model.TaskLog) error
}

// TaskLogHook captures logs containing review_id or report_id fields
// and writes them to a separate task log database.
type TaskLogHook struct {
	writer TaskLogWriter

	// Buffer for batch writes
	buffer []model.TaskLog
	mu     sync.Mutex

	// Background flushing
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewTaskLogHook creates a new TaskLogHook with the given writer.
func NewTaskLogHook(writer TaskLogWriter) *TaskLogHook {
	hook := &TaskLogHook{
		writer: writer,
		buffer: make([]model.TaskLog, 0, bufferSize),
		stopCh: make(chan struct{}),
	}

	// Start background flushing goroutine
	hook.wg.Add(1)
	go hook.backgroundFlush()

	return hook
}

// taskLogCore wraps a zapcore.Core to intercept logs and capture task-related entries.
type taskLogCore struct {
	zapcore.Core
	hook   *TaskLogHook
	fields []zapcore.Field
}

// WrapCore wraps a zapcore.Core with the TaskLogHook to capture task-related logs.
func (h *TaskLogHook) WrapCore(core zapcore.Core) zapcore.Core {
	return &taskLogCore{
		Core:   core,
		hook:   h,
		fields: nil,
	}
}

// With creates a new Core with additional fields.
func (c *taskLogCore) With(fields []zapcore.Field) zapcore.Core {
	// Merge fields
	newFields := make([]zapcore.Field, 0, len(c.fields)+len(fields))
	newFields = append(newFields, c.fields...)
	newFields = append(newFields, fields...)

	return &taskLogCore{
		Core:   c.Core.With(fields),
		hook:   c.hook,
		fields: newFields,
	}
}

// Check determines whether the supplied Entry should be logged.
func (c *taskLogCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return ce.AddCore(entry, c)
	}
	return ce
}

// Write intercepts log writes to capture task-related logs.
func (c *taskLogCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	// First, write to the underlying core
	if err := c.Core.Write(entry, fields); err != nil {
		return err
	}

	// Combine with context fields
	allFields := make([]zapcore.Field, 0, len(c.fields)+len(fields))
	allFields = append(allFields, c.fields...)
	allFields = append(allFields, fields...)

	// Check if this log contains task-related fields
	taskType, taskID := extractTaskInfo(allFields)
	if taskType == "" || taskID == "" {
		return nil
	}

	// Create TaskLog entry
	taskLog := model.TaskLog{
		CreatedAt: entry.Time,
		TaskType:  taskType,
		TaskID:    taskID,
		Level:     convertLevel(entry.Level),
		Message:   entry.Message,
		Caller:    entry.Caller.String(),
		Fields:    serializeFields(allFields),
	}

	// Add to buffer
	c.hook.addToBuffer(taskLog)

	return nil
}

// Sync flushes any buffered logs.
func (c *taskLogCore) Sync() error {
	c.hook.Flush()
	return c.Core.Sync()
}

// addToBuffer adds a log entry to the buffer.
func (h *TaskLogHook) addToBuffer(log model.TaskLog) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.buffer = append(h.buffer, log)

	// Flush if buffer is full
	if len(h.buffer) >= bufferSize {
		h.flushLocked()
	}
}

// Flush writes all buffered logs to storage.
func (h *TaskLogHook) Flush() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.flushLocked()
}

// flushLocked writes buffered logs to storage (must be called with lock held).
func (h *TaskLogHook) flushLocked() {
	if len(h.buffer) == 0 {
		return
	}

	logs := h.buffer
	h.buffer = make([]model.TaskLog, 0, bufferSize)

	// Write to storage (non-blocking)
	go func(logs []model.TaskLog) {
		if err := h.writer.Write(logs); err != nil {
			// Log error to stderr (avoid recursive logging)
			fmt.Fprintf(os.Stderr, "Failed to write task logs: %v\n", err)
		}
	}(logs)
}

// backgroundFlush periodically flushes the buffer.
func (h *TaskLogHook) backgroundFlush() {
	defer h.wg.Done()

	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.Flush()
		case <-h.stopCh:
			h.Flush()
			return
		}
	}
}

// Close stops the background flushing and flushes remaining logs.
func (h *TaskLogHook) Close() {
	close(h.stopCh)
	h.wg.Wait()
}

// extractTaskInfo extracts task type and ID from log fields.
func extractTaskInfo(fields []zapcore.Field) (model.TaskType, string) {
	for _, field := range fields {
		switch field.Key {
		case FieldReviewID:
			if field.String != "" {
				return model.TaskTypeReview, field.String
			}
		case FieldReportID:
			if field.String != "" {
				return model.TaskTypeReport, field.String
			}
		}
	}
	return "", ""
}

// convertLevel converts zapcore.Level to model.LogLevel.
func convertLevel(level zapcore.Level) model.LogLevel {
	switch level {
	case zapcore.DebugLevel:
		return model.LogLevelDebug
	case zapcore.InfoLevel:
		return model.LogLevelInfo
	case zapcore.WarnLevel:
		return model.LogLevelWarn
	case zapcore.ErrorLevel:
		return model.LogLevelError
	case zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		return model.LogLevelFatal
	default:
		return model.LogLevelInfo
	}
}

// serializeFields converts zapcore.Field slice to model.JSONMap.
func serializeFields(fields []zapcore.Field) model.JSONMap {
	if len(fields) == 0 {
		return model.JSONMap{}
	}

	data := make(model.JSONMap)
	for _, field := range fields {
		// Skip task identification fields (already stored separately)
		if field.Key == FieldReviewID || field.Key == FieldReportID {
			continue
		}

		switch field.Type {
		case zapcore.StringType:
			data[field.Key] = field.String
		case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
			data[field.Key] = field.Integer
		case zapcore.Uint64Type, zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type:
			data[field.Key] = uint64(field.Integer)
		case zapcore.Float64Type, zapcore.Float32Type:
			data[field.Key] = field.Integer
		case zapcore.BoolType:
			data[field.Key] = field.Integer == 1
		case zapcore.DurationType:
			data[field.Key] = time.Duration(field.Integer).String()
		case zapcore.TimeType, zapcore.TimeFullType:
			if t, ok := field.Interface.(time.Time); ok {
				data[field.Key] = t.Format(time.RFC3339)
			}
		case zapcore.ErrorType:
			if err, ok := field.Interface.(error); ok && err != nil {
				data[field.Key] = err.Error()
			}
		default:
			if field.Interface != nil {
				data[field.Key] = fmt.Sprint(field.Interface)
			}
		}
	}

	return data
}
