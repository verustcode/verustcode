package cursor

import (
	"testing"

	"github.com/verustcode/verustcode/internal/agent/base"
)

func TestNewAgent(t *testing.T) {
	agent, err := NewAgent()
	if err != nil {
		t.Fatalf("Failed to create Cursor agent: %v", err)
	}

	if agent.Name() != AgentName {
		t.Errorf("Expected agent name %s, got %s", AgentName, agent.Name())
	}

	if agent.Version() != Version {
		t.Errorf("Expected version %s, got %s", Version, agent.Version())
	}

	t.Logf("✓ Cursor agent created successfully")
	t.Logf("  - Name: %s", agent.Name())
	t.Logf("  - Version: %s", agent.Version())
	t.Logf("  - Available: %v", agent.Available())
}

func TestAgentRegistration(t *testing.T) {
	agents := base.List()
	found := false
	for _, name := range agents {
		if name == AgentName {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("Cursor agent is not registered")
	}

	agent, err := base.Create(AgentName)
	if err != nil {
		t.Fatalf("Failed to create Cursor agent via factory: %v", err)
	}

	if agent.Name() != AgentName {
		t.Errorf("Expected agent name %s, got %s", AgentName, agent.Name())
	}

	t.Logf("✓ Cursor agent registered and created successfully")
}

func TestAvailable(t *testing.T) {
	agent, err := NewAgent()
	if err != nil {
		t.Fatalf("Failed to create Cursor agent: %v", err)
	}

	// Note: This will return false if cursor-agent CLI is not installed
	available := agent.Available()
	t.Logf("Cursor CLI available: %v", available)
}

func TestSetStore(t *testing.T) {
	agent, err := NewAgent()
	if err != nil {
		t.Fatalf("Failed to create Cursor agent: %v", err)
	}

	// SetStore with nil should not panic
	agent.SetStore(nil)
	t.Logf("✓ Cursor agent SetStore(nil) succeeded")
}
