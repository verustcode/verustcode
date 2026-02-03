package dsl

import (
	"testing"
)

func TestDefaultReportConfig(t *testing.T) {
	config := DefaultReportConfig()

	if config.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", config.Version)
	}

	if config.Agent.Type != "cursor" {
		t.Errorf("expected agent type cursor, got %s", config.Agent.Type)
	}

	if config.Output.Style.Language != "Chinese" {
		t.Errorf("expected output.style.language Chinese, got %s", config.Output.Style.Language)
	}

	// Check structure defaults
	if config.Structure.Description == "" {
		t.Error("expected structure description to be set")
	}

	// Check section defaults
	if config.Section.Description == "" {
		t.Error("expected section description to be set")
	}

	// Check output.style defaults
	if config.Output.Style.Tone != "professional" {
		t.Errorf("expected tone professional, got %s", config.Output.Style.Tone)
	}

	if !config.Output.Style.GetUseMermaid() {
		t.Error("expected use_mermaid to be true")
	}

	if !config.Output.Style.GetConcise() {
		t.Error("expected concise to be true")
	}

	if !config.Output.Style.GetNoEmoji() {
		t.Error("expected no_emoji to be true")
	}

	if config.Output.Style.GetHeadingLevel() != 2 {
		t.Errorf("expected heading_level 2, got %d", config.Output.Style.GetHeadingLevel())
	}

	if !config.Output.Style.GetIncludeLineNumbers() {
		t.Error("expected include_line_numbers to be true")
	}
}

func TestReportConfigApplyDefaults(t *testing.T) {
	config := &ReportConfig{
		ID:   "test",
		Name: "Test Report",
	}

	config.ApplyDefaults()

	if config.Agent.Type != "cursor" {
		t.Errorf("expected agent type cursor, got %s", config.Agent.Type)
	}

	if config.Output.Style.Language != "Chinese" {
		t.Errorf("expected output.style.language Chinese, got %s", config.Output.Style.Language)
	}

	if config.Structure.Description == "" {
		t.Error("expected structure description to be set")
	}

	if config.Section.Description == "" {
		t.Error("expected section description to be set")
	}

	if config.Output.Style.Tone != "professional" {
		t.Errorf("expected tone professional, got %s", config.Output.Style.Tone)
	}
}

func TestReportConfigApplyDefaultsPreservesExisting(t *testing.T) {
	config := &ReportConfig{
		ID:    "test",
		Name:  "Test Report",
		Agent: AgentConfig{Type: "gemini"},
		Output: ReportOutputConfig{
			Style: ReportStyleConfig{
				Language: "English",
				Tone:     "strict",
			},
		},
		Structure: StructurePhase{
			Description: "Custom structure description",
		},
		Section: SectionPhase{
			Description: "Custom section description",
		},
	}

	config.ApplyDefaults()

	// Should preserve existing values
	if config.Agent.Type != "gemini" {
		t.Errorf("expected agent type gemini, got %s", config.Agent.Type)
	}

	if config.Output.Style.Language != "English" {
		t.Errorf("expected output.style.language English, got %s", config.Output.Style.Language)
	}

	if config.Structure.Description != "Custom structure description" {
		t.Errorf("expected custom structure description, got %s", config.Structure.Description)
	}

	if config.Section.Description != "Custom section description" {
		t.Errorf("expected custom section description, got %s", config.Section.Description)
	}

	if config.Output.Style.Tone != "strict" {
		t.Errorf("expected tone strict, got %s", config.Output.Style.Tone)
	}
}

func TestReportStyleConfigGetters(t *testing.T) {
	// Test with nil values (should return defaults)
	style := &ReportStyleConfig{}

	if !style.GetUseMermaid() {
		t.Error("GetUseMermaid should return true by default")
	}

	if !style.GetConcise() {
		t.Error("GetConcise should return true by default")
	}

	if !style.GetNoEmoji() {
		t.Error("GetNoEmoji should return true by default")
	}

	if style.GetHeadingLevel() != 2 {
		t.Errorf("GetHeadingLevel should return 2 by default, got %d", style.GetHeadingLevel())
	}

	if !style.GetIncludeLineNumbers() {
		t.Error("GetIncludeLineNumbers should return true by default")
	}

	if style.GetLanguage() != "Chinese" {
		t.Errorf("GetLanguage should return Chinese by default, got %s", style.GetLanguage())
	}

	// Test with explicit false values
	falseVal := false
	style2 := &ReportStyleConfig{
		UseMermaid:         &falseVal,
		Concise:            &falseVal,
		NoEmoji:            &falseVal,
		HeadingLevel:       3,
		IncludeLineNumbers: &falseVal,
		Language:           "English",
	}

	if style2.GetUseMermaid() {
		t.Error("GetUseMermaid should return false when set to false")
	}

	if style2.GetConcise() {
		t.Error("GetConcise should return false when set to false")
	}

	if style2.GetNoEmoji() {
		t.Error("GetNoEmoji should return false when set to false")
	}

	if style2.GetHeadingLevel() != 3 {
		t.Errorf("GetHeadingLevel should return 3, got %d", style2.GetHeadingLevel())
	}

	if style2.GetIncludeLineNumbers() {
		t.Error("GetIncludeLineNumbers should return false when set to false")
	}

	if style2.GetLanguage() != "English" {
		t.Errorf("GetLanguage should return English, got %s", style2.GetLanguage())
	}
}

