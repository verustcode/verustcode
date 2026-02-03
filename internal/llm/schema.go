package llm

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// SchemaGenerator generates JSON Schema from Go types
type SchemaGenerator struct{}

// NewSchemaGenerator creates a new SchemaGenerator
func NewSchemaGenerator() *SchemaGenerator {
	return &SchemaGenerator{}
}

// Generate generates a JSON Schema from the given value
// The value can be a struct, map, or any other Go type
func (g *SchemaGenerator) Generate(v interface{}) (map[string]interface{}, error) {
	if v == nil {
		return nil, fmt.Errorf("cannot generate schema from nil")
	}

	t := reflect.TypeOf(v)
	// Dereference pointer types
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return g.generateFromType(t)
}

// generateFromType generates schema from a reflect.Type
func (g *SchemaGenerator) generateFromType(t reflect.Type) (map[string]interface{}, error) {
	schema := make(map[string]interface{})

	switch t.Kind() {
	case reflect.Struct:
		return g.generateStructSchema(t)
	case reflect.Map:
		schema["type"] = "object"
		if t.Key().Kind() == reflect.String {
			valueSchema, err := g.generateFromType(t.Elem())
			if err != nil {
				return nil, err
			}
			schema["additionalProperties"] = valueSchema
		}
	case reflect.Slice, reflect.Array:
		schema["type"] = "array"
		itemSchema, err := g.generateFromType(t.Elem())
		if err != nil {
			return nil, err
		}
		schema["items"] = itemSchema
	case reflect.String:
		schema["type"] = "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema["type"] = "integer"
	case reflect.Float32, reflect.Float64:
		schema["type"] = "number"
	case reflect.Bool:
		schema["type"] = "boolean"
	case reflect.Interface:
		// Any type
		schema["type"] = "object"
	default:
		schema["type"] = "string"
	}

	return schema, nil
}

// generateStructSchema generates schema for a struct type
func (g *SchemaGenerator) generateStructSchema(t reflect.Type) (map[string]interface{}, error) {
	schema := map[string]interface{}{
		"type": "object",
	}

	properties := make(map[string]interface{})
	required := make([]string, 0)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get JSON tag
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		fieldName := field.Name
		isRequired := true

		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				fieldName = parts[0]
			}
			// Check for omitempty
			for _, part := range parts[1:] {
				if part == "omitempty" {
					isRequired = false
					break
				}
			}
		}

		// Generate schema for field type
		fieldSchema, err := g.generateFromType(field.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to generate schema for field %s: %w", field.Name, err)
		}

		// Add description from tag if available
		if desc := field.Tag.Get("description"); desc != "" {
			fieldSchema["description"] = desc
		}

		properties[fieldName] = fieldSchema
		if isRequired {
			required = append(required, fieldName)
		}
	}

	schema["properties"] = properties
	if len(required) > 0 {
		schema["required"] = required
	}

	return schema, nil
}

// ToJSONString converts a schema to a JSON string
func (g *SchemaGenerator) ToJSONString(schema map[string]interface{}) (string, error) {
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// BuildSchemaPrompt builds a prompt instruction for structured output
func BuildSchemaPrompt(schema *ResponseSchema) string {
	if schema == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n## Output Format\n")

	if schema.Description != "" {
		sb.WriteString(schema.Description)
		sb.WriteString("\n\n")
	}

	sb.WriteString("Please provide your response in the following JSON format:\n")

	// Generate schema if it's a struct
	generator := NewSchemaGenerator()
	schemaMap, err := generator.Generate(schema.Schema)
	if err == nil {
		jsonSchema, err := generator.ToJSONString(schemaMap)
		if err == nil {
			sb.WriteString("```json\n")
			sb.WriteString(jsonSchema)
			sb.WriteString("\n```\n")
		}
	}

	if schema.Strict {
		sb.WriteString("\nIMPORTANT: Your response MUST be valid JSON that strictly follows this schema. ")
		sb.WriteString("Do not include any text before or after the JSON object.\n")
	} else {
		sb.WriteString("\nPlease ensure your response contains valid JSON that follows this schema.\n")
	}

	return sb.String()
}

// ExtractJSON extracts JSON from a string that may contain other text
func ExtractJSON(content string) (string, error) {
	// Find the first { and last }
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")

	if start == -1 || end == -1 || end <= start {
		// Try to find array
		start = strings.Index(content, "[")
		end = strings.LastIndex(content, "]")
		if start == -1 || end == -1 || end <= start {
			return "", fmt.Errorf("no valid JSON found in content")
		}
	}

	return content[start : end+1], nil
}

// ParseResponseJSON extracts and parses JSON from the response content
func ParseResponseJSON(content string, target interface{}) error {
	jsonStr, err := ExtractJSON(content)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(jsonStr), target)
}

// MarkdownOutputPrompt returns the prompt instruction for Markdown output format
// Used when no ResponseSchema is provided, indicating free-form Markdown output
func MarkdownOutputPrompt() string {
	return `

## Output Format

Please provide your response in **Markdown format** with clear structure
`
}
