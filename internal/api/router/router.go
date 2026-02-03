// Package router sets up the API routes for the application.
// This is used in server mode for webhook triggers and API access.
// For CLI-only usage, the API layer is not required.
package router

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/verustcode/verustcode/consts"
	"github.com/verustcode/verustcode/internal/api/handler"
	"github.com/verustcode/verustcode/internal/api/middleware"
	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/database"
	"github.com/verustcode/verustcode/internal/engine"
	"github.com/verustcode/verustcode/internal/report"
	"github.com/verustcode/verustcode/internal/store"
)

// Setup configures all API routes
func Setup(r *gin.Engine, e *engine.Engine, cfg *config.Config, s store.Store) {
	SetupWithConfigPath(r, e, nil, cfg, config.BootstrapConfigPath, s)
}

// SetupWithReportEngine configures all API routes with report engine
func SetupWithReportEngine(r *gin.Engine, e *engine.Engine, re *report.Engine, cfg *config.Config, s store.Store) {
	SetupWithConfigPath(r, e, re, cfg, config.BootstrapConfigPath, s)
}

// SetupWithConfigPath configures all API routes with a custom config path
func SetupWithConfigPath(r *gin.Engine, e *engine.Engine, re *report.Engine, cfg *config.Config, configPath string, s store.Store) {
	// Apply global middleware
	r.Use(middleware.Recovery())
	r.Use(middleware.Logger(&middleware.LoggerConfig{
		AccessLog: cfg.Logging.AccessLog,
	}))
	r.Use(middleware.CORS(cfg.Server.CORSOrigins))
	r.Use(middleware.RequestID())
	// P2-1 Security improvement: Pass debug mode to hide sensitive errors in production
	r.Use(middleware.ErrorHandler(cfg.Server.Debug))

	// Apply OpenTelemetry tracing middleware
	r.Use(otelgin.Middleware(consts.ServiceName))

	// Health check endpoint (public)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API v1 routes
	v1 := r.Group("/api/v1")

	// ============== Public routes ==============

	// Schema routes (public - for frontend to fetch JSON schemas)
	schemaHandler := handler.NewSchemaHandler()
	v1.GET("/schemas/:name", schemaHandler.GetSchema)

	// Initialize admin handler
	adminHandler := handler.NewAdminHandler(cfg, configPath, s)

	// Webhook routes (public - requires webhook secret validation instead)
	webhookHandler := handler.NewWebhookHandler(e, s)
	webhooks := v1.Group("/webhooks")
	{
		webhooks.POST("/:provider", webhookHandler.HandleWebhook)
	}

	// ============== Auth routes ==============

	// Initialize auth handler with config path for password setup
	authHandler := handler.NewAuthHandlerWithConfigPath(cfg, configPath)

	// Auth routes (login and setup are public, me requires auth)
	auth := v1.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
		// Password setup routes (public - only available when password is not set)
		auth.GET("/setup/status", authHandler.GetSetupStatus)
		auth.POST("/setup", authHandler.SetupPassword)
	}

	// ============== API routes (protected) ==============

	// Initialize handlers
	reviewHandler := handler.NewReviewHandler(e, s)

	// Queue status endpoint - protected by JWT authentication
	v1.GET("/queue/status", middleware.JWTAuth(authHandler), func(c *gin.Context) {
		stats := e.GetQueueStats()
		c.JSON(200, gin.H{
			"total_pending": stats.TotalPending,
			"total_running": stats.TotalRunning,
			"repo_count":    stats.RepoCount,
			"repos":         stats.RepoStats,
		})
	})

	// Review routes - protected by JWT authentication
	reviews := v1.Group("/reviews")
	reviews.Use(middleware.JWTAuth(authHandler))
	{
		reviews.POST("", reviewHandler.CreateReview)
		reviews.GET("", reviewHandler.ListReviews)
		reviews.GET("/:id", reviewHandler.GetReview)
		reviews.POST("/:id/cancel", reviewHandler.CancelReview)
		reviews.POST("/:id/retry", reviewHandler.RetryReview)
		reviews.POST("/:id/rules/:rule_id/retry", reviewHandler.RetryReviewRule) // Retry single rule
	}

	// Report routes - protected by JWT authentication
	if re != nil {
		reportHandler := handler.NewReportHandler(re, cfg, s)

		// Report types (public endpoint for UI dropdown)
		v1.GET("/report-types", reportHandler.GetReportTypes)

		reports := v1.Group("/reports")
		reports.Use(middleware.JWTAuth(authHandler))
		{
			reports.POST("", reportHandler.CreateReport)
			reports.GET("", reportHandler.ListReports)
			reports.GET("/repositories", reportHandler.GetRepositories)
			reports.GET("/branches", reportHandler.GetBranches)
			reports.GET("/:id", reportHandler.GetReport)
			reports.GET("/:id/progress", reportHandler.GetReportProgress)
			reports.POST("/:id/cancel", reportHandler.CancelReport)
			reports.POST("/:id/retry", reportHandler.RetryReport)
			reports.GET("/:id/export", reportHandler.ExportReport)
		}

		// Add logs endpoints under reports (using task_logs.db)
		taskLogDB := database.GetTaskLogDB()
		if taskLogDB != nil {
			taskLogStore := store.NewTaskLogStore(taskLogDB)
			taskLogHandler := handler.NewTaskLogHandler(taskLogStore)
			reports.GET("/:id/logs", taskLogHandler.GetReportLogs)
		}
	}

	// ============== Admin routes (protected) ==============

	// Initialize rules and repository handlers
	rulesHandler := handler.NewRulesHandler()
	reportTypesHandler := handler.NewReportTypesHandler()
	repoHandler := handler.NewRepositoryHandler(s)

	// Admin routes with JWT authentication
	admin := v1.Group("/admin")
	admin.Use(middleware.JWTAuth(authHandler))
	{
		// Auth - me endpoint
		admin.GET("/me", authHandler.Me)

		// App meta endpoint (protected - requires auth)
		// Returns application name, subtitle, and version
		admin.GET("/meta", adminHandler.GetAppMeta)

		// Server status
		admin.GET("/status", adminHandler.GetStatus)

		// Stats
		admin.GET("/stats", adminHandler.GetStats)

		// Statistics - repository level analytics
		statsHandler := handler.NewStatsHandler(s)
		admin.GET("/stats/repo", statsHandler.GetRepoStats)

		// Findings - aggregated findings list across all repositories
		findingsHandler := handler.NewFindingsHandler(s)
		admin.GET("/findings", findingsHandler.ListFindings)

		// Rules management
		admin.GET("/rules", rulesHandler.ListRules)
		admin.POST("/rules", rulesHandler.CreateRuleFile)
		admin.GET("/rules/:name", rulesHandler.GetRule)
		admin.PUT("/rules/:name", rulesHandler.SaveRule)
		admin.POST("/rules/validate", rulesHandler.ValidateRule)

		// Review files list (available review config files)
		admin.GET("/review-files", rulesHandler.ListReviewFiles)

		// Report types management (report configuration files)
		admin.GET("/report-types", reportTypesHandler.ListReportTypes)
		admin.POST("/report-types", reportTypesHandler.CreateReportType)
		admin.GET("/report-types/:name", reportTypesHandler.GetReportType)
		admin.PUT("/report-types/:name", reportTypesHandler.SaveReportType)
		admin.POST("/report-types/validate", reportTypesHandler.ValidateReportType)

		// Repository review config management
		admin.GET("/repositories", repoHandler.ListRepositories)
		admin.POST("/repositories", repoHandler.CreateRepositoryConfig)
		admin.PUT("/repositories/:id", repoHandler.UpdateRepositoryConfig)
		admin.DELETE("/repositories/:id", repoHandler.DeleteRepositoryConfig)
		admin.POST("/parse-repo-url", repoHandler.ParseRepoUrl)

		// Settings management (database-backed runtime settings)
		// All components read configuration directly from database, no hot-reload needed
		settingsHandler := handler.NewSettingsHandler(s)
		admin.GET("/settings", settingsHandler.GetAllSettings)
		admin.POST("/settings/apply", settingsHandler.ApplySettings)
		admin.GET("/settings/:category", settingsHandler.GetSettingsByCategory)
		admin.PUT("/settings/:category", settingsHandler.UpdateSettingsByCategory)
		admin.POST("/settings/git/test", settingsHandler.TestGitProvider)
		admin.POST("/settings/agents/test", settingsHandler.TestAgent)
		admin.POST("/settings/notifications/test", settingsHandler.TestNotificationConfig)

		// Notification management
		notificationHandler := handler.NewNotificationHandler()
		admin.GET("/notifications/status", notificationHandler.GetNotificationStatus)
		admin.POST("/notifications/test", notificationHandler.TestNotification)
	}

	// Also add /auth/me under admin protection
	v1.GET("/auth/me", middleware.JWTAuth(authHandler), authHandler.Me)

	// Task log routes - protected by JWT authentication
	// Uses separate task_logs.db database
	// Add logs endpoints under reviews
	taskLogDB := database.GetTaskLogDB()
	if taskLogDB != nil {
		taskLogStore := store.NewTaskLogStore(taskLogDB)
		taskLogHandler := handler.NewTaskLogHandler(taskLogStore)
		reviews.GET("/:id/logs", taskLogHandler.GetReviewLogs)
	}
}
