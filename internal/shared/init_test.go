// Package shared provides common initialization utilities shared across different engines.
// This file contains unit tests for the shared package.
package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/logger"

	// Import providers to register them
	_ "github.com/verustcode/verustcode/internal/git/providers"
	// Import agents to register them
	_ "github.com/verustcode/verustcode/internal/agent/agents"
)

func init() {
	// Initialize logger for tests
	logger.Init(logger.Config{
		Level:  "error",
		Format: "text",
	})
}

// TestInitProviders_EmptyConfig tests initializing providers with empty config
func TestInitProviders_EmptyConfig(t *testing.T) {
	cfg := &config.Config{
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{},
		},
	}

	providers, providerConfigs := InitProviders(cfg)

	assert.Empty(t, providers)
	assert.Empty(t, providerConfigs)
}

// TestInitProviders_SingleProvider tests initializing a single provider
func TestInitProviders_SingleProvider(t *testing.T) {
	cfg := &config.Config{
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{
				{
					Type:               "github",
					URL:                "https://github.com",
					Token:              "test-token",
					WebhookSecret:      "test-secret",
					InsecureSkipVerify: false,
				},
			},
		},
	}

	providers, providerConfigs := InitProviders(cfg)

	require.Len(t, providers, 1)
	require.Len(t, providerConfigs, 1)

	prov, exists := providers["github"]
	assert.True(t, exists)
	assert.NotNil(t, prov)
	assert.Equal(t, "github", prov.Name())

	cfgVal, exists := providerConfigs["github"]
	assert.True(t, exists)
	assert.NotNil(t, cfgVal)
	assert.Equal(t, "github", cfgVal.Type)
	assert.Equal(t, "https://github.com", cfgVal.URL)
	assert.Equal(t, "test-token", cfgVal.Token)
	assert.Equal(t, "test-secret", cfgVal.WebhookSecret)
	assert.False(t, cfgVal.InsecureSkipVerify)
}

// TestInitProviders_MultipleProviders tests initializing multiple providers
func TestInitProviders_MultipleProviders(t *testing.T) {
	cfg := &config.Config{
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{
				{
					Type:  "github",
					URL:   "https://github.com",
					Token: "github-token",
				},
				{
					Type:  "gitlab",
					URL:   "https://gitlab.com",
					Token: "gitlab-token",
				},
			},
		},
	}

	providers, providerConfigs := InitProviders(cfg)

	require.Len(t, providers, 2)
	require.Len(t, providerConfigs, 2)

	githubProv, exists := providers["github"]
	assert.True(t, exists)
	assert.NotNil(t, githubProv)
	assert.Equal(t, "github", githubProv.Name())

	gitlabProv, exists := providers["gitlab"]
	assert.True(t, exists)
	assert.NotNil(t, gitlabProv)
	assert.Equal(t, "gitlab", gitlabProv.Name())

	githubCfg, exists := providerConfigs["github"]
	assert.True(t, exists)
	assert.Equal(t, "github", githubCfg.Type)

	gitlabCfg, exists := providerConfigs["gitlab"]
	assert.True(t, exists)
	assert.Equal(t, "gitlab", gitlabCfg.Type)
}

// TestInitProviders_InvalidProviderType tests handling invalid provider type
func TestInitProviders_InvalidProviderType(t *testing.T) {
	cfg := &config.Config{
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{
				{
					Type:  "invalid-provider",
					URL:   "https://invalid.com",
					Token: "token",
				},
			},
		},
	}

	providers, providerConfigs := InitProviders(cfg)

	// Invalid provider should be skipped
	assert.Empty(t, providers)
	assert.Empty(t, providerConfigs)
}

// TestInitProviders_SelfHosted tests initializing self-hosted provider
func TestInitProviders_SelfHosted(t *testing.T) {
	cfg := &config.Config{
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{
				{
					Type:               "github",
					URL:                "https://github.example.com",
					Token:              "enterprise-token",
					InsecureSkipVerify: true,
				},
			},
		},
	}

	providers, providerConfigs := InitProviders(cfg)

	require.Len(t, providers, 1)
	require.Len(t, providerConfigs, 1)

	cfgVal := providerConfigs["github"]
	assert.Equal(t, "https://github.example.com", cfgVal.URL)
	assert.True(t, cfgVal.InsecureSkipVerify)
}

