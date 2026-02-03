package config

import (
	"testing"

	"github.com/verustcode/verustcode/internal/store"
)

// TestNewDBConfigProvider tests NewDBConfigProvider
func TestNewDBConfigProvider(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	provider := NewDBConfigProvider(testStore)

	if provider == nil {
		t.Fatal("NewDBConfigProvider returned nil")
	}
	if provider.store != testStore {
		t.Error("DBConfigProvider.store not set correctly")
	}
}

// TestNewStaticConfigProvider tests NewStaticConfigProvider
func TestNewStaticConfigProvider(t *testing.T) {
	cfg := &Config{
		Review: ReviewConfig{
			MaxRetries: 5,
		},
		Report: ReportConfig{
			MaxRetries: 3,
		},
	}

	provider := NewStaticConfigProvider(cfg)

	if provider == nil {
		t.Fatal("NewStaticConfigProvider returned nil")
	}
	if provider.cfg != cfg {
		t.Error("StaticConfigProvider.cfg not set correctly")
	}
}

// TestNewStaticConfigProvider_Nil tests NewStaticConfigProvider with nil config
func TestNewStaticConfigProvider_Nil(t *testing.T) {
	provider := NewStaticConfigProvider(nil)

	if provider == nil {
		t.Fatal("NewStaticConfigProvider returned nil")
	}
	if provider.cfg != nil {
		t.Error("StaticConfigProvider.cfg should be nil")
	}
}

// TestStaticConfigProvider_GetReviewConfig tests GetReviewConfig
func TestStaticConfigProvider_GetReviewConfig(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *Config
		expectNil   bool
		expectError bool
	}{
		{
			name: "Valid config",
			cfg: &Config{
				Review: ReviewConfig{
					MaxRetries: 5,
				},
			},
			expectNil:   false,
			expectError: false,
		},
		{
			name:        "Nil config",
			cfg:         nil,
			expectNil:   true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewStaticConfigProvider(tt.cfg)
			result, err := provider.GetReviewConfig()

			if (err != nil) != tt.expectError {
				t.Errorf("GetReviewConfig() error = %v, expectError = %v", err, tt.expectError)
				return
			}
			if (result == nil) != tt.expectNil {
				t.Errorf("GetReviewConfig() result = %v, expectNil = %v", result, tt.expectNil)
			}
			if result != nil && tt.cfg != nil {
				if result.MaxRetries != tt.cfg.Review.MaxRetries {
					t.Errorf("GetReviewConfig() MaxRetries = %d, want %d", result.MaxRetries, tt.cfg.Review.MaxRetries)
				}
			}
		})
	}
}

// TestStaticConfigProvider_GetReportConfig tests GetReportConfig
func TestStaticConfigProvider_GetReportConfig(t *testing.T) {
	tests := []struct {
		name           string
		cfg            *Config
		expectNil      bool
		expectError    bool
		checkDefaults  bool
		expectedValues map[string]interface{}
	}{
		{
			name: "Valid config with defaults",
			cfg: &Config{
				Report: ReportConfig{
					MaxRetries:     0,  // Should default to 3
					RetryDelay:     0,  // Should default to 10
					MaxConcurrent:  0,  // Should default to 2
					OutputLanguage: "", // Should default to "en"
				},
			},
			expectNil:     false,
			expectError:   false,
			checkDefaults: true,
			expectedValues: map[string]interface{}{
				"MaxRetries":     3,
				"RetryDelay":     10,
				"MaxConcurrent":  2,
				"OutputLanguage": "en",
			},
		},
		{
			name: "Valid config with values",
			cfg: &Config{
				Report: ReportConfig{
					MaxRetries:     5,
					RetryDelay:     20,
					MaxConcurrent:  4,
					OutputLanguage: "zh-CN",
				},
			},
			expectNil:     false,
			expectError:   false,
			checkDefaults: false,
			expectedValues: map[string]interface{}{
				"MaxRetries":     5,
				"RetryDelay":     20,
				"MaxConcurrent":  4,
				"OutputLanguage": "zh-CN",
			},
		},
		{
			name:        "Nil config",
			cfg:         nil,
			expectNil:   true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewStaticConfigProvider(tt.cfg)
			result, err := provider.GetReportConfig()

			if (err != nil) != tt.expectError {
				t.Errorf("GetReportConfig() error = %v, expectError = %v", err, tt.expectError)
				return
			}
			if (result == nil) != tt.expectNil {
				t.Errorf("GetReportConfig() result = %v, expectNil = %v", result, tt.expectNil)
				return
			}
			if result != nil && tt.expectedValues != nil {
				if result.MaxRetries != tt.expectedValues["MaxRetries"] {
					t.Errorf("GetReportConfig() MaxRetries = %d, want %d", result.MaxRetries, tt.expectedValues["MaxRetries"])
				}
				if result.RetryDelay != tt.expectedValues["RetryDelay"] {
					t.Errorf("GetReportConfig() RetryDelay = %d, want %d", result.RetryDelay, tt.expectedValues["RetryDelay"])
				}
				if result.MaxConcurrent != tt.expectedValues["MaxConcurrent"] {
					t.Errorf("GetReportConfig() MaxConcurrent = %d, want %d", result.MaxConcurrent, tt.expectedValues["MaxConcurrent"])
				}
				if result.OutputLanguage != tt.expectedValues["OutputLanguage"] {
					t.Errorf("GetReportConfig() OutputLanguage = %s, want %s", result.OutputLanguage, tt.expectedValues["OutputLanguage"])
				}
			}
		})
	}
}

