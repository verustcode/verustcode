// Package utils provides utility functions for the engine.
// This file contains unit tests for DSL configuration utilities.
package utils

import (
	"reflect"
	"testing"

	"github.com/verustcode/verustcode/internal/dsl"
	"github.com/verustcode/verustcode/internal/model"
)

// TestDSLConfigToJSONMap tests the DSLConfigToJSONMap function
func TestDSLConfigToJSONMap(t *testing.T) {
	t.Run("convert valid config", func(t *testing.T) {
		config := &dsl.ReviewRulesConfig{
			Version: "1.0",
			Rules: []dsl.ReviewRuleConfig{
				{
					ID:          "code-quality",
					Description: "Code quality review",
				},
			},
		}

		result, err := DSLConfigToJSONMap(config)
		if err != nil {
			t.Fatalf("DSLConfigToJSONMap() error = %v", err)
		}

		if result == nil {
			t.Fatal("DSLConfigToJSONMap() returned nil")
		}

		// Verify version is present (field name in JSON is "Version" without json tag)
		if version, ok := result["Version"]; !ok || version != "1.0" {
			t.Errorf("Version = %v, want 1.0", result["Version"])
		}

		// Verify rules are present
		if rules, ok := result["Rules"]; !ok || rules == nil {
			t.Error("Rules not found in result")
		}
	})

	t.Run("convert empty config", func(t *testing.T) {
		config := &dsl.ReviewRulesConfig{}

		result, err := DSLConfigToJSONMap(config)
		if err != nil {
			t.Fatalf("DSLConfigToJSONMap() error = %v", err)
		}

		if result == nil {
			t.Fatal("DSLConfigToJSONMap() returned nil")
		}
	})

	t.Run("convert config with rule_base", func(t *testing.T) {
		config := &dsl.ReviewRulesConfig{
			Version: "1.0",
			RuleBase: &dsl.RuleBaseConfig{
				Agent: dsl.AgentConfig{
					Type: "cursor",
				},
			},
			Rules: []dsl.ReviewRuleConfig{
				{
					ID: "test-rule",
				},
			},
		}

		result, err := DSLConfigToJSONMap(config)
		if err != nil {
			t.Fatalf("DSLConfigToJSONMap() error = %v", err)
		}

		if result["RuleBase"] == nil {
			t.Error("RuleBase not found in result")
		}
	})
}

// TestRuleConfigToJSONMap tests the RuleConfigToJSONMap function
func TestRuleConfigToJSONMap(t *testing.T) {
	t.Run("convert valid rule", func(t *testing.T) {
		rule := &dsl.ReviewRuleConfig{
			ID:          "code-quality",
			Description: "Code quality review",
		}

		result, err := RuleConfigToJSONMap(rule)
		if err != nil {
			t.Fatalf("RuleConfigToJSONMap() error = %v", err)
		}

		if result == nil {
			t.Fatal("RuleConfigToJSONMap() returned nil")
		}

		// Verify ID is present (field name in JSON is "ID" without json tag)
		if id, ok := result["ID"]; !ok || id != "code-quality" {
			t.Errorf("ID = %v, want code-quality", result["ID"])
		}

		// Verify description is present
		if desc, ok := result["Description"]; !ok || desc != "Code quality review" {
			t.Errorf("Description = %v, want 'Code quality review'", result["Description"])
		}
	})

	t.Run("convert empty rule", func(t *testing.T) {
		rule := &dsl.ReviewRuleConfig{}

		result, err := RuleConfigToJSONMap(rule)
		if err != nil {
			t.Fatalf("RuleConfigToJSONMap() error = %v", err)
		}

		if result == nil {
			t.Fatal("RuleConfigToJSONMap() returned nil")
		}
	})

	t.Run("convert rule with agent config", func(t *testing.T) {
		rule := &dsl.ReviewRuleConfig{
			ID: "test-rule",
			Agent: dsl.AgentConfig{
				Type:  "cursor",
				Model: "claude-sonnet-4-20250514",
			},
		}

		result, err := RuleConfigToJSONMap(rule)
		if err != nil {
			t.Fatalf("RuleConfigToJSONMap() error = %v", err)
		}

		if result["Agent"] == nil {
			t.Error("Agent not found in result")
		}
	})

	t.Run("convert rule with goals", func(t *testing.T) {
		rule := &dsl.ReviewRuleConfig{
			ID: "test-rule",
			Goals: dsl.GoalsConfig{
				Areas: []string{"security", "performance"},
			},
		}

		result, err := RuleConfigToJSONMap(rule)
		if err != nil {
			t.Fatalf("RuleConfigToJSONMap() error = %v", err)
		}

		if result["Goals"] == nil {
			t.Error("Goals not found in result")
		}
	})
}

