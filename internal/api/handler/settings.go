// Package handler provides HTTP handlers for the API.
// This file handles settings management API endpoints.
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/agent/base"
	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/notification"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/errors"
	"github.com/verustcode/verustcode/pkg/logger"

	// Import git providers to register them
	_ "github.com/verustcode/verustcode/internal/git/gitea"
	_ "github.com/verustcode/verustcode/internal/git/github"
	_ "github.com/verustcode/verustcode/internal/git/gitlab"
)

// sensitiveKeyPatterns defines patterns for sensitive field names that should be masked
var sensitiveKeyPatterns = []string{
	"token",
	"api_key",
	"secret",
	"password",
	"private_key",
	"webhook_secret",
	"jwt_secret",
}

// maskedPlaceholder is the placeholder used in masked values
const maskedPlaceholder = "****"

// isSensitiveKey checks if a key name indicates a sensitive field
func isSensitiveKey(key string) bool {
	keyLower := strings.ToLower(key)
	for _, pattern := range sensitiveKeyPatterns {
		if strings.Contains(keyLower, pattern) {
			return true
		}
	}
	return false
}

// maskSensitiveValue masks a sensitive string value, keeping first 4 and last 4 characters
// Returns the masked value or empty string if input is too short
func maskSensitiveValue(value string) string {
	if value == "" {
		return ""
	}
	// For very short values, just return ****
	if len(value) <= 8 {
		return maskedPlaceholder
	}
	// Show first 4 and last 4 characters
	return value[:4] + maskedPlaceholder + value[len(value)-4:]
}

// isMaskedValue checks if a value appears to be a masked value (contains ****)
func isMaskedValue(value string) bool {
	return strings.Contains(value, maskedPlaceholder)
}

// maskSettingsValue recursively masks sensitive values in a settings structure
func maskSettingsValue(key string, value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		if isSensitiveKey(key) && v != "" {
			return maskSensitiveValue(v)
		}
		return v
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, val := range v {
			result[k] = maskSettingsValue(k, val)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			// For arrays of objects (like providers), process each item
			if obj, ok := item.(map[string]interface{}); ok {
				masked := make(map[string]interface{})
				for k, val := range obj {
					masked[k] = maskSettingsValue(k, val)
				}
				result[i] = masked
			} else {
				result[i] = item
			}
		}
		return result
	default:
		return v
	}
}

// SettingsHandler handles settings-related API requests
type SettingsHandler struct {
	service *config.SettingsService
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(s store.Store) *SettingsHandler {
	return &SettingsHandler{
		service: config.NewSettingsService(s),
	}
}

// GetAllSettings returns all settings grouped by category
// GET /api/admin/settings
// Sensitive values (tokens, secrets, api_keys) are masked to show only first 4 and last 4 characters
func (h *SettingsHandler) GetAllSettings(c *gin.Context) {
	settings, err := h.service.GetAll()
	if err != nil {
		logger.Error("Failed to get all settings", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeDBQuery,
			"message": "Failed to retrieve settings",
		})
		return
	}

	// Convert to response format with sensitive value masking
	result := make(map[string]map[string]interface{})
	for category, categorySettings := range settings {
		result[category] = make(map[string]interface{})
		for _, setting := range categorySettings {
			// Parse JSON value back to native type
			var value interface{}
			if err := unmarshalSettingValue(setting.Value, &value); err != nil {
				value = setting.Value
			}
			// Mask sensitive values
			result[category][setting.Key] = maskSettingsValue(setting.Key, value)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"settings": result,
	})
}