// TestStaticConfigProvider_GetGitProviders tests GetGitProviders
func TestStaticConfigProvider_GetGitProviders(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *Config
		expectNil   bool
		expectError bool
		expectedLen int
	}{
		{
			name: "Valid config with providers",
			cfg: &Config{
				Git: GitConfig{
					Providers: []ProviderConfig{
						{Type: "github"},
						{Type: "gitlab"},
					},
				},
			},
			expectNil:   false,
			expectError: false,
			expectedLen: 2,
		},
		{
			name: "Valid config with empty providers",
			cfg: &Config{
				Git: GitConfig{
					Providers: []ProviderConfig{},
				},
			},
			expectNil:   false,
			expectError: false,
			expectedLen: 0,
		},
		{
			name:        "Nil config",
			cfg:         nil,
			expectNil:   true,
			expectError: false,
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewStaticConfigProvider(tt.cfg)
			result, err := provider.GetGitProviders()

			if (err != nil) != tt.expectError {
				t.Errorf("GetGitProviders() error = %v, expectError = %v", err, tt.expectError)
				return
			}
			if tt.expectNil {
				if result != nil {
					t.Errorf("GetGitProviders() result = %v, want nil", result)
				}
			} else {
				if result == nil {
					t.Error("GetGitProviders() result should not be nil")
					return
				}
				if len(result) != tt.expectedLen {
					t.Errorf("GetGitProviders() len = %d, want %d", len(result), tt.expectedLen)
				}
			}
		})
	}
}

// TestStaticConfigProvider_GetAgents tests GetAgents
func TestStaticConfigProvider_GetAgents(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *Config
		expectNil   bool
		expectError bool
		expectedLen int
	}{
		{
			name: "Valid config with agents",
			cfg: &Config{
				Agents: map[string]AgentDetail{
					"cursor": {CLIPath: "/usr/bin/cursor-agent"},
					"gemini": {CLIPath: "/usr/bin/gemini"},
				},
			},
			expectNil:   false,
			expectError: false,
			expectedLen: 2,
		},
		{
			name: "Valid config with empty agents",
			cfg: &Config{
				Agents: map[string]AgentDetail{},
			},
			expectNil:   false,
			expectError: false,
			expectedLen: 0,
		},
		{
			name:        "Nil config",
			cfg:         nil,
			expectNil:   true,
			expectError: false,
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewStaticConfigProvider(tt.cfg)
			result, err := provider.GetAgents()

			if (err != nil) != tt.expectError {
				t.Errorf("GetAgents() error = %v, expectError = %v", err, tt.expectError)
				return
			}
			if tt.expectNil {
				if result != nil {
					t.Errorf("GetAgents() result = %v, want nil", result)
				}
			} else {
				if result == nil {
					t.Error("GetAgents() result should not be nil")
					return
				}
				if len(result) != tt.expectedLen {
					t.Errorf("GetAgents() len = %d, want %d", len(result), tt.expectedLen)
				}
			}
		})
	}
}

// TestStaticConfigProvider_GetNotificationConfig tests GetNotificationConfig
func TestStaticConfigProvider_GetNotificationConfig(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *Config
		expectNil   bool
		expectError bool
	}{
		{
			name: "Valid config",
			cfg: &Config{
				Notifications: NotificationConfig{
					Channel: NotificationChannelWebhook,
				},
			},
			expectNil:   false,
			expectError: false,
		},
		{
			name:        "Nil config",
			cfg:         nil,
			expectNil:   true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewStaticConfigProvider(tt.cfg)
			result, err := provider.GetNotificationConfig()

			if (err != nil) != tt.expectError {
				t.Errorf("GetNotificationConfig() error = %v, expectError = %v", err, tt.expectError)
				return
			}
			if (result == nil) != tt.expectNil {
				t.Errorf("GetNotificationConfig() result = %v, expectNil = %v", result, tt.expectNil)
			}
			if result != nil && tt.cfg != nil {
				if result.Channel != tt.cfg.Notifications.Channel {
					t.Errorf("GetNotificationConfig() Channel = %v, want %v", result.Channel, tt.cfg.Notifications.Channel)
				}
			}
		})
	}
}

// TestDBConfigProvider_GetReviewConfig tests DBConfigProvider GetReviewConfig
// This requires a test store, so we test the basic structure
func TestDBConfigProvider_GetReviewConfig(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	provider := NewDBConfigProvider(testStore)

	// The actual implementation calls SettingsService which requires database setup
	// We just verify the provider is created correctly
	if provider.store != testStore {
		t.Error("DBConfigProvider.store not set correctly")
	}

	// Test that it doesn't panic (actual functionality tested in integration tests)
	_, _ = provider.GetReviewConfig()
}

// TestDBConfigProvider_GetReportConfig tests DBConfigProvider GetReportConfig
func TestDBConfigProvider_GetReportConfig(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	provider := NewDBConfigProvider(testStore)

	// Test that it doesn't panic
	_, _ = provider.GetReportConfig()
}

// TestDBConfigProvider_GetGitProviders tests DBConfigProvider GetGitProviders
func TestDBConfigProvider_GetGitProviders(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	provider := NewDBConfigProvider(testStore)

	// Test that it doesn't panic
	_, _ = provider.GetGitProviders()
}

// TestDBConfigProvider_GetAgents tests DBConfigProvider GetAgents
func TestDBConfigProvider_GetAgents(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	provider := NewDBConfigProvider(testStore)

	// Test that it doesn't panic
	_, _ = provider.GetAgents()
}

// TestDBConfigProvider_GetNotificationConfig tests DBConfigProvider GetNotificationConfig
func TestDBConfigProvider_GetNotificationConfig(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	provider := NewDBConfigProvider(testStore)

	// Test that it doesn't panic
	_, _ = provider.GetNotificationConfig()
}
