package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDefaultBootstrapConfig tests the DefaultBootstrapConfig function
func TestDefaultBootstrapConfig(t *testing.T) {
	cfg := DefaultBootstrapConfig()

	// Verify server defaults
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host = %v, want 0.0.0.0", cfg.Server.Host)
	}
	if cfg.Server.Port != 8091 {
		t.Errorf("Server.Port = %v, want 8091", cfg.Server.Port)
	}
	if cfg.Server.Debug {
		t.Error("Server.Debug should be false by default")
	}

	// Verify database defaults
	if cfg.Database.Path != "./data/verustcode.db" {
		t.Errorf("Database.Path = %v, want ./data/verustcode.db", cfg.Database.Path)
	}

	// Verify admin defaults
	if cfg.Admin == nil {
		t.Fatal("Admin config should not be nil")
	}
	if !cfg.Admin.Enabled {
		t.Error("Admin.Enabled should be true by default")
	}
	if cfg.Admin.Username != "admin" {
		t.Errorf("Admin.Username = %v, want admin", cfg.Admin.Username)
	}
	if cfg.Admin.PasswordHash != "" {
		t.Error("Admin.PasswordHash should be empty by default")
	}
	if cfg.Admin.TokenExpiration != 24 {
		t.Errorf("Admin.TokenExpiration = %v, want 24", cfg.Admin.TokenExpiration)
	}

	// Verify logging defaults
	if cfg.Logging.Level != "info" {
		t.Errorf("Logging.Level = %v, want info", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "text" {
		t.Errorf("Logging.Format = %v, want text", cfg.Logging.Format)
	}

	// Verify telemetry defaults
	if cfg.Telemetry.Enabled {
		t.Error("Telemetry.Enabled should be false by default")
	}
	if cfg.Telemetry.ServiceName != "verustcode" {
		t.Errorf("Telemetry.ServiceName = %v, want verustcode", cfg.Telemetry.ServiceName)
	}
}

// TestLoadBootstrap tests loading bootstrap configuration from file
func TestLoadBootstrap(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "bootstrap.yaml")

	configContent := `
server:
  host: "127.0.0.1"
  port: 9000
  debug: true

database:
  path: "./test/db.sqlite"

admin:
  enabled: true
  username: testadmin
  password_hash: '$2a$10$testhashhashhashhashhashhashhashhashhashhashhashhashha'
  jwt_secret: "test-secret-key-must-be-at-least-32-characters-long"
  expiry_hours: 48

logging:
  level: debug
  format: json
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	cfg, err := LoadBootstrap(configPath)
	if err != nil {
		t.Fatalf("LoadBootstrap() unexpected error: %v", err)
	}

	// Verify loaded values
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Server.Host = %v, want 127.0.0.1", cfg.Server.Host)
	}
	if cfg.Server.Port != 9000 {
		t.Errorf("Server.Port = %v, want 9000", cfg.Server.Port)
	}
	if !cfg.Server.Debug {
		t.Error("Server.Debug should be true")
	}
	if cfg.Database.Path != "./test/db.sqlite" {
		t.Errorf("Database.Path = %v, want ./test/db.sqlite", cfg.Database.Path)
	}
	if cfg.Admin.Username != "testadmin" {
		t.Errorf("Admin.Username = %v, want testadmin", cfg.Admin.Username)
	}
	if cfg.Admin.TokenExpiration != 48 {
		t.Errorf("Admin.TokenExpiration = %v, want 48", cfg.Admin.TokenExpiration)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Logging.Level = %v, want debug", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "json" {
		t.Errorf("Logging.Format = %v, want json", cfg.Logging.Format)
	}
}

// TestLoadBootstrap_EnvVarExpansion tests environment variable expansion
func TestLoadBootstrap_EnvVarExpansion(t *testing.T) {
	// Set environment variable
	os.Setenv("TEST_DB_PATH", "/var/lib/verustcode/test.db")
	defer os.Unsetenv("TEST_DB_PATH")

	// Create temporary config file with env var
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "bootstrap.yaml")

	configContent := `
database:
  path: ${TEST_DB_PATH}
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	cfg, err := LoadBootstrap(configPath)
	if err != nil {
		t.Fatalf("LoadBootstrap() unexpected error: %v", err)
	}

	if cfg.Database.Path != "/var/lib/verustcode/test.db" {
		t.Errorf("Database.Path = %v, want /var/lib/verustcode/test.db", cfg.Database.Path)
	}
}

