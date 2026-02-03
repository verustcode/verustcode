package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/store"
)

// TestLoadRuntimeConfig_DefaultAgent tests that default cursor agent is provided
func TestLoadRuntimeConfig_DefaultAgent(t *testing.T) {
	// Create temporary bootstrap config
	tmpDir := t.TempDir()
	bootstrapPath := tmpDir + "/bootstrap.yaml"
	
	err := CreateDefaultBootstrap(bootstrapPath)
	require.NoError(t, err)

	// Load runtime config without database (no agent settings)
	cfg, err := LoadRuntimeConfig(bootstrapPath, nil)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify default cursor agent exists
	assert.Contains(t, cfg.Agents, "cursor", "Default cursor agent should exist")
	
	cursorAgent := cfg.Agents["cursor"]
	assert.Equal(t, defaultCLIPath, cursorAgent.CLIPath, "Should have default CLI path")
	assert.Equal(t, defaultAgentTimeout, cursorAgent.Timeout, "Should have default timeout")
}

// TestLoadRuntimeConfig_AgentMerge tests that database settings merge with defaults
func TestLoadRuntimeConfig_AgentMerge(t *testing.T) {
	// Setup test database
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	// Create temporary bootstrap config
	tmpDir := t.TempDir()
	bootstrapPath := tmpDir + "/bootstrap.yaml"
	
	err := CreateDefaultBootstrap(bootstrapPath)
	require.NoError(t, err)

	// Save partial agent config to database (only api_key)
	svc := NewSettingsService(testStore)
	agentSettings := map[string]interface{}{
		"cursor": AgentDetail{
			APIKey: "test-api-key-123",
			// CLIPath and Timeout intentionally omitted
		},
	}
	err = svc.SetCategory(string(model.SettingCategoryAgents), agentSettings, "test-user")
	require.NoError(t, err)

	// Load runtime config with database
	cfg, err := LoadRuntimeConfig(bootstrapPath, testStore)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify cursor agent exists and has merged config
	assert.Contains(t, cfg.Agents, "cursor", "Cursor agent should exist")
	
	cursorAgent := cfg.Agents["cursor"]
	// Database value should be used
	assert.Equal(t, "test-api-key-123", cursorAgent.APIKey, "Should use API key from database")
	// Default values should be preserved
	assert.Equal(t, defaultCLIPath, cursorAgent.CLIPath, "Should preserve default CLI path")
	assert.Equal(t, defaultAgentTimeout, cursorAgent.Timeout, "Should preserve default timeout")
}

// TestLoadRuntimeConfig_AgentOverride tests that database settings fully override when all fields present
func TestLoadRuntimeConfig_AgentOverride(t *testing.T) {
	// Setup test database
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	// Create temporary bootstrap config
	tmpDir := t.TempDir()
	bootstrapPath := tmpDir + "/bootstrap.yaml"
	
	err := CreateDefaultBootstrap(bootstrapPath)
	require.NoError(t, err)

	// Save complete agent config to database
	svc := NewSettingsService(testStore)
	agentSettings := map[string]interface{}{
		"cursor": AgentDetail{
			CLIPath:      "/custom/path/cursor",
			APIKey:       "custom-api-key",
			Timeout:      1200,
			DefaultModel: "custom-model",
		},
	}
	err = svc.SetCategory(string(model.SettingCategoryAgents), agentSettings, "test-user")
	require.NoError(t, err)

	// Load runtime config with database
	cfg, err := LoadRuntimeConfig(bootstrapPath, testStore)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify cursor agent has custom config
	assert.Contains(t, cfg.Agents, "cursor", "Cursor agent should exist")
	
	cursorAgent := cfg.Agents["cursor"]
	assert.Equal(t, "/custom/path/cursor", cursorAgent.CLIPath, "Should use custom CLI path")
	assert.Equal(t, "custom-api-key", cursorAgent.APIKey, "Should use custom API key")
	assert.Equal(t, 1200, cursorAgent.Timeout, "Should use custom timeout")
	assert.Equal(t, "custom-model", cursorAgent.DefaultModel, "Should use custom model")
}
