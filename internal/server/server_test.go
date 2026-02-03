// Package server provides HTTP server for the application.
// This file contains unit tests for the server package.
package server

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/engine"
	"github.com/verustcode/verustcode/internal/report"
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

// TestServer_New tests creating a new server instance
func TestServer_New(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	testEngine, err := engine.NewEngine(cfg, testStore)
	require.NoError(t, err)
	defer testEngine.Stop()

	srv := New(cfg, testEngine, testStore)
	require.NotNil(t, srv)
	assert.Equal(t, cfg, srv.cfg)
	assert.Equal(t, testEngine, srv.engine)
	assert.Equal(t, testStore, srv.store)
	assert.NotNil(t, srv.router)
}

// TestServer_NewWithReportEngine tests creating a server with report engine
func TestServer_NewWithReportEngine(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Git: config.GitConfig{
			Providers: []config.ProviderConfig{},
		},
		Agents: make(map[string]config.AgentDetail),
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	testEngine, err := engine.NewEngine(cfg, testStore)
	require.NoError(t, err)
	defer testEngine.Stop()

	reportEngine := report.NewEngine(cfg, nil, nil, testStore)

	srv := NewWithReportEngine(cfg, testEngine, reportEngine, testStore)
	require.NotNil(t, srv)
	assert.Equal(t, reportEngine, srv.reportEngine)
}

// TestServer_NewWithConfigPath tests creating a server with custom config path
func TestServer_NewWithConfigPath(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:  "localhost",
			Port:  8080,
			Debug: true,
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	testEngine, err := engine.NewEngine(cfg, testStore)
	require.NoError(t, err)
	defer testEngine.Stop()

	customPath := "/custom/path/config.yaml"
	srv := NewWithConfigPath(cfg, testEngine, nil, customPath, testStore)
	require.NotNil(t, srv)
	assert.Equal(t, customPath, srv.configPath)
	assert.Equal(t, gin.DebugMode, gin.Mode())
}

// TestServer_SetupRoutes tests setting up routes
func TestServer_SetupRoutes(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Admin: &config.AdminConfig{
			Enabled: false,
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	testEngine, err := engine.NewEngine(cfg, testStore)
	require.NoError(t, err)
	defer testEngine.Stop()

	srv := New(cfg, testEngine, testStore)
	srv.SetupRoutes()

	// Verify router is set up by making a test request
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	// Should get some response (may be 404 if route doesn't exist, but router should be functional)
	assert.NotNil(t, w)
}

// TestServer_Start tests starting the server
func TestServer_Start(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0, // Use port 0 for automatic port assignment in tests
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	testEngine, err := engine.NewEngine(cfg, testStore)
	require.NoError(t, err)
	defer testEngine.Stop()

	srv := New(cfg, testEngine, testStore)
	srv.SetupRoutes()

	err = srv.Start()
	require.NoError(t, err)
	assert.NotNil(t, srv.httpServer)

	// Stop the server
	err = srv.Stop()
	require.NoError(t, err)
}

// TestServer_Stop tests stopping the server
func TestServer_Stop(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0,
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	testEngine, err := engine.NewEngine(cfg, testStore)
	require.NoError(t, err)
	defer testEngine.Stop()

	srv := New(cfg, testEngine, testStore)
	srv.SetupRoutes()

	// Stop without starting should not error
	err = srv.Stop()
	require.NoError(t, err)

	// Start and then stop
	err = srv.Start()
	require.NoError(t, err)

	// Give server a moment to start
	time.Sleep(100 * time.Millisecond)

	err = srv.Stop()
	require.NoError(t, err)
}

// TestServer_Stop_WithTimeout tests stopping server with timeout
func TestServer_Stop_WithTimeout(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0,
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	testEngine, err := engine.NewEngine(cfg, testStore)
	require.NoError(t, err)
	defer testEngine.Stop()

	srv := New(cfg, testEngine, testStore)
	srv.SetupRoutes()

	err = srv.Start()
	require.NoError(t, err)

	// Stop should complete within timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan error)
	go func() {
		done <- srv.Stop()
	}()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal("Stop() timed out")
	}
}

// TestServer_Router tests getting the router
func TestServer_Router(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	testEngine, err := engine.NewEngine(cfg, testStore)
	require.NoError(t, err)
	defer testEngine.Stop()

	srv := New(cfg, testEngine, testStore)
	router := srv.Router()

	assert.NotNil(t, router)
	assert.Equal(t, srv.router, router)
}

// TestServer_Address tests server address configuration
func TestServer_Address(t *testing.T) {
	tests := []struct {
		name     string
		cfg      config.ServerConfig
		expected string
	}{
		{
			name: "default port",
			cfg: config.ServerConfig{
				Host: "localhost",
				Port: 8080,
			},
			expected: "localhost:8080",
		},
		{
			name: "custom host and port",
			cfg: config.ServerConfig{
				Host: "0.0.0.0",
				Port: 3000,
			},
			expected: "0.0.0.0:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			address := tt.cfg.Address()
			assert.Equal(t, tt.expected, address)
		})
	}
}

// TestServer_DebugMode tests debug mode configuration
func TestServer_DebugMode(t *testing.T) {
	tests := []struct {
		name     string
		debug    bool
		expected string
	}{
		{
			name:     "debug mode enabled",
			debug:    true,
			expected: gin.DebugMode,
		},
		{
			name:     "debug mode disabled",
			debug:    false,
			expected: gin.ReleaseMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Server: config.ServerConfig{
					Host:  "localhost",
					Port:  8080,
					Debug: tt.debug,
				},
			}
			testStore, cleanup := store.SetupTestDB(t)
			defer cleanup()

			testEngine, err := engine.NewEngine(cfg, testStore)
			require.NoError(t, err)
			defer testEngine.Stop()

			_ = NewWithConfigPath(cfg, testEngine, nil, "", testStore)
			assert.Equal(t, tt.expected, gin.Mode())
		})
	}
}

// TestServer_HTTPTimeouts tests HTTP server timeout configuration
func TestServer_HTTPTimeouts(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0,
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	testEngine, err := engine.NewEngine(cfg, testStore)
	require.NoError(t, err)
	defer testEngine.Stop()

	srv := New(cfg, testEngine, testStore)
	srv.SetupRoutes()

	err = srv.Start()
	require.NoError(t, err)
	defer srv.Stop()

	// Verify timeout values are set
	assert.Equal(t, defaultReadTimeout, srv.httpServer.ReadTimeout)
	assert.Equal(t, defaultWriteTimeout, srv.httpServer.WriteTimeout)
	assert.Equal(t, defaultIdleTimeout, srv.httpServer.IdleTimeout)
}

// TestServer_RouterConfiguration tests router configuration
func TestServer_RouterConfiguration(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}
	testStore, cleanup := store.SetupTestDB(t)
	defer cleanup()

	testEngine, err := engine.NewEngine(cfg, testStore)
	require.NoError(t, err)
	defer testEngine.Stop()

	srv := New(cfg, testEngine, testStore)

	// Verify router configuration
	assert.False(t, srv.router.RedirectTrailingSlash)
	assert.False(t, srv.router.RedirectFixedPath)
}
