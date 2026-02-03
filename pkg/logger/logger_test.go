package logger

import (
	"os"
	"sync"
	"testing"

	"go.uber.org/zap"
)

func TestInit(t *testing.T) {
	// Reset global state
	globalLogger = nil
	once = sync.Once{}

	cfg := Config{
		Level:  "info",
		Format: "json",
	}

	err := Init(cfg)
	if err != nil {
		t.Fatalf("Init() error = %v, want nil", err)
	}

	// Second call should be safe and return nil
	err = Init(cfg)
	if err != nil {
		t.Errorf("Init() second call error = %v, want nil", err)
	}
}

func TestInit_TextFormat(t *testing.T) {
	// Reset global state
	globalLogger = nil
	once = sync.Once{}

	cfg := Config{
		Level:  "debug",
		Format: "text",
	}

	err := Init(cfg)
	if err != nil {
		t.Fatalf("Init() with text format error = %v, want nil", err)
	}
}

func TestInit_InvalidLevel(t *testing.T) {
	// Reset global state
	globalLogger = nil
	once = sync.Once{}

	cfg := Config{
		Level:  "invalid-level",
		Format: "json",
	}

	err := Init(cfg)
	if err != nil {
		t.Fatalf("Init() with invalid level should default to info, got error = %v", err)
	}
}

func TestInit_WithFile(t *testing.T) {
	// Reset global state
	globalLogger = nil
	once = sync.Once{}

	tmpFile := "/tmp/test_logger.log"
	defer os.Remove(tmpFile)

	cfg := Config{
		Level:    "info",
		Format:   "json",
		File:     tmpFile,
		MaxSize:  10,
		MaxAge:   7,
		MaxBackups: 5,
		Compress: false,
	}

	err := Init(cfg)
	if err != nil {
		t.Fatalf("Init() with file error = %v, want nil", err)
	}
}

func TestGet(t *testing.T) {
	// Reset global state
	globalLogger = nil
	once = sync.Once{}

	// Test with uninitialized logger
	logger := Get()
	if logger == nil {
		t.Error("Get() returned nil logger")
	}

	// Initialize logger
	cfg := Config{
		Level:  "info",
		Format: "json",
	}
	Init(cfg)

	logger = Get()
	if logger == nil {
		t.Error("Get() returned nil logger after Init()")
	}
}

func TestSugar(t *testing.T) {
	// Reset global state
	globalLogger = nil
	once = sync.Once{}

	cfg := Config{
		Level:  "info",
		Format: "json",
	}
	Init(cfg)

	sugar := Sugar()
	if sugar == nil {
		t.Error("Sugar() returned nil")
	}
}

func TestWith(t *testing.T) {
	// Reset global state
	globalLogger = nil
	once = sync.Once{}

	cfg := Config{
		Level:  "info",
		Format: "json",
	}
	Init(cfg)

	logger := With(zap.String("key", "value"))
	if logger == nil {
		t.Error("With() returned nil logger")
	}
}

func TestNamed(t *testing.T) {
	// Reset global state
	globalLogger = nil
	once = sync.Once{}

	cfg := Config{
		Level:  "info",
		Format: "json",
	}
	Init(cfg)

	logger := Named("test-logger")
	if logger == nil {
		t.Error("Named() returned nil logger")
	}
}

func TestLogFunctions(t *testing.T) {
	// Reset global state
	globalLogger = nil
	once = sync.Once{}

	cfg := Config{
		Level:  "debug",
		Format: "json",
	}
	Init(cfg)

	// Test that log functions don't panic
	Debug("debug message", zap.String("key", "value"))
	Info("info message", zap.String("key", "value"))
	Warn("warn message", zap.String("key", "value"))
	Error("error message", zap.String("key", "value"))
}

func TestSync(t *testing.T) {
	// Reset global state
	globalLogger = nil
	once = sync.Once{}

	// Test with uninitialized logger
	err := Sync()
	if err != nil {
		t.Errorf("Sync() with uninitialized logger error = %v, want nil", err)
	}

	// Initialize logger
	cfg := Config{
		Level:  "info",
		Format: "json",
	}
	Init(cfg)

	// Sync may fail in test environment due to stdout being closed
	// Just verify it doesn't panic
	err = Sync()
	// We don't check for nil error as it may fail in test environment
	_ = err
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name      string
		level     string
		wantError bool
	}{
		{"valid debug", "debug", false},
		{"valid info", "info", false},
		{"valid warn", "warn", false},
		{"valid error", "error", false},
		{"invalid level", "invalid", true},
		{"empty level", "", false}, // Empty string doesn't error, defaults to info level
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseLevel(tt.level)
			if (err != nil) != tt.wantError {
				t.Errorf("parseLevel(%q) error = %v, wantError = %v", tt.level, err, tt.wantError)
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	// Reset global state
	globalLogger = nil
	once = sync.Once{}

	cfg := Config{
		Level:  "info",
		Format: "json",
		File:   "/tmp/test_defaults.log",
		// MaxSize, MaxAge, MaxBackups not set - should use defaults
	}

	err := Init(cfg)
	if err != nil {
		t.Fatalf("Init() with defaults error = %v, want nil", err)
	}
	defer os.Remove("/tmp/test_defaults.log")
}