// TestLoadBootstrap_EnvVarOverrides tests environment variable overrides
func TestLoadBootstrap_EnvVarOverrides(t *testing.T) {
	// Set environment variables
	os.Setenv("SC_SERVER_HOST", "192.168.1.100")
	os.Setenv("SC_SERVER_PORT", "9999")
	os.Setenv("SC_SERVER_DEBUG", "true")
	os.Setenv("SC_DATABASE_PATH", "/override/path.db")
	os.Setenv("SC_LOG_LEVEL", "error")
	defer func() {
		os.Unsetenv("SC_SERVER_HOST")
		os.Unsetenv("SC_SERVER_PORT")
		os.Unsetenv("SC_SERVER_DEBUG")
		os.Unsetenv("SC_DATABASE_PATH")
		os.Unsetenv("SC_LOG_LEVEL")
	}()

	// Create temporary config file with default values
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "bootstrap.yaml")

	configContent := `
server:
  host: "0.0.0.0"
  port: 8080
  debug: false

database:
  path: "./default.db"

logging:
  level: info
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	cfg, err := LoadBootstrap(configPath)
	if err != nil {
		t.Fatalf("LoadBootstrap() unexpected error: %v", err)
	}

	// Verify environment variables override file values
	if cfg.Server.Host != "192.168.1.100" {
		t.Errorf("Server.Host = %v, want 192.168.1.100 (from env)", cfg.Server.Host)
	}
	if cfg.Server.Port != 9999 {
		t.Errorf("Server.Port = %v, want 9999 (from env)", cfg.Server.Port)
	}
	if !cfg.Server.Debug {
		t.Error("Server.Debug should be true (from env)")
	}
	if cfg.Database.Path != "/override/path.db" {
		t.Errorf("Database.Path = %v, want /override/path.db (from env)", cfg.Database.Path)
	}
	if cfg.Logging.Level != "error" {
		t.Errorf("Logging.Level = %v, want error (from env)", cfg.Logging.Level)
	}
}

// TestLoadBootstrap_FileNotFound tests loading from non-existent file
func TestLoadBootstrap_FileNotFound(t *testing.T) {
	_, err := LoadBootstrap("/nonexistent/path/bootstrap.yaml")
	if err == nil {
		t.Error("LoadBootstrap() expected error for nonexistent file, got nil")
	}
}

// TestLoadBootstrap_InvalidYAML tests loading invalid YAML
func TestLoadBootstrap_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "bootstrap.yaml")

	// Invalid YAML content
	configContent := `
server:
  host: [invalid
  port: not-a-number
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	_, err := LoadBootstrap(configPath)
	if err == nil {
		t.Error("LoadBootstrap() expected error for invalid YAML, got nil")
	}
}

// TestBootstrapExists tests the BootstrapExists function
func TestBootstrapExists(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "bootstrap.yaml")

	// File doesn't exist yet
	if BootstrapExists(configPath) {
		t.Error("BootstrapExists() should return false for non-existent file")
	}

	// Create the file
	if err := os.WriteFile(configPath, []byte("server:\n  port: 8080"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// File should exist now
	if !BootstrapExists(configPath) {
		t.Error("BootstrapExists() should return true for existing file")
	}
}

// TestCreateDefaultBootstrap tests creating a default bootstrap configuration
func TestCreateDefaultBootstrap(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "bootstrap.yaml")

	// Create default bootstrap config
	if err := CreateDefaultBootstrap(configPath); err != nil {
		t.Fatalf("CreateDefaultBootstrap() error: %v", err)
	}

	// Verify file was created
	if !BootstrapExists(configPath) {
		t.Error("Bootstrap config file should exist after creation")
	}

	// Load and verify content
	cfg, err := LoadBootstrap(configPath)
	if err != nil {
		t.Fatalf("LoadBootstrap() error: %v", err)
	}

	// Verify default values
	if cfg.Server.Port != 8091 {
		t.Errorf("Server.Port = %v, want 8091 (default)", cfg.Server.Port)
	}
	if cfg.Admin.Username != "admin" {
		t.Errorf("Admin.Username = %v, want admin (default)", cfg.Admin.Username)
	}
}

// TestWriteBootstrap tests writing bootstrap configuration
func TestWriteBootstrap(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "bootstrap.yaml")

	cfg := &BootstrapConfig{
		Server: ServerConfig{
			Host:  "localhost",
			Port:  3000,
			Debug: true,
		},
		Database: BootstrapDBConfig{
			Path: "/custom/path.db",
		},
		Admin: &AdminConfig{
			Enabled:      true,
			Username:     "custom-admin",
			PasswordHash: "hash123",
		},
	}

	if err := WriteBootstrap(configPath, cfg); err != nil {
		t.Fatalf("WriteBootstrap() error: %v", err)
	}

	// Reload and verify
	loaded, err := LoadBootstrap(configPath)
	if err != nil {
		t.Fatalf("LoadBootstrap() error: %v", err)
	}

	if loaded.Server.Host != "localhost" {
		t.Errorf("Server.Host = %v, want localhost", loaded.Server.Host)
	}
	if loaded.Server.Port != 3000 {
		t.Errorf("Server.Port = %v, want 3000", loaded.Server.Port)
	}
	if loaded.Database.Path != "/custom/path.db" {
		t.Errorf("Database.Path = %v, want /custom/path.db", loaded.Database.Path)
	}
	if loaded.Admin.Username != "custom-admin" {
		t.Errorf("Admin.Username = %v, want custom-admin", loaded.Admin.Username)
	}
}

// TestParseBool tests the parseBool helper function
func TestParseBool(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"1", true},
		{"yes", true},
		{"YES", true},
		{"on", true},
		{"ON", true},
		{"false", false},
		{"FALSE", false},
		{"0", false},
		{"no", false},
		{"off", false},
		{"", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		result := parseBool(tt.input)
		if result != tt.expected {
			t.Errorf("parseBool(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

// TestGetDatabasePath tests the GetDatabasePath function
func TestGetDatabasePath(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with non-existent bootstrap file - should return default
	nonExistentPath := filepath.Join(tmpDir, "nonexistent.yaml")
	path := GetDatabasePath(nonExistentPath)
	if path != "./data/verustcode.db" {
		t.Errorf("GetDatabasePath() = %v, want ./data/verustcode.db (default)", path)
	}

	// Test with bootstrap file containing custom path
	configPath := filepath.Join(tmpDir, "bootstrap.yaml")
	configContent := `
database:
  path: "/custom/db/path.db"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	path = GetDatabasePath(configPath)
	if path != "/custom/db/path.db" {
		t.Errorf("GetDatabasePath() = %v, want /custom/db/path.db", path)
	}
}
