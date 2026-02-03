// Package report provides report generation functionality.
// This file contains unit tests for report section generator.
package report

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/dsl"
	"github.com/verustcode/verustcode/internal/model"
)

// mockConfigProvider is a simple mock for ConfigProvider
type mockConfigProvider struct {
	reportConfig *config.ReportConfig
	err          error
}

func (m *mockConfigProvider) GetReportConfig() (*config.ReportConfig, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.reportConfig == nil {
		return &config.ReportConfig{
			MaxRetries:     DefaultReportMaxRetries,
			RetryDelay:     10,
			OutputLanguage: "en-US",
		}, nil
	}
	return m.reportConfig, nil
}

func (m *mockConfigProvider) GetReviewConfig() (*config.ReviewConfig, error) {
	return nil, nil
}

func (m *mockConfigProvider) GetGitProviders() ([]config.ProviderConfig, error) {
	return nil, nil
}

func (m *mockConfigProvider) GetAgents() (map[string]config.AgentDetail, error) {
	return nil, nil
}

func (m *mockConfigProvider) GetNotificationConfig() (*config.NotificationConfig, error) {
	return nil, nil
}

// TestNewSectionGenerator_Basic tests basic NewSectionGenerator creation
func TestNewSectionGenerator_Basic(t *testing.T) {
	configProvider := &mockConfigProvider{}

	gen := NewSectionGenerator(nil, configProvider)
	require.NotNil(t, gen)
	assert.Equal(t, configProvider, gen.configProvider)
}

// TestSectionGenerator_GetReportConfig tests getReportConfig
func TestSectionGenerator_GetReportConfig(t *testing.T) {
	tests := []struct {
		name           string
		configProvider config.ConfigProvider
		wantErr        bool
		checkConfig    func(*testing.T, *config.ReportConfig)
	}{
		{
			name: "success",
			configProvider: &mockConfigProvider{
				reportConfig: &config.ReportConfig{
					MaxRetries:     5,
					RetryDelay:     20,
					OutputLanguage: "zh-CN",
				},
			},
			checkConfig: func(t *testing.T, cfg *config.ReportConfig) {
				assert.Equal(t, 5, cfg.MaxRetries)
				assert.Equal(t, 20, cfg.RetryDelay)
				assert.Equal(t, "zh-CN", cfg.OutputLanguage)
			},
		},
		{
			name: "default values",
			configProvider: &mockConfigProvider{
				reportConfig: nil, // Will return defaults
			},
			checkConfig: func(t *testing.T, cfg *config.ReportConfig) {
				assert.Equal(t, DefaultReportMaxRetries, cfg.MaxRetries)
				assert.Equal(t, 10, cfg.RetryDelay)
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
			gen := NewSectionGenerator(nil, tt.configProvider)
			cfg, err := gen.getReportConfig()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, cfg)
				if tt.checkConfig != nil {
					tt.checkConfig(t, cfg)
				}
			}
		})
	}
}

// TestSectionGenerator_ParseStructuredResponse_EdgeCases tests edge cases for parseStructuredResponse
func TestSectionGenerator_ParseStructuredResponse_EdgeCases(t *testing.T) {
	gen := NewSectionGenerator(nil, &mockConfigProvider{})

	tests := []struct {
		name            string
		response        string
		expectedContent string
		expectedSummary string
	}{
		{
			name: "summary before content",
			response: `[SUMMARY]
Summary first
[CONTENT]
Content second`,
			expectedContent: "",
			expectedSummary: "Summary first\n[CONTENT]\nContent second",
		},
		{
			name: "empty content",
			response: `[CONTENT]

[SUMMARY]
Summary only`,
			expectedContent: "",
			expectedSummary: "Summary only",
		},
		{
			name: "whitespace only content",
			response: `[CONTENT]
   
[SUMMARY]
Summary`,
			expectedContent: "",
			expectedSummary: "Summary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, summary := gen.parseStructuredResponse(tt.response)
			assert.Equal(t, tt.expectedContent, content)
			assert.Equal(t, tt.expectedSummary, summary)
		})
	}
}

