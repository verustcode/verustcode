// Package main is the entry point for the VerustCode application.
// VerustCode is a DSL-driven AI-powered code review webhook service.
package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/verustcode/verustcode/consts"
	"github.com/verustcode/verustcode/internal/check"
	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/database"
	"github.com/verustcode/verustcode/internal/engine"
	"github.com/verustcode/verustcode/internal/notification"
	"github.com/verustcode/verustcode/internal/report"
	"github.com/verustcode/verustcode/internal/server"
	"github.com/verustcode/verustcode/internal/shared"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/errors"
	"github.com/verustcode/verustcode/pkg/idgen"
	"github.com/verustcode/verustcode/pkg/logger"
	"github.com/verustcode/verustcode/pkg/telemetry"

	// Import agent implementations to register them
	// All agents are registered through the agents package
	_ "github.com/verustcode/verustcode/internal/agent/agents"

	// Import git provider implementations to register them
	// All providers are registered through the providers package
	_ "github.com/verustcode/verustcode/internal/git/providers"
)

// Build information - set via ldflags during build
// These variables are linked to consts package for global access
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// init synchronizes build info to consts package for global access
func init() {
	consts.Version = Version
	consts.BuildTime = BuildTime
	consts.GitCommit = GitCommit
}

// bootstrapPath holds the path to the bootstrap configuration file
var bootstrapPath string

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "verustcode",
	Short: "VerustCode - DSL-Driven AI-Powered Code Review Webhook Service",
	Long: `VerustCode is a DSL-driven code review webhook service that uses pluggable 
AI CLI backends to perform deep, multi-dimensional code analysis.`,
}

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the VerustCode server",
	Long: `Start the HTTP server to handle API requests and webhook triggers.

On first run, use --check flag to interactively set up your environment:
  verustcode serve --check

This will guide you through:
  - Creating configuration files from templates
  - Validating configuration formats

After initial setup, simply run:
  verustcode serve`,
	Run: runServe,
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("VerustCode %s\n", Version)
		fmt.Printf("  Build Time: %s\n", BuildTime)
		fmt.Printf("  Git Commit: %s\n", GitCommit)
	},
}

