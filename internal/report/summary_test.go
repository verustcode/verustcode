// Package report provides report generation functionality.
// This file contains unit tests for report summary generator.
package report

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/dsl"
	"github.com/verustcode/verustcode/internal/model"
)

// TestNewSummaryGenerator tests NewSummaryGenerator
func TestNewSummaryGenerator(t *testing.T) {
	configProvider := &mockConfigProvider{}

	gen := NewSummaryGenerator(nil, configProvider)
	require.NotNil(t, gen)
	assert.Equal(t, configProvider, gen.configProvider)
}

// TestSummaryGenerator_GetReportConfig tests getReportConfig
func TestSummaryGenerator_GetReportConfig(t *testing.T) {
	tests := []struct {
		name           string
		configProvider config.ConfigProvider
		wantErr        bool
	}{
		{
			name: "success",
			configProvider: &mockConfigProvider{
				reportConfig: &config.ReportConfig{
					MaxRetries: 3,
				},
			},
		},
		{
			name: "error",
			configProvider: &mockConfigProvider{
				err: errors.New("config error"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewSummaryGenerator(nil, tt.configProvider)
			cfg, err := gen.getReportConfig()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
			}
		})
	}
}

// TestSummaryGenerator_CollectSectionSummaries tests collectSectionSummaries
func TestSummaryGenerator_CollectSectionSummaries(t *testing.T) {
	gen := NewSummaryGenerator(nil, &mockConfigProvider{})

	structure := &ReportStructure{
		Sections: []GeneratedSection{
			{ID: "section-1", Title: "Section 1"},
			{ID: "section-2", Title: "Section 2"},
			{ID: "section-3", Title: "Section 3"},
		},
	}

	sections := []model.ReportSection{
		{
			SectionID: "section-1",
			Title:     "Section 1",
			Summary:   "Summary 1",
		},
		{
			SectionID: "section-2",
			Title:     "Section 2",
			Summary:   "Summary 2",
		},
		{
			SectionID: "section-3",
			Title:     "Section 3",
			Summary:   "", // Empty summary
		},
	}

	summaries := gen.collectSectionSummaries(sections, structure)

	assert.Len(t, summaries, 2) // Only non-empty summaries
	found1 := false
	found2 := false
	for _, item := range summaries {
		if item.Summary == "Summary 1" {
			found1 = true
		}
		if item.Summary == "Summary 2" {
			found2 = true
		}
	}
	assert.True(t, found1)
	assert.True(t, found2)
}

// TestSummaryGenerator_CollectSectionSummaries_Empty tests with empty sections
func TestSummaryGenerator_CollectSectionSummaries_Empty(t *testing.T) {
	gen := NewSummaryGenerator(nil, &mockConfigProvider{})

	structure := &ReportStructure{Sections: []GeneratedSection{}}
	summaries := gen.collectSectionSummaries([]model.ReportSection{}, structure)
	assert.Empty(t, summaries)
}

// TestSummaryGenerator_CollectSectionSummaries_AllEmpty tests with all empty summaries
func TestSummaryGenerator_CollectSectionSummaries_AllEmpty(t *testing.T) {
	gen := NewSummaryGenerator(nil, &mockConfigProvider{})

	structure := &ReportStructure{
		Sections: []GeneratedSection{
			{ID: "s1"},
			{ID: "s2"},
		},
	}

	sections := []model.ReportSection{
		{SectionID: "s1", Summary: ""},
		{SectionID: "s2", Summary: "   "}, // Whitespace only
	}

	summaries := gen.collectSectionSummaries(sections, structure)
	assert.Empty(t, summaries)
}