// TestSectionGenerator_PostProcessContent_EdgeCases tests edge cases for postProcessContent
func TestSectionGenerator_PostProcessContent_EdgeCases(t *testing.T) {
	gen := NewSectionGenerator(nil, &mockConfigProvider{})

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "remove multiple trailing patterns",
			input:    "## Title\n\nContent.\n\n文档已保存到\n\n已生成\n\n生成完成",
			expected: "## Title\n\nContent.",
		},
		{
			name:     "fence with content after",
			input:    "```markdown\n## Title\n\nContent.\n```\n\nExtra text",
			expected: "## Title\n\nContent.\n```\n\nExtra text", // Only removes first fence
		},
		{
			name:     "multiple code fences",
			input:    "```\n## Title\n```go\ncode\n```\nContent",
			expected: "## Title\n```go\ncode\n```\nContent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.postProcessContent(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSectionGenerator_PostProcessContentWithSummary_EdgeCases tests edge cases
func TestSectionGenerator_PostProcessContentWithSummary_EdgeCases(t *testing.T) {
	gen := NewSectionGenerator(nil, &mockConfigProvider{})

	reportConfig := &dsl.ReportConfig{
		Section: dsl.SectionPhase{
			Summary: dsl.SectionSummaryConfig{
				MaxLength: 50,
			},
		},
	}

	t.Run("truncate at word boundary", func(t *testing.T) {
		response := `[CONTENT]
## Title

Content.

[SUMMARY]
` + strings.Repeat("word ", 20) // Long summary

		result := gen.postProcessContentWithSummary(response, reportConfig)
		assert.LessOrEqual(t, len(result.Summary), 50)
	})

	t.Run("exact max length", func(t *testing.T) {
		response := `[CONTENT]
## Title

Content.

[SUMMARY]
` + strings.Repeat("a", 50)

		result := gen.postProcessContentWithSummary(response, reportConfig)
		assert.LessOrEqual(t, len(result.Summary), 50)
	})

	t.Run("shorter than max", func(t *testing.T) {
		response := `[CONTENT]
## Title

Content.

[SUMMARY]
Short summary`

		result := gen.postProcessContentWithSummary(response, reportConfig)
		assert.Equal(t, "Short summary", result.Summary)
	})
}

// TestSectionGenerator_BuildSectionPrompt tests buildSectionPrompt
func TestSectionGenerator_BuildSectionPrompt(t *testing.T) {
	gen := NewSectionGenerator(nil, &mockConfigProvider{
		reportConfig: &config.ReportConfig{
			OutputLanguage: "en-US",
		},
	})

	typeDef := &ReportTypeDefinition{
		ID:          "test-report",
		Name:        "Test Report",
		Description: "A test report type",
		Config: &dsl.ReportConfig{
			Section: dsl.SectionPhase{
				Description: "You are a section generator",
				Goals: dsl.ReportGoalsConfig{
					Topics: []string{"topic1", "topic2"},
					Avoid:  []string{"avoid1"},
				},
				Constraints: []string{"constraint1"},
				Summary: dsl.SectionSummaryConfig{
					MaxLength: 200,
				},
			},
			Output: dsl.ReportOutputConfig{
				Style: dsl.ReportStyleConfig{
					HeadingLevel: 2,
					Tone:         "professional",
					Language:     "en-US",
				},
			},
		},
	}

	section := &model.ReportSection{
		SectionID:       "section-1",
		Title:           "Test Section",
		Description:     "Section description",
		ParentSectionID: nil,
	}

	structure := &ReportStructure{
		Title:   "Test Report",
		Summary: "Report summary",
		Sections: []GeneratedSection{
			{
				ID:          "section-1",
				Title:       "Test Section",
				Description: "Section description",
			},
		},
	}

	prompt := gen.buildSectionPrompt(typeDef, typeDef.Config, section, structure)

	// Verify prompt contains expected sections
	assert.Contains(t, prompt, "Role")
	assert.Contains(t, prompt, "section generator")
	assert.Contains(t, prompt, "Report Context")
	assert.Contains(t, prompt, "Test Report")
	assert.Contains(t, prompt, "Section to Generate")
	assert.Contains(t, prompt, "section-1")
	assert.Contains(t, prompt, "Test Section")
	assert.Contains(t, prompt, "Focus Topics")
	assert.Contains(t, prompt, "topic1")
	assert.Contains(t, prompt, "Things to Avoid")
	assert.Contains(t, prompt, "avoid1")
	assert.Contains(t, prompt, "Constraints")
	assert.Contains(t, prompt, "constraint1")
	assert.Contains(t, prompt, "Output Format")
	assert.Contains(t, prompt, "[CONTENT]")
	assert.Contains(t, prompt, "[SUMMARY]")
}

// TestSectionGenerator_BuildSectionPrompt_WithParentSection tests buildSectionPrompt with parent section
func TestSectionGenerator_BuildSectionPrompt_WithParentSection(t *testing.T) {
	gen := NewSectionGenerator(nil, &mockConfigProvider{})

	parentID := "parent-1"
	typeDef := &ReportTypeDefinition{
		ID:   "test-report",
		Name: "Test Report",
		Config: &dsl.ReportConfig{
			Section: dsl.SectionPhase{},
			Output: dsl.ReportOutputConfig{
				Style: dsl.ReportStyleConfig{
					HeadingLevel: 2,
				},
			},
		},
	}

	section := &model.ReportSection{
		SectionID:       "subsection-1",
		Title:           "Subsection",
		Description:     "Subsection description",
		ParentSectionID: &parentID,
	}

	structure := &ReportStructure{
		Title:   "Test Report",
		Summary: "Summary",
		Sections: []GeneratedSection{
			{
				ID:          "parent-1",
				Title:       "Parent Section",
				Description: "Parent description",
				Subsections: []GeneratedSection{
					{
						ID:          "subsection-1",
						Title:       "Subsection",
						Description: "Subsection description",
					},
				},
			},
		},
	}

	prompt := gen.buildSectionPrompt(typeDef, typeDef.Config, section, structure)

	assert.Contains(t, prompt, "Parent Section ID")
	assert.Contains(t, prompt, "parent-1")
	assert.Contains(t, prompt, "Parent Section Title")
	assert.Contains(t, prompt, "Parent Section")
}

// TestMergeSections tests MergeSections function
func TestMergeSections(t *testing.T) {
	report := &model.Report{
		ID:         "report-001",
		RepoURL:    "https://github.com/test/repo",
		ReportType: "wiki",
	}

	sections := []model.ReportSection{
		{
			ReportID:     "report-001",
			SectionIndex: 0,
			SectionID:    "section-1",
			Title:        "Section 1",
			Content:      "Content 1",
			Status:       model.SectionStatusCompleted,
		},
		{
			ReportID:     "report-001",
			SectionIndex: 1,
			SectionID:    "section-2",
			Title:        "Section 2",
			Content:      "Content 2",
			Status:       model.SectionStatusCompleted,
		},
	}

	result := MergeSections(report, sections)
	assert.Contains(t, result, "Content 1")
	assert.Contains(t, result, "Content 2")
}

// TestSectionGenerator_PostProcessContentWithSummary_SummaryTruncation tests summary truncation edge cases
func TestSectionGenerator_PostProcessContentWithSummary_SummaryTruncation(t *testing.T) {
	gen := NewSectionGenerator(nil, &mockConfigProvider{})

	reportConfig := &dsl.ReportConfig{
		Section: dsl.SectionPhase{
			Summary: dsl.SectionSummaryConfig{
				MaxLength: 50,
			},
		},
	}

	t.Run("truncate at word boundary", func(t *testing.T) {
		response := `[CONTENT]
## Title

Content.

[SUMMARY]
` + strings.Repeat("word ", 20) // Long summary

		result := gen.postProcessContentWithSummary(response, reportConfig)
		assert.LessOrEqual(t, len(result.Summary), 50)
	})

	t.Run("exact max length", func(t *testing.T) {
		response := `[CONTENT]
## Title

Content.

[SUMMARY]
` + strings.Repeat("a", 50)

		result := gen.postProcessContentWithSummary(response, reportConfig)
		assert.LessOrEqual(t, len(result.Summary), 50)
	})

	t.Run("shorter than max", func(t *testing.T) {
		response := `[CONTENT]
## Title

Content.

[SUMMARY]
Short summary`

		result := gen.postProcessContentWithSummary(response, reportConfig)
		assert.Equal(t, "Short summary", result.Summary)
	})
}

// TestSectionGenerator_BuildSectionPrompt_OutputLanguage tests output language handling
func TestSectionGenerator_BuildSectionPrompt_OutputLanguage(t *testing.T) {
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
			expectLanguage: "中文",
		},
		{
			name:           "fallback to db language",
			configLanguage: "",
			dbLanguage:     "en-US",
			expectLanguage: "English",
		},
		{
			name:           "no language specified",
			configLanguage: "",
			dbLanguage:     "",
			expectLanguage: "", // Should not appear in prompt
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewSectionGenerator(nil, &mockConfigProvider{
				reportConfig: &config.ReportConfig{
					OutputLanguage: tt.dbLanguage,
				},
			})

			typeDef := &ReportTypeDefinition{
				ID:   "test",
				Name: "Test",
				Config: &dsl.ReportConfig{
					Section: dsl.SectionPhase{},
					Output: dsl.ReportOutputConfig{
						Style: dsl.ReportStyleConfig{
							Language: tt.configLanguage,
						},
					},
				},
			}

			section := &model.ReportSection{
				SectionID: "section-1",
				Title:     "Test",
			}

			structure := &ReportStructure{
				Title:   "Test",
				Summary: "Summary",
				Sections: []GeneratedSection{
					{ID: "section-1", Title: "Test"},
				},
			}

			prompt := gen.buildSectionPrompt(typeDef, typeDef.Config, section, structure)

			if tt.expectLanguage != "" {
				assert.Contains(t, prompt, tt.expectLanguage)
			} else {
				// Should not have Language section
				assert.NotContains(t, prompt, "### Language")
			}
		})
	}
}

