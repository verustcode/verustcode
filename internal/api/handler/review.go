// Package handler provides HTTP handlers for the API.
package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/verustcode/verustcode/internal/engine"
	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/prompt"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/errors"
	"github.com/verustcode/verustcode/pkg/idgen"
	"github.com/verustcode/verustcode/pkg/logger"
)

// Pagination configuration
const (
	defaultPage     = 1
	defaultPageSize = 20
	minPageSize     = 1 // Allow small page sizes for dashboard widgets
	maxPageSize     = 100
)

// ReviewHandler handles review-related HTTP requests
type ReviewHandler struct {
	engine *engine.Engine
	store  store.Store
}

// NewReviewHandler creates a new review handler
func NewReviewHandler(e *engine.Engine, s store.Store) *ReviewHandler {
	h := &ReviewHandler{engine: e, store: s}

	// Set engine callbacks
	e.SetCallbacks(h.onReviewComplete, h.onReviewError)

	return h
}

// CreateReviewRequest represents the request body for creating a review
type CreateReviewRequest struct {
	Repository string `json:"repository" binding:"required"` // owner/repo or full URL
	Ref        string `json:"ref" binding:"required"`        // branch, tag, or commit
	Provider   string `json:"provider"`                      // github, gitlab (auto-detect if empty)
	Agent      string `json:"agent"`                         // AI agent to use
	PRNumber   int    `json:"pr_number"`                     // optional PR number
}

// CreateReview handles POST /api/v1/reviews
func (h *ReviewHandler) CreateReview(c *gin.Context) {
	var req CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	// Parse repository string to get repo URL
	repoURL := req.Repository
	var owner, repoName, provider string
	if !strings.HasPrefix(repoURL, "http://") && !strings.HasPrefix(repoURL, "https://") {
		// If not a full URL, construct it
		owner, repoName, provider = parseRepository(req.Repository)
		if owner == "" || repoName == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    errors.ErrCodeValidation,
				"message": "Invalid repository format, expected 'owner/repo' or full URL",
			})
			return
		}

		// Use provided provider or detected one
		if req.Provider != "" {
			provider = req.Provider
		}
		if provider == "" {
			provider = "github" // default
		}

		repoURL = fmt.Sprintf("https://%s.com/%s/%s", provider, owner, repoName)
	} else {
		// Extract owner and repo from full URL for PR URL construction
		owner, repoName, provider = parseRepository(req.Repository)
		if provider == "" {
			// Try to detect from URL
			if strings.Contains(repoURL, "github.com") {
				provider = "github"
			} else if strings.Contains(repoURL, "gitlab.com") {
				provider = "gitlab"
			}
		}
		if req.Provider != "" {
			provider = req.Provider
		}
		if provider == "" {
			provider = "github" // default
		}
	}

	// Build PR URL if PR number is provided
	var prURL string
	if req.PRNumber > 0 && owner != "" && repoName != "" {
		if provider == "github" {
			prURL = fmt.Sprintf("https://github.com/%s/%s/pull/%d", owner, repoName, req.PRNumber)
		} else if provider == "gitlab" {
			prURL = fmt.Sprintf("https://gitlab.com/%s/%s/-/merge_requests/%d", owner, repoName, req.PRNumber)
		}
	}

	// Create review record with unique ID
	review := &model.Review{
		ID:       idgen.NewReviewID(),
		RepoURL:  repoURL,
		Ref:      req.Ref,
		Status:   model.ReviewStatusPending,
		PRNumber: req.PRNumber,
		PRURL:    prURL,
		Source:   "api",
	}

	if err := h.store.Review().Create(review); err != nil {
		logger.Error("Failed to create review", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeDBQuery,
			"message": "Failed to create review",
		})
		return
	}

	// Ensure repository config exists
	if _, err := h.store.RepositoryConfig().EnsureConfig(repoURL); err != nil {
		logger.Warn("Failed to ensure repository config",
			zap.String("repo_url", repoURL), zap.Error(err))
		// Non-fatal, continue with review
	}

	// Submit to engine
	_, err := h.engine.Submit(review, nil)
	if err != nil {
		// Update review status to failed
		h.store.Review().UpdateStatusWithError(review.ID, model.ReviewStatusFailed, err.Error())

		appErr, _ := errors.AsAppError(err)
		status := http.StatusInternalServerError
		if appErr != nil {
			status = appErr.HTTPStatus()
		}

		c.JSON(status, gin.H{
			"code":    errors.ErrCodeReviewFailed,
			"message": err.Error(),
		})
		return
	}

	// Note: Don't update status to running here
	// The status will be updated by processTask when the task is actually dequeued
	// This ensures consistency between queue state and database state
	logger.Info("Review created and submitted to queue",
		zap.String("review_id", review.ID),
		zap.String("repo_url", repoURL),
		zap.String("status", string(review.Status)),
	)

	c.JSON(http.StatusAccepted, gin.H{
		"id":         review.ID,
		"status":     review.Status,
		"repo_url":   repoURL,
		"ref":        req.Ref,
		"created_at": review.CreatedAt,
	})
}

