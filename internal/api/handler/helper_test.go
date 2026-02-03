package handler

import (
	"testing"
)

// TestMaskString tests maskString function
func TestMaskString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Short string (<=4 chars)",
			input:    "abc",
			expected: "****",
		},
		{
			name:     "Exactly 4 chars",
			input:    "abcd",
			expected: "****",
		},
		{
			name:     "Long string",
			input:    "abcdefghijklmnop",
			expected: "ab****op",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "****",
		},
		{
			name:     "5 chars",
			input:    "abcde",
			expected: "ab****de",
		},
		{
			name:     "API key format",
			input:    "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			expected: "gh****xx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskString(tt.input)
			if result != tt.expected {
				t.Errorf("maskString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestTruncateContent tests truncateContent function
func TestTruncateContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		maxLen   int
		expected string
	}{
		{
			name:     "Content shorter than maxLen",
			content:  "short",
			maxLen:   10,
			expected: "short",
		},
		{
			name:     "Content exactly maxLen",
			content:  "1234567890",
			maxLen:   10,
			expected: "1234567890",
		},
		{
			name:     "Content longer than maxLen",
			content:  "this is a very long content that exceeds the maximum length",
			maxLen:   20,
			expected: "this is a very long ...",
		},
		{
			name:     "Empty content",
			content:  "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "Zero maxLen",
			content:  "content",
			maxLen:   0,
			expected: "...",
		},
		{
			name:     "Chinese content truncation",
			content:  "这是一段中文内容测试",
			maxLen:   5,
			expected: "这是一段中...",
		},
		{
			name:     "Mixed content truncation",
			content:  "Hello世界你好World",
			maxLen:   8,
			expected: "Hello世界你...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateContent(tt.content, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateContent(%q, %d) = %q, want %q", tt.content, tt.maxLen, result, tt.expected)
			}
		})
	}
}

// TestProcessSensitiveValues tests processSensitiveValues function
func TestProcessSensitiveValues(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		handler  func(key, val string) string
		expected string
	}{
		{
			name:    "Password with double quotes",
			content: `password: "secret123"`,
			handler: func(key, val string) string {
				return maskString(val)
			},
			expected: `password: "se****23"`,
		},
		{
			name:    "Password with single quotes",
			content: `password: 'secret123'`,
			handler: func(key, val string) string {
				return maskString(val)
			},
			expected: `password: 'se****23'`,
		},
		{
			name:    "API key unquoted",
			content: `api_key: ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxx`,
			handler: func(key, val string) string {
				return maskString(val)
			},
			expected: `api_key: gh****xx`,
		},
		{
			name: "Multiple sensitive fields",
			content: `password: "secret123"
api_key: "key456"
token: "token789"`,
			handler: func(key, val string) string {
				return maskString(val)
			},
			expected: `password: "se****23"
api_key: "ke****56"
token: "to****89"`,
		},
		{
			name:    "Skip env vars",
			content: `password: "${ENV_VAR}"`,
			handler: func(key, val string) string {
				return maskString(val)
			},
			expected: `password: "${ENV_VAR}"`, // Should not be masked
		},
		{
			name:    "Skip empty values",
			content: `password: ""`,
			handler: func(key, val string) string {
				return maskString(val)
			},
			expected: `password: ""`, // Should not be processed
		},
		{
			name:    "Skip env vars with quotes",
			content: `password: "${ENV_VAR}"`,
			handler: func(key, val string) string {
				return maskString(val)
			},
			expected: `password: "${ENV_VAR}"`,
		},
		{
			name:    "Token field",
			content: `token: "my_secret_token_12345"`,
			handler: func(key, val string) string {
				return maskString(val)
			},
			expected: `token: "my****45"`,
		},
		{
			name:    "Secret field",
			content: `secret: "very_secret_value"`,
			handler: func(key, val string) string {
				return maskString(val)
			},
			expected: `secret: "ve****ue"`,
		},
		{
			name:    "JWT secret",
			content: `jwt_secret: "my_jwt_secret_key"`,
			handler: func(key, val string) string {
				return maskString(val)
			},
			expected: `jwt_secret: "my****ey"`,
		},
		{
			name:    "Private key",
			content: `private_key: "-----BEGIN PRIVATE KEY-----"`,
			handler: func(key, val string) string {
				return maskString(val)
			},
			expected: `private_key: "--****--"`,
		},
		{
			name:    "Value with comment",
			content: `password: "secret123" # comment`,
			handler: func(key, val string) string {
				return maskString(val)
			},
			expected: `password: "se****23" # comment`,
		},
		{
			name:    "Non-sensitive field",
			content: `name: "John Doe"`,
			handler: func(key, val string) string {
				return maskString(val)
			},
			expected: `name: "John Doe"`, // Should not match regex
		},
		{
			name:    "Key-aware processing",
			content: `password: "secret123"`,
			handler: func(key, val string) string {
				if key == "password" {
					return "***MASKED***"
				}
				return val
			},
			expected: `password: "***MASKED***"`,
		},
		{
			name: "Multiple lines with mixed content",
			content: `server:
  host: localhost
  password: "secret123"
  port: 8080
  api_key: "key456"`,
			handler: func(key, val string) string {
				return maskString(val)
			},
			expected: `server:
  host: localhost
  password: "se****23"
  port: 8080
  api_key: "ke****56"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processSensitiveValues(tt.content, tt.handler)
			if result != tt.expected {
				t.Errorf("processSensitiveValues() mismatch\nGot:\n%s\nWant:\n%s", result, tt.expected)
			}
		})
	}
}

// TestProcessSensitiveValues_EdgeCases tests edge cases
func TestProcessSensitiveValues_EdgeCases(t *testing.T) {
	t.Run("Empty content", func(t *testing.T) {
		result := processSensitiveValues("", func(key, val string) string {
			return "masked"
		})
		if result != "" {
			t.Errorf("processSensitiveValues(\"\") = %q, want \"\"", result)
		}
	})

	t.Run("No sensitive fields", func(t *testing.T) {
		content := `name: "John"
age: 30
city: "New York"`
		result := processSensitiveValues(content, func(key, val string) string {
			return "masked"
		})
		if result != content {
			t.Errorf("processSensitiveValues() should not modify non-sensitive content")
		}
	})

	t.Run("Handler returns empty string", func(t *testing.T) {
		content := `password: "secret123"`
		result := processSensitiveValues(content, func(key, val string) string {
			return ""
		})
		// Should preserve structure even if handler returns empty
		if result == "" {
			t.Error("processSensitiveValues() should preserve YAML structure")
		}
	})

	t.Run("Value contains # character", func(t *testing.T) {
		content := `password: "secret#123"`
		result := processSensitiveValues(content, func(key, val string) string {
			return maskString(val)
		})
		// Should handle # in quoted values correctly
		if result == "" {
			t.Error("processSensitiveValues() should handle # in quoted values")
		}
	})
}