func init() {
	// Disable auto-generated completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Global flags
	rootCmd.PersistentFlags().StringVar(&bootstrapPath, "bootstrap", "", "bootstrap config file path (default: config/bootstrap.yaml)")

	// Add commands
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(versionCmd)

	// Serve command flags
	serveCmd.Flags().String("host", "", "server host (overrides config)")
	serveCmd.Flags().Int("port", 0, "server port (overrides config)")
	serveCmd.Flags().Bool("debug", false, "enable debug mode")
	serveCmd.Flags().Bool("check", false, "run interactive environment check before starting server")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// runServe starts the VerustCode server
func runServe(cmd *cobra.Command, args []string) {
	// Check if interactive check is enabled via --check flag
	interactiveCheck, _ := cmd.Flags().GetBool("check")

	if interactiveCheck {
		// Run full interactive environment check
		checker := check.NewChecker()
		if err := checker.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Environment check failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\n✓ Environment check completed successfully")
	} else {
		// Run non-interactive basic check
		checker := check.NewChecker()
		result := checker.RunNonInteractive()

		if !result.Success {
			// Print errors and exit
			check.PrintCheckResult(result)
			os.Exit(1)
		}

		// Print warnings if any (but don't block startup)
		if len(result.Warnings) > 0 {
			for _, warn := range result.Warnings {
				fmt.Fprintf(os.Stderr, "[WARNING] %s\n", warn)
			}
			fmt.Fprintln(os.Stderr)
		}
	}

	// Record server start time
	consts.SetStartedAt(time.Now())

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Override config with command line flags
	if host, _ := cmd.Flags().GetString("host"); host != "" {
		cfg.Server.Host = host
	}
	if port, _ := cmd.Flags().GetInt("port"); port != 0 {
		cfg.Server.Port = port
	}
	if debug, _ := cmd.Flags().GetBool("debug"); debug {
		cfg.Server.Debug = true
		cfg.Logging.Level = "debug"
		cfg.Logging.Format = "text"
	}

	// Auto-generate JWT secret if empty and save to config file
	if cfg.Admin != nil && cfg.Admin.Enabled && strings.TrimSpace(cfg.Admin.JWTSecret) == "" {
		newSecret := idgen.NewSecureSecret(32)
		cfg.Admin.JWTSecret = newSecret

		// Determine config path
		configPath := bootstrapPath
		if configPath == "" {
			configPath = config.BootstrapConfigPath
		}

		// Save to config file
		if err := config.UpdateJWTSecretInConfig(configPath, newSecret); err != nil {
			fmt.Fprintf(os.Stderr, "[WARNING] Failed to save JWT secret to config file: %v\n", err)
			fmt.Fprintf(os.Stderr, "Using auto-generated JWT secret for this session only.\n")
			fmt.Fprintf(os.Stderr, "Please manually add jwt_secret to your config file to persist across restarts.\n\n")
		} else {
			fmt.Fprintf(os.Stderr, "[INFO] JWT secret was empty, auto-generated and saved to config file.\n\n")
		}
	}

	// Validate admin configuration
	// Note: password_hash is NOT validated - can be set via Web UI
	if validationErr := config.ValidateAdminConfig(cfg.Admin); validationErr != nil {
		fmt.Fprintf(os.Stderr, "\n[ERROR] Admin configuration validation failed\n")
		fmt.Fprintf(os.Stderr, "Error Code: %s\n", validationErr.Code)
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", validationErr)

		// Print context-specific configuration hints based on error type
		switch validationErr.Code {
		case errors.ErrCodeJWTSecretInvalid:
			fmt.Fprintf(os.Stderr, "JWT secret is invalid or too short.\n")
			fmt.Fprintf(os.Stderr, "Please configure JWT secret in your config file:\n")
			fmt.Fprintf(os.Stderr, "  admin:\n")
			fmt.Fprintf(os.Stderr, "    jwt_secret: \"%s\"\n\n", idgen.NewSecureSecret(32))
		case errors.ErrCodeAdminCredentialsEmpty:
			fmt.Fprintf(os.Stderr, "Please configure admin username in your config file:\n")
			fmt.Fprintf(os.Stderr, "  admin:\n")
			fmt.Fprintf(os.Stderr, "    username: \"admin\"\n\n")
		default:
			fmt.Fprintf(os.Stderr, "Please check admin configuration in your config file.\n\n")
		}

		os.Exit(errors.ExitCodeConfigValidation)
	}

	// Log warning if password is not set, but allow server to start
	if cfg.Admin != nil && cfg.Admin.Enabled && strings.TrimSpace(cfg.Admin.PasswordHash) == "" {
		fmt.Fprintf(os.Stderr, "[WARNING] Admin password not set\n")
		fmt.Fprintf(os.Stderr, "Please set password via web UI at http://%s/admin after starting the server.\n\n", cfg.Server.Address())
	}

	// Initialize logger
	if err := logger.Init(cfg.Logging); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Initialize task log database (separate from main database)
	var taskLogCleanupService *store.TaskLogCleanupService
	if err := database.InitTaskLogDB(); err != nil {
		fmt.Fprintf(os.Stderr, "[WARNING] Failed to initialize task log database: %v\n", err)
		// Continue without task logging - not fatal
	} else {
		defer database.CloseTaskLogDB()

		// Create TaskLogStore and set up the logger hook for dual-write mode
		taskLogStore := store.NewTaskLogStore(database.GetTaskLogDB())
		logger.SetTaskLogHook(taskLogStore)
		defer logger.CloseTaskLogHook()

		// Start task log cleanup service (runs daily at 2 AM)
		taskLogCleanupService = store.NewTaskLogCleanupService(taskLogStore, store.DefaultTaskLogRetentionDays)
		if err := taskLogCleanupService.Start(); err != nil {
			logger.Warn("Failed to start task log cleanup service", zap.Error(err))
			// Continue without cleanup - not fatal
		} else {
			defer taskLogCleanupService.Stop()
		}
	}

	logger.Info("Starting VerustCode",
		zap.String("version", Version),
	)

	// Initialize telemetry (OpenTelemetry traces and metrics)
	tel, err := telemetry.New(cfg.Telemetry)
	if err != nil {
		logger.Fatal("Failed to initialize telemetry", zap.Error(err))
	}
	defer func() {
		// Graceful shutdown with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := tel.Shutdown(ctx); err != nil {
			logger.Error("Failed to shutdown telemetry", zap.Error(err))
		}
	}()

	// Database is already initialized in loadConfig
	// Just ensure cleanup on exit
	defer database.Close()

	// Create store instance for dependency injection
	dataStore := store.NewStore(database.Get())

	// Initialize notification manager - now uses store for real-time DB access
	notification.Init(dataStore)

	// Create review engine
	reviewEngine, err := engine.NewEngine(cfg, dataStore)
	if err != nil {
		logger.Fatal("Failed to create review engine", zap.Error(err))
	}

	// Start review engine
	reviewEngine.Start()
	defer reviewEngine.Stop()

	// Create report engine (independent initialization)
	// Each engine initializes its own providers and agents
	reportProviders, _ := shared.InitProviders(cfg)
	reportAgents := shared.InitAgents(cfg, dataStore)
	reportEngine := report.NewEngine(cfg, reportProviders, reportAgents, dataStore)

	// Start report engine
	reportEngine.Start()
	defer reportEngine.Stop()

	// Create and configure server
	srv := server.NewWithReportEngine(cfg, reviewEngine, reportEngine, dataStore)
	srv.SetupRoutes()

	// Start server
	if err := srv.Start(); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}

	logger.Info("VerustCode server is running",
		zap.String("address", cfg.Server.Address()),
	)

	// Log access URLs for user convenience
	port := cfg.Server.Port
	logger.Info(fmt.Sprintf("  Local:   http://localhost:%d/admin", port))
	if lanIP := getLocalIP(); lanIP != "" {
		logger.Info(fmt.Sprintf("  Network: http://%s:%d/admin", lanIP, port))
	}

	// Wait for shutdown
	srv.WaitForShutdown()

	logger.Info("VerustCode stopped")
}

