// Package config provides configuration management for the application.
// It supports YAML configuration files with environment variable overrides.
package config

import (
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/verustcode/verustcode/consts"
	"github.com/verustcode/verustcode/pkg/logger"
	"github.com/verustcode/verustcode/pkg/telemetry"
)

// Default configuration values
const (
	defaultWorkspace           = "./workspace"
	defaultReportWorkspace     = "./report_workspace"
	defaultCLIPath             = "/usr/local/bin/cursor"
	defaultAgentTimeout        = 600
	defaultMaxConcurrent       = 3
	defaultReportMaxConcurrent = 2
	defaultRetentionDays       = 30
	defaultMaxRetries          = 3
	defaultRetryDelay          = 5
	defaultReportMaxRetries    = 3
	defaultReportRetryDelay    = 10
	defaultOTLPEndpoint        = "localhost:4317"
	defaultPrometheusPort      = 9090
	defaultTaskTimeoutHours    = 24
	defaultMaxRetryCount       = 3
)

// Review configuration path constants
const (
	// ReviewsDir is the directory containing review configuration files
	ReviewsDir = "config/reviews"
	// DefaultReviewFile is the default review configuration file name
	DefaultReviewFile = "default.yaml"
	// RepoRootReviewPath is the path for review config at repository root (highest priority)
	RepoRootReviewPath = ".verust-review.yaml"
	// RepoEmbeddedReviewPath is the path for embedded review config in repositories (deprecated, use RepoRootReviewPath)
	RepoEmbeddedReviewPath = ".verustcode/review.yaml"
)

// ReviewFilePath returns the full path to the default review configuration file
// Kept for backward compatibility
var ReviewFilePath = ReviewsDir + "/" + DefaultReviewFile

// Config represents the complete application configuration
type Config struct {
	Subtitle      string                 `yaml:"subtitle"` // Application subtitle (displayed in browser title and sidebar)
	Server        ServerConfig           `yaml:"server"`
	Database      DatabaseConfig         `yaml:"database"`
	Auth          AuthConfig             `yaml:"auth"`
	Git           GitConfig              `yaml:"git"`
	Agents        map[string]AgentDetail `yaml:"agents"`
	Review        ReviewConfig           `yaml:"review"`
	Report        ReportConfig           `yaml:"report"`        // Report generation configuration
	Recovery      RecoveryConfig         `yaml:"recovery"`      // Task recovery configuration
	Notifications NotificationConfig     `yaml:"notifications"` // Notification configuration
	Logging       logger.Config          `yaml:"logging"`
	Telemetry     telemetry.Config       `yaml:"telemetry"`
	Admin         *AdminConfig           `yaml:"admin"`
}

// AdminConfig holds admin console configuration
type AdminConfig struct {
	Enabled         bool   `yaml:"enabled"`       // Enable admin console
	Username        string `yaml:"username"`      // Admin username
	PasswordHash    string `yaml:"password_hash"` // Admin password (bcrypt hash)
	JWTSecret       string `yaml:"jwt_secret"`    // JWT signing secret
	TokenExpiration int    `yaml:"expiry_hours"`  // Token expiration in hours
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host        string   `yaml:"host"`
	Port        int      `yaml:"port"`
	Debug       bool     `yaml:"debug"`
	CORSOrigins []string `yaml:"cors_origins"` // Allowed CORS origins whitelist
}

// DatabaseConfig holds database configuration
// Note: Database path is now hardcoded in the database package to prevent data loss from configuration errors
type DatabaseConfig struct {
	// Reserved for future database configuration options
}

// GitConfig holds Git provider configuration
type GitConfig struct {
	Providers []ProviderConfig `yaml:"providers"`
}

