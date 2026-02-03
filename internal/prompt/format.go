// Package prompt provides format instruction generation for LLM output.
package prompt

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/verustcode/verustcode/internal/dsl"
	"github.com/verustcode/verustcode/pkg/logger"
	"go.uber.org/zap"
)

// FormatInstructionBuilder builds output format instructions for LLM prompts
type FormatInstructionBuilder struct{}

// NewFormatInstructionBuilder creates a new FormatInstructionBuilder
func NewFormatInstructionBuilder() *FormatInstructionBuilder {
	return &FormatInstructionBuilder{}
}

// Build generates format instructions based on format type and schema configuration
// Parameters:
//   - format: output format ("markdown" or "json")
//   - schemaConfig: schema configuration (can be nil for default behavior)
//   - language: output language (e.g., "Chinese", "English")
//
// Deprecated: Use BuildWithOptions for new code to support history_compare.
func (b *FormatInstructionBuilder) Build(format string, schemaConfig *dsl.OutputSchemaConfig, language string) string {
	return b.BuildWithOptions(format, schemaConfig, language, false)
}

// BuildWithOptions generates format instructions with additional options.
// Parameters:
//   - format: output format ("markdown" or "json")
//   - schemaConfig: schema configuration (can be nil for default behavior)
//   - language: output language (e.g., "Chinese", "English")
//   - historyCompareEnabled: when true, status field becomes required
func (b *FormatInstructionBuilder) BuildWithOptions(format string, schemaConfig *dsl.OutputSchemaConfig, language string, historyCompareEnabled bool) string {
	logger.Debug("Building format instructions",
		zap.String("format", format),
		zap.String("language", language),
		zap.Bool("has_schema_config", schemaConfig != nil),
		zap.Bool("history_compare_enabled", historyCompareEnabled),
	)

	// Build JSON Schema from configuration (merges extra_fields into default schema)
	// When historyCompareEnabled is true, status field becomes required
	jsonSchema := dsl.BuildJSONSchemaWithOptions(schemaConfig, historyCompareEnabled)
	logger.Debug("Built JSON schema",
		zap.Bool("has_extra_fields", schemaConfig != nil && len(schemaConfig.ExtraFields) > 0),
		zap.Bool("history_compare_enabled", historyCompareEnabled),
	)

	switch format {
	case "json":
		result := b.buildJSONFormatInstructions(jsonSchema, language)
		logger.Debug("Generated JSON format instructions",
			zap.Int("length", len(result)),
		)
		return result
	case "markdown":
		result := b.buildMarkdownFormatInstructions(jsonSchema, language)
		logger.Debug("Generated Markdown format instructions",
			zap.Int("length", len(result)),
		)
		return result
	default:
		// Default to markdown
		logger.Debug("Unknown format, defaulting to markdown",
			zap.String("format", format),
		)
		result := b.buildMarkdownFormatInstructions(jsonSchema, language)
		logger.Debug("Generated Markdown format instructions (default)",
			zap.Int("length", len(result)),
		)
		return result
	}
}

// buildJSONFormatInstructions generates JSON format instructions
func (b *FormatInstructionBuilder) buildJSONFormatInstructions(jsonSchema map[string]interface{}, language string) string {
	var sb strings.Builder

	sb.WriteString("\n\n## Output Format\n\n")
	sb.WriteString("Please provide your response in the following **JSON format**:\n\n")

	// Pretty print the schema
	schemaJSON, err := json.MarshalIndent(jsonSchema, "", "  ")
	if err == nil {
		sb.WriteString("```json\n")
		sb.WriteString(string(schemaJSON))
		sb.WriteString("\n```\n\n")
	}

	sb.WriteString("**IMPORTANT:**\n")
	sb.WriteString("- Your response MUST be valid JSON that strictly follows this schema.\n")
	sb.WriteString("- Do not include any text before or after the JSON object.\n")
	sb.WriteString("- Ensure all required fields are present.\n")

	// Add language instruction if specified
	if language != "" {
		sb.WriteString(fmt.Sprintf("\nAll output content MUST be in %s.\n", language))
		sb.WriteString("Also use appropriate field names in the target language.\n")
		sb.WriteString("For example: If language is Chinese, use the following field names: Summary → 汇总, Location → 位置, ...\n")
	}

	return sb.String()
}

