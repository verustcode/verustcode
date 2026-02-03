// Package server provides the HTTP server for the application.
// It handles server lifecycle, API routes, and graceful shutdown.
package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/admin"
	"github.com/verustcode/verustcode/internal/api/router"
	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/engine"
	"github.com/verustcode/verustcode/internal/report"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/logger"
)

// HTTP server timeout configuration
const (
	defaultReadTimeout     = 30 * time.Second
	defaultWriteTimeout    = 30 * time.Second
	defaultIdleTimeout     = 60 * time.Second
	defaultShutdownTimeout = 30 * time.Second
	defaultStopTimeout     = 5 * time.Second
)

// Server represents the HTTP server
type Server struct {
	cfg          *config.Config
	configPath   string
	httpServer   *http.Server
	engine       *engine.Engine
	reportEngine *report.Engine
	router       *gin.Engine
	store        store.Store
}

// New creates a new server instance
func New(cfg *config.Config, e *engine.Engine, s store.Store) *Server {
	return NewWithConfigPath(cfg, e, nil, config.BootstrapConfigPath, s)
}

// NewWithReportEngine creates a new server instance with both engines
func NewWithReportEngine(cfg *config.Config, e *engine.Engine, re *report.Engine, s store.Store) *Server {
	return NewWithConfigPath(cfg, e, re, config.BootstrapConfigPath, s)
}

// NewWithConfigPath creates a new server instance with a custom config path
func NewWithConfigPath(cfg *config.Config, e *engine.Engine, re *report.Engine, configPath string, s store.Store) *Server {
	// Set Gin mode based on debug flag
	if cfg.Server.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create Gin router
	r := gin.New()

	// Disable automatic trailing slash redirect to avoid redirect loops with SPA routing
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

	return &Server{
		cfg:          cfg,
		configPath:   configPath,
		engine:       e,
		reportEngine: re,
		router:       r,
		store:        s,
	}
}

// SetupRoutes configures all routes including admin console
func (s *Server) SetupRoutes() {
	// Setup API routes with both engines
	router.SetupWithConfigPath(s.router, s.engine, s.reportEngine, s.cfg, s.configPath, s.store)

	// Setup admin console if enabled and assets are embedded
	if s.cfg.Admin != nil && s.cfg.Admin.Enabled && admin.HasEmbeddedAssets() {
		if err := admin.RegisterRoutes(s.router); err != nil {
			logger.Warn("Failed to register admin console routes", zap.Error(err))
		} else {
			logger.Info("Admin console enabled at /admin")
		}
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:         s.cfg.Server.Address(),
		Handler:      s.router,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		IdleTimeout:  defaultIdleTimeout,
	}

	logger.Info("Starting HTTP server",
		zap.String("address", s.cfg.Server.Address()),
		zap.Bool("debug", s.cfg.Server.Debug),
	)

	// Start server in goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	return nil
}

// WaitForShutdown waits for shutdown signal and gracefully stops the server
// First signal triggers graceful shutdown, second signal forces immediate exit
func (s *Server) WaitForShutdown() {
	// Create channel for shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Wait for first signal
	sig := <-quit
	logger.Info("Received shutdown signal, starting graceful shutdown (press Ctrl+C again to force exit)",
		zap.String("signal", sig.String()))

	// Start a goroutine to listen for second signal (force exit)
	go func() {
		sig := <-quit
		logger.Warn("Received second shutdown signal, forcing exit",
			zap.String("signal", sig.String()))
		os.Exit(1)
	}()

	// Create context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server stopped")
}

// Stop stops the server immediately
func (s *Server) Stop() error {
	if s.httpServer == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultStopTimeout)
	defer cancel()

	return s.httpServer.Shutdown(ctx)
}

// Router returns the underlying Gin router
func (s *Server) Router() *gin.Engine {
	return s.router
}
