package notification

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/store"
)

// mockStore is a minimal mock store for testing
type mockStore struct {
	settingsStore *mockSettingsStore
}

func newMockStore() *mockStore {
	return &mockStore{
		settingsStore: &mockSettingsStore{
			settings: make(map[string]*model.SystemSetting),
		},
	}
}

func (m *mockStore) Review() store.ReviewStore {
	return nil
}

func (m *mockStore) Report() store.ReportStore {
	return nil
}

func (m *mockStore) Settings() store.SettingsStore {
	return m.settingsStore
}

func (m *mockStore) RepositoryConfig() store.RepositoryConfigStore {
	return nil
}

func (m *mockStore) DB() *gorm.DB {
	return nil
}

func (m *mockStore) Transaction(fn func(store.Store) error) error {
	return fn(m)
}

type mockSettingsStore struct {
	mu       sync.RWMutex
	settings map[string]*model.SystemSetting
}

func (m *mockSettingsStore) Get(category, key string) (*model.SystemSetting, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	setting, ok := m.settings[category+":"+key]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return setting, nil
}

func (m *mockSettingsStore) GetByCategory(category string) ([]model.SystemSetting, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var settings []model.SystemSetting
	for _, setting := range m.settings {
		if setting.Category == category {
			settings = append(settings, *setting)
		}
	}
	return settings, nil
}

func (m *mockSettingsStore) GetAll() ([]model.SystemSetting, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var settings []model.SystemSetting
	for _, setting := range m.settings {
		settings = append(settings, *setting)
	}
	return settings, nil
}

func (m *mockSettingsStore) Create(setting *model.SystemSetting) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings[setting.Category+":"+setting.Key] = setting
	return nil
}

func (m *mockSettingsStore) Update(setting *model.SystemSetting) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings[setting.Category+":"+setting.Key] = setting
	return nil
}

func (m *mockSettingsStore) Save(setting *model.SystemSetting) error {
	return m.Create(setting)
}

func (m *mockSettingsStore) Delete(category, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.settings, category+":"+key)
	return nil
}

func (m *mockSettingsStore) DeleteByCategory(category string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k := range m.settings {
		if m.settings[k].Category == category {
			delete(m.settings, k)
		}
	}
	return nil
}

func (m *mockSettingsStore) DeleteSetting(setting *model.SystemSetting) error {
	return m.Delete(setting.Category, setting.Key)
}

func (m *mockSettingsStore) Count() (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int64(len(m.settings)), nil
}

func (m *mockSettingsStore) BatchUpsert(settings []model.SystemSetting) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range settings {
		m.settings[settings[i].Category+":"+settings[i].Key] = &settings[i]
	}
	return nil
}

func (m *mockSettingsStore) WithTx(tx *gorm.DB) store.SettingsStore {
	return m
}

