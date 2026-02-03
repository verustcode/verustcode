package dsl

import (
	"os"
	"path/filepath"
	"testing"
)

// TestExpandEnvVars tests the expandEnvVars function
func TestExpandEnvVars(t *testing.T) {
	// Set test environment variables with allowed prefix
	os.Setenv("SCOPEVIEW_TEST_VAR_1", "value1")
	os.Setenv("SCOPEVIEW_TEST_VAR_2", "value2")
	os.Setenv("BLOCKED_VAR", "blocked_value") // This should be blocked
	defer func() {
		os.Unsetenv("SCOPEVIEW_TEST_VAR_1")
		os.Unsetenv("SCOPEVIEW_TEST_VAR_2")
		os.Unsetenv("BLOCKED_VAR")
	}()

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "no env vars",
			content: "hello world",
			want:    "hello world",
		},
		{
			name:    "single ${VAR} syntax with allowed prefix",
			content: "hello ${SCOPEVIEW_TEST_VAR_1}",
			want:    "hello value1",
		},
		{
			name:    "multiple env vars with allowed prefix",
			content: "${SCOPEVIEW_TEST_VAR_1} and ${SCOPEVIEW_TEST_VAR_2}",
			want:    "value1 and value2",
		},
		{
			name:    "$VAR syntax not supported (to avoid bcrypt hash conflicts)",
			content: "hello $SCOPEVIEW_TEST_VAR_1",
			want:    "hello $SCOPEVIEW_TEST_VAR_1",
		},
		{
			name:    "undefined var returns empty",
			content: "hello ${SCOPEVIEW_UNDEFINED_VAR}",
			want:    "hello ",
		},
		{
			name:    "default value when var undefined",
			content: "hello ${SCOPEVIEW_UNDEFINED_VAR:-default_value}",
			want:    "hello default_value",
		},
		{
			name:    "default value ignored when var defined",
			content: "hello ${SCOPEVIEW_TEST_VAR_1:-default_value}",
			want:    "hello value1",
		},
		{
			name:    "embedded in yaml",
			content: "agent: ${SCOPEVIEW_TEST_VAR_1}\nid: test",
			want:    "agent: value1\nid: test",
		},
		{
			name:    "empty content",
			content: "",
			want:    "",
		},
		{
			name:    "blocked var returns original",
			content: "hello ${BLOCKED_VAR}",
			want:    "hello ${BLOCKED_VAR}",
		},
		{
			name:    "blocked var with default uses default",
			content: "hello ${BLOCKED_VAR:-safe_default}",
			want:    "hello safe_default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandEnvVars(tt.content)
			if got != tt.want {
				t.Errorf("expandEnvVars() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestLoader_Load tests loading a single DSL configuration file
func TestLoader_Load(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "dsl_loader_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a valid test configuration file
	validConfig := `
version: "1.0"
rules:
  - id: test-rule
    description: Test Reviewer
    goals:
      areas:
        - security
`
	validFilePath := filepath.Join(tmpDir, "valid.yaml")
	if err := os.WriteFile(validFilePath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create an invalid test configuration file
	invalidConfig := `
version: "1.0"
rules: []
`
	invalidFilePath := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(invalidFilePath, []byte(invalidConfig), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	t.Run("load valid config", func(t *testing.T) {
		loader := NewLoader()
		config, err := loader.Load(validFilePath)

		if err != nil {
			t.Fatalf("Load() unexpected error: %v", err)
		}

		if config == nil {
			t.Fatal("Load() returned nil config")
		}

		if len(config.Rules) != 1 {
			t.Errorf("Load() returned %d rules, want 1", len(config.Rules))
		}

		if config.Rules[0].ID != "test-rule" {
			t.Errorf("Rule ID = %s, want test-rule", config.Rules[0].ID)
		}
	})

	t.Run("load non-existent file", func(t *testing.T) {
		loader := NewLoader()
		_, err := loader.Load(filepath.Join(tmpDir, "nonexistent.yaml"))

		if err == nil {
			t.Error("Load() expected error for non-existent file, got nil")
		}
	})

	t.Run("load invalid config", func(t *testing.T) {
		loader := NewLoader()
		_, err := loader.Load(invalidFilePath)

		if err == nil {
			t.Error("Load() expected error for invalid config, got nil")
		}
	})
}

// TestLoader_LoadWithEnvVars tests loading config with environment variable expansion
func TestLoader_LoadWithEnvVars(t *testing.T) {
	// Set test environment variable with allowed prefix
	os.Setenv("SCOPEVIEW_TEST_AGENT_TYPE", "gemini")
	defer os.Unsetenv("SCOPEVIEW_TEST_AGENT_TYPE")

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "dsl_loader_env_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create config with env var using allowed prefix
	// Note: environment variables are used for agent.type field
	configWithEnv := `
version: "1.0"
rule_base:
  agent:
    type: ${SCOPEVIEW_TEST_AGENT_TYPE}
rules:
  - id: test-rule
    description: Test Reviewer
    goals:
      areas:
        - security
`
	filePath := filepath.Join(tmpDir, "config_with_env.yaml")
	if err := os.WriteFile(filePath, []byte(configWithEnv), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	loader := NewLoader()
	config, err := loader.Load(filePath)

	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	// The agent type should be from env var
	if config.Rules[0].Agent.Type != "gemini" {
		t.Errorf("Agent.Type = %s, want gemini (from env var)", config.Rules[0].Agent.Type)
	}
}

// TestLoader_LoadFromDir tests loading multiple config files from a directory
func TestLoader_LoadFromDir(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "dsl_loader_dir_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create multiple valid config files
	config1 := `
version: "1.0"
rules:
  - id: rule1
    description: Reviewer 1
    goals:
      areas:
        - security
`
	config2 := `
version: "1.0"
rules:
  - id: rule2
    description: Reviewer 2
    goals:
      areas:
        - performance
`

	if err := os.WriteFile(filepath.Join(tmpDir, "config1.yaml"), []byte(config1), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "config2.yml"), []byte(config2), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	t.Run("load from directory with valid configs", func(t *testing.T) {
		loader := NewLoader()
		configs, err := loader.LoadFromDir(tmpDir)

		if err != nil {
			t.Fatalf("LoadFromDir() unexpected error: %v", err)
		}

		if len(configs) != 2 {
			t.Errorf("LoadFromDir() returned %d configs, want 2", len(configs))
		}
	})

	t.Run("load from empty directory", func(t *testing.T) {
		emptyDir, err := os.MkdirTemp("", "dsl_loader_empty_test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(emptyDir)

		loader := NewLoader()
		_, err = loader.LoadFromDir(emptyDir)

		if err == nil {
			t.Error("LoadFromDir() expected error for empty directory, got nil")
		}
	})
}

// TestMergeConfigs tests merging multiple configurations
func TestMergeConfigs(t *testing.T) {
	t.Run("merge empty configs", func(t *testing.T) {
		result := MergeConfigs()

		if result == nil {
			t.Fatal("MergeConfigs() returned nil")
		}

		if len(result.Rules) != 0 {
			t.Errorf("MergeConfigs() returned %d rules, want 0", len(result.Rules))
		}
	})

	t.Run("merge single config", func(t *testing.T) {
		config := &ReviewRulesConfig{
			Version: "1.0",
			Rules: []ReviewRuleConfig{
				{ID: "rule1", Description: "Reviewer 1"},
			},
		}

		result := MergeConfigs(config)

		if result.Version != "1.0" {
			t.Errorf("Version = %s, want 1.0", result.Version)
		}

		if len(result.Rules) != 1 {
			t.Errorf("MergeConfigs() returned %d rules, want 1", len(result.Rules))
		}
	})

	t.Run("merge multiple configs", func(t *testing.T) {
		config1 := &ReviewRulesConfig{
			Version: "1.0",
			Rules: []ReviewRuleConfig{
				{ID: "rule1", Description: "Reviewer 1"},
				{ID: "rule2", Description: "Reviewer 2"},
			},
		}

		config2 := &ReviewRulesConfig{
			Version: "1.0",
			Rules: []ReviewRuleConfig{
				{ID: "rule3", Description: "Reviewer 3"},
			},
		}

		result := MergeConfigs(config1, config2)

		if len(result.Rules) != 3 {
			t.Errorf("MergeConfigs() returned %d rules, want 3", len(result.Rules))
		}

		// Verify rule IDs are preserved
		ruleIDs := make(map[string]bool)
		for _, rule := range result.Rules {
			ruleIDs[rule.ID] = true
		}

		for _, id := range []string{"rule1", "rule2", "rule3"} {
			if !ruleIDs[id] {
				t.Errorf("Missing rule ID: %s", id)
			}
		}
	})

	t.Run("merge with duplicate IDs", func(t *testing.T) {
		config1 := &ReviewRulesConfig{
			Version: "1.0",
			Rules: []ReviewRuleConfig{
				{ID: "rule1", Description: "Reviewer 1"},
			},
		}

		config2 := &ReviewRulesConfig{
			Version: "1.0",
			Rules: []ReviewRuleConfig{
				{ID: "rule1", Description: "Duplicate Reviewer"}, // Duplicate ID
				{ID: "rule2", Description: "Reviewer 2"},
			},
		}

		result := MergeConfigs(config1, config2)

		// Should have 2 rules (duplicate skipped)
		if len(result.Rules) != 2 {
			t.Errorf("MergeConfigs() returned %d rules, want 2 (duplicate should be skipped)", len(result.Rules))
		}

		// First rule1 should be kept
		if result.Rules[0].Description != "Reviewer 1" {
			t.Errorf("First rule description = %s, want 'Reviewer 1'", result.Rules[0].Description)
		}
	})
}

// TestNewLoader tests loader creation
func TestNewLoader(t *testing.T) {
	t.Run("new loader", func(t *testing.T) {
		loader := NewLoader()
		if loader == nil {
			t.Error("NewLoader() returned nil")
		}
		if loader.parser == nil {
			t.Error("NewLoader() parser is nil")
		}
	})

	t.Run("new strict loader", func(t *testing.T) {
		loader := NewStrictLoader()
		if loader == nil {
			t.Error("NewStrictLoader() returned nil")
		}
		if loader.parser == nil {
			t.Error("NewStrictLoader() parser is nil")
		}
	})
}

// TestLoader_ValidateFile tests file validation
func TestLoader_ValidateFile(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "dsl_validate_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create valid config
	validConfig := `
version: "1.0"
rules:
  - id: test-rule
    description: Test Reviewer
    goals:
      areas:
        - security
`
	validFilePath := filepath.Join(tmpDir, "valid.yaml")
	if err := os.WriteFile(validFilePath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create invalid config
	invalidConfig := `
version: "1.0"
rules: []
`
	invalidFilePath := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(invalidFilePath, []byte(invalidConfig), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	t.Run("validate valid file", func(t *testing.T) {
		loader := NewLoader()
		err := loader.ValidateFile(validFilePath)
		if err != nil {
			t.Errorf("ValidateFile() unexpected error: %v", err)
		}
	})

	t.Run("validate invalid file", func(t *testing.T) {
		loader := NewLoader()
		err := loader.ValidateFile(invalidFilePath)
		if err == nil {
			t.Error("ValidateFile() expected error for invalid file, got nil")
		}
	})
}

// TestFindConfigFile tests finding config files in common locations
func TestFindConfigFile(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "dsl_find_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test config file
	testFile := filepath.Join(tmpDir, "test_config.yaml")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	t.Run("find existing file with full path", func(t *testing.T) {
		result := FindConfigFile(testFile)
		if result != testFile {
			t.Errorf("FindConfigFile() = %s, want %s", result, testFile)
		}
	})

	t.Run("find non-existent file", func(t *testing.T) {
		result := FindConfigFile("nonexistent_config_12345.yaml")
		if result != "" {
			t.Errorf("FindConfigFile() = %s, want empty string", result)
		}
	})
}

// TestLoader_LoadFromRepoRoot tests loading config from .verust-review.yaml at repository root
func TestLoader_LoadFromRepoRoot(t *testing.T) {
	// Create temp directory simulating a repository
	tmpDir, err := os.MkdirTemp("", "dsl_repo_root_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Valid config content
	validConfig := `
version: "1.0"
rules:
  - id: repo-root-rule
    description: Review rule from repository root
    goals:
      areas:
        - security
`

	t.Run("load from existing .verust-review.yaml", func(t *testing.T) {
		// Create .verust-review.yaml at repo root
		configPath := filepath.Join(tmpDir, ".verust-review.yaml")
		if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		defer os.Remove(configPath)

		loader := NewLoader()
		config, err := loader.LoadFromRepoRoot(tmpDir)

		if err != nil {
			t.Fatalf("LoadFromRepoRoot() unexpected error: %v", err)
		}

		if config == nil {
			t.Fatal("LoadFromRepoRoot() returned nil config")
		}

		if len(config.Rules) != 1 {
			t.Errorf("LoadFromRepoRoot() returned %d rules, want 1", len(config.Rules))
		}

		if config.Rules[0].ID != "repo-root-rule" {
			t.Errorf("Rule ID = %s, want repo-root-rule", config.Rules[0].ID)
		}
	})

	t.Run("return nil when .verust-review.yaml does not exist", func(t *testing.T) {
		// Use a directory without .verust-review.yaml
		emptyDir, err := os.MkdirTemp("", "dsl_empty_repo_test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(emptyDir)

		loader := NewLoader()
		config, err := loader.LoadFromRepoRoot(emptyDir)

		if err != nil {
			t.Fatalf("LoadFromRepoRoot() unexpected error: %v", err)
		}

		if config != nil {
			t.Errorf("LoadFromRepoRoot() expected nil for non-existent file, got config with %d rules", len(config.Rules))
		}
	})

	t.Run("return error for invalid config", func(t *testing.T) {
		// Create invalid .verust-review.yaml
		invalidConfig := `
version: "1.0"
rules: []
`
		configPath := filepath.Join(tmpDir, ".verust-review.yaml")
		if err := os.WriteFile(configPath, []byte(invalidConfig), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		defer os.Remove(configPath)

		loader := NewLoader()
		_, err := loader.LoadFromRepoRoot(tmpDir)

		if err == nil {
			t.Error("LoadFromRepoRoot() expected error for invalid config, got nil")
		}
	})
}

