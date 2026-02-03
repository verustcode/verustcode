// Package config provides configuration management for the application.
// This file handles bootstrap configuration which requires server restart to take effect.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/verustcode/verustcode/pkg/logger"
	"github.com/verustcode/verustcode/pkg/telemetry"
)

// BootstrapConfig holds configuration that requires server restart to take effect.
// These are core system settings that cannot be changed at runtime.
type BootstrapConfig struct {
	Server    ServerConfig      `yaml:"server"`
	Database  BootstrapDBConfig `yaml:"database"`
	Admin     *AdminConfig      `yaml:"admin"`
	Logging   logger.Config     `yaml:"logging"`
	Telemetry telemetry.Config  `yaml:"telemetry"`
}

// BootstrapDBConfig holds database configuration for bootstrap
type BootstrapDBConfig struct {
	Path string `yaml:"path"`
}

// BootstrapConfigPath is the default path for bootstrap configuration
const BootstrapConfigPath = "config/bootstrap.yaml"

// DefaultBootstrapConfig returns default bootstrap configuration
func DefaultBootstrapConfig() *BootstrapConfig {
	return &BootstrapConfig{
		Server: ServerConfig{
			Host:  "0.0.0.0",
			Port:  8091,
			Debug: false,
		},
		Database: BootstrapDBConfig{
			Path: "./data/verustcode.db",
		},
		Admin: &AdminConfig{
			Enabled:         true,
			Username:        "admin",
			PasswordHash:    "",
			JWTSecret:       "",
			TokenExpiration: 24,
		},
		Logging: logger.Config{
			Level:      "info",
			Format:     "text",
			File:       "",
			MaxSize:    100,
			MaxAge:     7,
			MaxBackups: 5,
			Compress:   false,
			AccessLog:  false,
		},
		Telemetry: telemetry.Config{
			Enabled:     false,
			ServiceName: "verustcode",
			OTLP: telemetry.OTLPConfig{
				Enabled:  false,
				Endpoint: "localhost:4317",
				Insecure: true,
			},
			Prometheus: telemetry.PrometheusConfig{
				Enabled: false,
				Port:    9090,
			},
		},
	}
}

// LoadBootstrap loads bootstrap configuration from file with environment variable support.
// Environment variables can override values using SC_ prefix:
//   - SC_SERVER_HOST, SC_SERVER_PORT, SC_SERVER_DEBUG
//   - SC_DATABASE_PATH
//   - SC_ADMIN_USERNAME, SC_ADMIN_PASSWORD_HASH, SC_ADMIN_JWT_SECRET
//   - SC_LOG_LEVEL, SC_LOG_FORMAT, SC_LOG_FILE
func LoadBootstrap(path string) (*BootstrapConfig, error) {
	cfg := DefaultBootstrapConfig()

	// Read configuration file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read bootstrap config: %w", err)
	}

	// Expand environment variables in the configuration
	expanded := expandEnvVars(string(data))

	// Parse YAML
	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, fmt.Errorf("failed to parse bootstrap config: %w", err)
	}

	// Apply environment variable overrides
	applyBootstrapEnvOverrides(cfg)

	return cfg, nil
}

// BootstrapExists checks if bootstrap configuration file exists
func BootstrapExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// CreateDefaultBootstrap creates a default bootstrap configuration file
func CreateDefaultBootstrap(path string) error {
	cfg := DefaultBootstrapConfig()
	return WriteBootstrap(path, cfg)
}

// WriteBootstrap writes bootstrap configuration to file
func WriteBootstrap(path string, cfg *BootstrapConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal bootstrap config: %w", err)
	}

	// Add header comment
	content := bootstrapHeader + string(data)

	// Write file with proper permissions
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write bootstrap config: %w", err)
	}

	return nil
}