// GetSettingsByCategory returns settings for a specific category
// GET /api/admin/settings/:category
// Sensitive values (tokens, secrets, api_keys) are masked to show only first 4 and last 4 characters
func (h *SettingsHandler) GetSettingsByCategory(c *gin.Context) {
	category := c.Param("category")

	// Validate category
	if !isValidSettingCategory(category) {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid category",
		})
		return
	}

	settings, err := h.service.GetByCategory(category)
	if err != nil {
		logger.Error("Failed to get settings by category", zap.String("category", category), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeDBQuery,
			"message": "Failed to retrieve settings",
		})
		return
	}

	// Convert to response format with sensitive value masking
	result := make(map[string]interface{})
	for _, setting := range settings {
		var value interface{}
		if err := unmarshalSettingValue(setting.Value, &value); err != nil {
			value = setting.Value
		}
		// Mask sensitive values
		result[setting.Key] = maskSettingsValue(setting.Key, value)
	}

	c.JSON(http.StatusOK, gin.H{
		"category": category,
		"settings": result,
	})
}

// UpdateSettingsByCategoryRequest represents the request body for updating settings
type UpdateSettingsByCategoryRequest struct {
	Settings map[string]interface{} `json:"settings" binding:"required"`
}

// UpdateSettingsByCategory updates settings for a specific category
// PUT /api/admin/settings/:category
// When updating "git" category, triggers hot-reload of Git providers in all engines
func (h *SettingsHandler) UpdateSettingsByCategory(c *gin.Context) {
	category := c.Param("category")

	// Validate category
	if !isValidSettingCategory(category) {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid category",
		})
		return
	}

	var req UpdateSettingsByCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid request body",
		})
		return
	}

	// Get username from context (set by auth middleware)
	username, _ := c.Get("username")
	usernameStr, _ := username.(string)
	if usernameStr == "" {
		usernameStr = "admin"
	}

	// Update settings
	if err := h.service.SetCategory(category, req.Settings, usernameStr); err != nil {
		logger.Error("Failed to update settings", zap.String("category", category), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeDBQuery,
			"message": "Failed to update settings",
		})
		return
	}

	logger.Info("Settings updated", zap.String("category", category), zap.String("username", usernameStr))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Settings updated successfully",
	})
}

// ApplySettingsRequest represents the request body for applying all settings
type ApplySettingsRequest struct {
	Settings map[string]map[string]interface{} `json:"settings" binding:"required"`
}

// ApplySettings applies all settings at once
// POST /api/admin/settings/apply
// When git settings are included, triggers hot-reload of Git providers in all engines
func (h *SettingsHandler) ApplySettings(c *gin.Context) {
	var req ApplySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid request body",
		})
		return
	}

	// Get username from context
	username, _ := c.Get("username")
	usernameStr, _ := username.(string)
	if usernameStr == "" {
		usernameStr = "admin"
	}

	// Update each category
	for category, settings := range req.Settings {
		if !isValidSettingCategory(category) {
			continue
		}

		if err := h.service.SetCategory(category, settings, usernameStr); err != nil {
			logger.Error("Failed to apply settings", zap.String("category", category), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    errors.ErrCodeDBQuery,
				"message": "Failed to apply settings for category: " + category,
			})
			return
		}
	}

	logger.Info("All settings applied", zap.String("username", usernameStr))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "All settings applied successfully",
	})
}

// isValidSettingCategory validates if a category is valid
func isValidSettingCategory(category string) bool {
	validCategories := model.AllSettingCategories()
	for _, c := range validCategories {
		if string(c) == category {
			return true
		}
	}
	return false
}

// unmarshalSettingValue unmarshals a JSON string value
func unmarshalSettingValue(value string, target *interface{}) error {
	return json.Unmarshal([]byte(value), target)
}

// TestGitProviderRequest defines the request for testing git provider connection
type TestGitProviderRequest struct {
	Type               string `json:"type" binding:"required,oneof=github gitlab gitea"`
	URL                string `json:"url"`
	Token              string `json:"token" binding:"required"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify"`
}

