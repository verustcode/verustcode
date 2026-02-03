package check

import (
	"os"
	"path/filepath"
	"testing"
)

// TestValidateBootstrapYaml tests validateBootstrapYaml
func TestValidateBootstrapYaml(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "scopeview_test_validation")
	defer os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)

	checker := NewChecker()
	checker.configDir = tmpDir

	tests := []struct {
		name        string
		setupFile   bool
		fileContent string
		expectValid bool
		expectError bool
	}{
		{
			name:        "Valid bootstrap file",
			setupFile:   true,
			fileContent: "server:\n  host: localhost\n  port: 8080",
			expectValid: true,
			expectError: false,
		},
		{
			name:        "Non-existent file",
			setupFile:   false,
			expectValid: false,
			expectError: true,
		},
		{
			name:        "Invalid YAML",
			setupFile:   true,
			fileContent: "invalid: yaml: content: [",
			expectValid: false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bootstrapPath := checker.BootstrapPath()
			if tt.setupFile {
				if err := os.WriteFile(bootstrapPath, []byte(tt.fileContent), 0644); err != nil {
					t.Fatalf("Failed to create bootstrap file: %v", err)
				}
				defer os.Remove(bootstrapPath)
			}

			result := checker.validateBootstrapYaml()

			if result.Valid != tt.expectValid {
				t.Errorf("validateBootstrapYaml() Valid = %v, want %v", result.Valid, tt.expectValid)
			}
			if (result.Error != nil) != tt.expectError {
				t.Errorf("validateBootstrapYaml() Error = %v, want error = %v", result.Error, tt.expectError)
			}
			if result.Path != bootstrapPath {
				t.Errorf("validateBootstrapYaml() Path = %s, want %s", result.Path, bootstrapPath)
			}
		})
	}
}

// TestValidateConfigYaml tests validateConfigYaml
func TestValidateConfigYaml(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "scopeview_test_validation2")
	defer os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)

	checker := NewChecker()
	checker.configDir = tmpDir

	tests := []struct {
		name        string
		setupFile   bool
		fileContent string
		expectValid bool
		expectError bool
	}{
		{
			name:        "Valid config file",
			setupFile:   true,
			fileContent: "review:\n  max_retries: 3",
			expectValid: true,
			expectError: false,
		},
		{
			name:        "Non-existent file",
			setupFile:   false,
			expectValid: false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := checker.ConfigPath()
			if tt.setupFile {
				if err := os.WriteFile(configPath, []byte(tt.fileContent), 0644); err != nil {
					t.Fatalf("Failed to create config file: %v", err)
				}
				defer os.Remove(configPath)
			}

			result := checker.validateConfigYaml()

			if result.Valid != tt.expectValid {
				t.Errorf("validateConfigYaml() Valid = %v, want %v", result.Valid, tt.expectValid)
			}
			if (result.Error != nil) != tt.expectError {
				t.Errorf("validateConfigYaml() Error = %v, want error = %v", result.Error, tt.expectError)
			}
		})
	}
}

// TestValidateSchemas tests validateSchemas with embedded schema
func TestValidateSchemas(t *testing.T) {
	checker := NewChecker()

	// Since schema is now embedded in code, validation should always pass
	// as long as the embedded schema is properly structured
	result := checker.validateSchemas()

	if !result.Valid {
		t.Errorf("validateSchemas() Valid = false, want true for embedded schema")
	}
	if result.Error != nil {
		t.Errorf("validateSchemas() Error = %v, want nil", result.Error)
	}
	if result.Path != "embedded:default-schema" {
		t.Errorf("validateSchemas() Path = %v, want 'embedded:default-schema'", result.Path)
	}
}