// bootstrapHeader is the comment header for bootstrap.yaml
const bootstrapHeader = `# VerustCode Bootstrap Configuration
# This file contains core system settings that require server restart to take effect.
# For runtime configuration, use the admin web interface.
#
# Environment Variable Support:
#   - Use ${VAR_NAME} syntax in values to reference environment variables
#   - Or use SC_* prefix environment variables to override:
#     SC_SERVER_HOST, SC_SERVER_PORT, SC_SERVER_DEBUG
#     SC_DATABASE_PATH
#     SC_ADMIN_USERNAME, SC_ADMIN_JWT_SECRET
#     SC_LOG_LEVEL, SC_LOG_FORMAT
#

`

// applyBootstrapEnvOverrides applies environment variable overrides to bootstrap config
func applyBootstrapEnvOverrides(cfg *BootstrapConfig) {
	// Server overrides
	if v := os.Getenv("SC_SERVER_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("SC_SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("SC_SERVER_DEBUG"); v != "" {
		cfg.Server.Debug = parseBool(v)
	}

	// Database overrides
	if v := os.Getenv("SC_DATABASE_PATH"); v != "" {
		cfg.Database.Path = v
	}

	// Admin overrides
	if cfg.Admin != nil {
		if v := os.Getenv("SC_ADMIN_USERNAME"); v != "" {
			cfg.Admin.Username = v
		}
		if v := os.Getenv("SC_ADMIN_PASSWORD_HASH"); v != "" {
			cfg.Admin.PasswordHash = v
		}
		if v := os.Getenv("SC_ADMIN_JWT_SECRET"); v != "" {
			cfg.Admin.JWTSecret = v
		}
	}

	// Logging overrides
	if v := os.Getenv("SC_LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}
	if v := os.Getenv("SC_LOG_FORMAT"); v != "" {
		cfg.Logging.Format = v
	}
	if v := os.Getenv("SC_LOG_FILE"); v != "" {
		cfg.Logging.File = v
	}

	// Telemetry overrides
	if v := os.Getenv("SC_TELEMETRY_ENABLED"); v != "" {
		cfg.Telemetry.Enabled = parseBool(v)
	}
	if v := os.Getenv("SC_OTLP_ENABLED"); v != "" {
		cfg.Telemetry.OTLP.Enabled = parseBool(v)
	}
	if v := os.Getenv("SC_OTLP_ENDPOINT"); v != "" {
		cfg.Telemetry.OTLP.Endpoint = v
	}
	if v := os.Getenv("SC_PROMETHEUS_ENABLED"); v != "" {
		cfg.Telemetry.Prometheus.Enabled = parseBool(v)
	}
	if v := os.Getenv("SC_PROMETHEUS_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Telemetry.Prometheus.Port = port
		}
	}
}

// parseBool parses a boolean string value
func parseBool(v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	return v == "true" || v == "1" || v == "yes" || v == "on"
}

// UpdateJWTSecretInConfig updates the jwt_secret field in the config file.
// It uses YAML parsing to safely update only the jwt_secret field while preserving all other fields.
func UpdateJWTSecretInConfig(configPath, jwtSecret string) error {
	// Read current config file
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Backup current config before making changes
	backupPath := configPath + ".backup"
	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		// Continue anyway, backup is optional
		fmt.Fprintf(os.Stderr, "[WARNING] Failed to create backup: %v\n", err)
	}

	// Parse YAML into a generic map to preserve all fields
	var cfg map[string]interface{}
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Get or create admin section
	adminSection, ok := cfg["admin"].(map[string]interface{})
	if !ok {
		// Admin section doesn't exist or is not a map, create it
		adminSection = make(map[string]interface{})
		cfg["admin"] = adminSection
	}

	// Update only the jwt_secret field, preserving all other fields
	adminSection["jwt_secret"] = jwtSecret

	// Marshal back to YAML
	newContent, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Add header comment back
	finalContent := bootstrapHeader + string(newContent)

	// Write the updated config file
	if err := os.WriteFile(configPath, []byte(finalContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
