package check

import (
	"testing"
)

func TestNewReport(t *testing.T) {
	report := NewReport()
	if report == nil {
		t.Fatal("NewReport() returned nil")
	}
	if report.FileResults == nil {
		t.Error("NewReport() FileResults should be initialized")
	}
	if report.ValidationResults == nil {
		t.Error("NewReport() ValidationResults should be initialized")
	}
}

func TestAddFileResult(t *testing.T) {
	report := NewReport()
	result := FileCheckResult{
		Path:   "test.yaml",
		Exists: true,
	}

	report.AddFileResult(result)

	if len(report.FileResults) != 1 {
		t.Errorf("AddFileResult() added %d results, want 1", len(report.FileResults))
	}
	if report.FileResults[0].Path != "test.yaml" {
		t.Errorf("AddFileResult() path = %q, want %q", report.FileResults[0].Path, "test.yaml")
	}
}

func TestAddValidationResult(t *testing.T) {
	report := NewReport()
	result := ValidationResult{
		Path:  "test.yaml",
		Valid: true,
	}

	report.AddValidationResult(result)

	if len(report.ValidationResults) != 1 {
		t.Errorf("AddValidationResult() added %d results, want 1", len(report.ValidationResults))
	}
}

func TestCalculateSummary(t *testing.T) {
	report := NewReport()

	// Add file results
	report.AddFileResult(FileCheckResult{Path: "file1.yaml", Exists: true})
	report.AddFileResult(FileCheckResult{Path: "file2.yaml", Exists: false})
	report.AddFileResult(FileCheckResult{Path: "file3.yaml", Exists: true, Created: true})

	// Add validation results
	report.AddValidationResult(ValidationResult{Path: "valid.yaml", Valid: true})
	report.AddValidationResult(ValidationResult{Path: "invalid.yaml", Valid: false})

	summary := report.calculateSummary()

	if summary.TotalFiles != 3 {
		t.Errorf("calculateSummary() TotalFiles = %d, want 3", summary.TotalFiles)
	}
	if summary.FilesExist != 2 {
		t.Errorf("calculateSummary() FilesExist = %d, want 2", summary.FilesExist)
	}
	if summary.FilesCreated != 1 {
		t.Errorf("calculateSummary() FilesCreated = %d, want 1", summary.FilesCreated)
	}
	if summary.FilesMissing != 1 {
		t.Errorf("calculateSummary() FilesMissing = %d, want 1", summary.FilesMissing)
	}
	if summary.TotalValidations != 2 {
		t.Errorf("calculateSummary() TotalValidations = %d, want 2", summary.TotalValidations)
	}
	if summary.ValidationsValid != 1 {
		t.Errorf("calculateSummary() ValidationsValid = %d, want 1", summary.ValidationsValid)
	}
	if summary.ValidationErrors != 1 {
		t.Errorf("calculateSummary() ValidationErrors = %d, want 1", summary.ValidationErrors)
	}
}

func TestCalculateSummary_WithErrors(t *testing.T) {
	report := NewReport()

	// Add file result with error
	report.AddFileResult(FileCheckResult{
		Path:  "error.yaml",
		Error: &mockError{msg: "file error"},
	})

	// Add validation result with error
	report.AddValidationResult(ValidationResult{
		Path:  "invalid.yaml",
		Valid: false,
		Error: &mockError{msg: "validation error"},
	})

	summary := report.calculateSummary()

	if !summary.HasErrors {
		t.Error("calculateSummary() HasErrors = false, want true")
	}
}

func TestCalculateSummary_WithWarnings(t *testing.T) {
	report := NewReport()

	report.AddValidationResult(ValidationResult{
		Path:     "warning.yaml",
		Valid:    true,
		Warnings: []string{"warning 1", "warning 2"},
	})

	summary := report.calculateSummary()

	if !summary.HasWarnings {
		t.Error("calculateSummary() HasWarnings = false, want true")
	}
}

func TestPrintDetailedReport(t *testing.T) {
	r := NewReport()

	// Add some results
	r.AddFileResult(FileCheckResult{Path: "test.yaml", Exists: true})
	r.AddValidationResult(ValidationResult{Path: "test.yaml", Valid: true})

	// Should not panic
	r.PrintDetailedReport()
}

func TestPrintSeparator(t *testing.T) {
	r := NewReport()
	// Should not panic
	r.printSeparator()
}

func TestPrintSummary(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Report)
		wantErr bool
	}{
		{
			name: "all passed",
			setup: func(r *Report) {
				r.AddFileResult(FileCheckResult{Exists: true})
				r.AddValidationResult(ValidationResult{Valid: true})
			},
		},
		{
			name: "with errors",
			setup: func(r *Report) {
				r.AddFileResult(FileCheckResult{Error: &mockError{msg: "error"}})
			},
		},
		{
			name: "with warnings",
			setup: func(r *Report) {
				r.AddValidationResult(ValidationResult{
					Valid:    true,
					Warnings: []string{"warning"},
				})
			},
		},
		{
			name: "with missing files",
			setup: func(r *Report) {
				r.AddFileResult(FileCheckResult{Exists: false})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReport()
			tt.setup(r)
			summary := r.calculateSummary()
			// Should not panic
			r.printSummary(summary)
		})
	}
}

func TestPrintFileSection(t *testing.T) {
	r := NewReport()

	r.AddFileResult(FileCheckResult{
		Path:    "test.yaml",
		Exists:  true,
		Created: false,
	})
	r.AddFileResult(FileCheckResult{
		Path:    "new.yaml",
		Exists:  true,
		Created: true,
	})
	r.AddFileResult(FileCheckResult{
		Path:   "missing.yaml",
		Exists: false,
	})
	r.AddFileResult(FileCheckResult{
		Path:  "error.yaml",
		Error: &mockError{msg: "file error"},
	})

	// Should not panic
	r.printFileSection()
}

func TestPrintValidationSection(t *testing.T) {
	r := NewReport()

	r.AddValidationResult(ValidationResult{
		Path:      "valid.yaml",
		Valid:     true,
		RuleCount: 5,
	})
	r.AddValidationResult(ValidationResult{
		Path:  "invalid.yaml",
		Valid: false,
		Error: &mockError{msg: "validation error"},
	})
	r.AddValidationResult(ValidationResult{
		Path:     "warning.yaml",
		Valid:    true,
		Warnings: []string{"warning 1"},
	})

	// Should not panic
	r.printValidationSection()
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

