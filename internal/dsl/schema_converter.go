// Package dsl provides DSL parsing and schema conversion utilities.
package dsl

import (
	"fmt"
	"sort"
	"strings"

	"github.com/verustcode/verustcode/pkg/logger"
	"go.uber.org/zap"
)

// MarkdownFormatSpec represents the Markdown format specification
// derived from a JSON Schema
type MarkdownFormatSpec struct {
	// Sections defines the structure of Markdown sections
	Sections []SectionSpec

	// FindingFormat defines how each finding should be formatted
	FindingFormat *FindingFormatSpec
}

// SectionSpec defines a Markdown section
type SectionSpec struct {
	// Name is the section name (e.g., "Summary", "Findings")
	Name string

	// Level is the Markdown heading level (1-6)
	Level int

	// Description provides additional context for the section
	Description string

	// Required indicates if the section must be included
	Required bool
}

// FindingFormatSpec defines how each finding should be formatted
type FindingFormatSpec struct {
	// RequiredFields are fields that must be included in each finding
	RequiredFields []FieldSpec

	// OptionalFields are fields that may be included in each finding
	OptionalFields []FieldSpec
}

// FieldSpec defines a field in the output
type FieldSpec struct {
	// Name is the field name
	Name string

	// Description provides context for the field
	Description string

	// Type is the field type (string, integer, array, etc.)
	Type string

	// EnumValues are the allowed values for enum fields
	EnumValues []string

	// Required indicates if the field is required
	Required bool
}

// BuildJSONSchema builds a JSON Schema by merging extra fields into the default schema.
// The base schema (summary, findings with core fields) is immutable.
// Extra fields are added to findings.items.properties.
// Deprecated: Use BuildJSONSchemaWithOptions for new code.
func BuildJSONSchema(cfg *OutputSchemaConfig) map[string]interface{} {
	return BuildJSONSchemaWithOptions(cfg, false)
}

// BuildJSONSchemaWithOptions builds a JSON Schema with configurable options.
// When historyCompareEnabled is true, the "status" field becomes required.
func BuildJSONSchemaWithOptions(cfg *OutputSchemaConfig, historyCompareEnabled bool) map[string]interface{} {
	// Start with the default schema
	schema := GetDefaultJSONSchema()

	// Get findings.items to potentially modify required fields
	properties := schema["properties"].(map[string]interface{})
	findings := properties["findings"].(map[string]interface{})
	items := findings["items"].(map[string]interface{})
	itemRequired := items["required"].([]string)

	// If history_compare is enabled, add "status" to required fields
	if historyCompareEnabled {
		itemRequired = append(itemRequired, "status")
		items["required"] = itemRequired
		logger.Debug("History compare enabled, status field is now required")
	}

	// If no config or no extra fields, return schema (with potential status required)
	if cfg == nil || len(cfg.ExtraFields) == 0 {
		logger.Debug("No extra fields configured, using default schema",
			zap.Bool("history_compare_enabled", historyCompareEnabled),
		)
		return schema
	}

	logger.Debug("Building JSON schema with extra fields",
		zap.Int("extra_fields_count", len(cfg.ExtraFields)),
		zap.Bool("history_compare_enabled", historyCompareEnabled),
	)

	// Get findings.items.properties to add extra fields
	itemProps := items["properties"].(map[string]interface{})

	// Add each extra field to findings.items.properties
	for _, field := range cfg.ExtraFields {
		if field.Name == "" {
			logger.Warn("Skipping extra field with empty name")
			continue
		}

		// Build field schema
		fieldSchema := map[string]interface{}{
			"type":        field.Type,
			"description": field.Description,
		}

		// Add enum values if specified
		if len(field.Enum) > 0 {
			fieldSchema["enum"] = field.Enum
		}

		// Add to properties
		itemProps[field.Name] = fieldSchema

		// Add to required list if marked as required
		if field.Required {
			itemRequired = append(itemRequired, field.Name)
		}

		logger.Debug("Added extra field to schema",
			zap.String("field_name", field.Name),
			zap.String("field_type", field.Type),
			zap.Bool("required", field.Required),
		)
	}

	// Update the required list in items
	items["required"] = itemRequired

	return schema
}

// ConvertJSONSchemaToMarkdownSpec converts a JSON Schema to a Markdown format specification
func ConvertJSONSchemaToMarkdownSpec(jsonSchema map[string]interface{}) (*MarkdownFormatSpec, error) {
	if jsonSchema == nil {
		return nil, fmt.Errorf("json schema is nil")
	}

	spec := &MarkdownFormatSpec{
		Sections: []SectionSpec{},
	}

	// Extract properties from the schema
	properties, ok := jsonSchema["properties"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("json schema does not have properties")
	}

	// Get required fields at the root level
	rootRequired := extractRequiredFields(jsonSchema)

	// Process each property as a potential section
	for propName, propValue := range properties {
		propSchema, ok := propValue.(map[string]interface{})
		if !ok {
			continue
		}

		// Determine section level (default: 2)
		level := 2

		// Get description
		description := ""
		if desc, ok := propSchema["description"].(string); ok {
			description = desc
		}

		// Check if required
		isRequired := containsString(rootRequired, propName)

		section := SectionSpec{
			Name:        propName,
			Level:       level,
			Description: description,
			Required:    isRequired,
		}
		spec.Sections = append(spec.Sections, section)

		// If this is the "findings" array, extract the finding format
		if propName == "findings" {
			findingFormat := extractFindingFormat(propSchema)
			if findingFormat != nil {
				spec.FindingFormat = findingFormat
			}
		}
	}

	// Sort sections by a reasonable order: summary first, then findings, then others
	sortSections(spec.Sections)

	return spec, nil
}