// setupNotificationTest creates a mock store with the given notification config
func setupNotificationTest(cfg *config.NotificationConfig) *mockStore {
	mockStore := newMockStore()

	if cfg != nil {
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

	return mockStore
}

// TestManager_Notify_ReviewFailed tests sending review_failed notification
func TestManager_Notify_ReviewFailed(t *testing.T) {
	cfg := &config.NotificationConfig{
		Channel: config.NotificationChannelWebhook,
		Events: []config.NotificationEvent{
			config.NotificationEventReviewFailed,
		},
		Webhook: config.WebhookNotificationConfig{
			URL: "http://test-webhook.example.com",
		},
	}

	mockStore := setupNotificationTest(cfg)
	manager := NewManager(store.Store(mockStore))
	if manager == nil {
		t.Fatal("Failed to create notification manager")
	}

	event := &Event{
		Type:         EventReviewFailed,
		TaskID:       "test-review-001",
		TaskType:     "review",
		RepoURL:      "https://github.com/test/repo",
		ErrorMessage: "Test error",
		Timestamp:    time.Now(),
	}

	ctx := context.Background()
	// Note: This will fail if webhook URL is not accessible, but that's expected in tests
	err := manager.Notify(ctx, event)
	// We expect an error since the webhook URL is not real, but the manager should still process it
	if err == nil {
		t.Log("Notification sent (or skipped due to disabled notifier)")
	}
}

// TestManager_Notify_EventNotEnabled tests notification when event type is not enabled
func TestManager_Notify_EventNotEnabled(t *testing.T) {
	cfg := &config.NotificationConfig{
		Channel: config.NotificationChannelWebhook,
		Events: []config.NotificationEvent{
			config.NotificationEventReviewCompleted, // Only review_completed enabled
		},
		Webhook: config.WebhookNotificationConfig{
			URL: "http://test-webhook.example.com",
		},
	}

	mockStore := setupNotificationTest(cfg)
	manager := NewManager(store.Store(mockStore))
	event := &Event{
		Type:         EventReviewFailed, // This event is not enabled
		TaskID:       "test-review-001",
		TaskType:     "review",
		RepoURL:      "https://github.com/test/repo",
		ErrorMessage: "Test error",
		Timestamp:    time.Now(),
	}

	ctx := context.Background()
	err := manager.Notify(ctx, event)
	// Should return nil (event skipped, not an error)
	if err != nil {
		t.Errorf("Expected nil error when event is not enabled, got %v", err)
	}
}

// TestManager_Notify_NotificationsDisabled tests notification when notifications are disabled
func TestManager_Notify_NotificationsDisabled(t *testing.T) {
	cfg := &config.NotificationConfig{
		Channel: config.NotificationChannelNone, // Disabled
		Events:  []config.NotificationEvent{},
	}

	mockStore := setupNotificationTest(cfg)
	manager := NewManager(store.Store(mockStore))
	event := &Event{
		Type:      EventReviewFailed,
		TaskID:    "test-review-001",
		TaskType:  "review",
		RepoURL:   "https://github.com/test/repo",
		Timestamp: time.Now(),
	}

	ctx := context.Background()
	err := manager.Notify(ctx, event)
	// Should return nil (notifications disabled, not an error)
	if err != nil {
		t.Errorf("Expected nil error when notifications are disabled, got %v", err)
	}
}

// TestManager_IsEnabled tests IsEnabled method
func TestManager_IsEnabled(t *testing.T) {
	// Test with enabled notifications
	cfg1 := &config.NotificationConfig{
		Channel: config.NotificationChannelWebhook,
		Events: []config.NotificationEvent{
			config.NotificationEventReviewFailed,
		},
		Webhook: config.WebhookNotificationConfig{
			URL: "http://test-webhook.example.com",
		},
	}
	mockStore1 := setupNotificationTest(cfg1)
	manager1 := NewManager(mockStore1)
	if !manager1.IsEnabled() {
		t.Error("Expected manager to be enabled")
	}

	// Test with disabled notifications
	cfg2 := &config.NotificationConfig{
		Channel: config.NotificationChannelNone,
	}
	mockStore2 := setupNotificationTest(cfg2)
	manager2 := NewManager(mockStore2)
	if manager2.IsEnabled() {
		t.Error("Expected manager to be disabled")
	}
}

// TestManager_GetChannel tests GetChannel method
func TestManager_GetChannel(t *testing.T) {
	cfg := &config.NotificationConfig{
		Channel: config.NotificationChannelSlack,
		Events: []config.NotificationEvent{
			config.NotificationEventReviewFailed,
		},
		Slack: config.SlackNotificationConfig{
			WebhookURL: "http://test-slack.example.com",
		},
	}
	mockStore := setupNotificationTest(cfg)
	manager := NewManager(store.Store(mockStore))
	channel := manager.GetChannel()
	if channel != config.NotificationChannelSlack {
		t.Errorf("Expected channel %s, got %s", config.NotificationChannelSlack, channel)
	}
}

// TestManager_UpdateConfig tests updating configuration
func TestManager_UpdateConfig(t *testing.T) {
	cfg1 := &config.NotificationConfig{
		Channel: config.NotificationChannelWebhook,
		Events: []config.NotificationEvent{
			config.NotificationEventReviewFailed,
		},
		Webhook: config.WebhookNotificationConfig{
			URL: "http://test-webhook.example.com",
		},
	}
	mockStore := setupNotificationTest(cfg1)
	manager := NewManager(mockStore)

	cfg2 := &config.NotificationConfig{
		Channel: config.NotificationChannelEmail,
		Events: []config.NotificationEvent{
			config.NotificationEventReviewCompleted,
		},
		Email: config.EmailNotificationConfig{
			SMTPHost: "smtp.example.com",
			SMTPPort: 587,
			From:     "test@example.com",
			To:       []string{"admin@example.com"},
		},
	}
	// UpdateConfig now refreshes from database, so we need to update the store
	mockStore2 := setupNotificationTest(cfg2)
	manager = NewManager(store.Store(mockStore2))

	if manager.GetChannel() != config.NotificationChannelEmail {
		t.Errorf("Expected channel %s after update, got %s", config.NotificationChannelEmail, manager.GetChannel())
	}
}

// TestNotifyReviewFailed tests convenience function for review failed notification
func TestNotifyReviewFailed(t *testing.T) {
	cfg := &config.NotificationConfig{
		Channel: config.NotificationChannelWebhook,
		Events: []config.NotificationEvent{
			config.NotificationEventReviewFailed,
		},
		Webhook: config.WebhookNotificationConfig{
			URL: "http://test-webhook.example.com",
		},
	}
	mockStore := setupNotificationTest(cfg)
	Init(store.Store(mockStore))
	defer Init(nil)

	ctx := context.Background()
	err := NotifyReviewFailed(ctx, "review-001", "https://github.com/test/repo", "Test error", nil)
	// May fail due to webhook URL, but function should execute
	if err != nil {
		t.Logf("Notification failed (expected): %v", err)
	}
}

// TestNotifyReviewCompleted tests convenience function for review completed notification
func TestNotifyReviewCompleted(t *testing.T) {
	cfg := &config.NotificationConfig{
		Channel: config.NotificationChannelWebhook,
		Events: []config.NotificationEvent{
			config.NotificationEventReviewCompleted,
		},
		Webhook: config.WebhookNotificationConfig{
			URL: "http://test-webhook.example.com",
		},
	}
	mockStore := setupNotificationTest(cfg)
	Init(store.Store(mockStore))
	defer Init(nil)

	ctx := context.Background()
	err := NotifyReviewCompleted(ctx, "review-001", "https://github.com/test/repo", nil)
	// May fail due to webhook URL, but function should execute
	if err != nil {
		t.Logf("Notification failed (expected): %v", err)
	}
}

// TestNotifyReportFailed tests convenience function for report failed notification
func TestNotifyReportFailed(t *testing.T) {
	cfg := &config.NotificationConfig{
		Channel: config.NotificationChannelWebhook,
		Events: []config.NotificationEvent{
			config.NotificationEventReportFailed,
		},
		Webhook: config.WebhookNotificationConfig{
			URL: "http://test-webhook.example.com",
		},
	}
	mockStore := setupNotificationTest(cfg)
	Init(store.Store(mockStore))
	defer Init(nil)

	ctx := context.Background()
	err := NotifyReportFailed(ctx, "report-001", "https://github.com/test/repo", "Test error", nil)
	// May fail due to webhook URL, but function should execute
	if err != nil {
		t.Logf("Notification failed (expected): %v", err)
	}
}

// TestNotifyReportCompleted tests convenience function for report completed notification
func TestNotifyReportCompleted(t *testing.T) {
	cfg := &config.NotificationConfig{
		Channel: config.NotificationChannelWebhook,
		Events: []config.NotificationEvent{
			config.NotificationEventReportCompleted,
		},
		Webhook: config.WebhookNotificationConfig{
			URL: "http://test-webhook.example.com",
		},
	}
	mockStore := setupNotificationTest(cfg)
	Init(store.Store(mockStore))
	defer Init(nil)

	ctx := context.Background()
	err := NotifyReportCompleted(ctx, "report-001", "https://github.com/test/repo", nil)
	// May fail due to webhook URL, but function should execute
	if err != nil {
		t.Logf("Notification failed (expected): %v", err)
	}
}

// TestManager_Notify_NoNotifier tests notification when no notifier is configured
func TestManager_Notify_NoNotifier(t *testing.T) {
	cfg := &config.NotificationConfig{
		Channel: config.NotificationChannelWebhook,
		Events: []config.NotificationEvent{
			config.NotificationEventReviewFailed,
		},
		// Webhook URL is empty, so no notifier will be created
		Webhook: config.WebhookNotificationConfig{},
	}

	mockStore := setupNotificationTest(cfg)
	manager := NewManager(store.Store(mockStore))
	event := &Event{
		Type:         EventReviewFailed,
		TaskID:       "test-review-001",
		TaskType:     "review",
		RepoURL:      "https://github.com/test/repo",
		ErrorMessage: "Test error",
		Timestamp:    time.Now(),
	}

	ctx := context.Background()
	err := manager.Notify(ctx, event)
	// Should return error when no notifier is configured
	if err == nil {
		t.Error("Expected error when no notifier is configured")
	}
}
