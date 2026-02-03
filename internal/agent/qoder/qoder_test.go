package qoder

import (
	"testing"

	"github.com/verustcode/verustcode/internal/agent/base"
)

func TestNewAgent(t *testing.T) {
	agent, err := NewAgent()
	if err != nil {
		t.Fatalf("Failed to create Qoder agent: %v", err)
	}

	if agent.Name() != AgentName {
		t.Errorf("Expected agent name %s, got %s", AgentName, agent.Name())
	}

	if agent.Version() != Version {
		t.Errorf("Expected version %s, got %s", Version, agent.Version())
	}

	t.Logf("✓ Qoder agent created successfully")
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
		t.Fatalf("Qoder agent is not registered")
	}

	agent, err := base.Create(AgentName)
	if err != nil {
		t.Fatalf("Failed to create Qoder agent via factory: %v", err)
	}

	if agent.Name() != AgentName {
		t.Errorf("Expected agent name %s, got %s", AgentName, agent.Name())
	}

	t.Logf("✓ Qoder agent registered and created successfully")
}

func TestAvailable(t *testing.T) {
	agent, err := NewAgent()
	if err != nil {
		t.Fatalf("Failed to create Qoder agent: %v", err)
	}

	// Note: This will return false if qodercli CLI is not installed
	available := agent.Available()
	t.Logf("Qoder CLI available: %v", available)
}

func TestSetStore(t *testing.T) {
	agent, err := NewAgent()
	if err != nil {
		t.Fatalf("Failed to create Qoder agent: %v", err)
	}

	// SetStore with nil should not panic
	agent.SetStore(nil)
	t.Logf("✓ Qoder agent SetStore(nil) succeeded")
}
