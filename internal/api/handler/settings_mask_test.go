// Package handler provides HTTP handlers for the API.
// This file contains tests for sensitive value masking functionality.
package handler

import (
	"testing"
)

// TestIsSensitiveKey tests the isSensitiveKey function
func TestIsSensitiveKey(t *testing.T) {
	testCases := []struct {
		key      string
		expected bool
	}{
		// Sensitive keys - should return true
		{"api_key", true},
		{"API_KEY", true},
		{"token", true},
		{"access_token", true},
		{"private_token", true},
		{"secret", true},
		{"client_secret", true},
		{"webhook_secret", true},
		{"password", true},
		{"private_key", true},
		{"jwt_secret", true},

		// Non-sensitive keys - should return false
		{"name", false},
		{"url", false},
		{"base_url", false},
		{"enabled", false},
		{"timeout", false},
		{"model", false},
		{"max_retries", false},
		{"provider_type", false},
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

// TestMaskSensitiveValue tests the maskSensitiveValue function
func TestMaskSensitiveValue(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Short value (less than 8 chars)",
			input:    "short",
			expected: "****",
		},
		{
			name:     "Exactly 8 chars",
			input:    "12345678",
			expected: "****",
		},
		{
			name:     "Normal API key",
			input:    "sk-1234567890abcdef",
			expected: "sk-1****cdef",
		},
		{
			name:     "Long token",
			input:    "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			expected: "ghp_****xxxx",
		},
		{
			name:     "GitLab personal token",
			input:    "glpat-xxxxxxxxxxxxxxxx",
			expected: "glpa****xxxx",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := maskSensitiveValue(tc.input)
			if result != tc.expected {
				t.Errorf("maskSensitiveValue(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

// TestIsMaskedValue tests the isMaskedValue function
func TestIsMaskedValue(t *testing.T) {
	testCases := []struct {
		value    string
		expected bool
	}{
		{"sk-1****cdef", true},
		{"ghp_****xxxx", true},
		{"****", true},
		{"prefix****suffix", true},
		{"plain-value", false},
		{"no-asterisks", false},
		{"***", false},  // Only 3 asterisks, not 4
		{"*****", true}, // 5 asterisks contains ****
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

// TestMaskSettingsValue tests the maskSettingsValue function with various value types
func TestMaskSettingsValue(t *testing.T) {
	t.Run("String value - sensitive key", func(t *testing.T) {
		result := maskSettingsValue("api_key", "sk-1234567890abcdef")
		if result != "sk-1****cdef" {
			t.Errorf("Expected masked value, got %v", result)
		}
	})

	t.Run("String value - non-sensitive key", func(t *testing.T) {
		result := maskSettingsValue("base_url", "https://api.example.com")
		if result != "https://api.example.com" {
			t.Errorf("Expected unmasked value, got %v", result)
		}
	})

	t.Run("Empty string - sensitive key", func(t *testing.T) {
		result := maskSettingsValue("api_key", "")
		if result != "" {
			t.Errorf("Expected empty string, got %v", result)
		}
	})

	t.Run("Map value with sensitive fields", func(t *testing.T) {
		input := map[string]interface{}{
			"name":      "test-provider",
			"api_key":   "sk-secret-key-12345",
			"base_url":  "https://api.example.com",
			"token":     "glpat-abcdefghijklmnop",
			"is_active": true,
		}

		result := maskSettingsValue("provider", input)
		resultMap, ok := result.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map result, got %T", result)
		}

		// Non-sensitive fields should be unchanged
		if resultMap["name"] != "test-provider" {
			t.Errorf("name should be unchanged, got %v", resultMap["name"])
		}
		if resultMap["base_url"] != "https://api.example.com" {
			t.Errorf("base_url should be unchanged, got %v", resultMap["base_url"])
		}
		if resultMap["is_active"] != true {
			t.Errorf("is_active should be unchanged, got %v", resultMap["is_active"])
		}

		// Sensitive fields should be masked
		if resultMap["api_key"] == "sk-secret-key-12345" {
			t.Errorf("api_key should be masked, got %v", resultMap["api_key"])
		}
		if resultMap["token"] == "glpat-abcdefghijklmnop" {
			t.Errorf("token should be masked, got %v", resultMap["token"])
		}
	})

	t.Run("Array of maps with sensitive fields", func(t *testing.T) {
		input := []interface{}{
			map[string]interface{}{
				"name":    "Provider 1",
				"api_key": "key-provider-1-secret",
			},
			map[string]interface{}{
				"name":    "Provider 2",
				"api_key": "key-provider-2-secret",
			},
		}

		result := maskSettingsValue("providers", input)
		resultArray, ok := result.([]interface{})
		if !ok {
			t.Fatalf("Expected array result, got %T", result)
		}

		if len(resultArray) != 2 {
			t.Fatalf("Expected 2 items, got %d", len(resultArray))
		}

		for i, item := range resultArray {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				t.Errorf("Item %d should be a map", i)
				continue
			}

			// api_key should be masked
			if apiKey, ok := itemMap["api_key"].(string); ok {
				if !isMaskedValue(apiKey) {
					t.Errorf("Item %d api_key should be masked, got %v", i, apiKey)
				}
			}

			// name should be unchanged
			if name, ok := itemMap["name"].(string); ok {
				if isMaskedValue(name) {
					t.Errorf("Item %d name should not be masked, got %v", i, name)
				}
			}
		}
	})

	t.Run("Non-string values are passed through", func(t *testing.T) {
		// Integer
		result := maskSettingsValue("timeout", 30)
		if result != 30 {
			t.Errorf("Integer should be unchanged, got %v", result)
		}

		// Boolean
		result = maskSettingsValue("enabled", true)
		if result != true {
			t.Errorf("Boolean should be unchanged, got %v", result)
		}

		// Float
		result = maskSettingsValue("rate", 0.5)
		if result != 0.5 {
			t.Errorf("Float should be unchanged, got %v", result)
		}
	})
}
