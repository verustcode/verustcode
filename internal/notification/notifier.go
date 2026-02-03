// Package notification provides notification services for task failure alerts.
// It supports multiple notification channels including Webhook, Email, Slack, and Feishu.
package notification

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/logger"
)

// EventType represents the type of notification event
type EventType string

const (
	// EventReviewFailed is triggered when a review task fails
	EventReviewFailed EventType = "review_failed"
	// EventReviewCompleted is triggered when a review task completes successfully
	EventReviewCompleted EventType = "review_completed"
	// EventReportFailed is triggered when a report generation fails
	EventReportFailed EventType = "report_failed"
	// EventReportCompleted is triggered when a report generation completes successfully
	EventReportCompleted EventType = "report_completed"
)

// Event represents a notification event with context information
type Event struct {
	// Type is the event type (review_failed, report_failed)
	Type EventType `json:"type"`
	// TaskID is the unique identifier of the failed task
	TaskID string `json:"task_id"`
	// TaskType is either "review" or "report"
	TaskType string `json:"task_type"`
	// RepoURL is the repository URL associated with the task
	RepoURL string `json:"repo_url"`
	// ErrorMessage is the error that caused the failure
	ErrorMessage string `json:"error_message"`
	// Timestamp is when the failure occurred
	Timestamp time.Time `json:"timestamp"`
	// Extra contains additional context-specific information
	Extra map[string]interface{} `json:"extra,omitempty"`
}

// Notifier is the interface that all notification channels must implement
type Notifier interface {
	// Name returns the name of the notifier (e.g., "webhook", "email")
	Name() string
	// Send sends a notification for the given event
	// Returns an error if the notification fails to send
	Send(ctx context.Context, event *Event) error
}

// Manager manages notification channels and dispatches events.
// Manager now uses ConfigProvider for real-time database access to notification configurations.
type Manager struct {
	mu             sync.RWMutex
	configProvider config.ConfigProvider
	// cachedConfig is used only for notifier initialization state
	// The actual config is always fetched from DB when sending notifications
	cachedConfig *config.NotificationConfig
	notifier     Notifier
}

// globalManager is the singleton manager instance
var (
	globalManager *Manager
	managerOnce   sync.Once
)

// NewManager creates a new notification manager.
// It now requires a store.Store for real-time database access to notification configurations.
func NewManager(s store.Store) *Manager {
	m := &Manager{
		configProvider: config.NewSettingsService(s),
	}

	// Initialize the notifier based on current database configuration
	m.refreshNotifier()

	return m
}

// Init initializes the global notification manager.
// It now requires a store.Store for real-time database access to notification configurations.
func Init(s store.Store) {
	managerOnce.Do(func() {
		globalManager = NewManager(s)
		if globalManager.notifier != nil && globalManager.cachedConfig != nil {
			logger.Info("Notification manager initialized",
				zap.String("channel", string(globalManager.cachedConfig.Channel)),
				zap.Int("events_count", len(globalManager.cachedConfig.Events)),
			)
		} else {
			logger.Info("Notification manager initialized (disabled or no config in DB)")
		}
	})
}

// GetManager returns the global notification manager
func GetManager() *Manager {
	return globalManager
}

// ResetForTesting resets the global notification manager for testing purposes.
// This should only be used in tests to allow re-initialization with different configurations.
func ResetForTesting(s store.Store) {
	globalManager = NewManager(s)
}

// refreshNotifier fetches the latest configuration from database and initializes the notifier.
// Must be called with write lock held.
func (m *Manager) refreshNotifier() {
	// Fetch latest config from database
	cfg, err := m.configProvider.GetNotificationConfig()
	if err != nil {
		logger.Error("Failed to get notification config from database",
			zap.Error(err),
		)
		return
	}

	m.cachedConfig = cfg
	m.notifier = nil

	if cfg == nil || !cfg.IsEnabled() {
		return
	}

	m.initNotifierFromConfig(cfg)
}

// initNotifierFromConfig initializes the appropriate notifier based on the given configuration.
func (m *Manager) initNotifierFromConfig(cfg *config.NotificationConfig) {
	switch cfg.Channel {
	case config.NotificationChannelWebhook:
		m.notifier = NewWebhookNotifier(&cfg.Webhook)
	case config.NotificationChannelEmail:
		m.notifier = NewEmailNotifier(&cfg.Email)
	case config.NotificationChannelSlack:
		m.notifier = NewSlackNotifier(&cfg.Slack)
	case config.NotificationChannelFeishu:
		m.notifier = NewFeishuNotifier(&cfg.Feishu)
	default:
		logger.Warn("Unknown notification channel",
			zap.String("channel", string(cfg.Channel)),
		)
	}
}

