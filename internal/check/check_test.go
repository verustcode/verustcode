package check

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewChecker tests the NewChecker function
func TestNewChecker(t *testing.T) {
	checker := NewChecker()
	if checker == nil {
		t.Fatal("NewChecker returned nil")
	}
	if checker.configDir != "config" {
		t.Errorf("Expected configDir 'config', got '%s'", checker.configDir)
	}
	if checker.report == nil {
		t.Error("Report should be initialized")
	}
}

// TestRequiredFiles tests the RequiredFiles method
func TestRequiredFiles(t *testing.T) {
	checker := NewChecker()
	files := checker.RequiredFiles()
	
	if len(files) != 2 {
		t.Errorf("Expected 2 required files, got %d", len(files))
	}
	
	// Check first file is bootstrap.yaml
	if files[0].Path != "config/bootstrap.yaml" {
		t.Errorf("First file should be config/bootstrap.yaml, got %s", files[0].Path)
	}
}

// TestFileExists tests the fileExists function
func TestFileExists(t *testing.T) {
	// Test with existing file
	tmpFile := filepath.Join(os.TempDir(), "test_exists.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)
	
	if !fileExists(tmpFile) {
		t.Error("fileExists should return true for existing file")
	}
	
	// Test with non-existing file
	if fileExists("/non/existent/file.txt") {
		t.Error("fileExists should return false for non-existing file")
	}
}

// TestEnsureDir tests the ensureDir function
func TestEnsureDir(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "scopeview_test_dir", "subdir")
	defer os.RemoveAll(filepath.Join(os.TempDir(), "scopeview_test_dir"))
	
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := ensureDir(testFile); err != nil {
		t.Errorf("ensureDir failed: %v", err)
	}
	
	// Check directory was created
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("Directory should have been created")
	}
}

// TestReport tests the Report struct
func TestReport(t *testing.T) {
	report := NewReport()
	
	// Add file result
	report.AddFileResult(FileCheckResult{
		Path:   "test.yaml",
		Exists: true,
	})
	
	if len(report.FileResults) != 1 {
		t.Error("File result should be added")
	}
	
	// Add validation result
	report.AddValidationResult(ValidationResult{
		Path:  "test.yaml",
		Valid: true,
	})
	
	if len(report.ValidationResults) != 1 {
		t.Error("Validation result should be added")
	}
}

// TestReportSummary tests the report summary calculation
func TestReportSummary(t *testing.T) {
	report := NewReport()

	report.AddFileResult(FileCheckResult{Path: "a.yaml", Exists: true})
	report.AddFileResult(FileCheckResult{Path: "b.yaml", Exists: false})
	report.AddFileResult(FileCheckResult{Path: "c.yaml", Created: true, Exists: true})

	report.AddValidationResult(ValidationResult{Path: "a.yaml", Valid: true})
	report.AddValidationResult(ValidationResult{Path: "b.yaml", Valid: false})

	summary := report.calculateSummary()

	if summary.TotalFiles != 3 {
		t.Errorf("TotalFiles = %d, want 3", summary.TotalFiles)
	}
	if summary.FilesExist != 2 {
		t.Errorf("FilesExist = %d, want 2", summary.FilesExist)
	}
	if summary.FilesCreated != 1 {
		t.Errorf("FilesCreated = %d, want 1", summary.FilesCreated)
	}
	if summary.FilesMissing != 1 {
		t.Errorf("FilesMissing = %d, want 1", summary.FilesMissing)
	}
	if summary.TotalValidations != 2 {
		t.Errorf("TotalValidations = %d, want 2", summary.TotalValidations)
	}
	if summary.ValidationsValid != 1 {
		t.Errorf("ValidationsValid = %d, want 1", summary.ValidationsValid)
	}
}
