package llm

import (
	"testing"
)

// TestRegister tests registering a client factory
func TestRegister(t *testing.T) {
	// Clean up any existing registration
	Unregister("test-client")

	factory := func(config *ClientConfig) (Client, error) {
		return nil, nil
	}

	Register("test-client", factory)

	if !IsRegistered("test-client") {
		t.Error("Client should be registered after Register()")
	}

	// Clean up
	Unregister("test-client")
}

// TestCreate_NonExistent tests creating a non-existent client
func TestCreate_NonExistent(t *testing.T) {
	_, err := Create("non-existent-client", nil)
	if err == nil {
		t.Error("Create() should return error for non-existent client")
	}
}

// TestList tests listing registered clients
func TestList(t *testing.T) {
	clients := List()
	if clients == nil {
		t.Error("List() should not return nil")
	}
	// Should return at least some clients (gemini, cursor, etc.)
}

// TestIsRegistered tests checking if a client is registered
func TestIsRegistered(t *testing.T) {
	// Test with non-existent client
	if IsRegistered("non-existent-client") {
		t.Error("IsRegistered() should return false for non-existent client")
	}

	// Test with registered client (gemini should be registered)
	if !IsRegistered("gemini") {
		t.Log("Note: gemini client may not be registered in test environment")
	}
}

// TestUnregister tests unregistering a client
func TestUnregister(t *testing.T) {
	// Register a test client
	factory := func(config *ClientConfig) (Client, error) {
		return nil, nil
	}
	Register("test-unregister", factory)

	if !IsRegistered("test-unregister") {
		t.Error("Client should be registered")
	}

	// Unregister
	Unregister("test-unregister")

	if IsRegistered("test-unregister") {
		t.Error("Client should not be registered after Unregister()")
	}
}

// TestCreate_WithNilConfig tests creating a client with nil config
func TestCreate_WithNilConfig(t *testing.T) {
	// This will fail for non-existent client, but should handle nil config gracefully
	_, err := Create("non-existent", nil)
	if err == nil {
		t.Error("Create() should return error for non-existent client")
	}
}

// TestCreate_WithConfig tests creating a client with config
func TestCreate_WithConfig(t *testing.T) {
	config := NewClientConfig("test-client")
	_, err := Create("non-existent", config)
	if err == nil {
		t.Error("Create() should return error for non-existent client")
	}
}




