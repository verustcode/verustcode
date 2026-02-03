// Package agent implements AI agent management for the review engine.
// It handles initialization and access to AI agents.
package agent

import (
	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/agent/base"
	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/logger"
)

// Manager handles AI agent initialization and access.
type Manager struct {
	agents map[string]base.Agent
	cfg    *config.Config
	store  store.Store
}

// NewManager creates a new Agent Manager.
func NewManager(cfg *config.Config, s store.Store) *Manager {
	return &Manager{
		agents: make(map[string]base.Agent),
		cfg:    cfg,
		store:  s,
	}
}

// Initialize initializes all AI agents with database store for runtime configuration.
// Agents read configuration from database on each execution.
func (m *Manager) Initialize() error {
	for name := range m.cfg.Agents {
		agent, err := base.Create(name)
		if err != nil {
			logger.Warn("Failed to create agent",
				zap.String("name", name),
				zap.Error(err),
			)
			continue
		}

		// Inject store for runtime configuration reading
		agent.SetStore(m.store)

		m.agents[name] = agent
		logger.Info("Initialized AI agent",
			zap.String("name", name),
			zap.Bool("available", agent.Available()),
		)
	}

	return nil
}

// Get returns an agent by name.
// Returns nil if the agent is not found.
func (m *Manager) Get(name string) base.Agent {
	return m.agents[name]
}

// GetWithOK returns an agent by name with a boolean indicating if it exists.
func (m *Manager) GetWithOK(name string) (base.Agent, bool) {
	a, ok := m.agents[name]
	return a, ok
}

// List returns all available agent names.
func (m *Manager) List() []string {
	names := make([]string, 0, len(m.agents))
	for name := range m.agents {
		names = append(names, name)
	}
	return names
}

// All returns the map of all agents.
// This is used for components that need direct access to the agent map.
func (m *Manager) All() map[string]base.Agent {
	return m.agents
}

// Count returns the number of configured agents.
func (m *Manager) Count() int {
	return len(m.agents)
}