// TestGitProvider tests a git provider connection by validating the token
// POST /api/admin/settings/git/test
func (h *SettingsHandler) TestGitProvider(c *gin.Context) {
	var req TestGitProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid request body: type and token are required",
		})
		return
	}

	token := req.Token
	url := req.URL
	insecureSkipVerify := req.InsecureSkipVerify

	// If token is masked, fetch real token from database
	if isMaskedValue(token) {
		realCreds, err := h.getRealProviderCredentials(req.Type)
		if err != nil {
			logger.Error("Failed to get real provider credentials from database",
				zap.String("type", req.Type),
				zap.Error(err),
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    errors.ErrCodeDBQuery,
				"message": "Failed to retrieve saved configuration",
			})
			return
		}
		if realCreds == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    errors.ErrCodeValidation,
				"message": "No saved configuration found for " + req.Type,
			})
			return
		}
		token = realCreds.Token
		// Also use real URL if the provided one is empty or masked
		if url == "" || isMaskedValue(url) {
			url = realCreds.URL
		}
		insecureSkipVerify = realCreds.InsecureSkipVerify
	}

	// For gitlab and gitea, URL is required for self-hosted instances
	// GitHub doesn't require URL as it defaults to github.com
	if req.Type != "github" && url == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "URL is required for " + req.Type,
		})
		return
	}

	// Create provider options
	opts := &provider.ProviderOptions{
		Token:              token,
		BaseURL:            url,
		InsecureSkipVerify: insecureSkipVerify,
	}

	// Create temporary provider instance
	p, err := provider.Create(req.Type, opts)
	if err != nil {
		logger.Error("Failed to create git provider for testing",
			zap.String("type", req.Type),
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Failed to create provider: " + err.Error(),
		})
		return
	}

	// Test the connection with a timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	if err := p.ValidateToken(ctx); err != nil {
		logger.Warn("Git provider token validation failed",
			zap.String("type", req.Type),
			zap.String("url", req.URL),
			zap.Error(err),
		)
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	logger.Info("Git provider connection test successful",
		zap.String("type", req.Type),
		zap.String("url", req.URL),
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Connection successful",
	})
}

// getRealProviderCredentials retrieves the real (non-masked) provider credentials from database
func (h *SettingsHandler) getRealProviderCredentials(providerType string) (*config.ProviderConfig, error) {
	providers, err := h.service.GetGitProviders()
	if err != nil {
		return nil, err
	}

	for _, p := range providers {
		if p.Type == providerType {
			return &p, nil
		}
	}

	return nil, nil
}

// TestAgentRequest represents the request body for testing an agent
type TestAgentRequest struct {
	Name           string   `json:"name" binding:"required"`
	CLIPath        string   `json:"cli_path"`
	APIKey         string   `json:"api_key"`
	DefaultModel   string   `json:"default_model"`
	FallbackModels []string `json:"fallback_models"`
	Timeout        int      `json:"timeout"`
}

// TestAgent tests an agent connection by running a simple prompt
// POST /api/admin/settings/agents/test
func (h *SettingsHandler) TestAgent(c *gin.Context) {
	var req TestAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	// Handle masked secrets
	if isMaskedValue(req.APIKey) {
		realConfig, err := h.getRealAgentConfig(req.Name)
		if err != nil {
			logger.Error("Failed to get real agent config", zap.String("agent", req.Name), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    errors.ErrCodeDBQuery,
				"message": "Failed to retrieve saved configuration",
			})
			return
		}
		if realConfig != nil {
			req.APIKey = realConfig.APIKey
		}
	}

	// Create a temporary store with the test configuration
	// We use the MockStore from testutil.go which is part of the handler package
	testStore := NewMockStore()

	// Create agent config object
	agentDetail := config.AgentDetail{
		CLIPath:        req.CLIPath,
		APIKey:         req.APIKey,
		DefaultModel:   req.DefaultModel,
		FallbackModels: req.FallbackModels,
		Timeout:        req.Timeout,
	}

	// Marshal to JSON string
	configBytes, err := json.Marshal(agentDetail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeInternal,
			"message": "Failed to marshal agent config",
		})
		return
	}

	// Save to mock store
	if err := testStore.Settings().Create(&model.SystemSetting{
		Category:  string(model.SettingCategoryAgents),
		Key:       req.Name,
		Value:     string(configBytes),
		ValueType: string(model.SettingValueTypeObject),
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeInternal,
			"message": "Failed to setup test environment",
		})
		return
	}

	// Create agent instance
	agentInstance, err := base.Create(req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Failed to create agent: " + err.Error(),
		})
		return
	}

	// Set the store with our temporary config
	agentInstance.SetStore(testStore)

	// Execute prompt
	// Use a timeout context
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	reviewReq := &base.ReviewRequest{
		RequestID: "test-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		RepoPath:  ".", // Dummy path, agent might need valid path but for simple prompt it might be ignored or handled
	}

	// Note: Some agents might require valid RepoPath even for prompt execution if they check for .git
	// But for simple "2+2=?" prompt, they should work or we might need to mock it.
	// Gemini/Cursor usually just send the prompt.

	result, err := agentInstance.ExecuteWithPrompt(ctx, reviewReq, "2+2=?")
	if err != nil {
		logger.Warn("Agent connection test failed",
			zap.String("agent", req.Name),
			zap.Error(err),
		)
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	logger.Info("Agent connection test successful",
		zap.String("agent", req.Name),
		zap.String("result", result.Text),
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Connection successful",
		"data":    result.Text,
	})
}

