// Package handler provides HTTP handlers for the API.
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/notification"
	"github.com/verustcode/verustcode/pkg/errors"
	"github.com/verustcode/verustcode/pkg/logger"
)

// NotificationHandler handles notification-related HTTP requests
type NotificationHandler struct{}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{}
}

// TestNotificationRequest represents the request body for testing notifications
type TestNotificationRequest struct {
	// EventType is the event type to simulate (review_failed, review_completed, report_failed, report_completed)
	EventType string `json:"event_type" binding:"required,oneof=review_failed review_completed report_failed report_completed"`
}

// TestNotificationResponse represents the response for test notification
type TestNotificationResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Channel string `json:"channel,omitempty"`
}

// TestNotification handles POST /api/v1/notifications/test
// This endpoint sends a test notification using the current configuration
func (h *NotificationHandler) TestNotification(c *gin.Context) {
	var req TestNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	manager := notification.GetManager()
	if manager == nil || !manager.IsEnabled() {
		c.JSON(http.StatusOK, TestNotificationResponse{
			Success: false,
			Message: "Notifications are not enabled",
		})
		return
	}

	// Build a test event
	var event *notification.Event
	switch req.EventType {
	case "review_failed":
		event = &notification.Event{
			Type:         notification.EventReviewFailed,
			TaskID:       "test-review-001",
			TaskType:     "review",
			RepoURL:      "https://github.com/example/test-repo",
			ErrorMessage: "This is a test notification for review failure",
			Timestamp:    time.Now(),
			Extra: map[string]interface{}{
				"ref":        "main",
				"commit_sha": "abc1234",
				"source":     "test",
			},
		}
	case "review_completed":
		event = &notification.Event{
			Type:      notification.EventReviewCompleted,
			TaskID:    "test-review-001",
			TaskType:  "review",
			RepoURL:   "https://github.com/example/test-repo",
			Timestamp: time.Now(),
			Extra: map[string]interface{}{
				"ref":         "main",
				"commit_sha":  "abc1234",
				"source":      "test",
				"duration_ms": 12345,
			},
		}
	case "report_failed":
		event = &notification.Event{
			Type:         notification.EventReportFailed,
			TaskID:       "test-report-001",
			TaskType:     "report",
			RepoURL:      "https://github.com/example/test-repo",
			ErrorMessage: "This is a test notification for report failure",
			Timestamp:    time.Now(),
			Extra: map[string]interface{}{
				"report_type": "wiki",
				"ref":         "main",
				"context":     "test",
			},
		}
	case "report_completed":
		event = &notification.Event{
			Type:      notification.EventReportCompleted,
			TaskID:    "test-report-001",
			TaskType:  "report",
			RepoURL:   "https://github.com/example/test-repo",
			Timestamp: time.Now(),
			Extra: map[string]interface{}{
				"report_type": "wiki",
				"ref":         "main",
				"title":       "Test Report",
				"duration_ms": 54321,
			},
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid event_type: " + req.EventType,
		})
		return
	}

	// Send test notification with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	logger.Info("Sending test notification",
		zap.String("event_type", req.EventType),
		zap.String("channel", string(manager.GetChannel())),
	)

	if err := manager.Notify(ctx, event); err != nil {
		logger.Error("Test notification failed",
			zap.String("event_type", req.EventType),
			zap.Error(err),
		)
		c.JSON(http.StatusOK, TestNotificationResponse{
			Success: false,
			Message: "Failed to send notification: " + err.Error(),
			Channel: string(manager.GetChannel()),
		})
		return
	}

	logger.Info("Test notification sent successfully",
		zap.String("event_type", req.EventType),
		zap.String("channel", string(manager.GetChannel())),
	)

	c.JSON(http.StatusOK, TestNotificationResponse{
		Success: true,
		Message: "Test notification sent successfully",
		Channel: string(manager.GetChannel()),
	})
}

// GetNotificationStatus handles GET /api/v1/notifications/status
// Returns the current notification configuration status
func (h *NotificationHandler) GetNotificationStatus(c *gin.Context) {
	manager := notification.GetManager()
	if manager == nil {
		c.JSON(http.StatusOK, gin.H{
			"enabled": false,
			"channel": "",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled": manager.IsEnabled(),
		"channel": string(manager.GetChannel()),
	})
}