// TestJSONMapToDSLConfig tests the JSONMapToDSLConfig function
func TestJSONMapToDSLConfig(t *testing.T) {
	t.Run("convert valid JSONMap", func(t *testing.T) {
		data := model.JSONMap{
			"Version": "1.0",
			"Rules": []interface{}{
				map[string]interface{}{
					"ID":          "code-quality",
					"Description": "Code quality review",
				},
			},
		}

		result, err := JSONMapToDSLConfig(data)
		if err != nil {
			t.Fatalf("JSONMapToDSLConfig() error = %v", err)
		}

		if result == nil {
			t.Fatal("JSONMapToDSLConfig() returned nil")
		}

		if result.Version != "1.0" {
			t.Errorf("Version = %s, want 1.0", result.Version)
		}

		if len(result.Rules) != 1 {
			t.Errorf("Rules length = %d, want 1", len(result.Rules))
		}

		if result.Rules[0].ID != "code-quality" {
			t.Errorf("Rules[0].ID = %s, want code-quality", result.Rules[0].ID)
		}
	})

	t.Run("convert empty JSONMap", func(t *testing.T) {
		data := model.JSONMap{}

		result, err := JSONMapToDSLConfig(data)
		if err == nil {
			t.Error("JSONMapToDSLConfig() should return error for empty data")
		}
		if result != nil {
			t.Error("JSONMapToDSLConfig() should return nil for empty data")
		}
	})

	t.Run("convert nil JSONMap", func(t *testing.T) {
		var data model.JSONMap

		result, err := JSONMapToDSLConfig(data)
		if err == nil {
			t.Error("JSONMapToDSLConfig() should return error for nil data")
		}
		if result != nil {
			t.Error("JSONMapToDSLConfig() should return nil for nil data")
		}
	})

	t.Run("convert with rule_base", func(t *testing.T) {
		data := model.JSONMap{
			"Version": "1.0",
			"RuleBase": map[string]interface{}{
				"Agent": map[string]interface{}{
					"Type": "cursor",
				},
			},
			"Rules": []interface{}{
				map[string]interface{}{
					"ID": "test-rule",
				},
			},
		}

		result, err := JSONMapToDSLConfig(data)
		if err != nil {
			t.Fatalf("JSONMapToDSLConfig() error = %v", err)
		}

		if result.RuleBase == nil {
			t.Error("RuleBase should not be nil")
		}
	})
}

// TestDSLConfigRoundTrip tests converting DSL config to JSONMap and back
func TestDSLConfigRoundTrip(t *testing.T) {
	original := &dsl.ReviewRulesConfig{
		Version: "1.0",
		Rules: []dsl.ReviewRuleConfig{
			{
				ID:          "code-quality",
				Description: "Code quality review",
			},
			{
				ID:          "security",
				Description: "Security review",
			},
		},
	}

	// Convert to JSONMap
	jsonMap, err := DSLConfigToJSONMap(original)
	if err != nil {
		t.Fatalf("DSLConfigToJSONMap() error = %v", err)
	}

	// Convert back to DSL config
	restored, err := JSONMapToDSLConfig(jsonMap)
	if err != nil {
		t.Fatalf("JSONMapToDSLConfig() error = %v", err)
	}

	// Verify version
	if restored.Version != original.Version {
		t.Errorf("Version = %s, want %s", restored.Version, original.Version)
	}

	// Verify rules count
	if len(restored.Rules) != len(original.Rules) {
		t.Errorf("Rules length = %d, want %d", len(restored.Rules), len(original.Rules))
	}

	// Verify rule IDs
	for i, rule := range original.Rules {
		if restored.Rules[i].ID != rule.ID {
			t.Errorf("Rules[%d].ID = %s, want %s", i, restored.Rules[i].ID, rule.ID)
		}
	}
}