// getRealAgentConfig retrieves the real (non-masked) agent config from database
func (h *SettingsHandler) getRealAgentConfig(agentName string) (*config.AgentDetail, error) {
	agents, err := h.service.GetAgents()
	if err != nil {
		return nil, err
	}

	if agent, ok := agents[agentName]; ok {
		return &agent, nil
	}

	return nil, nil
}

// TestNotificationConfigRequest represents the request body for testing notification config
type TestNotificationConfigRequest struct {
	Channel string `json:"channel" binding:"required"`
	// Webhook config
	WebhookURL    string `json:"webhook_url,omitempty"`
	WebhookSecret string `json:"webhook_secret,omitempty"`
	// Email config
	SMTPHost     string   `json:"smtp_host,omitempty"`
	SMTPPort     int      `json:"smtp_port,omitempty"`
	SMTPUsername string   `json:"smtp_username,omitempty"`
	SMTPPassword string   `json:"smtp_password,omitempty"`
	EmailFrom    string   `json:"email_from,omitempty"`
	EmailTo      []string `json:"email_to,omitempty"`
	// Slack config
	SlackWebhookURL string `json:"slack_webhook_url,omitempty"`
	SlackChannel    string `json:"slack_channel,omitempty"`
	// Feishu config
	FeishuWebhookURL string `json:"feishu_webhook_url,omitempty"`
	FeishuSecret     string `json:"feishu_secret,omitempty"`
}

