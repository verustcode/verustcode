// Package report provides report generation functionality.
package report

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/dsl"
)

// ====================
// Tests for types.go - Report Type Registry
// ====================

func TestRegisterReportType(t *testing.T) {
	// Reset registry before test
	ResetReportTypes()
	defer ResetReportTypes()

	// Register a report type
	def := &ReportTypeDefinition{
		ID:          "test-report",
		Name:        "Test Report",
		Description: "A test report type",
	}
	RegisterReportType(def)

	// Verify it was registered
	types := ListReportTypes()
	assert.Len(t, types, 1)
	assert.Equal(t, "test-report", types[0].ID)
	assert.Equal(t, "Test Report", types[0].Name)
}

func TestRegisterReportType_NoDuplicate(t *testing.T) {
	// Reset registry before test
	ResetReportTypes()
	defer ResetReportTypes()

	// Register same ID twice
	def1 := &ReportTypeDefinition{
		ID:          "test-report",
		Name:        "Test Report v1",
		Description: "Version 1",
	}
	def2 := &ReportTypeDefinition{
		ID:          "test-report",
		Name:        "Test Report v2",
		Description: "Version 2",
	}
	RegisterReportType(def1)
	RegisterReportType(def2)

	// Should still have only one entry, but updated to v2
	types := ListReportTypes()
	assert.Len(t, types, 1)
	assert.Equal(t, "Test Report v2", types[0].Name)
}

func TestListReportTypes_Order(t *testing.T) {
	// Reset registry before test
	ResetReportTypes()
	defer ResetReportTypes()

	// Register multiple types
	RegisterReportType(&ReportTypeDefinition{ID: "alpha", Name: "Alpha"})
	RegisterReportType(&ReportTypeDefinition{ID: "beta", Name: "Beta"})
	RegisterReportType(&ReportTypeDefinition{ID: "gamma", Name: "Gamma"})

	// Should maintain registration order
	types := ListReportTypes()
	require.Len(t, types, 3)
	assert.Equal(t, "alpha", types[0].ID)
	assert.Equal(t, "beta", types[1].ID)
	assert.Equal(t, "gamma", types[2].ID)
}

func TestResetReportTypes(t *testing.T) {
	// Reset registry before test
	ResetReportTypes()

	// Register some types
	RegisterReportType(&ReportTypeDefinition{ID: "test1", Name: "Test 1"})
	RegisterReportType(&ReportTypeDefinition{ID: "test2", Name: "Test 2"})

	// Verify they exist
	assert.Len(t, ListReportTypes(), 2)

	// Reset
	ResetReportTypes()

	// Should be empty now (but will try to load from filesystem)
	// Since we don't have the config directory, it should remain empty
	types := ListReportTypes()
	// Note: ListReportTypes calls initReportTypes which tries to load from filesystem
	// If config/reports doesn't exist, it should still be empty
	_ = types // Just verify no panic
}

// ====================
// Tests for generator.go - SectionGenerator
// ====================

func TestNewSectionGenerator(t *testing.T) {
	tests := []struct {
		name               string
		maxRetries         int
		retryDelay         int
		outputLanguage     string
		expectedMaxRetries int
	}{
		{
			name:               "default values",
			maxRetries:         0,
			retryDelay:         0,
			outputLanguage:     "zh-CN",
			expectedMaxRetries: DefaultReportMaxRetries,
		},
		{
			name:               "custom values",
			maxRetries:         5,
			retryDelay:         20,
			outputLanguage:     "en-US",
			expectedMaxRetries: 5,
		},
		{
			name:               "negative values use defaults",
			maxRetries:         -1,
			retryDelay:         -1,
			outputLanguage:     "zh-CN",
			expectedMaxRetries: DefaultReportMaxRetries,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Report: config.ReportConfig{
					MaxRetries:     tt.maxRetries,
					RetryDelay:     tt.retryDelay,
					OutputLanguage: tt.outputLanguage,
				},
			}
			configProvider := config.NewStaticConfigProvider(cfg)
			gen := NewSectionGenerator(nil, configProvider)
			assert.NotNil(t, gen)
			// Verify config can be retrieved
			reportCfg, err := gen.getReportConfig()
			require.NoError(t, err)
			if tt.maxRetries > 0 {
				assert.Equal(t, tt.maxRetries, reportCfg.MaxRetries)
			} else {
				assert.Equal(t, DefaultReportMaxRetries, reportCfg.MaxRetries)
			}
		})
	}
}