// ProviderConfig holds individual Git provider settings
type ProviderConfig struct {
	Type               string `yaml:"type"`                 // github, gitlab
	URL                string `yaml:"url"`                  // for self-hosted instances (supports both http:// and https://)
	Token              string `yaml:"token"`                // access token
	WebhookSecret      string `yaml:"webhook_secret"`       // webhook secret for validation
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"` // skip SSL certificate verification (for self-signed certs)
}

// AgentDetail holds specific agent configuration
type AgentDetail struct {
	CLIPath        string   `yaml:"cli_path" json:"cli_path"`
	APIKey         string   `yaml:"api_key" json:"api_key"`
	Timeout        int      `yaml:"timeout" json:"timeout"`                 // seconds
	DefaultModel   string   `yaml:"default_model" json:"default_model"`     // default model to use
	FallbackModels []string `yaml:"fallback_models" json:"fallback_models"` // fallback model list
}

// ReviewConfig holds review process configuration
type ReviewConfig struct {
	Workspace      string               `yaml:"workspace"`       // Working directory for cloned repositories
	MaxConcurrent  int                  `yaml:"max_concurrent"`  // Maximum concurrent review tasks
	RetentionDays  int                  `yaml:"retention_days"`  // Review result retention days
	MaxRetries     int                  `yaml:"max_retries"`     // Maximum retry attempts for review
	RetryDelay     int                  `yaml:"retry_delay"`     // Delay between retry attempts in seconds
	OutputLanguage string               `yaml:"output_language"` // Output language for review results (ISO 639-1 code, e.g., en, zh-cn)
	OutputMetadata OutputMetadataConfig `yaml:"output_metadata"` // Output metadata configuration
}

