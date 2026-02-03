package prompt

import (
	"strings"
	"testing"

	"github.com/verustcode/verustcode/internal/dsl"
)

// TestNewFormatInstructionBuilder tests creating a new FormatInstructionBuilder
func TestNewFormatInstructionBuilder(t *testing.T) {
	builder := NewFormatInstructionBuilder()
	if builder == nil {
		t.Error("NewFormatInstructionBuilder() returned nil")
	}
}

// TestFormatInstructionBuilder_Build tests the Build method
func TestFormatInstructionBuilder_Build(t *testing.T) {
	builder := NewFormatInstructionBuilder()

	t.Run("json format", func(t *testing.T) {
		result := builder.Build("json", nil, "")

		if result == "" {
			t.Error("Build() returned empty string")
		}

		if !strings.Contains(result, "Output Format") {
			t.Error("Result should contain 'Output Format' header")
		}

		if !strings.Contains(result, "JSON format") {
			t.Error("Result should mention JSON format")
		}

		if !strings.Contains(result, "IMPORTANT") {
			t.Error("Result should contain IMPORTANT section")
		}
	})

	t.Run("markdown format", func(t *testing.T) {
		result := builder.Build("markdown", nil, "")

		if result == "" {
			t.Error("Build() returned empty string")
		}

		if !strings.Contains(result, "Output Format") {
			t.Error("Result should contain 'Output Format' header")
		}

		if !strings.Contains(result, "Markdown format") {
			t.Error("Result should mention Markdown format")
		}
	})

	t.Run("unknown format defaults to markdown", func(t *testing.T) {
		result := builder.Build("unknown_format", nil, "")

		if result == "" {
			t.Error("Build() returned empty string")
		}

		if !strings.Contains(result, "Markdown format") {
			t.Error("Unknown format should default to Markdown")
		}
	})

	t.Run("empty format defaults to markdown", func(t *testing.T) {
		result := builder.Build("", nil, "")

		if result == "" {
			t.Error("Build() returned empty string")
		}

		if !strings.Contains(result, "Markdown format") {
			t.Error("Empty format should default to Markdown")
		}
	})

	t.Run("json format with language", func(t *testing.T) {
		result := builder.Build("json", nil, "Chinese")

		if !strings.Contains(result, "Chinese") {
			t.Error("Result should contain language instruction for Chinese")
		}
	})

	t.Run("markdown format with language", func(t *testing.T) {
		result := builder.Build("markdown", nil, "English")

		if !strings.Contains(result, "English") {
			t.Error("Result should contain language instruction for English")
		}
	})
}

// TestFormatInstructionBuilder_BuildJSONFormat tests JSON format instructions
func TestFormatInstructionBuilder_BuildJSONFormat(t *testing.T) {
	builder := NewFormatInstructionBuilder()

	t.Run("json output contains schema", func(t *testing.T) {
		result := builder.Build("json", nil, "")

		// Should contain code block
		if !strings.Contains(result, "```json") {
			t.Error("Result should contain JSON code block")
		}

		// Should contain instructions about valid JSON
		if !strings.Contains(result, "valid JSON") {
			t.Error("Result should mention valid JSON requirement")
		}

		// Should mention required fields
		if !strings.Contains(result, "required fields") {
			t.Error("Result should mention required fields")
		}
	})

	t.Run("json with language instruction", func(t *testing.T) {
		result := builder.Build("json", nil, "Japanese")

		if !strings.Contains(result, "Japanese") {
			t.Error("Result should contain language instruction")
		}

		if !strings.Contains(result, "MUST be in Japanese") {
			t.Error("Result should contain MUST be in language instruction")
		}
	})
}

// TestFormatInstructionBuilder_BuildMarkdownFormat tests Markdown format instructions
func TestFormatInstructionBuilder_BuildMarkdownFormat(t *testing.T) {
	builder := NewFormatInstructionBuilder()

	t.Run("markdown contains structure", func(t *testing.T) {
		result := builder.Build("markdown", nil, "")

		// Should contain recommended structure
		if !strings.Contains(result, "Recommended Structure") || !strings.Contains(result, "Structure") {
			t.Error("Result should contain structure section")
		}

		// Should contain code block for structure
		if !strings.Contains(result, "```") {
			t.Error("Result should contain code block")
		}
	})

	t.Run("markdown with language instruction", func(t *testing.T) {
		result := builder.Build("markdown", nil, "German")

		if !strings.Contains(result, "German") {
			t.Error("Result should contain language instruction")
		}

		// Check for actual language instruction format used in implementation
		if !strings.Contains(result, "MUST be in") {
			t.Error("Result should contain 'MUST be in' instruction")
		}
	})
}

