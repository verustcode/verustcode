// Package handler provides HTTP handlers for the API.
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/errors"
	"github.com/verustcode/verustcode/pkg/logger"
)

// TaskLogHandler handles task log related HTTP requests
type TaskLogHandler struct {
	store store.TaskLogStore
}

// NewTaskLogHandler creates a new task log handler
func NewTaskLogHandler(s store.TaskLogStore) *TaskLogHandler {
	return &TaskLogHandler{store: s}
}

// taskLogPagination configuration
const (
	defaultLogPage     = 1
	defaultLogPageSize = 50
	minLogPageSize     = 1
	maxLogPageSize     = 500
)

// GetReviewLogs handles GET /api/v1/reviews/:id/logs
func (h *TaskLogHandler) GetReviewLogs(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid review ID",
		})
		return
	}

	logs, total, err := h.getLogs(c, model.TaskTypeReview, id)
	if err != nil {
		return // Error already handled in getLogs
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      logs,
		"total":     total,
		"task_type": model.TaskTypeReview,
		"task_id":   id,
	})
}

// GetReportLogs handles GET /api/v1/reports/:id/logs
func (h *TaskLogHandler) GetReportLogs(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid report ID",
		})
		return
	}

	logs, total, err := h.getLogs(c, model.TaskTypeReport, id)
	if err != nil {
		return // Error already handled in getLogs
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      logs,
		"total":     total,
		"task_type": model.TaskTypeReport,
		"task_id":   id,
	})
}

// getLogs is a shared helper for fetching logs with pagination
func (h *TaskLogHandler) getLogs(c *gin.Context, taskType model.TaskType, taskID string) ([]model.TaskLog, int64, error) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", strconv.Itoa(defaultLogPage)))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(defaultLogPageSize)))
	level := c.Query("level")

	if page < 1 {
		page = defaultLogPage
	}
	if pageSize < minLogPageSize || pageSize > maxLogPageSize {
		pageSize = defaultLogPageSize
	}

	// Validate level if provided
	var levelFilter model.LogLevel
	if level != "" {
		switch level {
		case "debug", "info", "warn", "error", "fatal":
			levelFilter = model.LogLevel(level)
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    errors.ErrCodeValidation,
				"message": "Invalid log level, must be one of: debug, info, warn, error, fatal",
			})
			return nil, 0, errors.New(errors.ErrCodeValidation, "invalid level")
		}
	}

	var logs []model.TaskLog
	var total int64
	var err error

	if levelFilter != "" {
		logs, total, err = h.store.GetByTaskIDAndLevel(taskType, taskID, levelFilter, page, pageSize)
	} else {
		logs, total, err = h.store.GetByTaskIDWithPagination(taskType, taskID, page, pageSize)
	}

	if err != nil {
		logger.Error("Failed to fetch task logs",
			zap.String("task_type", string(taskType)),
			zap.String("task_id", taskID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeDBQuery,
			"message": "Failed to fetch logs",
		})
		return nil, 0, err
	}

	return logs, total, nil
}
