package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/notification"
)

// setupNotificationTest creates a mock store with the given notification config
// and resets the notification manager for testing.
func setupNotificationTest(t *testing.T, cfg *config.NotificationConfig) *MockStore {
	t.Helper()
	mockStore := NewMockStore()

	if cfg != nil {
		// Store notification settings in mock store
		settings := []model.SystemSetting{}

		if cfg.Channel != "" {
			settings = append(settings, model.SystemSetting{
				Category: string(model.SettingCategoryNotifications),
				Key:      "channel",
				Value:    string(cfg.Channel),
			})
		}

		if len(cfg.Events) > 0 {
			eventsJSON, _ := json.Marshal(cfg.Events)
			settings = append(settings, model.SystemSetting{
				Category: string(model.SettingCategoryNotifications),
				Key:      "events",
				Value:    string(eventsJSON),
			})
		}

		if cfg.Webhook.URL != "" {
			webhookJSON, _ := json.Marshal(cfg.Webhook)
			settings = append(settings, model.SystemSetting{
				Category: string(model.SettingCategoryNotifications),
				Key:      "webhook",
				Value:    string(webhookJSON),
			})
		}

		if cfg.Email.SMTPHost != "" {
			emailJSON, _ := json.Marshal(cfg.Email)
			settings = append(settings, model.SystemSetting{
				Category: string(model.SettingCategoryNotifications),
				Key:      "email",
				Value:    string(emailJSON),
			})
		}

		if cfg.Slack.WebhookURL != "" {
			slackJSON, _ := json.Marshal(cfg.Slack)
			settings = append(settings, model.SystemSetting{
				Category: string(model.SettingCategoryNotifications),
				Key:      "slack",
				Value:    string(slackJSON),
			})
		}

		if cfg.Feishu.WebhookURL != "" {
			feishuJSON, _ := json.Marshal(cfg.Feishu)
			settings = append(settings, model.SystemSetting{
				Category: string(model.SettingCategoryNotifications),
				Key:      "feishu",
				Value:    string(feishuJSON),
			})
		}

		if len(settings) > 0 {
			_ = mockStore.Settings().BatchUpsert(settings)
		}
	}

	// Reset notification manager with mock store
	notification.ResetForTesting(mockStore)

	return mockStore
}