func TestReportGoalsConfig(t *testing.T) {
	goals := ReportGoalsConfig{
		Topics: []string{"topic1", "topic2"},
		Avoid:  []string{"avoid1", "avoid2"},
	}

	if len(goals.Topics) != 2 {
		t.Errorf("expected 2 topics, got %d", len(goals.Topics))
	}

	if len(goals.Avoid) != 2 {
		t.Errorf("expected 2 avoid items, got %d", len(goals.Avoid))
	}

	if goals.Topics[0] != "topic1" {
		t.Errorf("expected topic1, got %s", goals.Topics[0])
	}

	if goals.Avoid[0] != "avoid1" {
		t.Errorf("expected avoid1, got %s", goals.Avoid[0])
	}
}

func TestSectionSummaryConfigGetMaxLength(t *testing.T) {
	// Test zero value returns default (200)
	cfg := &SectionSummaryConfig{}
	if cfg.GetMaxLength() != 200 {
		t.Errorf("expected default 200, got %d", cfg.GetMaxLength())
	}

	// Test custom value is returned
	cfg2 := &SectionSummaryConfig{MaxLength: 500}
	if cfg2.GetMaxLength() != 500 {
		t.Errorf("expected 500, got %d", cfg2.GetMaxLength())
	}

	// Test smaller custom value
	cfg3 := &SectionSummaryConfig{MaxLength: 100}
	if cfg3.GetMaxLength() != 100 {
		t.Errorf("expected 100, got %d", cfg3.GetMaxLength())
	}
}

func TestSummaryPhaseDefaults(t *testing.T) {
	config := DefaultReportConfig()

	// Check that Summary phase has default description
	if config.Summary.Description == "" {
		t.Error("expected summary description to be set by default")
	}

	// Verify specific default value
	expectedDesc := "基于各章节摘要，生成项目的整体概览。"
	if config.Summary.Description != expectedDesc {
		t.Errorf("expected summary description %q, got %q", expectedDesc, config.Summary.Description)
	}

	// Goals and Constraints should be empty by default
	if len(config.Summary.Goals.Topics) != 0 {
		t.Errorf("expected no default topics, got %d", len(config.Summary.Goals.Topics))
	}

	if len(config.Summary.Goals.Avoid) != 0 {
		t.Errorf("expected no default avoid items, got %d", len(config.Summary.Goals.Avoid))
	}

	if len(config.Summary.Constraints) != 0 {
		t.Errorf("expected no default constraints, got %d", len(config.Summary.Constraints))
	}
}

func TestReportConfigApplyDefaultsWithSummary(t *testing.T) {
	// Test that ApplyDefaults sets Section.Summary.MaxLength
	config := &ReportConfig{
		ID:   "test",
		Name: "Test Report",
	}
	config.ApplyDefaults()

	if config.Section.Summary.MaxLength != 200 {
		t.Errorf("expected section summary max_length 200, got %d", config.Section.Summary.MaxLength)
	}

	// Test that ApplyDefaults sets Summary.Description
	if config.Summary.Description == "" {
		t.Error("expected summary description to be set by ApplyDefaults")
	}

	// Test that ApplyDefaults preserves existing Section.Summary.MaxLength
	config2 := &ReportConfig{
		ID:   "test",
		Name: "Test Report",
		Section: SectionPhase{
			Summary: SectionSummaryConfig{
				MaxLength: 500,
			},
		},
	}
	config2.ApplyDefaults()

	if config2.Section.Summary.MaxLength != 500 {
		t.Errorf("expected preserved section summary max_length 500, got %d", config2.Section.Summary.MaxLength)
	}

	// Test that ApplyDefaults preserves existing Summary.Description
	config3 := &ReportConfig{
		ID:   "test",
		Name: "Test Report",
		Summary: SummaryPhase{
			Description: "Custom summary description",
		},
	}
	config3.ApplyDefaults()

	if config3.Summary.Description != "Custom summary description" {
		t.Errorf("expected preserved summary description, got %q", config3.Summary.Description)
	}
}

func TestSummaryPhaseConfiguration(t *testing.T) {
	// Test full SummaryPhase configuration
	summary := SummaryPhase{
		Description: "Generate executive summary",
		Goals: ReportGoalsConfig{
			Topics: []string{"Key findings", "Recommendations"},
			Avoid:  []string{"Technical jargon", "Implementation details"},
		},
		Constraints: []string{"Max 500 words", "Use bullet points"},
	}

	if summary.Description != "Generate executive summary" {
		t.Errorf("expected description 'Generate executive summary', got %q", summary.Description)
	}

	if len(summary.Goals.Topics) != 2 {
		t.Errorf("expected 2 topics, got %d", len(summary.Goals.Topics))
	}

	if len(summary.Goals.Avoid) != 2 {
		t.Errorf("expected 2 avoid items, got %d", len(summary.Goals.Avoid))
	}

	if len(summary.Constraints) != 2 {
		t.Errorf("expected 2 constraints, got %d", len(summary.Constraints))
	}
}

func TestStructurePhaseNestedField(t *testing.T) {
	// Test default value (should be false)
	config := DefaultReportConfig()
	if config.Structure.Nested {
		t.Error("expected structure.nested to be false by default")
	}

	// Test explicit true value
	config2 := &ReportConfig{
		Structure: StructurePhase{
			Nested: true,
		},
	}
	config2.ApplyDefaults()

	if !config2.Structure.Nested {
		t.Error("expected structure.nested to remain true after ApplyDefaults")
	}
}