// TestSummaryGenerator_BuildSummaryPrompt tests buildSummaryPrompt
func TestSummaryGenerator_BuildSummaryPrompt(t *testing.T) {
	gen := NewSummaryGenerator(nil, &mockConfigProvider{
		reportConfig: &config.ReportConfig{
			OutputLanguage: "en-US",
		},
	})

	typeDef := &ReportTypeDefinition{
		ID:          "test-report",
		Name:        "Test Report",
		Description: "A test report",
		Config: &dsl.ReportConfig{
			Summary: dsl.SummaryPhase{
				Description: "You are a summary generator",
				Goals: dsl.ReportGoalsConfig{
					Topics: []string{"topic1"},
					Avoid:  []string{"avoid1"},
				},
				Constraints: []string{"constraint1"},
			},
			Output: dsl.ReportOutputConfig{
				Style: dsl.ReportStyleConfig{
					Language: "en-US",
				},
			},
		},
	}

	structure := &ReportStructure{
		Title:   "Test Report",
		Summary: "Structure summary",
		Sections: []GeneratedSection{
			{ID: "s1", Title: "Section 1"},
		},
	}

	sectionSummaries := []sectionSummaryItem{
		{Index: "1", Title: "Section 1", Summary: "Summary 1"},
		{Index: "2", Title: "Section 2", Summary: "Summary 2"},
	}

	prompt := gen.buildSummaryPrompt(typeDef, typeDef.Config, structure, sectionSummaries)

	assert.Contains(t, prompt, "Role")
	assert.Contains(t, prompt, "summary generator")
	assert.Contains(t, prompt, "Report Information")
	assert.Contains(t, prompt, "Test Report")
	assert.Contains(t, prompt, "Section Summaries")
	assert.Contains(t, prompt, "Summary 1")
	assert.Contains(t, prompt, "Summary 2")
	assert.Contains(t, prompt, "Section 1")
	assert.Contains(t, prompt, "Section 2")
	assert.Contains(t, prompt, "Focus Topics")
	assert.Contains(t, prompt, "topic1")
	assert.Contains(t, prompt, "Things to Avoid")
	assert.Contains(t, prompt, "avoid1")
	assert.Contains(t, prompt, "Constraints")
	assert.Contains(t, prompt, "constraint1")
	assert.Contains(t, prompt, "Output Requirements")
}

// TestSummaryGenerator_BuildSummaryPrompt_NoGoals tests prompt without goals
func TestSummaryGenerator_BuildSummaryPrompt_NoGoals(t *testing.T) {
	gen := NewSummaryGenerator(nil, &mockConfigProvider{})

	typeDef := &ReportTypeDefinition{
		ID:   "test",
		Name: "Test",
		Config: &dsl.ReportConfig{
			Summary: dsl.SummaryPhase{
				Description: "Generator",
			},
		},
	}

	structure := &ReportStructure{
		Title:    "Test",
		Summary:  "Summary",
		Sections: []GeneratedSection{},
	}

	prompt := gen.buildSummaryPrompt(typeDef, typeDef.Config, structure, []sectionSummaryItem{})

	assert.NotContains(t, prompt, "Focus Topics")
	assert.NotContains(t, prompt, "Things to Avoid")
}

