package dsl

import (
	"testing"
)

// TestBuildJSONSchema tests schema building with extra fields
func TestBuildJSONSchema(t *testing.T) {
	t.Run("nil config returns default schema", func(t *testing.T) {
		schema := BuildJSONSchema(nil)
		if schema == nil {
			t.Error("BuildJSONSchema(nil) should return default schema, got nil")
		}

		// Verify it has the expected structure
		properties, ok := schema["properties"].(map[string]interface{})
		if !ok {
			t.Fatal("Schema should have properties")
		}
		if _, ok := properties["summary"]; !ok {
			t.Error("Schema should have 'summary' property")
		}
		if _, ok := properties["findings"]; !ok {
			t.Error("Schema should have 'findings' property")
		}
	})

	t.Run("empty extra fields returns default schema", func(t *testing.T) {
		cfg := &OutputSchemaConfig{
			ExtraFields: []ExtraFieldConfig{},
		}
		schema := BuildJSONSchema(cfg)
		if schema == nil {
			t.Error("BuildJSONSchema() should return default schema, got nil")
		}
	})

	t.Run("extra fields are merged into findings items", func(t *testing.T) {
		cfg := &OutputSchemaConfig{
			ExtraFields: []ExtraFieldConfig{
				{
					Name:        "vulnerability_type",
					Type:        "string",
					Description: "Type of security vulnerability",
					Required:    true,
					Enum:        []string{"sql_injection", "xss", "csrf"},
				},
				{
					Name:        "cve_id",
					Type:        "string",
					Description: "CVE identifier if applicable",
					Required:    false,
				},
			},
		}
		schema := BuildJSONSchema(cfg)
		if schema == nil {
			t.Fatal("BuildJSONSchema() returned nil")
		}

		// Navigate to findings.items.properties
		properties := schema["properties"].(map[string]interface{})
		findings := properties["findings"].(map[string]interface{})
		items := findings["items"].(map[string]interface{})
		itemProps := items["properties"].(map[string]interface{})

		// Check vulnerability_type was added
		vulnType, ok := itemProps["vulnerability_type"].(map[string]interface{})
		if !ok {
			t.Error("Extra field 'vulnerability_type' should be added to schema")
		}
		if vulnType["type"] != "string" {
			t.Errorf("vulnerability_type.type = %v, want string", vulnType["type"])
		}
		if vulnType["description"] != "Type of security vulnerability" {
			t.Errorf("vulnerability_type.description = %v, want 'Type of security vulnerability'", vulnType["description"])
		}

		// Check enum values
		enumVals, ok := vulnType["enum"].([]string)
		if !ok {
			t.Error("vulnerability_type should have enum values")
		}
		if len(enumVals) != 3 {
			t.Errorf("vulnerability_type.enum has %d values, want 3", len(enumVals))
		}

		// Check cve_id was added
		if _, ok := itemProps["cve_id"].(map[string]interface{}); !ok {
			t.Error("Extra field 'cve_id' should be added to schema")
		}

		// Check required fields
		itemRequired := items["required"].([]string)
		hasVulnTypeRequired := false
		hasCveIdRequired := false
		for _, req := range itemRequired {
			if req == "vulnerability_type" {
				hasVulnTypeRequired = true
			}
			if req == "cve_id" {
				hasCveIdRequired = true
			}
		}
		if !hasVulnTypeRequired {
			t.Error("vulnerability_type should be in required list")
		}
		if hasCveIdRequired {
			t.Error("cve_id should NOT be in required list (required=false)")
		}
	})

	t.Run("skip extra fields with empty name", func(t *testing.T) {
		cfg := &OutputSchemaConfig{
			ExtraFields: []ExtraFieldConfig{
				{
					Name:        "",
					Type:        "string",
					Description: "Empty name field",
				},
				{
					Name:        "valid_field",
					Type:        "string",
					Description: "Valid field",
				},
			},
		}
		schema := BuildJSONSchema(cfg)
		if schema == nil {
			t.Fatal("BuildJSONSchema() returned nil")
		}

		// Navigate to findings.items.properties
		properties := schema["properties"].(map[string]interface{})
		findings := properties["findings"].(map[string]interface{})
		items := findings["items"].(map[string]interface{})
		itemProps := items["properties"].(map[string]interface{})

		// valid_field should exist
		if _, ok := itemProps["valid_field"]; !ok {
			t.Error("valid_field should be added to schema")
		}

		// Empty name field should not be added (and shouldn't cause issues)
		if _, ok := itemProps[""]; ok {
			t.Error("Empty name field should not be added to schema")
		}
	})
}