// Notify sends a notification for the given event.
// It fetches the latest configuration from database before sending.
// It checks if the event type is enabled before sending.
func (m *Manager) Notify(ctx context.Context, event *Event) error {
	// Fetch the latest configuration from database (real-time access)
	cfg, err := m.configProvider.GetNotificationConfig()
	if err != nil {
		logger.Error("Failed to get notification config from database",
			zap.Error(err),
		)
		return fmt.Errorf("failed to get notification config from database: %w", err)
	}

	// Check if notifications are enabled
	if cfg == nil || !cfg.IsEnabled() {
		logger.Debug("Notifications disabled, skipping",
			zap.String("event_type", string(event.Type)),
		)
		return nil
	}

	// Check if this event type is enabled
	eventType := config.NotificationEvent(event.Type)
	if !cfg.HasEvent(eventType) {
		logger.Debug("Event type not in notification list, skipping",
			zap.String("event_type", string(event.Type)),
		)
		return nil
	}

	// Create notifier based on current config (may have changed since initialization)
	m.mu.Lock()
	// Check if config has changed and we need to reinitialize
	if m.cachedConfig == nil || m.cachedConfig.Channel != cfg.Channel {
		m.cachedConfig = cfg
		m.initNotifierFromConfig(cfg)
	}
	notifier := m.notifier
	m.mu.Unlock()

	// Check if we have a notifier
	if notifier == nil {
		logger.Warn("No notifier configured")
		return fmt.Errorf("no notifier configured for channel: %s", cfg.Channel)
	}

	// Send the notification
	logger.Info("Sending notification",
		zap.String("channel", notifier.Name()),
		zap.String("event_type", string(event.Type)),
		zap.String("task_id", event.TaskID),
	)

	if err := notifier.Send(ctx, event); err != nil {
		logger.Error("Failed to send notification",
			zap.String("channel", notifier.Name()),
			zap.String("event_type", string(event.Type)),
			zap.String("task_id", event.TaskID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to send notification via %s: %w", notifier.Name(), err)
	}

	logger.Info("Notification sent successfully",
		zap.String("channel", notifier.Name()),
		zap.String("task_id", event.TaskID),
	)

	return nil
}

// NotifyReviewFailed is a convenience function to notify about review failures
func NotifyReviewFailed(ctx context.Context, reviewID, repoURL, errorMsg string, extra map[string]interface{}) error {
	if globalManager == nil {
		return nil
	}

	event := &Event{
		Type:         EventReviewFailed,
		TaskID:       reviewID,
		TaskType:     "review",
		RepoURL:      repoURL,
		ErrorMessage: errorMsg,
		Timestamp:    time.Now(),
		Extra:        extra,
	}

	return globalManager.Notify(ctx, event)
}

// NotifyReviewCompleted is a convenience function to notify about review completion
func NotifyReviewCompleted(ctx context.Context, reviewID, repoURL string, extra map[string]interface{}) error {
	if globalManager == nil {
		return nil
	}

	event := &Event{
		Type:      EventReviewCompleted,
		TaskID:    reviewID,
		TaskType:  "review",
		RepoURL:   repoURL,
		Timestamp: time.Now(),
		Extra:     extra,
	}

	return globalManager.Notify(ctx, event)
}

// NotifyReportFailed is a convenience function to notify about report failures
func NotifyReportFailed(ctx context.Context, reportID, repoURL, errorMsg string, extra map[string]interface{}) error {
	if globalManager == nil {
		return nil
	}

	event := &Event{
		Type:         EventReportFailed,
		TaskID:       reportID,
		TaskType:     "report",
		RepoURL:      repoURL,
		ErrorMessage: errorMsg,
		Timestamp:    time.Now(),
		Extra:        extra,
	}

	return globalManager.Notify(ctx, event)
}

// NotifyReportCompleted is a convenience function to notify about report completion
func NotifyReportCompleted(ctx context.Context, reportID, repoURL string, extra map[string]interface{}) error {
	if globalManager == nil {
		return nil
	}

	event := &Event{
		Type:      EventReportCompleted,
		TaskID:    reportID,
		TaskType:  "report",
		RepoURL:   repoURL,
		Timestamp: time.Now(),
		Extra:     extra,
	}

	return globalManager.Notify(ctx, event)
}

// UpdateConfig is deprecated - configuration is now read from database in real-time.
// This method is kept for backwards compatibility but now refreshes from database.
func (m *Manager) UpdateConfig(_ *config.NotificationConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Refresh from database instead of using the passed config
	m.refreshNotifier()

	if m.cachedConfig != nil && m.cachedConfig.IsEnabled() {
		logger.Info("Notification configuration refreshed from database",
			zap.String("channel", string(m.cachedConfig.Channel)),
		)
	} else {
		logger.Info("Notifications disabled")
	}
}

// Refresh refreshes the notification configuration from the database.
// This can be called when configuration changes at runtime.
func (m *Manager) Refresh() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refreshNotifier()
}

// IsEnabled returns true if notifications are enabled.
// It fetches the latest configuration from the database.
func (m *Manager) IsEnabled() bool {
	cfg, err := m.configProvider.GetNotificationConfig()
	if err != nil {
		logger.Error("Failed to get notification config from database",
			zap.Error(err),
		)
		return false
	}
	return cfg != nil && cfg.IsEnabled()
}

// GetChannel returns the current notification channel.
// It fetches the latest configuration from the database.
func (m *Manager) GetChannel() config.NotificationChannel {
	cfg, err := m.configProvider.GetNotificationConfig()
	if err != nil {
		logger.Error("Failed to get notification config from database",
			zap.Error(err),
		)
		return config.NotificationChannelNone
	}
	if cfg == nil {
		return config.NotificationChannelNone
	}
	return cfg.Channel
}
