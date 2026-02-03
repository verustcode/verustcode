package qoder

import (
	"testing"

	"github.com/verustcode/verustcode/internal/llm"
)

func TestNewClient(t *testing.T) {
	config := llm.NewClientConfig(ClientName)
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create Qoder client: %v", err)
	}

	if client.Name() != ClientName {
		t.Errorf("Expected client name %s, got %s", ClientName, client.Name())
	}

	t.Logf("✓ Qoder client created successfully")
	t.Logf("  - Name: %s", client.Name())
	t.Logf("  - Available: %v", client.Available())
}

func TestNewClientWithNilConfig(t *testing.T) {
	client, err := NewClient(nil)
	if err != nil {
		t.Fatalf("Failed to create Qoder client with nil config: %v", err)
	}

	if client.Name() != ClientName {
		t.Errorf("Expected client name %s, got %s", ClientName, client.Name())
	}
}

func TestNewClientWithCustomCLIPath(t *testing.T) {
	config := llm.NewClientConfig(ClientName)
	config.CLIPath = "/custom/path/to/qodercli"
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create Qoder client: %v", err)
	}

	if client.Name() != ClientName {
		t.Errorf("Expected client name %s, got %s", ClientName, client.Name())
	}

	// Verify CLI path is set (we can't easily test the actual path without mocking)
	_ = client
}

func TestClientRegistration(t *testing.T) {
	if !llm.IsRegistered(ClientName) {
		t.Fatalf("Qoder client is not registered")
	}

	config := llm.NewClientConfig(ClientName)
	client, err := llm.Create(ClientName, config)
	if err != nil {
		t.Fatalf("Failed to create Qoder client via factory: %v", err)
	}

	if client.Name() != ClientName {
		t.Errorf("Expected client name %s, got %s", ClientName, client.Name())
	}

	t.Logf("✓ Qoder client registered and created successfully")
}

func TestAvailable(t *testing.T) {
	config := llm.NewClientConfig(ClientName)
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create Qoder client: %v", err)
	}

	// Note: This will return false if qodercli CLI is not installed
	available := client.Available()
	t.Logf("Qoder CLI available: %v", available)
}

func TestClose(t *testing.T) {
	config := llm.NewClientConfig(ClientName)
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create Qoder client: %v", err)
	}

	// Close should not error
	err = client.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got: %v", err)
	}
}

func TestCreateSession(t *testing.T) {
	config := llm.NewClientConfig(ClientName)
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create Qoder client: %v", err)
	}

	// Qoder auto-creates sessions, so CreateSession should return empty string
	sessionID, err := client.CreateSession(nil)
	if err != nil {
		t.Errorf("CreateSession() should not return error, got: %v", err)
	}
	if sessionID != "" {
		t.Errorf("CreateSession() should return empty string for Qoder, got: %q", sessionID)
	}
}

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty key",
			input:    "",
			expected: "",
		},
		{
			name:     "short key",
			input:    "short",
			expected: "***",
		},
		{
			name:     "long key",
			input:    "abcdefghijklmnopqrstuvwxyz1234567890",
			expected: "abcd...7890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskAPIKey(tt.input)
			if result != tt.expected {
				t.Errorf("maskAPIKey(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsSensitiveFlag(t *testing.T) {
	tests := []struct {
		name     string
		flag     string
		expected bool
	}{
		{
			name:     "api-key flag",
			flag:     "--api-key",
			expected: true,
		},
		{
			name:     "token flag",
			flag:     "--token",
			expected: true,
		},
		{
			name:     "secret flag",
			flag:     "--secret",
			expected: true,
		},
		{
			name:     "password flag",
			flag:     "--password",
			expected: true,
		},
		{
			name:     "non-sensitive flag",
			flag:     "--model",
			expected: false,
		},
		{
			name:     "empty flag",
			flag:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSensitiveFlag(tt.flag)
			if result != tt.expected {
				t.Errorf("isSensitiveFlag(%q) = %v, want %v", tt.flag, result, tt.expected)
			}
		})
	}
}

func TestMaskSensitiveArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "no sensitive args",
			args:     []string{"--model", "auto", "--output-format", "text"},
			expected: []string{"--model", "auto", "--output-format", "text"},
		},
		{
			name:     "with api-key",
			args:     []string{"--api-key", "secret1234567890", "--model", "auto"},
			expected: []string{"--api-key", "secr...7890", "--model", "auto"},
		},
		{
			name:     "with token",
			args:     []string{"--token", "mytoken1234567890"},
			expected: []string{"--token", "myto...7890"},
		},
		{
			name:     "multiple sensitive args",
			args:     []string{"--api-key", "key123", "--token", "token456"},
			expected: []string{"--api-key", "***", "--token", "***"},
		},
		{
			name:     "sensitive flag at end",
			args:     []string{"--api-key"},
			expected: []string{"--api-key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskSensitiveArgs(tt.args)
			if len(result) != len(tt.expected) {
				t.Errorf("maskSensitiveArgs() length = %d, want %d", len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("maskSensitiveArgs()[%d] = %q, want %q", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestExtractTextFromMessage(t *testing.T) {
	client := &Client{}

	tests := []struct {
		name     string
		data     map[string]interface{}
		expected string
	}{
		{
			name:     "empty data",
			data:     map[string]interface{}{},
			expected: "",
		},
		{
			name: "valid message with text content",
			data: map[string]interface{}{
				"message": map[string]interface{}{
					"content": []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": "Hello, world!",
						},
					},
				},
			},
			expected: "Hello, world!",
		},
		{
			name: "multiple text parts",
			data: map[string]interface{}{
				"message": map[string]interface{}{
					"content": []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": "Part 1",
						},
						map[string]interface{}{
							"type": "text",
							"text": "Part 2",
						},
					},
				},
			},
			expected: "Part 1Part 2",
		},
		{
			name: "non-text content",
			data: map[string]interface{}{
				"message": map[string]interface{}{
					"content": []interface{}{
						map[string]interface{}{
							"type": "image",
							"url":  "https://example.com/image.png",
						},
					},
				},
			},
			expected: "",
		},
		{
			name: "no message field",
			data: map[string]interface{}{
				"type": "system",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.extractTextFromMessage(tt.data)
			if result != tt.expected {
				t.Errorf("extractTextFromMessage() = %q, want %q", result, tt.expected)
			}
		})
	}
}