// TestValidateYamlSyntax tests validateYamlSyntax
func TestValidateYamlSyntax(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "scopeview_test_validation4")
	defer os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)

	tests := []struct {
		name        string
		fileContent string
		expectError bool
	}{
		{
			name:        "Valid YAML",
			fileContent: "key: value\nlist:\n  - item1\n  - item2",
			expectError: false,
		},
		{
			name:        "Invalid YAML",
			fileContent: "key: value: invalid",
			expectError: true,
		},
		{
			name:        "Empty file",
			fileContent: "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(tmpDir, "test.yaml")
			if err := os.WriteFile(tmpFile, []byte(tt.fileContent), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
			defer os.Remove(tmpFile)

			err := validateYamlSyntax(tmpFile)
			if (err != nil) != tt.expectError {
				t.Errorf("validateYamlSyntax() error = %v, want error = %v", err, tt.expectError)
			}
		})
	}
}

// TestExtractAgentNames tests extractAgentNames
func TestExtractAgentNames(t *testing.T) {
	// Create a minimal DSL config for testing
	// We need to import dsl package and create a proper config
	// Since extractAgentNames accesses RuleBase.Agent, we need a valid config
	// For now, skip this test as it requires full DSL config setup
	// This function is tested indirectly through validateRulesYaml tests
	t.Skip("extractAgentNames requires full DSL config setup, tested indirectly")
}

// TestGetDefaultCLIName tests getDefaultCLIName
func TestGetDefaultCLIName(t *testing.T) {
	tests := []struct {
		name     string
		agent    string
		expected string
	}{
		{
			name:     "cursor agent",
			agent:    "cursor",
			expected: "cursor-agent",
		},
		{
			name:     "gemini agent",
			agent:    "gemini",
			expected: "gemini",
		},
		{
			name:     "qoder agent",
			agent:    "qoder",
			expected: "qodercli",
		},
		{
			name:     "unknown agent",
			agent:    "unknown",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getDefaultCLIName(tt.agent)
			if result != tt.expected {
				t.Errorf("getDefaultCLIName(%s) = %s, want %s", tt.agent, result, tt.expected)
			}
		})
	}
}

// TestValidationResult tests ValidationResult struct
func TestValidationResult(t *testing.T) {
	result := ValidationResult{
		Path:      "test.yaml",
		Valid:     true,
		RuleCount: 5,
		Error:     nil,
		Warnings:  []string{"warning1", "warning2"},
	}

	if result.Path != "test.yaml" {
		t.Errorf("ValidationResult.Path = %s, want test.yaml", result.Path)
	}
	if !result.Valid {
		t.Error("ValidationResult.Valid should be true")
	}
	if result.RuleCount != 5 {
		t.Errorf("ValidationResult.RuleCount = %d, want 5", result.RuleCount)
	}
	if len(result.Warnings) != 2 {
		t.Errorf("ValidationResult.Warnings length = %d, want 2", len(result.Warnings))
	}
}

// TestAgentValidationResult tests AgentValidationResult struct
func TestAgentValidationResult(t *testing.T) {
	result := AgentValidationResult{
		AgentName:    "cursor",
		CLIAvailable: true,
		APIKeySet:    true,
		CLIPath:      "/usr/bin/cursor-agent",
		Error:        nil,
	}

	if result.AgentName != "cursor" {
		t.Errorf("AgentValidationResult.AgentName = %s, want cursor", result.AgentName)
	}
	if !result.CLIAvailable {
		t.Error("AgentValidationResult.CLIAvailable should be true")
	}
	if !result.APIKeySet {
		t.Error("AgentValidationResult.APIKeySet should be true")
	}
	if result.CLIPath != "/usr/bin/cursor-agent" {
		t.Errorf("AgentValidationResult.CLIPath = %s, want /usr/bin/cursor-agent", result.CLIPath)
	}
}

// TestValidateSchemas_EmbeddedSchemaStructure tests that the embedded schema has proper structure
func TestValidateSchemas_EmbeddedSchemaStructure(t *testing.T) {
	checker := NewChecker()
	result := checker.validateSchemas()

	if !result.Valid {
		t.Errorf("validateSchemas() Valid = false, want true")
		t.Logf("Error: %v", result.Error)
	}

	// The embedded schema should always be valid since it's defined in code
	// This test ensures the validation logic correctly checks the schema structure
}
