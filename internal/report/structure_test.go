// Package report provides report generation functionality.
// This file contains unit tests for report structure generator.
package report

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verustcode/verustcode/internal/dsl"
)

// TestNewStructureGenerator tests NewStructureGenerator
func TestNewStructureGenerator(t *testing.T) {
	configProvider := &mockConfigProvider{}

	gen := NewStructureGenerator(nil, configProvider)
	require.NotNil(t, gen)
	assert.Equal(t, configProvider, gen.configProvider)
}

// Note: StructureGenerator doesn't have getReportConfig method
// It accesses configProvider directly when needed

// TestStructureGenerator_ParseStructureResponse tests parseStructureResponse
func TestStructureGenerator_ParseStructureJSON(t *testing.T) {
	gen := NewStructureGenerator(nil, &mockConfigProvider{})

	tests := []struct {
		name      string
		jsonStr   string
		wantErr   bool
		checkFunc func(*testing.T, *ReportStructure)
	}{
		{
			name: "valid flat structure",
			jsonStr: `{
				"title": "Test Report",
				"summary": "Report summary",
				"sections": [
					{
						"id": "section-1",
						"title": "Section 1",
						"description": "Description 1"
					}
				]
			}`,
			checkFunc: func(t *testing.T, s *ReportStructure) {
				assert.Equal(t, "Test Report", s.Title)
				assert.Equal(t, "Report summary", s.Summary)
				assert.Len(t, s.Sections, 1)
				assert.Equal(t, "section-1", s.Sections[0].ID)
			},
		},
		{
			name: "valid nested structure",
			jsonStr: `{
				"title": "Test Report",
				"summary": "Summary",
				"sections": [
					{
						"id": "parent-1",
						"title": "Parent",
						"description": "Parent desc",
						"subsections": [
							{
								"id": "sub-1",
								"title": "Subsection",
								"description": "Sub desc"
							}
						]
					}
				]
			}`,
			checkFunc: func(t *testing.T, s *ReportStructure) {
				assert.Len(t, s.Sections, 1)
				assert.Len(t, s.Sections[0].Subsections, 1)
				assert.Equal(t, "sub-1", s.Sections[0].Subsections[0].ID)
			},
		},
		{
			name:    "invalid JSON",
			jsonStr: `Some text before {invalid json} and after`,
			wantErr: true,
		},
		{
			name:    "missing required fields",
			jsonStr: `Here is the response:\n\n{"title": "Test"}`,
			wantErr: true,
		},
		{
			name:    "empty sections",
			jsonStr: `{"title": "Test", "summary": "Summary", "sections": []}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := gen.parseStructureResponse(tt.jsonStr)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				if tt.checkFunc != nil {
					tt.checkFunc(t, result)
				}
			}
		})
	}
}

// TestStructureGenerator_BuildStructurePrompt tests buildStructurePrompt
func TestStructureGenerator_BuildStructurePrompt(t *testing.T) {
	gen := NewStructureGenerator(nil, &mockConfigProvider{})

	typeDef := &ReportTypeDefinition{
		ID:          "test-report",
		Name:        "Test Report",
		Description: "A test report",
		Config: &dsl.ReportConfig{
			Structure: dsl.StructurePhase{
				Description: "You are a structure generator",
				Goals: dsl.ReportGoalsConfig{
					Topics: []string{"topic1", "topic2"},
					Avoid:  []string{"avoid1"},
				},
				Constraints:   []string{"constraint1"},
				ReferenceDocs: []string{"README.md"},
				Nested:        false,
			},
		},
	}

	prompt := gen.buildStructurePrompt(typeDef, typeDef.Config)

	assert.Contains(t, prompt, "Role")
	assert.Contains(t, prompt, "structure generator")
	assert.Contains(t, prompt, "Report Type")
	assert.Contains(t, prompt, "Test Report")
	assert.Contains(t, prompt, "Focus Topics")
	assert.Contains(t, prompt, "topic1")
	assert.Contains(t, prompt, "Things to Avoid")
	assert.Contains(t, prompt, "avoid1")
	assert.Contains(t, prompt, "Constraints")
	assert.Contains(t, prompt, "constraint1")
	assert.Contains(t, prompt, "README.md")
	assert.Contains(t, prompt, "Output Format")
	assert.Contains(t, prompt, "JSON")
}

// TestStructureGenerator_BuildStructurePrompt_Nested tests buildStructurePrompt with nested structure
func TestStructureGenerator_BuildStructurePrompt_Nested(t *testing.T) {
	gen := NewStructureGenerator(nil, &mockConfigProvider{})

	typeDef := &ReportTypeDefinition{
		ID:   "test-report",
		Name: "Test Report",
		Config: &dsl.ReportConfig{
			Structure: dsl.StructurePhase{
				Nested: true,
			},
		},
	}

	prompt := gen.buildStructurePrompt(typeDef, typeDef.Config)

	assert.Contains(t, prompt, "subsections")
	assert.Contains(t, prompt, "Parent")
}

// TestStructureGenerator_ParseStructureJSON_EdgeCases tests edge cases
func TestStructureGenerator_ParseStructureJSON_EdgeCases(t *testing.T) {
	gen := NewStructureGenerator(nil, &mockConfigProvider{})

	t.Run("whitespace in JSON", func(t *testing.T) {
		jsonStr := `   {
			"title": "Test",
			"summary": "Summary",
			"sections": [{"id": "s1", "title": "S1", "description": "D1"}]
		}   `
		result, err := gen.parseStructureResponse(jsonStr)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("extra fields", func(t *testing.T) {
		jsonStr := `{
			"title": "Test",
			"summary": "Summary",
			"sections": [{"id": "s1", "title": "S1", "description": "D1"}],
			"extra": "field"
		}`
		result, err := gen.parseStructureResponse(jsonStr)
		assert.NoError(t, err) // Should ignore extra fields
		assert.NotNil(t, result)
	})

	t.Run("empty strings", func(t *testing.T) {
		jsonStr := `{
			"title": "",
			"summary": "",
			"sections": [{"id": "s1", "title": "", "description": ""}]
		}`
		result, err := gen.parseStructureResponse(jsonStr)
		assert.Error(t, err) // Should fail validation
		assert.Nil(t, result)
	})
}

// TestReportStructure_JSONRoundTrip tests JSON marshaling/unmarshaling
func TestReportStructure_JSONRoundTrip(t *testing.T) {
	original := &ReportStructure{
		Title:   "Test Report",
		Summary: "Summary",
		Sections: []GeneratedSection{
			{
				ID:          "section-1",
				Title:       "Section 1",
				Description: "Description 1",
				Subsections: []GeneratedSection{
					{
						ID:          "sub-1",
						Title:       "Subsection 1",
						Description: "Sub description",
					},
				},
			},
		},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded ReportStructure
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Title, decoded.Title)
	assert.Equal(t, original.Summary, decoded.Summary)
	assert.Len(t, decoded.Sections, 1)
	assert.Len(t, decoded.Sections[0].Subsections, 1)
}

// TestStructureGenerator_BuildStructurePrompt_NoGoals tests prompt without goals
func TestStructureGenerator_BuildStructurePrompt_NoGoals(t *testing.T) {
	gen := NewStructureGenerator(nil, &mockConfigProvider{})

	typeDef := &ReportTypeDefinition{
		ID:   "test",
		Name: "Test",
		Config: &dsl.ReportConfig{
			Structure: dsl.StructurePhase{
				Description: "Generator",
			},
		},
	}

	prompt := gen.buildStructurePrompt(typeDef, typeDef.Config)

	assert.NotContains(t, prompt, "Focus Topics")
	assert.NotContains(t, prompt, "Things to Avoid")
}

// TestStructureGenerator_BuildStructurePrompt_NoConstraints tests prompt without constraints
func TestStructureGenerator_BuildStructurePrompt_NoConstraints(t *testing.T) {
	gen := NewStructureGenerator(nil, &mockConfigProvider{})

	typeDef := &ReportTypeDefinition{
		ID:   "test",
		Name: "Test",
		Config: &dsl.ReportConfig{
			Structure: dsl.StructurePhase{
				Description: "Generator",
			},
		},
	}

	prompt := gen.buildStructurePrompt(typeDef, typeDef.Config)

	assert.NotContains(t, prompt, "## Constraints")
}

// TestStructureGenerator_ParseStructureJSON_Validation tests validation logic
func TestStructureGenerator_ParseStructureJSON_Validation(t *testing.T) {
	gen := NewStructureGenerator(nil, &mockConfigProvider{})

	t.Run("missing title", func(t *testing.T) {
		jsonStr := `{
			"summary": "Summary",
			"sections": [{"id": "s1", "title": "S1", "description": "D1"}]
		}`
		_, err := gen.parseStructureResponse(jsonStr)
		assert.Error(t, err)
	})

	t.Run("missing summary", func(t *testing.T) {
		jsonStr := `{
			"title": "Test",
			"sections": [{"id": "s1", "title": "S1", "description": "D1"}]
		}`
		_, err := gen.parseStructureResponse(jsonStr)
		assert.Error(t, err)
	})

	t.Run("section missing id", func(t *testing.T) {
		jsonStr := `{
			"title": "Test",
			"summary": "Summary",
			"sections": [{"title": "S1", "description": "D1"}]
		}`
		_, err := gen.parseStructureResponse(jsonStr)
		assert.Error(t, err)
	})

	t.Run("section missing title", func(t *testing.T) {
		jsonStr := `{
			"title": "Test",
			"summary": "Summary",
			"sections": [{"id": "s1", "description": "D1"}]
		}`
		_, err := gen.parseStructureResponse(jsonStr)
		assert.Error(t, err)
	})

	t.Run("subsection validation", func(t *testing.T) {
		jsonStr := `{
			"title": "Test",
			"summary": "Summary",
			"sections": [{
				"id": "parent",
				"title": "Parent",
				"description": "Desc",
				"subsections": [{"title": "Sub", "description": "Sub desc"}]
			}]
		}`
		_, err := gen.parseStructureResponse(jsonStr)
		assert.Error(t, err) // Subsection missing id
	})
}
