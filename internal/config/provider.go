// Package config provides configuration management for the application.
package config

import (
	"github.com/verustcode/verustcode/internal/store"
)

// ConfigProvider interface defines methods for retrieving configuration from database.
// This enables runtime configuration reads instead of cached values.
type ConfigProvider interface {
	// GetReviewConfig retrieves review configuration from database
	GetReviewConfig() (*ReviewConfig, error)

	// GetReportConfig retrieves report configuration from database
	GetReportConfig() (*ReportConfig, error)

	// GetGitProviders retrieves Git providers configuration from database
	GetGitProviders() ([]ProviderConfig, error)

	// GetAgents retrieves all agent configurations from database
	GetAgents() (map[string]AgentDetail, error)

	// GetNotificationConfig retrieves notification configuration from database
	GetNotificationConfig() (*NotificationConfig, error)
}

// DBConfigProvider implements ConfigProvider using database access.
type DBConfigProvider struct {
	store store.Store
}

// NewDBConfigProvider creates a new database-backed configuration provider.
func NewDBConfigProvider(s store.Store) *DBConfigProvider {
	return &DBConfigProvider{store: s}
}

// GetReviewConfig retrieves review configuration from database.
func (p *DBConfigProvider) GetReviewConfig() (*ReviewConfig, error) {
	svc := NewSettingsService(p.store)
	return svc.GetReviewConfig()
}

// GetReportConfig retrieves report configuration from database.
func (p *DBConfigProvider) GetReportConfig() (*ReportConfig, error) {
	svc := NewSettingsService(p.store)
	return svc.GetReportConfig()
}

// GetGitProviders retrieves Git providers configuration from database.
func (p *DBConfigProvider) GetGitProviders() ([]ProviderConfig, error) {
	svc := NewSettingsService(p.store)
	return svc.GetGitProviders()
}

// GetAgents retrieves all agent configurations from database.
func (p *DBConfigProvider) GetAgents() (map[string]AgentDetail, error) {
	svc := NewSettingsService(p.store)
	return svc.GetAgents()
}

// GetNotificationConfig retrieves notification configuration from database.
func (p *DBConfigProvider) GetNotificationConfig() (*NotificationConfig, error) {
	svc := NewSettingsService(p.store)
	return svc.GetNotificationConfig()
}

// StaticConfigProvider implements ConfigProvider using static configuration.
// This is useful for testing or when database is not available.
type StaticConfigProvider struct {
	cfg *Config
}

// NewStaticConfigProvider creates a new static configuration provider.
func NewStaticConfigProvider(cfg *Config) *StaticConfigProvider {
	return &StaticConfigProvider{cfg: cfg}
}

// GetReviewConfig returns the static review configuration.
func (p *StaticConfigProvider) GetReviewConfig() (*ReviewConfig, error) {
	if p.cfg == nil {
		return nil, nil
	}
	return &p.cfg.Review, nil
}

// GetReportConfig returns the static report configuration.
// Applies default values for zero or negative values to match SettingsService behavior.
func (p *StaticConfigProvider) GetReportConfig() (*ReportConfig, error) {
	if p.cfg == nil {
		return nil, nil
	}
	cfg := p.cfg.Report
	// Apply defaults for zero or negative values
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3 // DefaultReportMaxRetries
	}
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = 10 // DefaultReportRetryDelay in seconds
	}
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 2
	}
	if cfg.OutputLanguage == "" {
		cfg.OutputLanguage = "en"
	}
	return &cfg, nil
}

// GetGitProviders returns the static Git providers configuration.
func (p *StaticConfigProvider) GetGitProviders() ([]ProviderConfig, error) {
	if p.cfg == nil {
		return nil, nil
	}
	return p.cfg.Git.Providers, nil
}

// GetAgents returns the static agent configurations.
func (p *StaticConfigProvider) GetAgents() (map[string]AgentDetail, error) {
	if p.cfg == nil {
		return nil, nil
	}
	return p.cfg.Agents, nil
}

// GetNotificationConfig returns the static notification configuration.
func (p *StaticConfigProvider) GetNotificationConfig() (*NotificationConfig, error) {
	if p.cfg == nil {
		return nil, nil
	}
	return &p.cfg.Notifications, nil
}