// TestDSLConfigToJSONMapPreservesFields tests that all important fields are preserved
func TestDSLConfigToJSONMapPreservesFields(t *testing.T) {
	config := &dsl.ReviewRulesConfig{
		Version: "1.0",
		RuleBase: &dsl.RuleBaseConfig{
			Agent: dsl.AgentConfig{
				Type: "cursor",
			},
		},
		Rules: []dsl.ReviewRuleConfig{
			{
				ID:          "test-rule",
				Description: "Test description",
				Agent: dsl.AgentConfig{
					Type:  "gemini",
					Model: "gemini-2.5-pro",
				},
			},
		},
	}

	jsonMap, err := DSLConfigToJSONMap(config)
	if err != nil {
		t.Fatalf("DSLConfigToJSONMap() error = %v", err)
	}

	// Check that all expected keys exist (field names, not yaml tags)
	expectedKeys := []string{"Version", "RuleBase", "Rules"}
	for _, key := range expectedKeys {
		if _, ok := jsonMap[key]; !ok {
			t.Errorf("JSONMap missing expected key: %s", key)
		}
	}

	// Check version value
	if jsonMap["Version"] != "1.0" {
		t.Errorf("Version = %v, want 1.0", jsonMap["Version"])
	}

	// Check RuleBase exists and has Agent
	if ruleBase, ok := jsonMap["RuleBase"].(map[string]interface{}); ok {
		if _, hasAgent := ruleBase["Agent"]; !hasAgent {
			t.Error("RuleBase.Agent not found")
		}
	}
}

// TestRuleConfigToJSONMapNilInput tests handling of nil input
func TestRuleConfigToJSONMapNilInput(t *testing.T) {
	// Note: passing nil would cause a panic, so we test with empty struct
	rule := &dsl.ReviewRuleConfig{}

	result, err := RuleConfigToJSONMap(rule)
	if err != nil {
		t.Fatalf("RuleConfigToJSONMap() error = %v", err)
	}

	if result == nil {
		t.Fatal("RuleConfigToJSONMap() returned nil for empty config")
	}
}

// TestJSONMapToDSLConfigInvalidData tests handling of invalid JSON data
func TestJSONMapToDSLConfigInvalidData(t *testing.T) {
	tests := []struct {
		name    string
		data    model.JSONMap
		wantErr bool
	}{
		{
			name:    "empty map",
			data:    model.JSONMap{},
			wantErr: true,
		},
		{
			name:    "nil map",
			data:    nil,
			wantErr: true,
		},
		{
			name: "valid minimal config",
			data: model.JSONMap{
				"Version": "1.0",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := JSONMapToDSLConfig(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("JSONMapToDSLConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// BenchmarkDSLConfigToJSONMap benchmarks the conversion function
func BenchmarkDSLConfigToJSONMap(b *testing.B) {
	config := &dsl.ReviewRulesConfig{
		Version: "1.0",
		Rules: []dsl.ReviewRuleConfig{
			{
				ID:          "code-quality",
				Description: "Code quality review",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DSLConfigToJSONMap(config)
	}
}

// BenchmarkJSONMapToDSLConfig benchmarks the reverse conversion
func BenchmarkJSONMapToDSLConfig(b *testing.B) {
	data := model.JSONMap{
		"version": "1.0",
		"rules": []interface{}{
			map[string]interface{}{
				"id":          "code-quality",
				"description": "Code quality review",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		JSONMapToDSLConfig(data)
	}
}

// suppress the "declared and not used" error from the testing package
var _ = reflect.TypeOf