// TestNotificationConfig tests a notification connection by sending a test message
// POST /api/admin/settings/notifications/test
func (h *SettingsHandler) TestNotificationConfig(c *gin.Context) {
	var req TestNotificationConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	// Handle masked secrets - retrieve real values from database
	realConfig, err := h.service.GetNotificationConfig()
	if err != nil {
		logger.Error("Failed to get real notification config", zap.Error(err))
		// Continue with provided values, might fail if masked
	}

	// Build notification config based on channel type
	cfg := &config.NotificationConfig{
		Channel: config.NotificationChannel(req.Channel),
	}

	switch req.Channel {
	case "webhook":
		webhookURL := req.WebhookURL
		webhookSecret := req.WebhookSecret

		// Handle masked values
		if realConfig != nil {
			if isMaskedValue(webhookSecret) {
				webhookSecret = realConfig.Webhook.Secret
			}
		}

		if webhookURL == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    errors.ErrCodeValidation,
				"message": "Webhook URL is required",
			})
			return
		}

		cfg.Webhook = config.WebhookNotificationConfig{
			URL:    webhookURL,
			Secret: webhookSecret,
		}

	case "email":
		smtpHost := req.SMTPHost
		smtpPort := req.SMTPPort
		smtpUsername := req.SMTPUsername
		smtpPassword := req.SMTPPassword
		emailFrom := req.EmailFrom
		emailTo := req.EmailTo

		// Handle masked values
		if realConfig != nil {
			if isMaskedValue(smtpPassword) {
				smtpPassword = realConfig.Email.Password
			}
		}

		if smtpHost == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    errors.ErrCodeValidation,
				"message": "SMTP host is required",
			})
			return
		}

		if emailFrom == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    errors.ErrCodeValidation,
				"message": "Sender email (from) is required",
			})
			return
		}

		if smtpPort == 0 {
			smtpPort = 587 // Default SMTP port
		}

		cfg.Email = config.EmailNotificationConfig{
			SMTPHost: smtpHost,
			SMTPPort: smtpPort,
			Username: smtpUsername,
			Password: smtpPassword,
			From:     emailFrom,
			To:       emailTo,
		}

	case "slack":
		slackWebhookURL := req.SlackWebhookURL
		slackChannel := req.SlackChannel

		// Handle masked values
		if realConfig != nil {
			if isMaskedValue(slackWebhookURL) {
				slackWebhookURL = realConfig.Slack.WebhookURL
			}
		}

		if slackWebhookURL == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    errors.ErrCodeValidation,
				"message": "Slack webhook URL is required",
			})
			return
		}

		cfg.Slack = config.SlackNotificationConfig{
			WebhookURL: slackWebhookURL,
			Channel:    slackChannel,
		}

	case "feishu":
		feishuWebhookURL := req.FeishuWebhookURL
		feishuSecret := req.FeishuSecret

		// Handle masked values
		if realConfig != nil {
			if isMaskedValue(feishuWebhookURL) {
				feishuWebhookURL = realConfig.Feishu.WebhookURL
			}
			if isMaskedValue(feishuSecret) {
				feishuSecret = realConfig.Feishu.Secret
			}
		}

		if feishuWebhookURL == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    errors.ErrCodeValidation,
				"message": "Feishu webhook URL is required",
			})
			return
		}

		cfg.Feishu = config.FeishuNotificationConfig{
			WebhookURL: feishuWebhookURL,
			Secret:     feishuSecret,
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Unsupported notification channel: " + req.Channel,
		})
		return
	}

	// Use timeout context
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Send test notification
	err = sendTestNotification(ctx, cfg)
	if err != nil {
		logger.Warn("Notification test failed",
			zap.String("channel", req.Channel),
			zap.Error(err),
		)
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	logger.Info("Notification test successful",
		zap.String("channel", req.Channel),
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Test notification sent successfully",
	})
}

// sendTestNotification sends a test notification using the provided config
func sendTestNotification(ctx context.Context, cfg *config.NotificationConfig) error {
	// Create test event
	testEvent := &notification.Event{
		Type:         notification.EventReviewCompleted,
		TaskID:       "test-config-001",
		TaskType:     "config_test",
		RepoURL:      "https://example.com/test-repo",
		ErrorMessage: "",
		Timestamp:    time.Now(),
		Extra: map[string]interface{}{
			"test":    true,
			"channel": string(cfg.Channel),
			"message": "This is a test notification from VerustCode to verify your notification configuration.",
		},
	}

	// Create the appropriate notifier based on channel type
	var notifier notification.Notifier

	switch cfg.Channel {
	case config.NotificationChannelWebhook:
		notifier = notification.NewWebhookNotifier(&cfg.Webhook)
	case config.NotificationChannelEmail:
		notifier = notification.NewEmailNotifier(&cfg.Email)
	case config.NotificationChannelSlack:
		notifier = notification.NewSlackNotifier(&cfg.Slack)
	case config.NotificationChannelFeishu:
		notifier = notification.NewFeishuNotifier(&cfg.Feishu)
	default:
		return errors.New(errors.ErrCodeValidation, "unsupported notification channel: "+string(cfg.Channel))
	}

	// Send test notification
	return notifier.Send(ctx, testEvent)
}