// TestConvertJSONSchemaToMarkdownSpec tests JSON schema to Markdown conversion
func TestConvertJSONSchemaToMarkdownSpec(t *testing.T) {
	t.Run("nil schema", func(t *testing.T) {
		_, err := ConvertJSONSchemaToMarkdownSpec(nil)
		if err == nil {
			t.Error("ConvertJSONSchemaToMarkdownSpec(nil) expected error, got nil")
		}
	})

	t.Run("schema without properties", func(t *testing.T) {
		schema := map[string]interface{}{
			"type": "object",
		}
		_, err := ConvertJSONSchemaToMarkdownSpec(schema)
		if err == nil {
			t.Error("ConvertJSONSchemaToMarkdownSpec() expected error for schema without properties, got nil")
		}
	})

	t.Run("valid schema with findings", func(t *testing.T) {
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"summary": map[string]interface{}{
					"type":        "string",
					"description": "Overall summary",
				},
				"findings": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"severity": map[string]interface{}{
								"type": "string",
								"enum": []interface{}{"low", "medium", "high"},
							},
							"title": map[string]interface{}{
								"type": "string",
							},
						},
						"required": []interface{}{"severity", "title"},
					},
				},
			},
			"required": []interface{}{"summary", "findings"},
		}

		spec, err := ConvertJSONSchemaToMarkdownSpec(schema)
		if err != nil {
			t.Errorf("ConvertJSONSchemaToMarkdownSpec() unexpected error: %v", err)
			return
		}

		if spec == nil {
			t.Error("ConvertJSONSchemaToMarkdownSpec() returned nil spec")
			return
		}

		if len(spec.Sections) == 0 {
			t.Error("ConvertJSONSchemaToMarkdownSpec() returned spec with no sections")
		}

		if spec.FindingFormat == nil {
			t.Error("ConvertJSONSchemaToMarkdownSpec() should extract finding format")
		}
	})
}

// TestGetDefaultJSONSchema tests the default JSON schema
func TestGetDefaultJSONSchema(t *testing.T) {
	schema := GetDefaultJSONSchema()

	if schema == nil {
		t.Fatal("GetDefaultJSONSchema() returned nil")
	}

	// Check that schema has required fields
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Default schema should have properties")
	}

	if _, ok := properties["summary"]; !ok {
		t.Error("Default schema should have 'summary' property")
	}

	if _, ok := properties["findings"]; !ok {
		t.Error("Default schema should have 'findings' property")
	}

	// Check findings structure
	findings, ok := properties["findings"].(map[string]interface{})
	if !ok {
		t.Fatal("findings property should be a map")
	}

	items, ok := findings["items"].(map[string]interface{})
	if !ok {
		t.Fatal("findings should have items")
	}

	itemProps, ok := items["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("findings.items should have properties")
	}

	// Check base finding fields
	baseFields := []string{"severity", "title", "description", "category", "location", "suggestion", "code_snippet"}
	for _, field := range baseFields {
		if _, ok := itemProps[field]; !ok {
			t.Errorf("Default schema should have '%s' field in findings items", field)
		}
	}

	// Check category field has pattern validation for English-only requirement
	category, ok := itemProps["category"].(map[string]interface{})
	if !ok {
		t.Fatal("category field should be a map")
	}
	pattern, ok := category["pattern"].(string)
	if !ok {
		t.Error("category field should have 'pattern' property for English-only validation")
	} else if pattern != "^[a-zA-Z0-9_-]+$" {
		t.Errorf("category pattern = %v, want ^[a-zA-Z0-9_-]+$", pattern)
	}

	// Check required fields
	required, ok := items["required"].([]string)
	if !ok {
		t.Fatal("findings.items should have required array")
	}
	requiredSet := make(map[string]bool)
	for _, r := range required {
		requiredSet[r] = true
	}
	if !requiredSet["severity"] || !requiredSet["title"] || !requiredSet["description"] {
		t.Error("severity, title, and description should be required in findings items")
	}
}