func TestSectionGenerator_ParseStructuredResponse(t *testing.T) {
	cfg := &config.Config{
		Report: config.ReportConfig{
			MaxRetries:     3,
			RetryDelay:     10,
			OutputLanguage: "zh-CN",
		},
	}
	configProvider := config.NewStaticConfigProvider(cfg)
	gen := NewSectionGenerator(nil, configProvider)

	tests := []struct {
		name            string
		response        string
		expectedContent string
		expectedSummary string
	}{
		{
			name: "with both markers",
			response: `[CONTENT]
## Section Title

This is the section content.

[SUMMARY]
Short summary of the section.`,
			expectedContent: "## Section Title\n\nThis is the section content.",
			expectedSummary: "Short summary of the section.",
		},
		{
			name: "only summary marker",
			response: `## Section Title

This is the section content.

[SUMMARY]
This is the summary.`,
			expectedContent: "## Section Title\n\nThis is the section content.",
			expectedSummary: "This is the summary.",
		},
		{
			name: "no markers",
			response: `## Section Title

This is some content without markers.`,
			expectedContent: "## Section Title\n\nThis is some content without markers.",
			expectedSummary: "This is some content without markers.", // fallback summary
		},
		{
			name: "case insensitive markers",
			response: `[content]
## Title

Content here.

[summary]
Summary here.`,
			expectedContent: "## Title\n\nContent here.",
			expectedSummary: "Summary here.",
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

func TestSectionGenerator_PostProcessContent(t *testing.T) {
	cfg := &config.Config{
		Report: config.ReportConfig{
			MaxRetries:     3,
			RetryDelay:     10,
			OutputLanguage: "zh-CN",
		},
	}
	configProvider := config.NewStaticConfigProvider(cfg)
	gen := NewSectionGenerator(nil, configProvider)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already clean",
			input:    "## Section Title\n\nThis is content.",
			expected: "## Section Title\n\nThis is content.",
		},
		{
			name:     "markdown fence wrapper",
			input:    "```markdown\n## Section Title\n\nContent here.\n```",
			expected: "## Section Title\n\nContent here.",
		},
		{
			name:     "md fence wrapper",
			input:    "```md\n## Section Title\n\nContent here.\n```",
			expected: "## Section Title\n\nContent here.",
		},
		{
			name:     "remove leading meta-commentary",
			input:    "Here is the generated section:\n\n## Section Title\n\nContent.",
			expected: "## Section Title\n\nContent.",
		},
		{
			name:     "remove trailing file save message (Chinese)",
			input:    "## Section Title\n\nContent.\n\n文档已保存到 docs/report.md",
			expected: "## Section Title\n\nContent.",
		},
		{
			name:     "remove trailing file save message (English)",
			input:    "## Section Title\n\nContent.\n\nDocument saved to docs/report.md",
			expected: "## Section Title\n\nContent.",
		},
		{
			name:     "preserve internal code blocks",
			input:    "## Section Title\n\n```go\nfunc main() {}\n```\n\nMore content.",
			expected: "## Section Title\n\n```go\nfunc main() {}\n```\n\nMore content.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.postProcessContent(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSectionGenerator_GenerateFallbackSummary(t *testing.T) {
	cfg := &config.Config{
		Report: config.ReportConfig{
			MaxRetries:     3,
			RetryDelay:     10,
			OutputLanguage: "zh-CN",
		},
	}
	configProvider := config.NewStaticConfigProvider(cfg)
	gen := NewSectionGenerator(nil, configProvider)

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "simple paragraph",
			content:  "## Title\n\nThis is a simple paragraph that should be used as the summary.",
			expected: "This is a simple paragraph that should be used as the summary.",
		},
		{
			name:     "removes markdown formatting",
			content:  "## Title\n\nThis is **bold** and *italic* content with `code`.",
			expected: "This is bold and italic content with code.",
		},
		{
			name:     "skips headings",
			content:  "# Main Title\n## Section\n\nActual content here.",
			expected: "Actual content here.",
		},
		{
			name:     "truncates long content",
			content:  "## Title\n\n" + string(make([]byte, 300)), // 300 bytes
			expected: "",                                         // Will truncate at word boundary or char limit
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.generateFallbackSummary(tt.content)
			if tt.expected != "" {
				assert.Equal(t, tt.expected, result)
			} else {
				// Just verify it doesn't panic and is within limit
				assert.LessOrEqual(t, len(result), 200)
			}
		})
	}
}

func TestSectionGenerator_PostProcessContentWithSummary(t *testing.T) {
	cfg := &config.Config{
		Report: config.ReportConfig{
			MaxRetries:     3,
			RetryDelay:     10,
			OutputLanguage: "zh-CN",
		},
	}
	configProvider := config.NewStaticConfigProvider(cfg)
	gen := NewSectionGenerator(nil, configProvider)

	// Create a mock DSL config
	config := &dsl.ReportConfig{
		Section: dsl.SectionPhase{
			Summary: dsl.SectionSummaryConfig{
				MaxLength: 100,
			},
		},
	}

	tests := []struct {
		name            string
		response        string
		expectedContent string
		summaryContains string
		summaryMaxLen   int
	}{
		{
			name: "structured response",
			response: `[CONTENT]
## Title

Content here.

[SUMMARY]
This is a short summary.`,
			expectedContent: "## Title\n\nContent here.",
			summaryContains: "summary",
			summaryMaxLen:   100,
		},
		{
			name: "summary truncation",
			response: `[CONTENT]
## Title

Content.

[SUMMARY]
` + string(make([]byte, 150)), // Long summary that should be truncated
			expectedContent: "## Title\n\nContent.",
			summaryMaxLen:   100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.postProcessContentWithSummary(tt.response, config)
			assert.Equal(t, tt.expectedContent, result.Content)
			if tt.summaryContains != "" {
				assert.Contains(t, result.Summary, tt.summaryContains)
			}
			assert.LessOrEqual(t, len(result.Summary), tt.summaryMaxLen)
		})
	}
}
