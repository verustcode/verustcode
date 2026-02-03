// Package handler provides HTTP handlers for the API.
package handler

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/verustcode/verustcode/consts"
	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/logger"
)

// AdminHandler handles admin-related HTTP requests
type AdminHandler struct {
	config    *config.Config
	configDir string
	store     store.Store
}

// NewAdminHandler creates a new admin handler
// Note: configPath parameter is kept for backward compatibility but no longer used
// since bootstrap config editing via web UI has been removed
func NewAdminHandler(cfg *config.Config, configPath string, s store.Store) *AdminHandler {
	return &AdminHandler{
		config:    cfg,
		configDir: "config",
		store:     s,
	}
}

// StatsResponse represents the dashboard statistics response
type StatsResponse struct {
	TodayReviews    int64   `json:"today_reviews"`
	RunningReviews  int64   `json:"running_reviews"`
	TotalReviews    int64   `json:"total_reviews"`
	SuccessRate     float64 `json:"success_rate"`
	AvgDuration     int64   `json:"avg_duration"`
	PendingCount    int64   `json:"pending_count"`
	CompletedToday  int64   `json:"completed_today"`
	FailedToday     int64   `json:"failed_today"`
	RepositoryCount int64   `json:"repository_count"` // Total number of repositories
	TotalReports    int64   `json:"total_reports"`    // Total number of reports
}

// GetStats handles GET /api/v1/admin/stats
func (h *AdminHandler) GetStats(c *gin.Context) {
	reviewStore := h.store.Review()
	repoConfigStore := h.store.RepositoryConfig()
	reportStore := h.store.Report()

	// Get today's date range
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekAgo := now.AddDate(0, 0, -7)

	var stats StatsResponse

	// Today's reviews count
	stats.TodayReviews, _ = reviewStore.CountCreatedAfter(todayStart)

	// Running reviews count
	stats.RunningReviews, _ = reviewStore.CountByStatusOnly(model.ReviewStatusRunning)

	// Total reviews count (all reviews)
	stats.TotalReviews, _ = reviewStore.CountAll()

	// Pending count
	stats.PendingCount, _ = reviewStore.CountByStatusOnly(model.ReviewStatusPending)

	// Completed today
	stats.CompletedToday, _ = reviewStore.CountByStatusAndCompletedAfter(model.ReviewStatusCompleted, todayStart)

	// Failed today
	stats.FailedToday, _ = reviewStore.CountByStatusAndCompletedAfter(model.ReviewStatusFailed, todayStart)

	// Calculate success rate (last 7 days)
	total, _ := reviewStore.CountCompletedOrFailedAfter(weekAgo)
	completed, _ := reviewStore.CountCompletedAfter(weekAgo)

	if total > 0 {
		stats.SuccessRate = float64(completed) / float64(total)
	}

	// Average duration (last 7 days, completed only)
	avgDuration, _ := reviewStore.GetAverageDurationAfter(weekAgo)
	stats.AvgDuration = int64(avgDuration)

	// Repository count (from RepositoryReviewConfig table)
	stats.RepositoryCount, _ = repoConfigStore.CountAll()

	// Total reports count
	stats.TotalReports, _ = reportStore.CountAll()

	c.JSON(http.StatusOK, stats)
}

// ServerStatusResponse represents the server status response
type ServerStatusResponse struct {
	Version     string `json:"version"`      // Application version
	BuildTime   string `json:"build_time"`   // Build timestamp
	GitCommit   string `json:"git_commit"`   // Git commit hash
	Uptime      int64  `json:"uptime"`       // Uptime in seconds
	StartedAt   string `json:"started_at"`   // Server start time in RFC3339 format
	GoVersion   string `json:"go_version"`   // Go runtime version
	MemoryUsage int64  `json:"memory_usage"` // Memory usage in bytes (heap alloc)
}

// GetStatus handles GET /api/v1/admin/status
// Returns server status information including version and uptime
func (h *AdminHandler) GetStatus(c *gin.Context) {
	startedAt := consts.GetStartedAt()
	uptime := consts.GetUptime()

	// Get memory statistics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	status := ServerStatusResponse{
		Version:     consts.Version,
		BuildTime:   consts.BuildTime,
		GitCommit:   consts.GitCommit,
		Uptime:      int64(uptime.Seconds()),
		StartedAt:   startedAt.Format(time.RFC3339),
		GoVersion:   runtime.Version(),
		MemoryUsage: int64(memStats.Alloc), // Current heap allocation
	}

	c.JSON(http.StatusOK, status)
}

// AppMetaResponse represents the public application metadata response
type AppMetaResponse struct {
	Name     string `json:"name"`     // Application name (always "VerustCode")
	Subtitle string `json:"subtitle"` // Application subtitle from config
	Version  string `json:"version"`  // Application version
}

// GetAppMeta handles GET /api/v1/meta
// Returns public application metadata (no auth required)
func (h *AdminHandler) GetAppMeta(c *gin.Context) {
	// Get subtitle from database
	var subtitle string
	if h.store != nil {
		settingsService := config.NewSettingsService(h.store)
		if setting, err := settingsService.Get(string(model.SettingCategoryApp), "subtitle"); err == nil && setting != nil {
			// Remove quotes from JSON string value
			subtitle = strings.Trim(setting.Value, "\"")
		}
	}

	meta := AppMetaResponse{
		Name:     "VerustCode",
		Subtitle: subtitle,
		Version:  consts.Version,
	}

	c.JSON(http.StatusOK, meta)
}

// updatePasswordHashInConfig updates the password_hash field in the config file
// This function is used by the setup password flow to persist the password
// It uses YAML parsing to safely update only the password_hash field while preserving all other fields
func updatePasswordHashInConfig(configPath, passwordHash string) error {
	// Read current config file
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Backup current config before making changes
	backupPath := configPath + ".backup"
	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		logger.Warn("Failed to create backup", zap.Error(err))
		// Continue anyway, backup is optional
	}

	// Parse YAML into a generic map to preserve all fields
	var cfg map[string]interface{}
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Get or create admin section
	adminSection, ok := cfg["admin"].(map[string]interface{})
	if !ok {
		// Admin section doesn't exist or is not a map, create it
		adminSection = make(map[string]interface{})
		cfg["admin"] = adminSection
	}

	// Update only the password_hash field, preserving all other fields
	adminSection["password_hash"] = passwordHash

	// If jwt_secret is empty or doesn't exist, generate a new one
	if _, exists := adminSection["jwt_secret"]; !exists {
		adminSection["jwt_secret"] = generateRandomString(32)
		logger.Info("Generated new jwt_secret for admin")
	} else if secret, ok := adminSection["jwt_secret"].(string); ok && secret == "" {
		adminSection["jwt_secret"] = generateRandomString(32)
		logger.Info("Generated new jwt_secret for admin (was empty)")
	}

	// Ensure enabled is set
	if _, exists := adminSection["enabled"]; !exists {
		adminSection["enabled"] = true
	}

	// Ensure username is set
	if _, exists := adminSection["username"]; !exists {
		adminSection["username"] = "admin"
	}

	// Ensure expiry_hours is set
	if _, exists := adminSection["expiry_hours"]; !exists {
		adminSection["expiry_hours"] = 24
	}

	// Marshal back to YAML
	newContent, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write updated config
	if err := os.WriteFile(configPath, newContent, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