// OutputMetadataConfig configures metadata appended to review output
type OutputMetadataConfig struct {
	// ShowAgent controls whether to show agent type (default: true)
	ShowAgent *bool `yaml:"show_agent,omitempty" json:"show_agent,omitempty"`

	// ShowModel controls whether to show model name (default: true)
	ShowModel *bool `yaml:"show_model,omitempty" json:"show_model,omitempty"`

	// CustomText is custom text to append (supports markdown links)
	// Default: "Generated by [VerustCode](https://github.com/verustcode/verustcode)"
	CustomText string `yaml:"custom_text,omitempty" json:"custom_text,omitempty"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret    string `yaml:"jwt_secret"`    // JWT signing secret key
	TokenExpiry  int    `yaml:"token_expiry"`  // Normal token expiry in hours (default: 24)
	RememberDays int    `yaml:"remember_days"` // Remember me token expiry in days (default: 7)
}

// ReportConfig holds report generation configuration
type ReportConfig struct {
	Workspace      string `yaml:"workspace"`       // Workspace directory for report repos (default: report_workspace)
	MaxConcurrent  int    `yaml:"max_concurrent"`  // Maximum concurrent report tasks (default: 2)
	MaxRetries     int    `yaml:"max_retries"`     // Maximum retry attempts for each agent call (default: 3)
	RetryDelay     int    `yaml:"retry_delay"`     // Delay between retry attempts in seconds (default: 10)
	OutputLanguage string `yaml:"output_language"` // Output language for report content (ISO 639-1 code, e.g., en, zh-cn)
}

// RecoveryConfig holds task recovery configuration
type RecoveryConfig struct {
	TaskTimeoutHours int `yaml:"task_timeout_hours"` // Task timeout in hours (default: 24)
	MaxRetryCount    int `yaml:"max_retry_count"`    // Maximum retry count for failed tasks (default: 3)
}

// NotificationChannel represents the type of notification channel
type NotificationChannel string

const (
	NotificationChannelNone    NotificationChannel = ""        // Disabled
	NotificationChannelWebhook NotificationChannel = "webhook" // Generic webhook
	NotificationChannelEmail   NotificationChannel = "email"   // Email via SMTP
	NotificationChannelSlack   NotificationChannel = "slack"   // Slack webhook
	NotificationChannelFeishu  NotificationChannel = "feishu"  // Feishu/Lark webhook
)

// NotificationEvent represents the type of event to notify
type NotificationEvent string

const (
	NotificationEventReviewFailed    NotificationEvent = "review_failed"    // Review task failed
	NotificationEventReviewCompleted NotificationEvent = "review_completed" // Review task completed successfully
	NotificationEventReportFailed    NotificationEvent = "report_failed"    // Report generation failed
	NotificationEventReportCompleted NotificationEvent = "report_completed" // Report generation completed successfully
)

// NotificationConfig holds notification configuration
type NotificationConfig struct {
	// Channel specifies the notification channel type (single choice)
	// Empty string means notifications are disabled
	// Valid values: webhook, email, slack, feishu
	Channel NotificationChannel `yaml:"channel"`

	// Events specifies which events trigger notifications (multiple choice)
	// Valid values: review_failed, report_failed
	Events []NotificationEvent `yaml:"events"`

	// Webhook configuration (used when channel is "webhook")
	Webhook WebhookNotificationConfig `yaml:"webhook"`

	// Email configuration (used when channel is "email")
	Email EmailNotificationConfig `yaml:"email"`

	// Slack configuration (used when channel is "slack")
	Slack SlackNotificationConfig `yaml:"slack"`

	// Feishu configuration (used when channel is "feishu")
	Feishu FeishuNotificationConfig `yaml:"feishu"`
}

// WebhookNotificationConfig holds webhook notification settings
type WebhookNotificationConfig struct {
	// URL is the webhook endpoint URL
	URL string `yaml:"url"`
	// Secret is optional, used for HMAC signature verification
	Secret string `yaml:"secret"`
}

// EmailNotificationConfig holds email notification settings
type EmailNotificationConfig struct {
	// SMTPHost is the SMTP server hostname
	SMTPHost string `yaml:"smtp_host"`
	// SMTPPort is the SMTP server port (typically 25, 465, or 587)
	SMTPPort int `yaml:"smtp_port"`
	// Username is the SMTP authentication username
	Username string `yaml:"username"`
	// Password is the SMTP authentication password
	Password string `yaml:"password"`
	// From is the sender email address
	From string `yaml:"from"`
	// To is the list of recipient email addresses
	To []string `yaml:"to"`
}

// SlackNotificationConfig holds Slack notification settings
type SlackNotificationConfig struct {
	// WebhookURL is the Slack incoming webhook URL
	WebhookURL string `yaml:"webhook_url"`
	// Channel is optional, overrides the default channel configured in webhook
	Channel string `yaml:"channel"`
}

// FeishuNotificationConfig holds Feishu/Lark notification settings
type FeishuNotificationConfig struct {
	// WebhookURL is the Feishu bot webhook URL
	WebhookURL string `yaml:"webhook_url"`
	// Secret is optional, used for signature verification
	Secret string `yaml:"secret"`
}

// IsEnabled returns true if notifications are enabled
func (c *NotificationConfig) IsEnabled() bool {
	return c.Channel != "" && c.Channel != NotificationChannelNone
}

// HasEvent returns true if the specified event is in the events list
func (c *NotificationConfig) HasEvent(event NotificationEvent) bool {
	for _, e := range c.Events {
		if e == event {
			return true
		}
	}
	return false
}

// Default returns a default configuration
func Default() *Config {
	trueVal := true
	return &Config{
		Subtitle: "", // Default empty, will show "VerustCode" only
		Server: ServerConfig{
			Host:  "0.0.0.0",
			Port:  8080,
			Debug: false,
			CORSOrigins: []string{
				"http://localhost:8091",
				"http://localhost:8092",
			},
		},
		Database: DatabaseConfig{},
		Auth: AuthConfig{
			JWTSecret:    "", // Should be set via config file or environment variable
			TokenExpiry:  24, // 24 hours
			RememberDays: 7,  // 7 days
		},
		Git: GitConfig{
			Providers: []ProviderConfig{},
		},
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
			OutputMetadata: OutputMetadataConfig{
				ShowAgent:  &trueVal,
				ShowModel:  &trueVal,
				CustomText: "Generated by [VerustCode](https://github.com/verustcode/verustcode)",
			},
		},
		Report: ReportConfig{
			Workspace:     defaultReportWorkspace,
			MaxConcurrent: defaultReportMaxConcurrent,
			MaxRetries:    defaultReportMaxRetries,
			RetryDelay:    defaultReportRetryDelay,
		},
		Recovery: RecoveryConfig{
			TaskTimeoutHours: defaultTaskTimeoutHours,
			MaxRetryCount:    defaultMaxRetryCount,
		},
		Notifications: NotificationConfig{
			Channel: NotificationChannelNone, // Disabled by default
			Events:  []NotificationEvent{},   // Empty by default
		},
		Logging: logger.Config{
			Level:      "info",
			Format:     "text", // Default to human-readable text format instead of JSON
			File:       "",
			MaxSize:    100, // Max 100MB per log file
			MaxAge:     7,   // Retain logs for 7 days
			MaxBackups: 5,   // Keep 5 backup files
			Compress:   false,
		},
		Telemetry: telemetry.Config{
			Enabled:     false,
			ServiceName: consts.ServiceName,
			OTLP: telemetry.OTLPConfig{
				Enabled:  false,
				Endpoint: defaultOTLPEndpoint,
				Insecure: true,
			},
			Prometheus: telemetry.PrometheusConfig{
				Enabled: false,
				Port:    defaultPrometheusPort,
			},
		},
	}
}

// Load loads configuration from a YAML file with environment variable expansion
func Load(path string) (*Config, error) {
	cfg := Default()

	// Read configuration file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Expand environment variables in the configuration
	expanded := expandEnvVars(string(data))

	// Parse YAML
	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// expandEnvVars replaces ${VAR_NAME} patterns with environment variable values
// Only matches ${VAR_NAME} format (not $VAR_NAME) to avoid conflicts with special characters like bcrypt hashes
func expandEnvVars(content string) string {
	// Match ${VAR_NAME} patterns only (not $VAR_NAME to avoid bcrypt hash conflicts)
	re := regexp.MustCompile(`\$\{([^}]+)\}`)

	return re.ReplaceAllStringFunc(content, func(match string) string {
		// Extract variable name from ${VAR_NAME}
		varName := match[2 : len(match)-1]

		// Support default values: ${VAR_NAME:-default}
		parts := strings.SplitN(varName, ":-", 2)
		varName = parts[0]

		if value := os.Getenv(varName); value != "" {
			return value
		}

		// Return default value if provided
		if len(parts) > 1 {
			return parts[1]
		}

		return ""
	})
}

// Address returns the server address string
func (c *ServerConfig) Address() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}

// GetProvider returns provider configuration by type
func (c *GitConfig) GetProvider(providerType string) *ProviderConfig {
	for i := range c.Providers {
		if c.Providers[i].Type == providerType {
			return &c.Providers[i]
		}
	}
	return nil
}

// GetAgent returns agent configuration by name
func (c *Config) GetAgent(name string) *AgentDetail {
	if detail, ok := c.Agents[name]; ok {
		return &detail
	}
	return nil
}

// ToLLMClientConfig converts AgentDetail to llm.ClientConfig
func (d *AgentDetail) ToLLMClientConfig(name string) map[string]interface{} {
	config := map[string]interface{}{
		"name":     name,
		"cli_path": d.CLIPath,
		"api_key":  d.APIKey,
	}

	if d.Timeout > 0 {
		config["timeout"] = d.Timeout
	}
	if d.DefaultModel != "" {
		config["default_model"] = d.DefaultModel
	}
	if len(d.FallbackModels) > 0 {
		config["fallback_models"] = d.FallbackModels
	}

	return config
}
