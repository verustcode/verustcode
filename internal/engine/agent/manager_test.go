// Package agent implements AI agent management for the review engine.
// This file contains unit tests for the agent manager.
package agent

import (
	"testing"

	"github.com/verustcode/verustcode/internal/config"
)

// TestNewManager tests the NewManager function
func TestNewManager(t *testing.T) {
	cfg := &config.Config{
		Agents: make(map[string]config.AgentDetail),
	}

	manager := NewManager(cfg, nil)
	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	if manager.agents == nil {
		t.Error("Manager.agents should not be nil")
	}

	if manager.cfg != cfg {
		t.Error("Manager.cfg not set correctly")
	}
}

// TestManagerInitialize tests the Initialize method
func TestManagerInitialize(t *testing.T) {
	t.Run("initialize with empty config", func(t *testing.T) {
		cfg := &config.Config{
			Agents: make(map[string]config.AgentDetail),
		}

		manager := NewManager(cfg, nil)
		err := manager.Initialize()

		if err != nil {
			t.Errorf("Initialize() returned error: %v", err)
		}

		if manager.Count() != 0 {
			t.Errorf("Count() = %d, want 0", manager.Count())
		}
	})

	t.Run("initialize with cursor agent", func(t *testing.T) {
		cfg := &config.Config{
			Agents: map[string]config.AgentDetail{
				"cursor": {
					CLIPath:      "/usr/local/bin/cursor",
					Timeout:      60,
					DefaultModel: "claude-3-5-sonnet",
				},
			},
		}

		manager := NewManager(cfg, nil)
		err := manager.Initialize()

		if err != nil {
			t.Errorf("Initialize() returned error: %v", err)
		}

		// Agent may or may not be available depending on CLI path
		// Just ensure no panic occurs
	})

	t.Run("initialize with gemini agent", func(t *testing.T) {
		cfg := &config.Config{
			Agents: map[string]config.AgentDetail{
				"gemini": {
					APIKey:       "test-api-key",
					Timeout:      60,
					DefaultModel: "gemini-1.5-pro",
				},
			},
		}

		manager := NewManager(cfg, nil)
		err := manager.Initialize()

		if err != nil {
			t.Errorf("Initialize() returned error: %v", err)
		}
	})

	t.Run("initialize with multiple agents", func(t *testing.T) {
		cfg := &config.Config{
			Agents: map[string]config.AgentDetail{
				"cursor": {
					CLIPath: "/usr/local/bin/cursor",
					Timeout: 60,
				},
				"gemini": {
					APIKey:  "test-api-key",
					Timeout: 60,
				},
				"qoder": {
					CLIPath: "/usr/local/bin/qoder",
					Timeout: 120,
				},
			},
		}

		manager := NewManager(cfg, nil)
		err := manager.Initialize()

		if err != nil {
			t.Errorf("Initialize() returned error: %v", err)
		}
	})
}

// TestManagerGet tests the Get method
func TestManagerGet(t *testing.T) {
	cfg := &config.Config{
		Agents: make(map[string]config.AgentDetail),
	}

	manager := NewManager(cfg, nil)
	manager.Initialize()

	t.Run("get non-existent agent", func(t *testing.T) {
		agent := manager.Get("nonexistent")
		if agent != nil {
			t.Error("Get() for non-existent agent should return nil")
		}
	})
}

// TestManagerGetWithOK tests the GetWithOK method
func TestManagerGetWithOK(t *testing.T) {
	cfg := &config.Config{
		Agents: make(map[string]config.AgentDetail),
	}

	manager := NewManager(cfg, nil)
	manager.Initialize()

	t.Run("get non-existent agent", func(t *testing.T) {
		agent, ok := manager.GetWithOK("nonexistent")
		if ok {
			t.Error("GetWithOK() for non-existent agent should return false")
		}
		if agent != nil {
			t.Error("GetWithOK() for non-existent agent should return nil agent")
		}
	})
}

// TestManagerList tests the List method
func TestManagerList(t *testing.T) {
	cfg := &config.Config{
		Agents: make(map[string]config.AgentDetail),
	}

	manager := NewManager(cfg, nil)
	manager.Initialize()

	names := manager.List()
	if names == nil {
		t.Error("List() should not return nil")
	}
	if len(names) != 0 {
		t.Errorf("List() length = %d, want 0", len(names))
	}
}

// TestManagerAll tests the All method
func TestManagerAll(t *testing.T) {
	cfg := &config.Config{
		Agents: make(map[string]config.AgentDetail),
	}

	manager := NewManager(cfg, nil)
	manager.Initialize()

	all := manager.All()
	if all == nil {
		t.Error("All() should not return nil")
	}
	if len(all) != 0 {
		t.Errorf("All() length = %d, want 0", len(all))
	}
}

// TestManagerCount tests the Count method
func TestManagerCount(t *testing.T) {
	cfg := &config.Config{
		Agents: make(map[string]config.AgentDetail),
	}

	manager := NewManager(cfg, nil)
	manager.Initialize()

	count := manager.Count()
	if count != 0 {
		t.Errorf("Count() = %d, want 0", count)
	}
}

// TestManagerWithInvalidAgentType tests initialization with invalid agent type
func TestManagerWithInvalidAgentType(t *testing.T) {
	cfg := &config.Config{
		Agents: map[string]config.AgentDetail{
			"invalid-agent-type": {
				CLIPath: "/some/path",
				Timeout: 60,
			},
		},
	}

	manager := NewManager(cfg, nil)
	err := manager.Initialize()

	// Should not return error, just log warning
	if err != nil {
		t.Errorf("Initialize() should not return error for invalid agent: %v", err)
	}

	// Invalid agent should not be added
	if manager.Get("invalid-agent-type") != nil {
		t.Error("Invalid agent should not be added to manager")
	}
}