// TestInitAgents_EmptyConfig tests initializing agents with empty config
func TestInitAgents_EmptyConfig(t *testing.T) {
	cfg := &config.Config{
		Agents: make(map[string]config.AgentDetail),
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	agents := InitAgents(cfg, testStore)

	assert.Empty(t, agents)
}

// TestInitAgents_SingleAgent tests initializing a single agent
func TestInitAgents_SingleAgent(t *testing.T) {
	cfg := &config.Config{
		Agents: map[string]config.AgentDetail{
			"mock": {
				CLIPath:      "/usr/local/bin/mock",
				APIKey:       "test-key",
				Timeout:      300,
				DefaultModel: "mock-model",
			},
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	agents := InitAgents(cfg, testStore)

	require.Len(t, agents, 1)

	agent, exists := agents["mock"]
	assert.True(t, exists)
	assert.NotNil(t, agent)
	assert.Equal(t, "mock", agent.Name())
	assert.True(t, agent.Available())
}

// TestInitAgents_MultipleAgents tests initializing multiple agents
func TestInitAgents_MultipleAgents(t *testing.T) {
	cfg := &config.Config{
		Agents: map[string]config.AgentDetail{
			"mock": {
				CLIPath: "/usr/local/bin/mock",
			},
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	agents := InitAgents(cfg, testStore)

	require.Len(t, agents, 1)

	mockAgent, exists := agents["mock"]
	assert.True(t, exists)
	assert.NotNil(t, mockAgent)
	assert.Equal(t, "mock", mockAgent.Name())
}

// TestInitAgents_InvalidAgentName tests handling invalid agent name
func TestInitAgents_InvalidAgentName(t *testing.T) {
	cfg := &config.Config{
		Agents: map[string]config.AgentDetail{
			"invalid-agent": {
				CLIPath: "/usr/local/bin/invalid",
			},
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	agents := InitAgents(cfg, testStore)

	// Invalid agent should be skipped
	assert.Empty(t, agents)
}

// TestInitAgents_StoreInjection tests that store is injected into agents
func TestInitAgents_StoreInjection(t *testing.T) {
	cfg := &config.Config{
		Agents: map[string]config.AgentDetail{
			"mock": {
				CLIPath: "/usr/local/bin/mock",
			},
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	agents := InitAgents(cfg, testStore)

	require.Len(t, agents, 1)

	agent := agents["mock"]
	// SetStore should not panic
	agent.SetStore(testStore)
	assert.NotNil(t, agent)
}

// TestInitAgents_AgentAvailability tests agent availability check
func TestInitAgents_AgentAvailability(t *testing.T) {
	cfg := &config.Config{
		Agents: map[string]config.AgentDetail{
			"mock": {
				CLIPath: "/usr/local/bin/mock",
			},
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	agents := InitAgents(cfg, testStore)

	require.Len(t, agents, 1)

	agent := agents["mock"]
	assert.True(t, agent.Available())
}

// TestInitProviders_ProviderConfigCopy tests that provider configs are copied correctly
func TestInitProviders_ProviderConfigCopy(t *testing.T) {
	cfg := &config.Config{
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{
				{
					Type:  "github",
					URL:   "https://github.com",
					Token: "original-token",
				},
			},
		},
	}

	providers, providerConfigs := InitProviders(cfg)

	require.Len(t, providerConfigs, 1)

	// Modify original config
	cfg.Git.Providers[0].Token = "modified-token"

	// Provider config copy should not be affected
	cfgVal := providerConfigs["github"]
	assert.Equal(t, "original-token", cfgVal.Token)
	assert.NotNil(t, providers["github"])
}

// TestInitProviders_MixedValidInvalid tests mixed valid and invalid providers
func TestInitProviders_MixedValidInvalid(t *testing.T) {
	cfg := &config.Config{
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{
				{
					Type:  "github",
					URL:   "https://github.com",
					Token: "token",
				},
				{
					Type:  "invalid-provider",
					URL:   "https://invalid.com",
					Token: "token",
				},
				{
					Type:  "gitlab",
					URL:   "https://gitlab.com",
					Token: "token",
				},
			},
		},
	}

	providers, providerConfigs := InitProviders(cfg)

	// Should only have valid providers
	require.Len(t, providers, 2)
	require.Len(t, providerConfigs, 2)

	_, githubExists := providers["github"]
	assert.True(t, githubExists)

	_, gitlabExists := providers["gitlab"]
	assert.True(t, gitlabExists)

	_, invalidExists := providers["invalid-provider"]
	assert.False(t, invalidExists)
}

// TestInitAgents_MixedValidInvalid tests mixed valid and invalid agents
func TestInitAgents_MixedValidInvalid(t *testing.T) {
	cfg := &config.Config{
		Agents: map[string]config.AgentDetail{
			"mock": {
				CLIPath: "/usr/local/bin/mock",
			},
			"invalid-agent": {
				CLIPath: "/usr/local/bin/invalid",
			},
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	agents := InitAgents(cfg, testStore)

	// Should only have valid agents
	require.Len(t, agents, 1)

	_, mockExists := agents["mock"]
	assert.True(t, mockExists)

	_, invalidExists := agents["invalid-agent"]
	assert.False(t, invalidExists)
}