// loadConfig loads configuration from bootstrap file and database settings
func loadConfig() (*config.Config, error) {
	// Use default bootstrap path if not specified
	if bootstrapPath == "" {
		bootstrapPath = config.BootstrapConfigPath
	}

	// Check if bootstrap.yaml exists
	if !config.BootstrapExists(bootstrapPath) {
		return nil, fmt.Errorf("bootstrap configuration not found: %s\nRun 'verustcode serve --check' to create it", bootstrapPath)
	}

	// Load bootstrap configuration
	bootstrapCfg, err := config.LoadBootstrap(bootstrapPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load bootstrap config: %w", err)
	}

	// Initialize database to load settings
	// Use database path from bootstrap config
	dbPath := bootstrapCfg.Database.Path
	if dbPath == "" {
		dbPath = database.DefaultDBPath
	}

	// Initialize database with custom path
	if err := database.InitWithPath(dbPath); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create store instance for database operations
	dataStore := store.NewStore(database.Get())

	// Load runtime config (bootstrap + database settings)
	runtimeCfg, err := config.LoadRuntimeConfig(bootstrapPath, dataStore)
	if err != nil {
		return nil, fmt.Errorf("failed to load runtime config: %w", err)
	}

	// Convert to legacy Config format for compatibility
	return runtimeCfg.ToConfig(), nil
}

// getLocalIP returns the first non-loopback IPv4 address
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