// TestFormatInstructionBuilder_GetDefaultFieldDescription tests default field descriptions
func TestFormatInstructionBuilder_GetDefaultFieldDescription(t *testing.T) {
	builder := NewFormatInstructionBuilder()

	t.Run("field with enum values", func(t *testing.T) {
		field := dsl.FieldSpec{
			Name:       "severity",
			Type:       "string",
			EnumValues: []string{"low", "medium", "high"},
		}

		desc := builder.getDefaultFieldDescription(field)

		if !strings.Contains(desc, "One of:") {
			t.Error("Enum field description should contain 'One of:'")
		}

		if !strings.Contains(desc, "low") {
			t.Error("Enum field description should contain enum values")
		}
	})

	t.Run("string field", func(t *testing.T) {
		field := dsl.FieldSpec{
			Name: "description",
			Type: "string",
		}

		desc := builder.getDefaultFieldDescription(field)

		if desc != "Text value" {
			t.Errorf("String field description = %s, want 'Text value'", desc)
		}
	})

	t.Run("integer field", func(t *testing.T) {
		field := dsl.FieldSpec{
			Name: "line_number",
			Type: "integer",
		}

		desc := builder.getDefaultFieldDescription(field)

		if desc != "Integer value" {
			t.Errorf("Integer field description = %s, want 'Integer value'", desc)
		}
	})

	t.Run("array field", func(t *testing.T) {
		field := dsl.FieldSpec{
			Name: "files",
			Type: "array",
		}

		desc := builder.getDefaultFieldDescription(field)

		if desc != "List of values" {
			t.Errorf("Array field description = %s, want 'List of values'", desc)
		}
	})

	t.Run("unknown type field", func(t *testing.T) {
		field := dsl.FieldSpec{
			Name: "custom",
			Type: "custom_type",
		}

		desc := builder.getDefaultFieldDescription(field)

		if desc != "" {
			t.Errorf("Unknown type field description = %s, want empty string", desc)
		}
	})
}

// TestFormatInstructionBuilder_GenerateMarkdownInstructions tests generating markdown from spec
func TestFormatInstructionBuilder_GenerateMarkdownInstructions(t *testing.T) {
	builder := NewFormatInstructionBuilder()

	t.Run("with sections", func(t *testing.T) {
		spec := &dsl.MarkdownFormatSpec{
			Sections: []dsl.SectionSpec{
				{Name: "Overview", Level: 2, Description: "High-level summary"},
				{Name: "Details", Level: 2, Description: "Detailed findings"},
			},
		}

		result := builder.generateMarkdownInstructions(spec, "")

		if !strings.Contains(result, "Overview") {
			t.Error("Result should contain Overview section")
		}

		if !strings.Contains(result, "Details") {
			t.Error("Result should contain Details section")
		}
	})

	t.Run("with finding format", func(t *testing.T) {
		spec := &dsl.MarkdownFormatSpec{
			Sections: []dsl.SectionSpec{
				{Name: "Findings", Level: 2},
			},
			FindingFormat: &dsl.FindingFormatSpec{
				RequiredFields: []dsl.FieldSpec{
					{Name: "severity", Type: "string", EnumValues: []string{"low", "high"}},
					{Name: "description", Type: "string", Description: "Issue description"},
				},
				OptionalFields: []dsl.FieldSpec{
					{Name: "file_path", Type: "string"},
					{Name: "line_number", Type: "integer"},
				},
			},
		}

		result := builder.generateMarkdownInstructions(spec, "")

		// Should contain required fields section
		if !strings.Contains(result, "required") {
			t.Error("Result should mention required fields")
		}

		// Should contain optional fields section
		if !strings.Contains(result, "Optional") {
			t.Error("Result should mention optional fields")
		}

		// Should contain field names
		if !strings.Contains(result, "severity") {
			t.Error("Result should contain severity field")
		}

		if !strings.Contains(result, "description") {
			t.Error("Result should contain description field")
		}

		if !strings.Contains(result, "file_path") {
			t.Error("Result should contain file_path field")
		}

		// Should contain formatting hint
		if !strings.Contains(result, "good readability") {
			t.Error("Result should contain formatting hint for good readability")
		}
	})

	t.Run("with language", func(t *testing.T) {
		spec := &dsl.MarkdownFormatSpec{
			Sections: []dsl.SectionSpec{
				{Name: "Summary", Level: 2},
			},
		}

		result := builder.generateMarkdownInstructions(spec, "French")

		if !strings.Contains(result, "French") {
			t.Error("Result should contain language instruction")
		}
	})

	t.Run("empty sections", func(t *testing.T) {
		spec := &dsl.MarkdownFormatSpec{
			Sections: []dsl.SectionSpec{},
		}

		result := builder.generateMarkdownInstructions(spec, "")

		// Should still generate valid output
		if !strings.Contains(result, "Output Format") {
			t.Error("Result should contain Output Format header")
		}
	})
}