// TestSectionGenerator_BuildSectionPrompt_HeadingLevel tests heading level in prompt
func TestSectionGenerator_BuildSectionPrompt_HeadingLevel(t *testing.T) {
	gen := NewSectionGenerator(nil, &mockConfigProvider{})

	tests := []struct {
		name         string
		headingLevel int
		expectPrefix string
	}{
		{
			name:         "level 2",
			headingLevel: 2,
			expectPrefix: "##",
		},
		{
			name:         "level 3",
			headingLevel: 3,
			expectPrefix: "###",
		},
		{
			name:         "level 1",
			headingLevel: 1,
			expectPrefix: "#",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeDef := &ReportTypeDefinition{
				ID:   "test",
				Name: "Test",
				Config: &dsl.ReportConfig{
					Section: dsl.SectionPhase{},
					Output: dsl.ReportOutputConfig{
						Style: dsl.ReportStyleConfig{
							HeadingLevel: tt.headingLevel,
						},
					},
				},
			}

			section := &model.ReportSection{
				SectionID: "section-1",
				Title:     "Test",
			}

			structure := &ReportStructure{
				Title:   "Test",
				Summary: "Summary",
				Sections: []GeneratedSection{
					{ID: "section-1", Title: "Test"},
				},
			}

			prompt := gen.buildSectionPrompt(typeDef, typeDef.Config, section, structure)
			assert.Contains(t, prompt, tt.expectPrefix)
			// Check that heading level is mentioned in prompt
			assert.Contains(t, prompt, fmt.Sprintf("level-%d", tt.headingLevel))
		})
	}
}