// TestNotificationHandler_TestNotification_ReviewFailed tests sending review_failed notification
func TestNotificationHandler_TestNotification_ReviewFailed(t *testing.T) {
	// Initialize notification manager for testing with mock store
	cfg := &config.NotificationConfig{
		Channel: config.NotificationChannelWebhook,
		Events: []config.NotificationEvent{
			config.NotificationEventReviewFailed,
		},
		Webhook: config.WebhookNotificationConfig{
			URL: "http://test-webhook.example.com",
		},
	}
	setupNotificationTest(t, cfg)

	router := SetupTestRouter()
	handler := NewNotificationHandler()
	router.POST("/api/v1/notifications/test", handler.TestNotification)

	reqBody := map[string]interface{}{
		"event_type": "review_failed",
	}
	req := CreateTestRequest("POST", "/api/v1/notifications/test", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 200 (even if notification fails, it returns 200 with success=false)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response TestNotificationResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Response should have success and message fields
	if response.Message == "" {
		t.Error("Response should contain message")
	}
}

// TestNotificationHandler_TestNotification_ReviewCompleted tests sending review_completed notification
func TestNotificationHandler_TestNotification_ReviewCompleted(t *testing.T) {
	cfg := &config.NotificationConfig{
		Channel: config.NotificationChannelWebhook,
		Events: []config.NotificationEvent{
			config.NotificationEventReviewCompleted,
		},
		Webhook: config.WebhookNotificationConfig{
			URL: "http://test-webhook.example.com",
		},
	}
	setupNotificationTest(t, cfg)

	router := SetupTestRouter()
	handler := NewNotificationHandler()
	router.POST("/api/v1/notifications/test", handler.TestNotification)

	reqBody := map[string]interface{}{
		"event_type": "review_completed",
	}
	req := CreateTestRequest("POST", "/api/v1/notifications/test", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestNotificationHandler_TestNotification_ReportFailed tests sending report_failed notification
func TestNotificationHandler_TestNotification_ReportFailed(t *testing.T) {
	cfg := &config.NotificationConfig{
		Channel: config.NotificationChannelWebhook,
		Events: []config.NotificationEvent{
			config.NotificationEventReportFailed,
		},
		Webhook: config.WebhookNotificationConfig{
			URL: "http://test-webhook.example.com",
		},
	}
	setupNotificationTest(t, cfg)

	router := SetupTestRouter()
	handler := NewNotificationHandler()
	router.POST("/api/v1/notifications/test", handler.TestNotification)

	reqBody := map[string]interface{}{
		"event_type": "report_failed",
	}
	req := CreateTestRequest("POST", "/api/v1/notifications/test", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestNotificationHandler_TestNotification_ReportCompleted tests sending report_completed notification
func TestNotificationHandler_TestNotification_ReportCompleted(t *testing.T) {
	cfg := &config.NotificationConfig{
		Channel: config.NotificationChannelWebhook,
		Events: []config.NotificationEvent{
			config.NotificationEventReportCompleted,
		},
		Webhook: config.WebhookNotificationConfig{
			URL: "http://test-webhook.example.com",
		},
	}
	setupNotificationTest(t, cfg)

	router := SetupTestRouter()
	handler := NewNotificationHandler()
	router.POST("/api/v1/notifications/test", handler.TestNotification)

	reqBody := map[string]interface{}{
		"event_type": "report_completed",
	}
	req := CreateTestRequest("POST", "/api/v1/notifications/test", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestNotificationHandler_TestNotification_InvalidEventType tests invalid event type
func TestNotificationHandler_TestNotification_InvalidEventType(t *testing.T) {
	router := SetupTestRouter()
	handler := NewNotificationHandler()
	router.POST("/api/v1/notifications/test", handler.TestNotification)

	reqBody := map[string]interface{}{
		"event_type": "invalid_event",
	}
	req := CreateTestRequest("POST", "/api/v1/notifications/test", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestNotificationHandler_TestNotification_NotificationsDisabled tests when notifications are disabled
func TestNotificationHandler_TestNotification_NotificationsDisabled(t *testing.T) {
	// Initialize with disabled notifications (channel is "none")
	cfg := &config.NotificationConfig{
		Channel: config.NotificationChannelNone,
	}
	setupNotificationTest(t, cfg)

	router := SetupTestRouter()
	handler := NewNotificationHandler()
	router.POST("/api/v1/notifications/test", handler.TestNotification)

	reqBody := map[string]interface{}{
		"event_type": "review_failed",
	}
	req := CreateTestRequest("POST", "/api/v1/notifications/test", reqBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response TestNotificationResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Success {
		t.Error("Expected success=false when notifications are disabled")
	}
}

// TestNotificationHandler_TestNotification_InvalidRequest tests invalid request body
func TestNotificationHandler_TestNotification_InvalidRequest(t *testing.T) {
	router := SetupTestRouter()
	handler := NewNotificationHandler()
	router.POST("/api/v1/notifications/test", handler.TestNotification)

	// Test with empty body
	req := CreateTestRequest("POST", "/api/v1/notifications/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)

	// Test with invalid JSON
	req, _ = http.NewRequest("POST", "/api/v1/notifications/test", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	AssertErrorResponse(t, w, http.StatusBadRequest)
}

// TestNotificationHandler_GetNotificationStatus tests getting notification status
func TestNotificationHandler_GetNotificationStatus(t *testing.T) {
	cfg := &config.NotificationConfig{
		Channel: config.NotificationChannelWebhook,
		Events: []config.NotificationEvent{
			config.NotificationEventReviewFailed,
		},
		Webhook: config.WebhookNotificationConfig{
			URL: "http://test-webhook.example.com",
		},
	}
	setupNotificationTest(t, cfg)

	router := SetupTestRouter()
	handler := NewNotificationHandler()
	router.GET("/api/v1/notifications/status", handler.GetNotificationStatus)

	req := CreateTestRequest("GET", "/api/v1/notifications/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if _, exists := response["enabled"]; !exists {
		t.Error("Response should contain 'enabled' field")
	}
	if _, exists := response["channel"]; !exists {
		t.Error("Response should contain 'channel' field")
	}
}

// TestNotificationHandler_GetNotificationStatus_NoManager tests when notifications are disabled
func TestNotificationHandler_GetNotificationStatus_NoManager(t *testing.T) {
	// Initialize with disabled notifications using ResetForTesting
	cfg := &config.NotificationConfig{
		Channel: config.NotificationChannelNone,
	}
	setupNotificationTest(t, cfg)

	router := SetupTestRouter()
	handler := NewNotificationHandler()
	router.GET("/api/v1/notifications/status", handler.GetNotificationStatus)

	req := CreateTestRequest("GET", "/api/v1/notifications/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check if enabled is false (channel is NotificationChannelNone)
	enabled, ok := response["enabled"].(bool)
	if !ok {
		t.Error("Response should contain 'enabled' field of type bool")
		return
	}

	// When channel is "none", enabled should be false
	if enabled {
		t.Error("Expected enabled=false when channel is NotificationChannelNone")
	}
}