// extractRequiredFields extracts the required field names from a schema
func extractRequiredFields(schema map[string]interface{}) []string {
	required, ok := schema["required"].([]interface{})
	if !ok {
		return nil
	}

	result := make([]string, 0, len(required))
	for _, r := range required {
		if s, ok := r.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// extractFindingFormat extracts the finding format from a findings array schema
func extractFindingFormat(findingsSchema map[string]interface{}) *FindingFormatSpec {
	// Get the items schema for array type
	items, ok := findingsSchema["items"].(map[string]interface{})
	if !ok {
		return nil
	}

	// Get properties of each finding
	properties, ok := items["properties"].(map[string]interface{})
	if !ok {
		return nil
	}

	// Get required fields
	requiredFields := extractRequiredFields(items)

	format := &FindingFormatSpec{
		RequiredFields: []FieldSpec{},
		OptionalFields: []FieldSpec{},
	}

	// Determine field order
	fieldOrder := getFieldOrder(properties)

	// Process each field in order
	for _, fieldName := range fieldOrder {
		fieldSchema, ok := properties[fieldName].(map[string]interface{})
		if !ok {
			continue
		}

		fieldSpec := extractFieldSpec(fieldName, fieldSchema)
		fieldSpec.Required = containsString(requiredFields, fieldName)

		if fieldSpec.Required {
			format.RequiredFields = append(format.RequiredFields, fieldSpec)
		} else {
			format.OptionalFields = append(format.OptionalFields, fieldSpec)
		}
	}

	return format
}

// extractFieldSpec extracts a field specification from a property schema
func extractFieldSpec(fieldName string, fieldSchema map[string]interface{}) FieldSpec {
	spec := FieldSpec{
		Name: fieldName,
	}

	// Get type
	if t, ok := fieldSchema["type"].(string); ok {
		spec.Type = t
	}

	// Get description
	if desc, ok := fieldSchema["description"].(string); ok {
		spec.Description = desc
	}

	// Get enum values
	if enum, ok := fieldSchema["enum"].([]interface{}); ok {
		spec.EnumValues = make([]string, 0, len(enum))
		for _, e := range enum {
			if s, ok := e.(string); ok {
				spec.EnumValues = append(spec.EnumValues, s)
			}
		}
	}

	return spec
}

// getFieldOrder returns the order of fields to display
func getFieldOrder(properties map[string]interface{}) []string {
	// Default order: collect all keys and sort alphabetically
	keys := make([]string, 0, len(properties))
	for k := range properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Prioritize common fields
	priority := []string{"severity", "title", "description", "file_path", "suggestion", "code_snippet"}
	result := make([]string, 0, len(keys))

	// Add priority fields first
	for _, p := range priority {
		if containsString(keys, p) {
			result = append(result, p)
		}
	}

	// Add remaining fields
	for _, k := range keys {
		if !containsString(result, k) {
			result = append(result, k)
		}
	}

	return result
}

// sortSections sorts sections by priority
func sortSections(sections []SectionSpec) {
	priority := map[string]int{
		"summary":  0,
		"findings": 1,
		"stats":    2,
	}

	sort.Slice(sections, func(i, j int) bool {
		pi := 99
		pj := 99
		if p, ok := priority[strings.ToLower(sections[i].Name)]; ok {
			pi = p
		}
		if p, ok := priority[strings.ToLower(sections[j].Name)]; ok {
			pj = p
		}
		if pi != pj {
			return pi < pj
		}
		return sections[i].Name < sections[j].Name
	})
}

// GetDefaultJSONSchema returns the default JSON Schema for ReviewResult
// Uses location field instead of separate file_path/start_line/end_line
func GetDefaultJSONSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"summary": map[string]interface{}{
				"type":        "string",
				"description": "Overall review summary",
			},
			"findings": map[string]interface{}{
				"type":        "array",
				"description": "List of code review findings",
				"items": map[string]interface{}{
					"type":     "object",
					"required": []string{"severity", "title", "description"},
					"properties": map[string]interface{}{
						"severity": map[string]interface{}{
							"type":        "string",
							"description": "Severity level of the finding",
							"enum":        []string{"critical", "high", "medium", "low", "info"},
						},
						"title": map[string]interface{}{
							"type":        "string",
							"description": "Brief title of the finding",
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "Detailed description of the issue",
						},
						"category": map[string]interface{}{
							"type":        "string",
							"description": "Category of the finding (e.g., security, performance). Must be in English (only letters, numbers, hyphens, and underscores allowed).",
							"pattern":     "^[a-zA-Z0-9_-]+$",
						},
						"location": map[string]interface{}{
							"type":        "string",
							"description": "Issue location in format: path:start-end (e.g., src/main.go:10-20)",
						},
						"suggestion": map[string]interface{}{
							"type":        "string",
							"description": "Suggested fix for the issue",
						},
						"code_snippet": map[string]interface{}{
							"type":        "string",
							"description": "Relevant code snippet",
						},
						"status": map[string]interface{}{
							"type":        "string",
							"description": "Status compared to previous review (only when history_compare enabled): fixed=issue resolved, new=new issue, persists=issue still exists",
							"enum":        []string{"fixed", "new", "persists"},
						},
					},
				},
			},
		},
		"required": []string{"summary", "findings"},
	}
}
