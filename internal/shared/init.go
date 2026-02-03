// Package shared provides common initialization utilities shared across different engines.
// This package enables code reuse between Review Engine and Report Engine without coupling them.
package shared

import (
	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/agent/base"
	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/logger"
)

// InitProviders initializes Git providers from configuration.
// Returns a map of provider type -> provider instance and provider configs.
// Each engine should call this to get its own set of providers.
func InitProviders(cfg *config.Config) (map[string]provider.Provider, map[string]*config.ProviderConfig) {
	providers := make(map[string]provider.Provider)
	providerConfigs := make(map[string]*config.ProviderConfig)

	for _, pc := range cfg.Git.Providers {
		// Create a copy of the config to store
		pcCopy := pc

		// Build provider options from config
		opts := &provider.ProviderOptions{
			Token:              pc.Token,
			BaseURL:            pc.URL,
			InsecureSkipVerify: pc.InsecureSkipVerify,
		}

		p, err := provider.Create(pc.Type, opts)
		if err != nil {
			logger.Warn("Failed to create provider",
				zap.String("type", pc.Type),
				zap.Error(err),
			)
			continue
		}
		providers[pc.Type] = p
		providerConfigs[pc.Type] = &pcCopy
		logger.Info("Initialized Git provider",
			zap.String("type", pc.Type),
			zap.String("url", pc.URL),
			zap.Bool("insecure_skip_verify", pc.InsecureSkipVerify),
		)
	}

	if len(providers) == 0 {
		logger.Warn("No Git providers configured")
	}

	return providers, providerConfigs
}

// InitAgents initializes AI agents with database store for runtime configuration.
// Agents read configuration from database on each execution.
// Returns a map of agent name -> agent instance.
func InitAgents(cfg *config.Config, s store.Store) map[string]base.Agent {
	agents := make(map[string]base.Agent)

	for name := range cfg.Agents {
		agent, err := base.Create(name)
		if err != nil {
			logger.Warn("Failed to create agent",
				zap.String("name", name),
				zap.Error(err),
			)
			continue
		}

		// Inject store for runtime configuration reading
		agent.SetStore(s)

		agents[name] = agent
		logger.Info("Initialized AI agent",
			zap.String("name", name),
			zap.Bool("available", agent.Available()),
		)
	}

	return agents
}
