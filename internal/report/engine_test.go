// Package report provides report generation functionality.
// This file contains unit tests for the report engine.
package report

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verustcode/verustcode/internal/agent/base"
	"github.com/verustcode/verustcode/internal/config"
	_ "github.com/verustcode/verustcode/internal/git/github" // Register GitHub provider
	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/logger"
)

func init() {
	// Initialize logger for tests
	logger.Init(logger.Config{
		Level:  "error",
		Format: "text",
	})
}

// TestEngine_NewEngine tests creating a new report engine
func TestEngine_NewEngine(t *testing.T) {
	cfg := &config.Config{
		Report: config.ReportConfig{
			MaxConcurrent: 2,
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	// Create empty agents map - engine will use nil agent (will fail at runtime but not at construction)
	agents := make(map[string]base.Agent)
	providers := make(map[string]provider.Provider)

	engine := NewEngine(cfg, providers, agents, testStore)
	require.NotNil(t, engine)
	assert.NotNil(t, engine.store)
	assert.NotNil(t, engine.taskQueue)
	assert.NotNil(t, engine.ctx)
	assert.NotNil(t, engine.cancel)
}

// TestEngine_StartStop tests starting and stopping the engine
func TestEngine_StartStop(t *testing.T) {
	cfg := &config.Config{
		Report: config.ReportConfig{
			MaxConcurrent: 1,
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	engine := NewEngine(cfg, nil, nil, testStore)

	// Start engine
	engine.Start()

	// Give workers a moment to start
	time.Sleep(100 * time.Millisecond)

	// Stop engine
	engine.Stop()

	// Engine should be stopped
	assert.NotNil(t, engine)
}

// TestEngine_Submit tests submitting a report
func TestEngine_Submit(t *testing.T) {
	cfg := &config.Config{
		Report: config.ReportConfig{
			MaxConcurrent: 1,
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	engine := NewEngine(cfg, nil, nil, testStore)
	engine.Start()
	defer engine.Stop()

	report := &model.Report{
		ID:         "test-report-001",
		RepoURL:    "https://github.com/test/repo",
		Ref:        "main",
		ReportType: "wiki",
		Status:     model.ReportStatusPending,
	}

	callback := func(*model.Report, error) {
		// Callback for testing
	}

	err := engine.Submit(report, callback)
	require.NoError(t, err)

	// Give task a moment to be queued
	time.Sleep(100 * time.Millisecond)

	// Callback may or may not be called depending on processing
	// Just verify submission succeeded
	assert.NoError(t, err)
}

// TestEngine_Submit_QueueFull tests submitting when queue is full
func TestEngine_Submit_QueueFull(t *testing.T) {
	cfg := &config.Config{
		Report: config.ReportConfig{
			MaxConcurrent: 1,
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	engine := NewEngine(cfg, nil, nil, testStore)
	// Don't start workers, so queue will fill up

	// Fill the queue (capacity is 100)
	for i := 0; i < 101; i++ {
		report := &model.Report{
			ID:         "test-report-" + string(rune(i)),
			RepoURL:    "https://github.com/test/repo",
			Ref:        "main",
			ReportType: "wiki",
			Status:     model.ReportStatusPending,
		}
		err := engine.Submit(report, nil)
		if i < 100 {
			assert.NoError(t, err)
		} else {
			// 101st submission should fail
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "queue is full")
		}
	}

	engine.Stop()
}

// TestEngine_GetProgress tests getting report progress
func TestEngine_GetProgress(t *testing.T) {
	cfg := &config.Config{
		Report: config.ReportConfig{
			MaxConcurrent: 1,
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	engine := NewEngine(cfg, nil, nil, testStore)

	// Test with non-existent report
	progress, err := engine.GetProgress("non-existent")
	assert.Error(t, err)
	assert.Nil(t, progress)
}

// TestEngine_Resume tests resuming a report
func TestEngine_Resume(t *testing.T) {
	cfg := &config.Config{
		Report: config.ReportConfig{
			MaxConcurrent: 1,
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	engine := NewEngine(cfg, nil, nil, testStore)

	ctx := context.Background()

	// Test with non-existent report
	err := engine.Resume(ctx, "non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestEngine_Resume_Completed tests resuming a completed report
func TestEngine_Resume_Completed(t *testing.T) {
	cfg := &config.Config{
		Report: config.ReportConfig{
			MaxConcurrent: 1,
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	engine := NewEngine(cfg, nil, nil, testStore)

	// Create completed report
	report := &model.Report{
		ID:         "test-report-001",
		RepoURL:    "https://github.com/test/repo",
		Ref:        "main",
		ReportType: "wiki",
		Status:     model.ReportStatusCompleted,
	}
	err := testStore.Report().Create(report)
	require.NoError(t, err)

	ctx := context.Background()

	// Try to resume completed report
	err = engine.Resume(ctx, report.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already completed")
}

// TestGetReportWorkers tests getting report workers count
func TestGetReportWorkers(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	configProvider := config.NewDBConfigProvider(testStore)

	// Test with no config (should return default)
	// GetReportConfig should return a config with defaults, not nil
	reportCfg, err := configProvider.GetReportConfig()
	require.NoError(t, err)
	require.NotNil(t, reportCfg) // Should always return a config with defaults

	workers := getReportWorkers(configProvider)
	assert.Equal(t, 2, workers) // Default value from GetReportConfig (not 3 from getReportWorkers fallback)
}

// TestEngine_Stop_WithoutStart tests stopping engine without starting
func TestEngine_Stop_WithoutStart(t *testing.T) {
	cfg := &config.Config{
		Report: config.ReportConfig{
			MaxConcurrent: 1,
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	engine := NewEngine(cfg, nil, nil, testStore)

	// Stop without starting should not panic
	engine.Stop()
}

// TestEngine_Submit_NilReport tests submitting nil report
func TestEngine_Submit_NilReport(t *testing.T) {
	cfg := &config.Config{
		Report: config.ReportConfig{
			MaxConcurrent: 1,
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	engine := NewEngine(cfg, nil, nil, testStore)
	engine.Start()
	defer engine.Stop()

	// Submit nil report - should return error immediately
	err := engine.Submit(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "report cannot be nil")
}

// TestEngine_Worker_ContextCancellation tests worker cancellation
func TestEngine_Worker_ContextCancellation(t *testing.T) {
	cfg := &config.Config{
		Report: config.ReportConfig{
			MaxConcurrent: 1,
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	engine := NewEngine(cfg, nil, nil, testStore)
	engine.Start()

	// Cancel context
	engine.cancel()

	// Stop should complete
	engine.Stop()
}

// TestEngine_ParseRepoURL_AnonymousGitHubProvider tests anonymous GitHub provider fallback
// When no GitHub provider is configured, parseRepoURL should fall back to anonymous provider
func TestEngine_ParseRepoURL_AnonymousGitHubProvider(t *testing.T) {
	cfg := &config.Config{
		Report: config.ReportConfig{
			MaxConcurrent: 1,
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	// Create engine with empty providers map (no configured providers)
	engine := NewEngine(cfg, make(map[string]provider.Provider), nil, testStore)

	// Test parsing a GitHub URL without any configured provider
	// Should fall back to anonymous GitHub provider
	prov, owner, repo, err := engine.parseRepoURL("https://github.com/frankbria/ralph-claude-code")
	require.NoError(t, err, "parseRepoURL should succeed with anonymous GitHub provider")
	assert.NotNil(t, prov, "provider should not be nil")
	assert.Equal(t, "frankbria", owner)
	assert.Equal(t, "ralph-claude-code", repo)
	assert.Equal(t, "github", prov.Name())
}

// TestEngine_ParseRepoURL_ConfiguredProviderPriority tests configured provider has priority
// When GitHub provider is configured, it should be used instead of anonymous provider
func TestEngine_ParseRepoURL_ConfiguredProviderPriority(t *testing.T) {
	cfg := &config.Config{
		Report: config.ReportConfig{
			MaxConcurrent: 1,
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	// Create a configured GitHub provider with token
	configuredProvider, err := provider.Create("github", &provider.ProviderOptions{
		Token:   "test-token",
		BaseURL: "",
	})
	require.NoError(t, err)

	// Create engine with configured GitHub provider
	providers := map[string]provider.Provider{
		"github": configuredProvider,
	}
	engine := NewEngine(cfg, providers, nil, testStore)

	// Test parsing a GitHub URL - should use configured provider
	prov, owner, repo, err := engine.parseRepoURL("https://github.com/test/repo")
	require.NoError(t, err)
	assert.NotNil(t, prov)
	assert.Equal(t, "test", owner)
	assert.Equal(t, "repo", repo)
	// Should be the same configured provider (with token)
	assert.Equal(t, configuredProvider, prov)
}

// TestEngine_ParseRepoURL_UnsupportedURL tests unsupported URL handling
// When URL doesn't match any known provider, should return error
func TestEngine_ParseRepoURL_UnsupportedURL(t *testing.T) {
	cfg := &config.Config{
		Report: config.ReportConfig{
			MaxConcurrent: 1,
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	// Create engine with empty providers map
	engine := NewEngine(cfg, make(map[string]provider.Provider), nil, testStore)

	// Test parsing a non-GitHub URL (e.g., GitLab) without configured provider
	// Should return error since we only support anonymous GitHub fallback currently
	_, _, _, err := engine.parseRepoURL("https://gitlab.com/test/repo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported repository URL")
}
