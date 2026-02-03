// Package config provides configuration management for the application.
// This file contains tests for sensitive value handling in settings.
package config

import (
	"encoding/json"
	"testing"

	"github.com/verustcode/verustcode/internal/model"
)

// TestIsSensitiveKeyConfig tests the isSensitiveKey function in config package
func TestIsSensitiveKeyConfig(t *testing.T) {
	testCases := []struct {
		key      string
		expected bool
	}{
		{"api_key", true},
		{"token", true},
		{"secret", true},
		{"password", true},
		{"private_key", true},
		{"name", false},
		{"url", false},
		{"enabled", false},
	}

	for _, tc := range testCases {
		t.Run(tc.key, func(t *testing.T) {
			result := isSensitiveKey(tc.key)
			if result != tc.expected {
				t.Errorf("isSensitiveKey(%q) = %v, want %v", tc.key, result, tc.expected)
			}
		})
	}
}

// TestIsMaskedValueConfig tests the isMaskedValue function in config package
func TestIsMaskedValueConfig(t *testing.T) {
	testCases := []struct {
		value    string
		expected bool
	}{
		{"sk-1****cdef", true},
		{"****", true},
		{"normal-value", false},
		{"", false},
	}

	for _, tc := range testCases {
		t.Run(tc.value, func(t *testing.T) {
			result := isMaskedValue(tc.value)
			if result != tc.expected {
				t.Errorf("isMaskedValue(%q) = %v, want %v", tc.value, result, tc.expected)
			}
		})
	}
}

// TestRestoreMaskedValues tests the restoreMaskedValues function
func TestRestoreMaskedValues(t *testing.T) {
	t.Run("Restore simple string value", func(t *testing.T) {
		existingMap := map[string]model.SystemSetting{
			"api_key": {
				Key:   "api_key",
				Value: `"sk-original-secret-key"`,
			},
		}

		// User submits masked value
		result := restoreMaskedValues("api_key", "sk-o****-key", existingMap)

		if result != "sk-original-secret-key" {
			t.Errorf("Expected original value to be restored, got %v", result)
		}
	})

	t.Run("Non-masked value is not restored", func(t *testing.T) {
		existingMap := map[string]model.SystemSetting{
			"api_key": {
				Key:   "api_key",
				Value: `"sk-original-secret-key"`,
			},
		}

		// User submits new value (not masked)
		newValue := "sk-brand-new-secret-key"
		result := restoreMaskedValues("api_key", newValue, existingMap)

		if result != newValue {
			t.Errorf("Expected new value to be kept, got %v", result)
		}
	})

	t.Run("Non-sensitive key with **** is not treated as masked", func(t *testing.T) {
		existingMap := map[string]model.SystemSetting{}

		// Non-sensitive key with asterisks (maybe a name with wildcards)
		value := "pattern****suffix"
		result := restoreMaskedValues("name", value, existingMap)

		if result != value {
			t.Errorf("Expected value to be unchanged, got %v", result)
		}
	})

	t.Run("Restore nested map values", func(t *testing.T) {
		existingMap := map[string]model.SystemSetting{
			"provider_config": {
				Key:   "provider_config",
				Value: `{"name":"test","api_key":"real-secret-key-123456"}`,
			},
		}

		// User submits with masked api_key
		input := map[string]interface{}{
			"name":    "test",
			"api_key": "real****3456", // masked value
		}

		result := restoreMaskedValues("provider_config", input, existingMap)
		resultMap, ok := result.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map result, got %T", result)
		}

		// api_key should be restored to original
		if resultMap["api_key"] != "real-secret-key-123456" {
			t.Errorf("Expected api_key to be restored, got %v", resultMap["api_key"])
		}

		// name should be unchanged
		if resultMap["name"] != "test" {
			t.Errorf("Expected name to be unchanged, got %v", resultMap["name"])
		}
	})

	t.Run("Restore array of objects", func(t *testing.T) {
		// Existing array with two providers
		existingArray := []interface{}{
			map[string]interface{}{
				"name":    "Provider 1",
				"api_key": "secret-key-provider-1",
			},
			map[string]interface{}{
				"name":    "Provider 2",
				"api_key": "secret-key-provider-2",
			},
		}
		existingJSON, _ := json.Marshal(existingArray)

		existingMap := map[string]model.SystemSetting{
			"providers": {
				Key:   "providers",
				Value: string(existingJSON),
			},
		}

		// User submits with masked values
		input := []interface{}{
			map[string]interface{}{
				"name":    "Provider 1",
				"api_key": "secr****er-1", // masked
			},
			map[string]interface{}{
				"name":    "Provider 2 Updated",     // name changed
				"api_key": "new-key-for-provider-2", // new value (not masked)
			},
		}

		result := restoreMaskedValues("providers", input, existingMap)
		resultArray, ok := result.([]interface{})
		if !ok {
			t.Fatalf("Expected array result, got %T", result)
		}

		// Provider 1's api_key should be restored
		p1, _ := resultArray[0].(map[string]interface{})
		if p1["api_key"] != "secret-key-provider-1" {
			t.Errorf("Provider 1 api_key should be restored, got %v", p1["api_key"])
		}

		// Provider 2's api_key is new (not masked), should be kept
		p2, _ := resultArray[1].(map[string]interface{})
		if p2["api_key"] != "new-key-for-provider-2" {
			t.Errorf("Provider 2 api_key should be the new value, got %v", p2["api_key"])
		}

		// Names should be as submitted
		if p2["name"] != "Provider 2 Updated" {
			t.Errorf("Provider 2 name should be updated, got %v", p2["name"])
		}
	})

	t.Run("Missing existing value returns masked value as-is", func(t *testing.T) {
		existingMap := map[string]model.SystemSetting{}

		// Masked value but no existing record
		result := restoreMaskedValues("api_key", "sk-1****cdef", existingMap)

		// Since there's no existing value to restore from, return as-is
		if result != "sk-1****cdef" {
			t.Errorf("Expected masked value to remain, got %v", result)
		}
	})
}