// GetReview handles GET /api/v1/reviews/:id
func (h *ReviewHandler) GetReview(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid review ID",
		})
		return
	}

	review, err := h.store.Review().GetByIDWithDetails(id)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    errors.ErrCodeReviewNotFound,
			"message": "Review not found",
		})
		return
	} else if err != nil {
		logger.Error("Database error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeDBQuery,
			"message": "Database error",
		})
		return
	}

	c.JSON(http.StatusOK, review)
}

// ListReviews handles GET /api/v1/reviews
func (h *ReviewHandler) ListReviews(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", strconv.Itoa(defaultPage)))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(defaultPageSize)))
	status := c.Query("status")

	if page < 1 {
		page = defaultPage
	}
	if pageSize < minPageSize || pageSize > maxPageSize {
		pageSize = defaultPageSize
	}

	offset := (page - 1) * pageSize
	reviews, total, err := h.store.Review().List(status, pageSize, offset)
	if err != nil {
		logger.Error("Database error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeDBQuery,
			"message": "Database error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      reviews,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// CancelReview handles POST /api/v1/reviews/:id/cancel
func (h *ReviewHandler) CancelReview(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid review ID",
		})
		return
	}

	rowsAffected, err := h.store.Review().UpdateStatusIfAllowed(id, model.ReviewStatusCancelled, []model.ReviewStatus{
		model.ReviewStatusPending,
		model.ReviewStatusRunning,
	})

	if err != nil {
		logger.Error("Failed to cancel review", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    errors.ErrCodeDBQuery,
			"message": "Failed to cancel review",
		})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    errors.ErrCodeReviewNotFound,
			"message": "Review not found or cannot be cancelled",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Review cancelled",
	})
}

// RetryReview handles POST /api/v1/reviews/:id/retry
func (h *ReviewHandler) RetryReview(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid review ID",
		})
		return
	}

	err := h.engine.Retry(id)
	if err != nil {
		appErr, ok := errors.AsAppError(err)
		status := http.StatusInternalServerError
		code := errors.ErrCodeInternal
		message := err.Error()

		if ok {
			status = appErr.HTTPStatus()
			code = appErr.Code
			message = appErr.Message
		}

		c.JSON(status, gin.H{
			"code":    code,
			"message": message,
		})
		return
	}

	logger.Info("Review retry initiated",
		zap.String("review_id", id),
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Review retry initiated",
	})
}

// RetryReviewRule handles POST /api/v1/reviews/:id/rules/:rule_id/retry
// This endpoint allows retrying a single failed rule within a review
// The rule retry runs asynchronously in a separate goroutine
func (h *ReviewHandler) RetryReviewRule(c *gin.Context) {
	reviewID := c.Param("id")
	ruleID := c.Param("rule_id")

	if reviewID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid review ID",
		})
		return
	}

	if ruleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    errors.ErrCodeValidation,
			"message": "Invalid rule ID",
		})
		return
	}

	err := h.engine.RetryRule(reviewID, ruleID)
	if err != nil {
		appErr, ok := errors.AsAppError(err)
		status := http.StatusInternalServerError
		code := errors.ErrCodeInternal
		message := err.Error()

		if ok {
			status = appErr.HTTPStatus()
			code = appErr.Code
			message = appErr.Message
		}

		c.JSON(status, gin.H{
			"code":    code,
			"message": message,
		})
		return
	}

	logger.Info("Rule retry initiated",
		zap.String("review_id", reviewID),
		zap.String("rule_id", ruleID),
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Rule retry initiated",
	})
}

// onReviewComplete is called when a review task completes
// Note: Results are already saved in runReviewWithTracking, so we just log completion
func (h *ReviewHandler) onReviewComplete(task *engine.Task, result *prompt.ReviewResult) {
	logger.Info("Review completed",
		zap.String("review_id", task.Review.ID),
	)
}

// onReviewError is called when a review task fails
func (h *ReviewHandler) onReviewError(task *engine.Task, err error) {
	if dbErr := h.store.Review().UpdateStatusWithErrorAndCompletedAt(task.Review.ID, model.ReviewStatusFailed, err.Error()); dbErr != nil {
		logger.Error("Failed to update review error status",
			zap.String("review_id", task.Review.ID),
			zap.Error(dbErr),
		)
	}
}

// parseRepository parses a repository string into owner, name, and provider
func parseRepository(repo string) (owner, name, provider string) {
	// Handle full URLs
	if len(repo) > 8 {
		if repo[:8] == "https://" || repo[:7] == "http://" {
			// Remove protocol
			repo = repo[8:]
			if repo[0] == '/' {
				repo = repo[7:]
			}

			// Detect provider from domain
			if len(repo) > 10 && repo[:10] == "github.com" {
				provider = "github"
				repo = repo[11:]
			} else if len(repo) > 10 && repo[:10] == "gitlab.com" {
				provider = "gitlab"
				repo = repo[11:]
			}

			// Remove .git suffix
			if len(repo) > 4 && repo[len(repo)-4:] == ".git" {
				repo = repo[:len(repo)-4]
			}
		}
	}

	// Parse owner/name
	for i, c := range repo {
		if c == '/' {
			owner = repo[:i]
			name = repo[i+1:]
			break
		}
	}

	return
}