// TestSummaryGenerator_PostProcessSummary tests postProcessSummary
func TestSummaryGenerator_PostProcessSummary(t *testing.T) {
	gen := NewSummaryGenerator(nil, &mockConfigProvider{})

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean summary",
			input:    "This is a clean summary.",
			expected: "This is a clean summary.",
		},
		{
			name:     "markdown fence wrapper",
			input:    "```markdown\nSummary content.\n```",
			expected: "Summary content.",
		},
		{
			name:     "remove leading meta",
			input:    "Here is the summary:\n\nActual summary content.",
			expected: "Actual summary content.",
		},
		{
			name:     "remove trailing patterns",
			input:    "Summary content.\n\n文档已保存",
			expected: "Summary content.",
		},
		{
			name:     "preserve internal formatting",
			input:    "Summary with **bold** and *italic*.",
			expected: "Summary with **bold** and *italic*.",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.postProcessSummary(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSummaryGenerator_BuildSummaryPrompt_OutputLanguage tests output language handling
func TestSummaryGenerator_BuildSummaryPrompt_OutputLanguage(t *testing.T) {
	tests := []struct {
		name           string
		configLanguage string
		dbLanguage     string
		expectLanguage string
	}{
		{
			name:           "config language takes priority",
			configLanguage: "zh-CN",
			dbLanguage:     "en-US",
			expectLanguage: "Chinese",
		},
		{
			name:           "fallback to db language",
			configLanguage: "",
			dbLanguage:     "en-US",
			expectLanguage: "English",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewSummaryGenerator(nil, &mockConfigProvider{
				reportConfig: &config.ReportConfig{
					OutputLanguage: tt.dbLanguage,
				},
			})

			typeDef := &ReportTypeDefinition{
				ID:   "test",
				Name: "Test",
				Config: &dsl.ReportConfig{
					Summary: dsl.SummaryPhase{},
					Output: dsl.ReportOutputConfig{
						Style: dsl.ReportStyleConfig{
							Language: tt.configLanguage,
						},
					},
				},
			}

			structure := &ReportStructure{
				Title:    "Test",
				Summary:  "Summary",
				Sections: []GeneratedSection{},
			}

			prompt := gen.buildSummaryPrompt(typeDef, typeDef.Config, structure, []sectionSummaryItem{})

			if tt.expectLanguage != "" {
				assert.Contains(t, prompt, tt.expectLanguage)
			}
		})
	}
}

// TestSummaryGenerator_CollectSectionSummaries_Truncation tests summary truncation
func TestSummaryGenerator_CollectSectionSummaries_Truncation(t *testing.T) {
	gen := NewSummaryGenerator(nil, &mockConfigProvider{})

	structure := &ReportStructure{
		Sections: []GeneratedSection{
			{ID: "section-1", Title: "Section 1"},
		},
	}

	longSummary := strings.Repeat("word ", 100)
	sections := []model.ReportSection{
		{
			SectionID: "section-1",
			Summary:   longSummary,
		},
	}

	summaries := gen.collectSectionSummaries(sections, structure)
	assert.Len(t, summaries, 1)
	// Should not truncate during collection (truncation happens in prompt building if needed)
	assert.Contains(t, summaries[0].Summary, "word")
}

// TestSummaryGenerator_BuildSummaryPrompt_EmptySummaries tests with empty section summaries
func TestSummaryGenerator_BuildSummaryPrompt_EmptySummaries(t *testing.T) {
	gen := NewSummaryGenerator(nil, &mockConfigProvider{})

	typeDef := &ReportTypeDefinition{
		ID:   "test",
		Name: "Test",
		Config: &dsl.ReportConfig{
			Summary: dsl.SummaryPhase{},
		},
	}

	structure := &ReportStructure{
		Title:   "Test",
		Summary: "Summary",
		Sections: []GeneratedSection{
			{ID: "s1", Title: "Section 1"},
		},
	}

	prompt := gen.buildSummaryPrompt(typeDef, typeDef.Config, structure, []sectionSummaryItem{})

	// Should still build prompt even with no summaries
	assert.Contains(t, prompt, "Report Information")
	assert.Contains(t, prompt, "Test")
}

// TestSummaryGenerator_PostProcessSummary_ComplexMarkdown tests complex markdown handling
func TestSummaryGenerator_PostProcessSummary_ComplexMarkdown(t *testing.T) {
	gen := NewSummaryGenerator(nil, &mockConfigProvider{})

	input := "```markdown\n## Summary\n\nThis is a **summary** with *formatting*.\n\n- Item 1\n- Item 2\n\n```\ncode block\n```\n```"

	result := gen.postProcessSummary(input)
	// Should remove outer fence but preserve internal formatting
	assert.Contains(t, result, "## Summary")
	assert.Contains(t, result, "**summary**")
	assert.Contains(t, result, "*formatting*")
	assert.NotContains(t, result, "```markdown")
}
