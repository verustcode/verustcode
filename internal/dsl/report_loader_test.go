package dsl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReportLoaderValidate(t *testing.T) {
	loader := NewReportLoader()

	tests := []struct {
		name    string
		config  *ReportConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "missing id",
			config:  &ReportConfig{Name: "Test"},
			wantErr: true,
			errMsg:  "missing required field: id",
		},
		{
			name:    "missing name",
			config:  &ReportConfig{ID: "test"},
			wantErr: true,
			errMsg:  "missing required field: name",
		},
		{
			name: "valid config",
			config: &ReportConfig{
				ID:   "test",
				Name: "Test Report",
			},
			wantErr: false,
		},
		{
			name: "invalid heading_level too low",
			config: &ReportConfig{
				ID:   "test",
				Name: "Test Report",
				Output: ReportOutputConfig{
					Style: ReportStyleConfig{
						HeadingLevel: -1,
					},
				},
			},
			wantErr: true,
			errMsg:  "heading_level must be between 1 and 4",
		},
		{
			name: "invalid heading_level too high",
			config: &ReportConfig{
				ID:   "test",
				Name: "Test Report",
				Output: ReportOutputConfig{
					Style: ReportStyleConfig{
						HeadingLevel: 5,
					},
				},
			},
			wantErr: true,
			errMsg:  "heading_level must be between 1 and 4",
		},
		{
			name: "valid heading_level",
			config: &ReportConfig{
				ID:   "test",
				Name: "Test Report",
				Output: ReportOutputConfig{
					Style: ReportStyleConfig{
						HeadingLevel: 3,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid max_section_length too small",
			config: &ReportConfig{
				ID:   "test",
				Name: "Test Report",
				Output: ReportOutputConfig{
					Style: ReportStyleConfig{
						MaxSectionLength: 100,
					},
				},
			},
			wantErr: true,
			errMsg:  "max_section_length must be at least 500",
		},
		{
			name: "valid max_section_length",
			config: &ReportConfig{
				ID:   "test",
				Name: "Test Report",
				Output: ReportOutputConfig{
					Style: ReportStyleConfig{
						MaxSectionLength: 1000,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "too many structure reference_docs",
			config: &ReportConfig{
				ID:   "test",
				Name: "Test Report",
				Structure: StructurePhase{
					ReferenceDocs: make([]string, 11),
				},
			},
			wantErr: true,
			errMsg:  "structure.reference_docs exceeds maximum",
		},
		{
			name: "too many section reference_docs",
			config: &ReportConfig{
				ID:   "test",
				Name: "Test Report",
				Section: SectionPhase{
					ReferenceDocs: make([]string, 11),
				},
			},
			wantErr: true,
			errMsg:  "section.reference_docs exceeds maximum",
		},
		{
			name: "valid reference_docs count",
			config: &ReportConfig{
				ID:   "test",
				Name: "Test Report",
				Structure: StructurePhase{
					ReferenceDocs: make([]string, 10),
				},
				Section: SectionPhase{
					ReferenceDocs: make([]string, 10),
				},
			},
			wantErr: false,
		},
		{
			name: "invalid section.summary.max_length negative",
			config: &ReportConfig{
				ID:   "test",
				Name: "Test Report",
				Section: SectionPhase{
					Summary: SectionSummaryConfig{
						MaxLength: -1,
					},
				},
			},
			wantErr: true,
			errMsg:  "section.summary.max_length must be non-negative",
		},
		{
			name: "invalid section.summary.max_length too large",
			config: &ReportConfig{
				ID:   "test",
				Name: "Test Report",
				Section: SectionPhase{
					Summary: SectionSummaryConfig{
						MaxLength: 1001,
					},
				},
			},
			wantErr: true,
			errMsg:  "section.summary.max_length must not exceed 1000",
		},
		{
			name: "valid section.summary.max_length",
			config: &ReportConfig{
				ID:   "test",
				Name: "Test Report",
				Section: SectionPhase{
					Summary: SectionSummaryConfig{
						MaxLength: 500,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := loader.validate(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMsg != "" && !stringContains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestReportLoaderLoadFromBytes(t *testing.T) {
	loader := NewReportLoader()

	yaml := `
version: "1.0"
id: test
name: Test Report
description: A test report

agent:
  type: cursor

output:
  style:
    tone: professional
    concise: true
    no_emoji: true
    language: English
    use_mermaid: true
    heading_level: 2

structure:
  description: Generate structure
  goals:
    topics:
      - Topic 1
      - Topic 2
    avoid:
      - Avoid 1
  constraints:
    - Constraint 1
  reference_docs:
    - README.md

section:
  description: Generate section content
  goals:
    topics:
      - Section topic 1
    avoid:
      - Section avoid 1
  constraints:
    - Section constraint 1
`

	config, err := loader.LoadFromBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.ID != "test" {
		t.Errorf("expected id test, got %s", config.ID)
	}

	if config.Name != "Test Report" {
		t.Errorf("expected name Test Report, got %s", config.Name)
	}

	if config.Agent.Type != "cursor" {
		t.Errorf("expected agent type cursor, got %s", config.Agent.Type)
	}

	if config.Output.Style.Language != "English" {
		t.Errorf("expected output.style.language English, got %s", config.Output.Style.Language)
	}

	// Check structure goals
	if len(config.Structure.Goals.Topics) != 2 {
		t.Errorf("expected 2 structure topics, got %d", len(config.Structure.Goals.Topics))
	}

	if len(config.Structure.Goals.Avoid) != 1 {
		t.Errorf("expected 1 structure avoid, got %d", len(config.Structure.Goals.Avoid))
	}

	// Check section goals
	if len(config.Section.Goals.Topics) != 1 {
		t.Errorf("expected 1 section topic, got %d", len(config.Section.Goals.Topics))
	}

	if len(config.Section.Goals.Avoid) != 1 {
		t.Errorf("expected 1 section avoid, got %d", len(config.Section.Goals.Avoid))
	}

	// Check output.style (now in output.style instead of section.style)
	if config.Output.Style.Tone != "professional" {
		t.Errorf("expected tone professional, got %s", config.Output.Style.Tone)
	}

	if !config.Output.Style.GetUseMermaid() {
		t.Error("expected use_mermaid to be true")
	}

	if config.Output.Style.GetHeadingLevel() != 2 {
		t.Errorf("expected heading_level 2, got %d", config.Output.Style.GetHeadingLevel())
	}
}

func TestReportLoaderLoadFile(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")

	yaml := `
version: "1.0"
id: file-test
name: File Test Report

structure:
  description: Test structure

section:
  description: Test section
  style:
    tone: technical
`

	err := os.WriteFile(tmpFile, []byte(yaml), 0644)
	if err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	loader := NewReportLoader()
	config, err := loader.LoadFile(tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.ID != "file-test" {
		t.Errorf("expected id file-test, got %s", config.ID)
	}

	// Check that defaults were applied
	if config.Agent.Type != "cursor" {
		t.Errorf("expected agent type cursor (default), got %s", config.Agent.Type)
	}

	if config.Output.Style.Language != "Chinese" {
		t.Errorf("expected output.style.language Chinese (default), got %s", config.Output.Style.Language)
	}

	// Check that the config was registered
	retrieved, ok := loader.Get("file-test")
	if !ok {
		t.Error("expected config to be registered")
	}

	if retrieved.ID != config.ID {
		t.Error("retrieved config doesn't match")
	}
}

func TestReportLoaderListAndClear(t *testing.T) {
	loader := NewReportLoader()

	yaml1 := `
id: test1
name: Test 1
structure:
  description: Test
section:
  description: Test
`
	yaml2 := `
id: test2
name: Test 2
structure:
  description: Test
section:
  description: Test
`

	_, err := loader.LoadFromBytes([]byte(yaml1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = loader.LoadFromBytes([]byte(yaml2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test List
	configs := loader.List()
	if len(configs) != 2 {
		t.Errorf("expected 2 configs, got %d", len(configs))
	}

	// Test ListIDs
	ids := loader.ListIDs()
	if len(ids) != 2 {
		t.Errorf("expected 2 ids, got %d", len(ids))
	}

	// Test Clear
	loader.Clear()
	configs = loader.List()
	if len(configs) != 0 {
		t.Errorf("expected 0 configs after clear, got %d", len(configs))
	}
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