// buildMarkdownFormatInstructions generates Markdown format instructions from JSON Schema
func (b *FormatInstructionBuilder) buildMarkdownFormatInstructions(jsonSchema map[string]interface{}, language string) string {
	// Convert JSON Schema to Markdown format spec
	logger.Debug("Converting JSON schema to Markdown format spec")
	spec, err := dsl.ConvertJSONSchemaToMarkdownSpec(jsonSchema)
	if err != nil {
		logger.Error("Failed to convert JSON schema to Markdown spec",
			zap.Error(err),
		)
		return ""
	}

	if spec == nil {
		logger.Error("Converted Markdown spec is nil")
		return ""
	}

	logger.Debug("Successfully converted JSON schema to Markdown spec",
		zap.Int("sections_count", len(spec.Sections)),
		zap.Bool("has_finding_format", spec.FindingFormat != nil),
	)

	result := b.generateMarkdownInstructions(spec, language)
	logger.Debug("Generated Markdown instructions from spec",
		zap.Int("result_length", len(result)),
	)
	return result
}

// generateMarkdownInstructions generates Markdown format instructions from a format spec
func (b *FormatInstructionBuilder) generateMarkdownInstructions(spec *dsl.MarkdownFormatSpec, language string) string {
	var sb strings.Builder

	sb.WriteString("\n\n## Output Format\n\n")
	sb.WriteString("Please provide your response in **Markdown format**.\n\n")

	// Add language instruction at the beginning
	if language != "" {
		sb.WriteString(fmt.Sprintf("All output content MUST be in %s.\n", language))
		sb.WriteString("Also use appropriate field names in the target language.\n")
		sb.WriteString("For example: If language is Chinese, use the following field names: Summary → 汇总, Location → 位置, ...\n\n")
	}

	// Generate section structure
	sb.WriteString("### Recommended Structure\n\n")
	sb.WriteString("```\n")
	for _, section := range spec.Sections {
		level := strings.Repeat("#", section.Level)
		sb.WriteString(fmt.Sprintf("%s %s\n", level, section.Name))
		if section.Description != "" {
			sb.WriteString(fmt.Sprintf("*%s*\n\n", section.Description))
		}
	}
	sb.WriteString("```\n\n")

	// Generate finding format
	if spec.FindingFormat != nil {
		sb.WriteString("\n### Finding Format\n\n")
		sb.WriteString("Format findings with good readability: use blank lines, code blocks, and clear structure.\n\n")

		if len(spec.FindingFormat.RequiredFields) > 0 {
			sb.WriteString("Each finding should include the following fields:\n")
			for _, field := range spec.FindingFormat.RequiredFields {
				desc := field.Description
				if desc == "" {
					desc = b.getDefaultFieldDescription(field)
				}
				sb.WriteString(fmt.Sprintf("- **%s** (required): %s\n", field.Name, desc))
			}
			sb.WriteString("\n")
		}

		if len(spec.FindingFormat.OptionalFields) > 0 {
			sb.WriteString("Optional fields:\n")
			for _, field := range spec.FindingFormat.OptionalFields {
				desc := field.Description
				if desc == "" {
					desc = b.getDefaultFieldDescription(field)
				}
				sb.WriteString(fmt.Sprintf("- **%s**: %s\n", field.Name, desc))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// getDefaultFieldDescription returns a default description for a field
func (b *FormatInstructionBuilder) getDefaultFieldDescription(field dsl.FieldSpec) string {
	if len(field.EnumValues) > 0 {
		return fmt.Sprintf("One of: %s", strings.Join(field.EnumValues, ", "))
	}

	switch field.Type {
	case "string":
		return "Text value"
	case "integer":
		return "Integer value"
	case "array":
		return "List of values"
	default:
		return ""
	}
}
