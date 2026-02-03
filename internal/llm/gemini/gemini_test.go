package gemini

import (
	"testing"

	"github.com/verustcode/verustcode/internal/llm"
)

func TestNewClient(t *testing.T) {
	config := llm.NewClientConfig(ClientName)
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create Gemini client: %v", err)
	}

	if client.Name() != ClientName {
		t.Errorf("Expected client name %s, got %s", ClientName, client.Name())
	}

	t.Logf("✓ Gemini client created successfully")
	t.Logf("  - Name: %s", client.Name())
	t.Logf("  - Available: %v", client.Available())
}

func TestClientRegistration(t *testing.T) {
	if !llm.IsRegistered(ClientName) {
		t.Fatalf("Gemini client is not registered")
	}

	config := llm.NewClientConfig(ClientName)
	client, err := llm.Create(ClientName, config)
	if err != nil {
		t.Fatalf("Failed to create Gemini client via factory: %v", err)
	}

	if client.Name() != ClientName {
		t.Errorf("Expected client name %s, got %s", ClientName, client.Name())
	}

	t.Logf("✓ Gemini client registered and created successfully")
}

func TestAvailable(t *testing.T) {
	config := llm.NewClientConfig(ClientName)
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create Gemini client: %v", err)
	}

	// Note: This will return false if gemini CLI is not installed
	available := client.Available()
	t.Logf("Gemini CLI available: %v", available)
}
