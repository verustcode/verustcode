// Package config provides configuration management for the application.
// This file handles merging bootstrap configuration with database settings.
package config

import (
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/logger"
)

// RuntimeConfig holds the complete merged configuration.
// It combines bootstrap configuration (file-based) with runtime settings (database-based).
type RuntimeConfig struct {
	*BootstrapConfig

	// Runtime settings from database
	Subtitle      string                 `yaml:"subtitle"`
	Git           GitConfig              `yaml:"git"`
	Agents        map[string]AgentDetail `yaml:"agents"`
	Review        ReviewConfig           `yaml:"review"`
	Report        ReportConfig           `yaml:"report"`
	Notifications NotificationConfig     `yaml:"notifications"`
}

// LoadRuntimeConfig loads bootstrap config and merges with database settings
func LoadRuntimeConfig(bootstrapPath string, s store.Store) (*RuntimeConfig, error) {
	// Load bootstrap configuration
	bootstrap, err := LoadBootstrap(bootstrapPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load bootstrap config: %w", err)
	}

	// Repair malformed settings in database before loading
	if s != nil {
		svc := NewSettingsService(s)
		if deleted, err := svc.RepairMalformedSettings(); err != nil {
			logger.Warn("Failed to repair malformed settings", zap.Error(err))
		} else if deleted > 0 {
			logger.Warn("Deleted malformed settings from database",
				zap.Int("deleted_count", deleted),
				zap.String("hint", "Reconfigure settings via the Settings page in admin interface"))
		}
	}

	// Create runtime config with bootstrap values
	cfg := &RuntimeConfig{
		BootstrapConfig: bootstrap,
		// Initialize with defaults
		Subtitle: "",
		Git: GitConfig{
			Providers: []ProviderConfig{},
		},
		// Initialize with default cursor agent to prevent nil pointer errors
		// Database settings will override these defaults if configured
		Agents: map[string]AgentDetail{
			"cursor": {
				CLIPath: defaultCLIPath,
				Timeout: defaultAgentTimeout,
			},
		},
		Review: ReviewConfig{
			Workspace:     defaultWorkspace,
			MaxConcurrent: defaultMaxConcurrent,
			RetentionDays: defaultRetentionDays,
			MaxRetries:    defaultMaxRetries,
			RetryDelay:    defaultRetryDelay,
		},
		Report: ReportConfig{
			Workspace:     defaultReportWorkspace,
			MaxConcurrent: defaultReportMaxConcurrent,
			MaxRetries:    defaultReportMaxRetries,
			RetryDelay:    defaultReportRetryDelay,
		},
		Notifications: NotificationConfig{
			Channel: NotificationChannelNone,
			Events:  []NotificationEvent{},
		},
	}

	// Load settings from database if available
	if s != nil {
		if err := cfg.loadFromDatabase(s); err != nil {
			logger.Warn("Failed to load settings from database, using defaults", zap.Error(err))
		}
	}

	return cfg, nil
}

// loadFromDatabase loads runtime settings from database and applies them
func (c *RuntimeConfig) loadFromDatabase(s store.Store) error {
	settings, err := s.Settings().GetAll()
	if err != nil {
		return err
	}

	// Group settings by category
	categorySettings := make(map[string]map[string]string)
	for _, s := range settings {
		if categorySettings[s.Category] == nil {
			categorySettings[s.Category] = make(map[string]string)
		}
		categorySettings[s.Category][s.Key] = s.Value
	}

	// Apply app settings
	if appSettings, ok := categorySettings[string(model.SettingCategoryApp)]; ok {
		if v, ok := appSettings["subtitle"]; ok {
			c.Subtitle = unquoteString(v)
		}
	}

	// Apply git settings
	if gitSettings, ok := categorySettings[string(model.SettingCategoryGit)]; ok {
		if v, ok := gitSettings["providers"]; ok {
			var providers []ProviderConfig
			if err := json.Unmarshal([]byte(v), &providers); err == nil {
				c.Git.Providers = providers
			}
		}
	}

	// Apply agents settings
	// Merge with existing defaults instead of replacing completely
	if agentSettings, ok := categorySettings[string(model.SettingCategoryAgents)]; ok {
		for key, v := range agentSettings {
			var dbAgent AgentDetail
			if err := json.Unmarshal([]byte(v), &dbAgent); err == nil {
				// Get existing agent config (may have defaults)
				existingAgent := c.Agents[key]
				
				// Merge: database values override defaults, but keep defaults for unset fields
				if dbAgent.CLIPath != "" {
					existingAgent.CLIPath = dbAgent.CLIPath
				}
				if dbAgent.APIKey != "" {
					existingAgent.APIKey = dbAgent.APIKey
				}
				if dbAgent.Timeout > 0 {
					existingAgent.Timeout = dbAgent.Timeout
				}
				if dbAgent.DefaultModel != "" {
					existingAgent.DefaultModel = dbAgent.DefaultModel
				}
				if len(dbAgent.FallbackModels) > 0 {
					existingAgent.FallbackModels = dbAgent.FallbackModels
				}
				
				c.Agents[key] = existingAgent
			}
		}
	}

	// Apply review settings
	if reviewSettings, ok := categorySettings[string(model.SettingCategoryReview)]; ok {
		applyJSONSettings(&c.Review, reviewSettings)
	}

	// Apply report settings
	if reportSettings, ok := categorySettings[string(model.SettingCategoryReport)]; ok {
		applyJSONSettings(&c.Report, reportSettings)
	}

	// Apply notification settings
	if notifSettings, ok := categorySettings[string(model.SettingCategoryNotifications)]; ok {
		applyNotificationSettings(&c.Notifications, notifSettings)
	}

	return nil
}

