// Package handler provides HTTP handlers for the API.
package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/dsl"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/errors"
	"github.com/verustcode/verustcode/pkg/logger"
)

// RepositoryHandler handles repository configuration related HTTP requests
type RepositoryHandler struct {
	store store.Store
}

// NewRepositoryHandler creates a new repository handler
func NewRepositoryHandler(s store.Store) *RepositoryHandler {
	return &RepositoryHandler{
		store: s,
	}
}

// RepositoryConfigItem represents a repository with its review config
type RepositoryConfigItem struct {
	ID           uint    `json:"id"`
	RepoURL      string  `json:"repo_url"`
	ReviewFile   string  `json:"review_file"`
	Description  string  `json:"description,omitempty"`
	ReviewCount  int64   `json:"review_count"`             // Number of reviews for this repo
	LastReviewAt *string `json:"last_review_at,omitempty"` // Last review timestamp
	CreatedAt    string  `json:"created_at,omitempty"`
	UpdatedAt    string  `json:"updated_at,omitempty"`
}

// ListRepositoriesResponse represents the response for listing repositories
type ListRepositoriesResponse struct {
	Data  []RepositoryConfigItem `json:"data"`
	Total int64                  `json:"total"`
	Page  int                    `json:"page"`
	Size  int                    `json:"page_size"`
}

// CreateRepositoryConfigRequest represents the request to create a repository config
type CreateRepositoryConfigRequest struct {
	RepoURL     string `json:"repo_url" binding:"required"`
	ReviewFile  string `json:"review_file"`
	Description string `json:"description"`
}

// UpdateRepositoryConfigRequest represents the request to update a repository config
type UpdateRepositoryConfigRequest struct {
	ReviewFile  string `json:"review_file"`
	Description string `json:"description"`
}

// ParseRepoUrlRequest represents the request body for parsing repository URL
type ParseRepoUrlRequest struct {
	URL string `json:"url" binding:"required"`
}

// ParseRepoUrlResponse represents the response for parsing repository URL
type ParseRepoUrlResponse struct {
	Provider string `json:"provider"`
	Owner    string `json:"owner"`
	Repo     string `json:"repo"`
}

// ListRepositories handles GET /api/v1/admin/repositories
// Returns all repositories with their review configs from repository_review_configs table.
// Repository configs are auto-created when reviews or reports are created, so all repos with activity
// will have a valid config entry with a non-zero ID.
// Supports sorting by: repo_url (default), review_count, last_review_at
// Query params: page, page_size, search, sort_by, sort_order (asc/desc)
func (h *RepositoryHandler) ListRepositories(c *gin.Context) {
	// Pagination parameters
	page := 1
	pageSize := 20
	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if s := c.Query("page_size"); s != "" {
		fmt.Sscanf(s, "%d", &pageSize)
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Filter parameters
	search := c.Query("search")

	// Sorting parameters
	sortBy := c.Query("sort_by")
	sortOrder := c.Query("sort_order")

	// Use store to get repositories with stats
	results, total, err := h.store.RepositoryConfig().ListWithStats(search, sortBy, sortOrder, page, pageSize)
	if err != nil {
		logger.Error("Failed to list repositories", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeDBQuery,
			"message": "Failed to list repositories",
		})
		return
	}

	// Build response
	items := make([]RepositoryConfigItem, 0, len(results))
	for _, r := range results {
		var lastReviewAtStr *string
		if r.LastReviewAt.Valid {
			formatted := r.LastReviewAt.Time.Format(time.RFC3339)
			lastReviewAtStr = &formatted
		}

		items = append(items, RepositoryConfigItem{
			ID:           r.ID,
			RepoURL:      r.RepoURL,
			ReviewFile:   r.ReviewFile,
			Description:  r.Description,
			ReviewCount:  r.ReviewCount,
			LastReviewAt: lastReviewAtStr,
			CreatedAt:    r.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    r.UpdatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, ListRepositoriesResponse{
		Data:  items,
		Total: total,
		Page:  page,
		Size:  pageSize,
	})
}

// CreateRepositoryConfig handles POST /api/v1/admin/repositories
func (h *RepositoryHandler) CreateRepositoryConfig(c *gin.Context) {
	var req CreateRepositoryConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate review file exists if specified
	if req.ReviewFile != "" && !dsl.ReviewFileExists(req.ReviewFile) {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Review file does not exist: " + req.ReviewFile,
		})
		return
	}

	// Check if config already exists
	existing, _ := h.store.RepositoryConfig().GetByRepoURL(req.RepoURL)
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{
			"code":    errors.ErrCodeConflict,
			"message": "Repository config already exists",
		})
		return
	}

	// Create new config
	cfg := &model.RepositoryReviewConfig{
		RepoURL:     req.RepoURL,
		ReviewFile:  req.ReviewFile,
		Description: req.Description,
	}

	if err := h.store.RepositoryConfig().Create(cfg); err != nil {
		logger.Error("Failed to create repository config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeDBQuery,
			"message": "Failed to create repository config",
		})
		return
	}

	logger.Info("Created repository config",
		zap.String("repo_url", req.RepoURL),
		zap.String("review_file", req.ReviewFile),
	)

	c.JSON(http.StatusCreated, gin.H{
		"id":      cfg.ID,
		"message": "Repository config created successfully",
	})
}

