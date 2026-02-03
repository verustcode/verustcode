package engine

import (
	"os"
	"testing"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/prompt"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/logger"
)

// TestMain initializes logger for all tests
func TestMain(m *testing.M) {
	// Initialize logger with minimal config for testing
	logger.Init(logger.Config{
		Level:  "error", // Only log errors to reduce noise
		Format: "text",
		File:   "", // No file output for tests
	})

	// Run tests
	code := m.Run()

	// Sync logger before exit
	logger.Sync()

	os.Exit(code)
}

// TestEngine_NewEngine tests creating a new engine
func TestEngine_NewEngine(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Review: config.ReviewConfig{
			MaxConcurrent: 1,
		},
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{},
		},
		Agents: make(map[string]config.AgentDetail),
	}

	engine, err := NewEngine(cfg, testStore)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	if engine == nil {
		t.Fatal("NewEngine() returned nil")
	}

	if engine.store != testStore {
		t.Error("Engine store not set correctly")
	}

	if engine.cfg != cfg {
		t.Error("Engine config not set correctly")
	}
}

// TestEngine_GetProvider tests getting a provider
func TestEngine_GetProvider(t *testing.T) {

	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Review: config.ReviewConfig{
			MaxConcurrent: 1,
		},
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{},
		},
		Agents: make(map[string]config.AgentDetail),
	}

	engine, err := NewEngine(cfg, testStore)

	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	defer func() {
		engine.Stop()
	}()

	// Test getting non-existent provider

	_, exists := engine.GetProvider("non-existent")

	if exists {
		t.Error("GetProvider() should return false for non-existent provider")
	}

}

// TestEngine_GetProviderConfig tests getting provider config
func TestEngine_GetProviderConfig(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Review: config.ReviewConfig{
			MaxConcurrent: 1,
		},
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{},
		},
		Agents: make(map[string]config.AgentDetail),
	}

	engine, err := NewEngine(cfg, testStore)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}
	defer engine.Stop()

	// Test getting non-existent provider config
	_, exists := engine.GetProviderConfig("non-existent")
	if exists {
		t.Error("GetProviderConfig() should return false for non-existent provider")
	}
}

// TestEngine_GetAgent tests getting an agent
func TestEngine_GetAgent(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Review: config.ReviewConfig{
			MaxConcurrent: 1,
		},
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{},
		},
		Agents: make(map[string]config.AgentDetail),
	}

	engine, err := NewEngine(cfg, testStore)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}
	defer engine.Stop()

	// Test getting non-existent agent
	_, exists := engine.GetAgent("non-existent")
	if exists {
		t.Error("GetAgent() should return false for non-existent agent")
	}
}

// TestEngine_ListAgents tests listing agents
func TestEngine_ListAgents(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Review: config.ReviewConfig{
			MaxConcurrent: 1,
		},
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{},
		},
		Agents: make(map[string]config.AgentDetail),
	}

	engine, err := NewEngine(cfg, testStore)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}
	defer engine.Stop()

	agents := engine.ListAgents()
	if agents == nil {
		t.Error("ListAgents() returned nil")
	}
	// Should return empty list if no agents configured
}

// TestEngine_ListProviders tests listing providers
func TestEngine_ListProviders(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Review: config.ReviewConfig{
			MaxConcurrent: 1,
		},
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{},
		},
		Agents: make(map[string]config.AgentDetail),
	}

	engine, err := NewEngine(cfg, testStore)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}
	defer engine.Stop()

	providers := engine.ListProviders()
	if providers == nil {
		t.Error("ListProviders() returned nil")
	}
	// Should return empty list if no providers configured
}

// TestEngine_Config tests getting config
func TestEngine_Config(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Review: config.ReviewConfig{
			MaxConcurrent: 1,
		},
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{},
		},
		Agents: make(map[string]config.AgentDetail),
	}

	engine, err := NewEngine(cfg, testStore)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}
	defer engine.Stop()

	retrievedCfg := engine.Config()
	if retrievedCfg != cfg {
		t.Error("Config() returned incorrect config")
	}
}

// TestEngine_SetCallbacks tests setting callbacks
func TestEngine_SetCallbacks(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Review: config.ReviewConfig{
			MaxConcurrent: 1,
		},
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{},
		},
		Agents: make(map[string]config.AgentDetail),
	}

	engine, err := NewEngine(cfg, testStore)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}
	defer engine.Stop()

	engine.SetCallbacks(
		func(task *Task, result *prompt.ReviewResult) {
			// Callback set successfully
		},
		func(task *Task, err error) {
			// Callback set successfully
		},
	)

	if engine.onComplete == nil {
		t.Error("onComplete callback not set")
	}
	if engine.onError == nil {
		t.Error("onError callback not set")
	}
}

// TestEngine_StartStop tests starting and stopping the engine
func TestEngine_StartStop(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Review: config.ReviewConfig{
			MaxConcurrent: 1,
		},
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{},
		},
		Agents: make(map[string]config.AgentDetail),
	}

	engine, err := NewEngine(cfg, testStore)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}

	// Start the engine
	engine.Start()

	// Stop the engine
	engine.Stop()

	// Verify context is cancelled
	select {
	case <-engine.ctx.Done():
		// Context cancelled, good
	default:
		t.Error("Engine context not cancelled after Stop()")
	}
}

// TestEngine_detectProviderFromURL tests detecting provider from URL
func TestEngine_detectProviderFromURL(t *testing.T) {
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		Review: config.ReviewConfig{
			MaxConcurrent: 1,
		},
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{},
		},
		Agents: make(map[string]config.AgentDetail),
	}

	engine, err := NewEngine(cfg, testStore)
	if err != nil {
		t.Fatalf("NewEngine() failed: %v", err)
	}
	defer engine.Stop()

	// Test GitHub URL
	provider := engine.detectProviderFromURL("https://github.com/test/repo")
	if provider != "github" {
		t.Errorf("Expected 'github', got '%s'", provider)
	}

	// Test GitLab URL
	provider = engine.detectProviderFromURL("https://gitlab.com/test/repo")
	if provider != "gitlab" {
		t.Errorf("Expected 'gitlab', got '%s'", provider)
	}

	// Test unknown URL
	provider = engine.detectProviderFromURL("https://unknown.com/test/repo")
	if provider != "" {
		t.Errorf("Expected empty string, got '%s'", provider)
	}
}