// ToConfig converts RuntimeConfig to the legacy Config format for compatibility
func (c *RuntimeConfig) ToConfig() *Config {
	return &Config{
		Subtitle: c.Subtitle,
		Server:   c.Server,
		Database: DatabaseConfig{},
		Auth: AuthConfig{
			JWTSecret:    c.Admin.JWTSecret,
			TokenExpiry:  c.Admin.TokenExpiration,
			RememberDays: 7,
		},
		Git:           c.Git,
		Agents:        c.Agents,
		Review:        c.Review,
		Report:        c.Report,
		Notifications: c.Notifications,
		Logging:       c.Logging,
		Telemetry:     c.Telemetry,
		Admin:         c.Admin,
	}
}

// SaveSettingsToDatabase saves runtime settings to database
func SaveSettingsToDatabase(cfg *Config, s store.Store, username string) error {
	svc := NewSettingsService(s)

	// Save app settings
	appSettings := map[string]interface{}{
		"subtitle": cfg.Subtitle,
	}
	if err := svc.SetCategory(string(model.SettingCategoryApp), appSettings, username); err != nil {
		return fmt.Errorf("failed to save app settings: %w", err)
	}

	// Save git settings
	gitSettings := map[string]interface{}{
		"providers": cfg.Git.Providers,
	}
	if err := svc.SetCategory(string(model.SettingCategoryGit), gitSettings, username); err != nil {
		return fmt.Errorf("failed to save git settings: %w", err)
	}

	// Save agents settings
	agentSettings := make(map[string]interface{})
	for name, agent := range cfg.Agents {
		agentSettings[name] = agent
	}
	if err := svc.SetCategory(string(model.SettingCategoryAgents), agentSettings, username); err != nil {
		return fmt.Errorf("failed to save agent settings: %w", err)
	}

	// Save review settings
	reviewSettings := map[string]interface{}{
		"workspace":       cfg.Review.Workspace,
		"max_concurrent":  cfg.Review.MaxConcurrent,
		"retention_days":  cfg.Review.RetentionDays,
		"max_retries":     cfg.Review.MaxRetries,
		"retry_delay":     cfg.Review.RetryDelay,
		"output_language": cfg.Review.OutputLanguage,
		"output_metadata": cfg.Review.OutputMetadata,
	}
	if err := svc.SetCategory(string(model.SettingCategoryReview), reviewSettings, username); err != nil {
		return fmt.Errorf("failed to save review settings: %w", err)
	}

	// Save report settings
	reportSettings := map[string]interface{}{
		"workspace":       cfg.Report.Workspace,
		"max_concurrent":  cfg.Report.MaxConcurrent,
		"max_retries":     cfg.Report.MaxRetries,
		"retry_delay":     cfg.Report.RetryDelay,
		"output_language": cfg.Report.OutputLanguage,
	}
	if err := svc.SetCategory(string(model.SettingCategoryReport), reportSettings, username); err != nil {
		return fmt.Errorf("failed to save report settings: %w", err)
	}

	// Save notification settings
	notifSettings := map[string]interface{}{
		"channel": cfg.Notifications.Channel,
		"events":  cfg.Notifications.Events,
		"webhook": cfg.Notifications.Webhook,
		"email":   cfg.Notifications.Email,
		"slack":   cfg.Notifications.Slack,
		"feishu":  cfg.Notifications.Feishu,
	}
	if err := svc.SetCategory(string(model.SettingCategoryNotifications), notifSettings, username); err != nil {
		return fmt.Errorf("failed to save notification settings: %w", err)
	}

	return nil
}

// applyJSONSettings applies JSON settings to a struct using reflection-like approach
func applyJSONSettings(target interface{}, settings map[string]string) {
	// Convert settings map to JSON and unmarshal into target
	// First build a proper JSON object
	jsonMap := make(map[string]interface{})
	for key, value := range settings {
		var v interface{}
		if err := json.Unmarshal([]byte(value), &v); err != nil {
			// If unmarshal fails, use string value
			jsonMap[key] = value
		} else {
			jsonMap[key] = v
		}
	}

	// Marshal back to JSON and unmarshal into target
	jsonBytes, err := json.Marshal(jsonMap)
	if err != nil {
		logger.Error("Failed to marshal settings", zap.Error(err))
		return
	}

	if err := json.Unmarshal(jsonBytes, target); err != nil {
		logger.Error("Failed to unmarshal settings into target", zap.Error(err))
	}
}

// applyNotificationSettings applies notification settings
func applyNotificationSettings(target *NotificationConfig, settings map[string]string) {
	if v, ok := settings["channel"]; ok {
		var channel NotificationChannel
		json.Unmarshal([]byte(v), &channel)
		target.Channel = channel
	}
	if v, ok := settings["events"]; ok {
		var events []NotificationEvent
		json.Unmarshal([]byte(v), &events)
		target.Events = events
	}
	if v, ok := settings["webhook"]; ok {
		json.Unmarshal([]byte(v), &target.Webhook)
	}
	if v, ok := settings["email"]; ok {
		json.Unmarshal([]byte(v), &target.Email)
	}
	if v, ok := settings["slack"]; ok {
		json.Unmarshal([]byte(v), &target.Slack)
	}
	if v, ok := settings["feishu"]; ok {
		json.Unmarshal([]byte(v), &target.Feishu)
	}
}

// unquoteString removes JSON quotes from a string value
func unquoteString(s string) string {
	var result string
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return s
	}
	return result
}

// GetDatabasePath returns the database path from bootstrap config or default
func GetDatabasePath(bootstrapPath string) string {
	if BootstrapExists(bootstrapPath) {
		cfg, err := LoadBootstrap(bootstrapPath)
		if err == nil && cfg.Database.Path != "" {
			return cfg.Database.Path
		}
	}
	return "./data/verustcode.db"
}