// UpdateRepositoryConfig handles PUT /api/v1/admin/repositories/:id
func (h *RepositoryHandler) UpdateRepositoryConfig(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid repository config ID",
		})
		return
	}

	var req UpdateRepositoryConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate review file exists if specified
	if req.ReviewFile != "" && !dsl.ReviewFileExists(req.ReviewFile) {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Review file does not exist: " + req.ReviewFile,
		})
		return
	}

	// Find existing config
	cfg, err := h.store.RepositoryConfig().GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    errors.ErrCodeNotFound,
			"message": "Repository config not found",
		})
		return
	}

	// Update fields
	cfg.ReviewFile = req.ReviewFile
	cfg.Description = req.Description

	if err := h.store.RepositoryConfig().Save(cfg); err != nil {
		logger.Error("Failed to update repository config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeDBQuery,
			"message": "Failed to update repository config",
		})
		return
	}

	logger.Info("Updated repository config",
		zap.Uint("id", id),
		zap.String("review_file", req.ReviewFile),
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Repository config updated successfully",
	})
}

// DeleteRepositoryConfig handles DELETE /api/v1/admin/repositories/:id
func (h *RepositoryHandler) DeleteRepositoryConfig(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid repository config ID",
		})
		return
	}

	// Find existing config (for logging purposes)
	cfg, err := h.store.RepositoryConfig().GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    errors.ErrCodeNotFound,
			"message": "Repository config not found",
		})
		return
	}

	// Delete (soft delete via GORM)
	if err := h.store.RepositoryConfig().Delete(id); err != nil {
		logger.Error("Failed to delete repository config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeDBQuery,
			"message": "Failed to delete repository config",
		})
		return
	}

	logger.Info("Deleted repository config",
		zap.Uint("id", id),
		zap.String("repo_url", cfg.RepoURL),
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Repository config deleted successfully",
	})
}

// ParseRepoUrl handles POST /api/v1/admin/parse-repo-url
// Parses a repository URL and returns provider, owner, and repo
func (h *RepositoryHandler) ParseRepoUrl(c *gin.Context) {
	var req ParseRepoUrlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid request body: url is required",
		})
		return
	}

	// Parse the repository URL using the same logic as review handler
	owner, repo, provider := parseRepoUrl(req.URL)

	if owner == "" || repo == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid repository URL format, expected 'owner/repo' or full URL like 'github.com/owner/repo'",
		})
		return
	}

	// Default provider to github if not detected
	if provider == "" {
		provider = "github"
	}

	c.JSON(http.StatusOK, ParseRepoUrlResponse{
		Provider: provider,
		Owner:    owner,
		Repo:     repo,
	})
}

// parseRepoUrl parses a repository URL string and extracts owner, repo, and provider
// Supports formats:
// - https://github.com/owner/repo
// - https://gitlab.com/owner/repo
// - github.com/owner/repo
// - gitlab.com/owner/repo
// - owner/repo (defaults to github)
func parseRepoUrl(repoUrl string) (owner, repo, provider string) {
	url := strings.TrimSpace(repoUrl)

	// Remove protocol prefix
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")

	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Remove trailing slash
	url = strings.TrimSuffix(url, "/")

	// Detect provider from domain
	if strings.HasPrefix(url, "github.com/") {
		provider = "github"
		url = strings.TrimPrefix(url, "github.com/")
	} else if strings.HasPrefix(url, "gitlab.com/") {
		provider = "gitlab"
		url = strings.TrimPrefix(url, "gitlab.com/")
	} else if strings.HasPrefix(url, "gitea.com/") {
		provider = "gitea"
		url = strings.TrimPrefix(url, "gitea.com/")
	}

	// Parse owner/repo
	parts := strings.SplitN(url, "/", 2)
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		owner = parts[0]
		repo = parts[1]
		// Remove any remaining path after repo name (e.g., /tree/main)
		if idx := strings.Index(repo, "/"); idx != -1 {
			repo = repo[:idx]
		}
	}

	return
}
