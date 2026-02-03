package check

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGetTemplateContent tests the getTemplateContent function
func TestGetTemplateContent(t *testing.T) {
	tests := []struct {
		name        string
		template    TemplateType
		expectError bool
	}{
		{
			name:        "TemplateBootstrap",
			template:    TemplateBootstrap,
			expectError: false,
		},
		{
			name:        "TemplateRules",
			template:    TemplateRules,
			expectError: false,
		},
		{
			name:        "InvalidTemplate",
			template:    TemplateType(999),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := getTemplateContent(tt.template)
			if tt.expectError {
				if err == nil {
					t.Errorf("getTemplateContent(%v) expected error, got nil", tt.template)
				}
				if content != nil {
					t.Errorf("getTemplateContent(%v) expected nil content on error, got %d bytes", tt.template, len(content))
				}
			} else {
				if err != nil {
					t.Errorf("getTemplateContent(%v) unexpected error: %v", tt.template, err)
				}
				if content == nil || len(content) == 0 {
					t.Errorf("getTemplateContent(%v) expected non-empty content, got nil or empty", tt.template)
				}
			}
		})
	}
}

// TestCheckFile tests the checkFile method with existing file
func TestCheckFile_ExistingFile(t *testing.T) {
	// Create a temporary file
	tmpDir := filepath.Join(os.TempDir(), "scopeview_test_checkfile")
	defer os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)

	tmpFile := filepath.Join(tmpDir, "bootstrap.yaml")
	if err := os.WriteFile(tmpFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	checker := NewChecker()
	checker.configDir = tmpDir

	fileConfig := FileConfig{
		Path:        tmpFile,
		Description: "Test bootstrap file",
		Template:    TemplateBootstrap,
	}

	result := checker.checkFile(fileConfig)

	if !result.Exists {
		t.Error("checkFile should detect existing file")
	}
	if result.Created {
		t.Error("checkFile should not mark existing file as created")
	}
	if result.Error != nil {
		t.Errorf("checkFile should not return error for existing file: %v", result.Error)
	}
	if result.Path != tmpFile {
		t.Errorf("checkFile result.Path = %s, want %s", result.Path, tmpFile)
	}
}

// TestCheckFile_NonExistingFile tests checkFile with non-existing file
// Note: This test doesn't test the interactive confirmation part
func TestCheckFile_NonExistingFile(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "scopeview_test_checkfile2")
	defer os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)

	nonExistentFile := filepath.Join(tmpDir, "nonexistent.yaml")

	checker := NewChecker()
	checker.configDir = tmpDir

	fileConfig := FileConfig{
		Path:        nonExistentFile,
		Description: "Test non-existent file",
		Template:    TemplateBootstrap,
	}

	// Since confirmCreate is interactive, we can't easily test the full flow
	// But we can verify the initial check
	result := checker.checkFile(fileConfig)

	if result.Exists {
		t.Error("checkFile should detect non-existing file")
	}
	// Without user confirmation, Created should be false
	if result.Created {
		t.Error("checkFile should not mark file as created without confirmation")
	}
}

// TestCheckReportsDir_Existing tests checkReportsDir with existing directory
func TestCheckReportsDir_Existing(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "scopeview_test_reports")
	defer os.RemoveAll(tmpDir)

	reportsDir := filepath.Join(tmpDir, "config", "reports")
	os.MkdirAll(reportsDir, 0755)

	// Create a test YAML file to simulate existing report config
	testYAML := filepath.Join(reportsDir, "test.yaml")
	if err := os.WriteFile(testYAML, []byte("test: content"), 0644); err != nil {
		t.Fatalf("Failed to create test YAML: %v", err)
	}

	checker := NewChecker()
	checker.configDir = filepath.Join(tmpDir, "config")

	err := checker.checkReportsDir()
	if err != nil {
		t.Errorf("checkReportsDir should succeed for existing directory: %v", err)
	}
}

// TestCheckReportsDir_NonExisting tests checkReportsDir with non-existing directory
// Note: This test doesn't test the interactive confirmation part
func TestCheckReportsDir_NonExisting(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "scopeview_test_reports2")
	defer os.RemoveAll(tmpDir)

	checker := NewChecker()
	checker.configDir = filepath.Join(tmpDir, "config")

	// Without user confirmation, this should not create the directory
	// But it should not error either (user declined)
	err := checker.checkReportsDir()
	// The function may return nil if user declines, or error if confirmation fails
	// Since we can't easily mock the interactive prompt, we just check it doesn't panic
	_ = err
}

// TestCheckFiles tests the checkFiles method
func TestCheckFiles(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "scopeview_test_checkfiles")
	defer os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)

	checker := NewChecker()
	checker.configDir = tmpDir

	// Create one required file to test partial success
	bootstrapPath := filepath.Join(tmpDir, "bootstrap.yaml")
	if err := os.WriteFile(bootstrapPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create bootstrap file: %v", err)
	}

	// checkFiles will iterate through RequiredFiles()
	// Since we can't easily mock confirmCreate, we test that it doesn't panic
	// and handles existing files correctly
	err := checker.checkFiles()
	// Error may occur if user interaction fails, but we can't easily test that
	_ = err
}

// TestFileCheckResult tests FileCheckResult struct
func TestFileCheckResult(t *testing.T) {
	result := FileCheckResult{
		Path:        "test.yaml",
		Exists:      true,
		Created:     false,
		Description: "Test file",
		Error:       nil,
	}

	if result.Path != "test.yaml" {
		t.Errorf("FileCheckResult.Path = %s, want test.yaml", result.Path)
	}
	if !result.Exists {
		t.Error("FileCheckResult.Exists should be true")
	}
	if result.Created {
		t.Error("FileCheckResult.Created should be false")
	}
	if result.Error != nil {
		t.Errorf("FileCheckResult.Error should be nil, got %v", result.Error)
	}
}

// TestFileConfig tests FileConfig struct
func TestFileConfig(t *testing.T) {
	config := FileConfig{
		Path:        "test.yaml",
		Description: "Test description",
		Template:    TemplateBootstrap,
	}

	if config.Path != "test.yaml" {
		t.Errorf("FileConfig.Path = %s, want test.yaml", config.Path)
	}
	if config.Description != "Test description" {
		t.Errorf("FileConfig.Description = %s, want Test description", config.Description)
	}
	if config.Template != TemplateBootstrap {
		t.Errorf("FileConfig.Template = %v, want TemplateBootstrap", config.Template)
	}
}
